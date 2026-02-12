package eventgoround

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// EventLoop manages the event scheduling and execution
type EventLoop struct {
	storage      *eventStorage
	eventChan    chan Event
	stopChan     chan struct{}
	pauseChan    chan bool
	isCatchingUp bool
	catchUpMu    sync.RWMutex
	isPaused     bool
	pauseMu      sync.RWMutex
	registry     IEventRegistry
	tickInterval time.Duration
	logger       *slog.Logger
	logWriter    *RotatingFileWriter
	includeInfo  bool
}

// NewEventLoop creates a new event loop with the specified tick interval
// logConfig is optional - pass nil to disable logging
func NewEventLoop(tickInterval time.Duration, registry IEventRegistry, logConfig *LogConfig) *EventLoop {
	el := &EventLoop{
		storage:      newEventStorage(),
		eventChan:    make(chan Event, 2000), // Buffered channel for better performance
		stopChan:     make(chan struct{}),
		pauseChan:    make(chan bool),
		isCatchingUp: false,
		isPaused:     false,
		registry:     registry,
		tickInterval: tickInterval,
	}

	// Initialize logger if config is provided
	if logConfig != nil && logConfig.Enabled {
		if writer, err := NewRotatingFileWriter(logConfig.FilePath, DefaultMaxBytes); err == nil {
			el.logWriter = writer
			el.logger = slog.New(slog.NewJSONHandler(writer, nil))
			el.includeInfo = logConfig.IncludeInfo
		}
	}

	return el
}

// Start begins the event loop processing
func (el *EventLoop) Start() {
	el.logInfo("event loop started", "tickInterval", el.tickInterval)
	go el.run()
}

// Stop gracefully stops the event loop
func (el *EventLoop) Stop() {
	el.logInfo("event loop stopping")
	close(el.stopChan)
	if el.logWriter != nil {
		el.logWriter.Close()
	}
}

// ScheduleEvent schedules an event to be executed at the specified timestamp
// This will block during catch-up mode until all past events are processed
// Events cannot be scheduled when the loop is paused
func (el *EventLoop) ScheduleEvent(timestamp int64, duration int64, handlername string, payload any) error {
	if el.IsPaused() {
		el.logError("event scheduling failed - loop is paused", "handler", handlername, "timestamp", timestamp)
		return fmt.Errorf("event loop is paused")
	}

	if el.IsCatchingUp() {
		el.logError("event scheduling failed - currently catching up", "handler", handlername, "timestamp", timestamp)
		return fmt.Errorf("currently catching up with past events")
	}

	handler, err := el.registry.GetHandler(handlername)

	if err != nil {
		el.logError("event scheduling failed - handler not found", "handler", handlername, "timestamp", timestamp)
		return fmt.Errorf("handler '%s' not found", handlername)
	}

	event := Event{
		Timestamp: timestamp, // Timestamp in seconds
		Duration:  duration,  // Duration in seconds
		handler:   handler,
		Handler:   handlername,
		Payload:   payload,
	}
	el.eventChan <- event
	el.logInfo("event scheduled", "handler", handlername, "timestamp", timestamp, "duration", duration)
	return nil
}

// IsCatchingUp returns whether the loop is currently in catch-up mode
func (el *EventLoop) IsCatchingUp() bool {
	el.catchUpMu.RLock()
	defer el.catchUpMu.RUnlock()
	return el.isCatchingUp
}

// IsPaused returns whether the loop is currently paused
func (el *EventLoop) IsPaused() bool {
	el.pauseMu.RLock()
	defer el.pauseMu.RUnlock()
	return el.isPaused
}

// Pause pauses the event loop, preventing event scheduling and processing
func (el *EventLoop) Pause() {
	el.pauseMu.Lock()
	if !el.isPaused {
		el.setCatchingUp(true)
		el.isPaused = true
		el.pauseMu.Unlock()
		el.logInfo("event loop paused")
		el.pauseChan <- true
	} else {
		el.pauseMu.Unlock()
	}
}

// Unpause resumes the event loop, allowing event scheduling and processing
func (el *EventLoop) Unpause() {
	el.pauseMu.Lock()
	if el.isPaused {
		el.isPaused = false
		el.pauseMu.Unlock()
		el.logInfo("event loop unpaused")
		el.pauseChan <- false
	} else {
		el.pauseMu.Unlock()
	}
}

// setCatchingUp sets the catch-up mode state
func (el *EventLoop) setCatchingUp(state bool) {
	el.catchUpMu.Lock()
	defer el.catchUpMu.Unlock()
	el.isCatchingUp = state
}

// run is the main event loop
func (el *EventLoop) run() {
	ticker := time.NewTicker(el.tickInterval)
	defer ticker.Stop()
	paused := false

	for {
		select {
		case <-el.stopChan:
			return

		case pauseState := <-el.pauseChan:
			paused = pauseState

		case <-ticker.C:
			if !paused {
				el.processTick()
			}

		case event := <-el.eventChan:
			el.storage.add(event)
		}
	}
}

// processTick handles the logic for each tick of the event loop
func (el *EventLoop) processTick() {
	currentTime := time.Now().Unix()

	// Check if we need to enter catch-up mode
	if el.storage.hasPastEvents(currentTime) {
		el.setCatchingUp(true)
		el.processCatchUp(currentTime)
		el.setCatchingUp(false)
	}

	// Process current time events
	el.processTimestamp(currentTime)
}

// processCatchUp processes all past events in chronological order
func (el *EventLoop) processCatchUp(currentTime int64) {
	timestamps := el.storage.getTimestampsUpTo(currentTime - 1) // Process only past events
	el.logInfo("entering catch-up mode", "pastEventCount", len(timestamps), "currentTime", currentTime)

	for _, ts := range timestamps {
		el.processTimestamp(ts)
	}

	el.logInfo("exiting catch-up mode")
}

// processTimestamp fires all events for a specific timestamp
func (el *EventLoop) processTimestamp(timestamp int64) {
	events := el.storage.getAndRemove(timestamp)

	if len(events) == 0 {
		return
	}

	el.logInfo("processing events", "timestamp", timestamp, "eventCount", len(events))

	// Fire all events for this timestamp in separate goroutines
	for _, event := range events {
		go el.executeHandler(event.handler, event.Payload)
	}
}

// executeHandler executes an event handler with panic recovery
func (el *EventLoop) executeHandler(handler func(any), payload any) {
	defer func() {
		if r := recover(); r != nil {
			el.logError("handler panicked", "panic", r)
		}
	}()

	handler(payload)
}

// logInfo logs informational messages (only if IncludeInfo is enabled)
func (el *EventLoop) logInfo(msg string, args ...any) {
	if el.logger != nil && el.includeInfo {
		el.logger.Info(msg, args...)
	}
}

// logError logs error messages (always logged when logger is enabled)
func (el *EventLoop) logError(msg string, args ...any) {
	if el.logger != nil {
		el.logger.Error(msg, args...)
	}
}
