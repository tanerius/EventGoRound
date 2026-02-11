package eventgoround

import (
	"fmt"
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
}

// NewEventLoop creates a new event loop with the specified tick interval
func NewEventLoop(tickInterval time.Duration, registry IEventRegistry) *EventLoop {
	return &EventLoop{
		storage:      newEventStorage(),
		eventChan:    make(chan Event, 100), // Buffered channel for better performance
		stopChan:     make(chan struct{}),
		pauseChan:    make(chan bool),
		isCatchingUp: false,
		isPaused:     false,
		registry:     registry,
		tickInterval: tickInterval,
	}
}

func (el *EventLoop) log(msg string) {

}

// Start begins the event loop processing
func (el *EventLoop) Start() {
	go el.run()
	el.log("START: Event loop started")
}

// Stop gracefully stops the event loop
func (el *EventLoop) Stop() {
	close(el.stopChan)
	el.log("STOP: Event loop stopped")
}

// ScheduleEvent schedules an event to be executed at the specified timestamp
// This will block during catch-up mode until all past events are processed
// Events cannot be scheduled when the loop is paused
func (el *EventLoop) ScheduleEvent(timestamp int64, duration int64, handlername string, payload any) {
	if el.IsPaused() {
		el.log("SCHEDULE: Cannot schedule event - event loop is paused")
		return
	}

	handler, err := el.registry.GetHandler(handlername)

	if err != nil {
		el.log(fmt.Sprintf("SCHEDULE: Cannot schedule event - handler '%s' not found", handlername))
		return
	}

	event := Event{
		Timestamp: timestamp, // Timestamp in seconds
		Duration:  duration,  // Duration in seconds
		handler:   handler,
		Handler:   handlername,
		Payload:   payload,
	}
	el.eventChan <- event
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
		el.log("PAUSE: Event loop paused")
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
		el.log("UNPAUSE: Event loop unpaused")
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
	if state {
		el.log("CATCH-UP START: Entering catch-up mode - blocking new event registration")

	} else {
		el.log("CATCH-UP END: Exiting catch-up mode - resuming normal operation")
	}
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
			// Only process new events when not catching up and not paused
			if !el.IsCatchingUp() && !paused {
				el.storage.add(event)
				el.log(fmt.Sprintf("SCHEDULE: Event scheduled for timestamp: %d", event.Timestamp+event.Duration))
			}
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

	el.log(fmt.Sprintf("CATCH-UP: Catching up on %d past timestamp(s)", len(timestamps)))

	for _, ts := range timestamps {
		el.log(fmt.Sprintf("CATCH-UP EVENT: Processing past events for timestamp: %d (current: %d)", ts, currentTime))
		el.processTimestamp(ts)
	}
}

// processTimestamp fires all events for a specific timestamp
func (el *EventLoop) processTimestamp(timestamp int64) {
	events := el.storage.getAndRemove(timestamp)

	if len(events) == 0 {
		return
	}

	el.log(fmt.Sprintf("Firing %d event(s) for timestamp: %d", len(events), timestamp))

	// Fire all events for this timestamp in separate goroutines
	for i, event := range events {
		go el.executeHandler(event.handler, event.Payload, timestamp, i)
	}
}

// executeHandler executes an event handler with panic recovery
func (el *EventLoop) executeHandler(handler func(any), payload any, timestamp int64, index int) {
	defer func() {
		if r := recover(); r != nil {
			el.log(fmt.Sprintf("HANDLE: Event handler panicked at timestamp %d, index %d: %v", timestamp, index, r))
		}
	}()

	handler(payload)
}

// GetStats returns current statistics about the event loop
func (el *EventLoop) GetStats() string {
	currentTime := time.Now().Unix()
	pastTimestamps := el.storage.getTimestampsUpTo(currentTime - 1)

	return fmt.Sprintf("STATISTICS: Catching up: %v, Past events: %d timestamps",
		el.IsCatchingUp(), len(pastTimestamps))
}
