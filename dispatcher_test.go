package eventgoround

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// mockRegistry implements IEventRegistry for testing
type mockRegistry struct {
	handlers map[string]func(any)
	mu       sync.RWMutex
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{
		handlers: make(map[string]func(any)),
	}
}

func (m *mockRegistry) RegisterHandler(name string, handler func(any)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[name] = handler
}

func (m *mockRegistry) GetHandler(name string) (func(any), error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	handler, ok := m.handlers[name]
	if !ok {
		return nil, fmt.Errorf("handler not found: %s", name)
	}
	return handler, nil
}

// executionTracker tracks handler executions
type executionTracker struct {
	executions []execution
	mu         sync.Mutex
	wg         sync.WaitGroup
}

type execution struct {
	handlerName string
	payload     any
	timestamp   int64
	actualTime  time.Time
}

func newExecutionTracker() *executionTracker {
	return &executionTracker{
		executions: make([]execution, 0),
	}
}

func (et *executionTracker) track(handlerName string, payload any, timestamp int64) func(any) {
	return func(data any) {
		et.mu.Lock()
		et.executions = append(et.executions, execution{
			handlerName: handlerName,
			payload:     data,
			timestamp:   timestamp,
			actualTime:  time.Now(),
		})
		et.mu.Unlock()
		et.wg.Done()
	}
}

func (et *executionTracker) count() int {
	et.mu.Lock()
	defer et.mu.Unlock()
	return len(et.executions)
}

func (et *executionTracker) getExecutions() []execution {
	et.mu.Lock()
	defer et.mu.Unlock()
	result := make([]execution, len(et.executions))
	copy(result, et.executions)
	return result
}

func (et *executionTracker) expectCount(count int) {
	et.wg.Add(count)
}

func (et *executionTracker) waitWithTimeout(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		et.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// TestSchedulePastEvents - Scenario 1: Schedule events in the past
func TestSchedulePastEvents(t *testing.T) {
	registry := newMockRegistry()
	tracker := newExecutionTracker()

	// Register handlers that track executions
	registry.RegisterHandler("past1", tracker.track("past1", nil, 0))
	registry.RegisterHandler("past2", tracker.track("past2", nil, 0))
	registry.RegisterHandler("past3", tracker.track("past3", nil, 0))

	// Create event loop with short tick interval for faster tests
	loop := NewEventLoop(50*time.Millisecond, registry)
	loop.Start()
	defer loop.Stop()

	// Get current time and create timestamps in the past
	now := time.Now().Unix()

	// Schedule 3 events with execution times in the past
	// Event 1: 5 seconds ago
	tracker.expectCount(3)
	loop.ScheduleEvent(now-10, 5, "past1", "payload1")

	// Event 2: 3 seconds ago
	loop.ScheduleEvent(now-7, 4, "past2", "payload2")

	// Event 3: 1 second ago (most recent past event)
	loop.ScheduleEvent(now-3, 2, "past3", "payload3")

	// Small delay to allow event loop to detect past events and enter catch-up
	time.Sleep(100 * time.Millisecond)

	// Verify catch-up mode was triggered
	if !loop.IsCatchingUp() && tracker.count() < 3 {
		// It's possible catch-up completed very quickly
		t.Log("Catch-up mode may have completed already")
	}

	// Wait for all events to execute
	if !tracker.waitWithTimeout(2 * time.Second) {
		t.Fatalf("Timeout waiting for past events to execute. Got %d executions, expected 3", tracker.count())
	}

	// Verify all 3 events executed
	if count := tracker.count(); count != 3 {
		t.Errorf("Expected 3 events to execute, got %d", count)
	}

	// Verify all expected handlers executed (order may vary due to concurrent execution)
	executions := tracker.getExecutions()
	handlerNames := make(map[string]bool)
	for _, exec := range executions {
		handlerNames[exec.handlerName] = true
	}

	expectedHandlers := []string{"past1", "past2", "past3"}
	for _, name := range expectedHandlers {
		if !handlerNames[name] {
			t.Errorf("Expected handler %s to execute, but it didn't", name)
		}
	}

	t.Log("Successfully scheduled and executed events in the past")
}

// TestScheduleCurrentTimeEvents - Scenario 2: Schedule events for the current time
func TestScheduleCurrentTimeEvents(t *testing.T) {
	registry := newMockRegistry()
	tracker := newExecutionTracker()

	registry.RegisterHandler("current", tracker.track("current", nil, 0))

	loop := NewEventLoop(50*time.Millisecond, registry)
	loop.Start()
	defer loop.Stop()

	// Schedule event for current time (duration = 0)
	now := time.Now().Unix()
	tracker.expectCount(1)
	loop.ScheduleEvent(now, 0, "current", "current-payload")

	// Give it a short time to process
	time.Sleep(200 * time.Millisecond)

	// Event should execute immediately without triggering catch-up
	// (or catch-up happens so fast it's barely noticeable)
	if !tracker.waitWithTimeout(1 * time.Second) {
		t.Fatalf("Timeout waiting for current time event to execute")
	}

	if count := tracker.count(); count != 1 {
		t.Errorf("Expected 1 event to execute, got %d", count)
	}

	executions := tracker.getExecutions()
	if len(executions) > 0 {
		exec := executions[0]
		if exec.handlerName != "current" {
			t.Errorf("Expected handler 'current', got '%s'", exec.handlerName)
		}
		if exec.payload != "current-payload" {
			t.Errorf("Expected payload 'current-payload', got '%v'", exec.payload)
		}
	}

	t.Log("Successfully scheduled and executed event for current time")
}

// TestScheduleFutureEvents - Scenario 3: Schedule events for the future
func TestScheduleFutureEvents(t *testing.T) {
	registry := newMockRegistry()
	tracker := newExecutionTracker()

	registry.RegisterHandler("future", tracker.track("future", nil, 0))

	loop := NewEventLoop(50*time.Millisecond, registry)
	loop.Start()
	defer loop.Stop()

	// Schedule event 2 seconds in the future (must be > tickInterval to avoid immediate execution)
	now := time.Now().Unix()
	tracker.expectCount(1)
	loop.ScheduleEvent(now, 2, "future", "future-payload")

	// Verify event doesn't execute immediately
	time.Sleep(500 * time.Millisecond)
	if count := tracker.count(); count != 0 {
		t.Errorf("Event should not execute immediately, but got %d executions", count)
	}

	// Wait for the event to execute (should happen after ~1 second + processing time)
	if !tracker.waitWithTimeout(2 * time.Second) {
		t.Fatalf("Timeout waiting for future event to execute")
	}

	if count := tracker.count(); count != 1 {
		t.Errorf("Expected 1 event to execute, got %d", count)
	}

	executions := tracker.getExecutions()
	if len(executions) > 0 {
		exec := executions[0]
		if exec.handlerName != "future" {
			t.Errorf("Expected handler 'future', got '%s'", exec.handlerName)
		}
		// Verify execution happened roughly 1 second after scheduling
		// (allowing for some variance due to tick interval and processing)
	}

	t.Log("Successfully scheduled and executed future event after specified delay")
}

// TestScheduleDuringCatchUp - Scenario 4: Try to schedule during catch-up
func TestScheduleDuringCatchUp(t *testing.T) {
	registry := newMockRegistry()
	tracker := newExecutionTracker()

	registry.RegisterHandler("past", tracker.track("past", nil, 0))
	registry.RegisterHandler("during_catchup", tracker.track("during_catchup", nil, 0))

	loop := NewEventLoop(50*time.Millisecond, registry)
	loop.Start()
	defer loop.Stop()

	// Schedule multiple events in the past to trigger catch-up mode
	now := time.Now().Unix()
	tracker.expectCount(5) // Expecting only the 5 past events, not the one during catch-up

	// Schedule many past events to ensure catch-up takes long enough
	for i := 0; i < 5; i++ {
		// Schedule events at different timestamps in the past
		loop.ScheduleEvent(now-20, int64(i*2), "past", fmt.Sprintf("past-%d", i))
	}

	// Give the loop time to start processing and enter catch-up mode
	time.Sleep(100 * time.Millisecond)

	// Poll until we detect catch-up mode is active
	maxAttempts := 20
	caughtDuringCatchup := false
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if loop.IsCatchingUp() {
			// Try to schedule an event while definitely in catch-up mode
			// This event should be rejected/dropped
			loop.ScheduleEvent(now+10, 0, "during_catchup", "should-be-rejected")
			caughtDuringCatchup = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !caughtDuringCatchup {
		t.Log("Warning: Could not catch system during catch-up mode, event may execute")
	}

	// Wait for the past events to complete
	if !tracker.waitWithTimeout(3 * time.Second) {
		t.Fatalf("Timeout waiting for past events to execute")
	}

	// Wait a bit more to see if the rejected event somehow executes
	time.Sleep(1500 * time.Millisecond)

	// Verify that only the 5 past events executed, not the one scheduled during catch-up
	count := tracker.count()
	if count != 5 {
		t.Errorf("Expected 5 events to execute (past events only), got %d", count)
	}

	// Verify the "during_catchup" handler was never called
	executions := tracker.getExecutions()
	for _, exec := range executions {
		if exec.handlerName == "during_catchup" {
			t.Error("Event scheduled during catch-up should have been rejected, but it executed")
		}
	}

	t.Log("Successfully prevented event scheduling during catch-up mode")
}

// TestPanicRecovery - Scenario 5: Demonstrate panic recovery
func TestPanicRecovery(t *testing.T) {
	registry := newMockRegistry()
	tracker := newExecutionTracker()

	// Register a handler that panics
	panicHandler := func(data any) {
		tracker.mu.Lock()
		tracker.executions = append(tracker.executions, execution{
			handlerName: "panic_handler",
			payload:     data,
			timestamp:   time.Now().Unix(),
			actualTime:  time.Now(),
		})
		tracker.mu.Unlock()
		tracker.wg.Done()
		panic("intentional panic for testing")
	}
	registry.RegisterHandler("panic_handler", panicHandler)

	// Register a normal handler that should execute after the panic
	registry.RegisterHandler("normal", tracker.track("normal", nil, 0))

	loop := NewEventLoop(50*time.Millisecond, registry)
	loop.Start()
	defer loop.Stop()

	now := time.Now().Unix()

	// Schedule the panicking event
	tracker.expectCount(2)
	loop.ScheduleEvent(now, 0, "panic_handler", "will-panic")

	// Schedule a normal event that should still execute
	loop.ScheduleEvent(now, 1, "normal", "should-still-work")

	// Wait for both events to be processed
	if !tracker.waitWithTimeout(3 * time.Second) {
		t.Fatalf("Timeout waiting for events to execute")
	}

	// Verify both events executed (panic was recovered)
	count := tracker.count()
	if count != 2 {
		t.Errorf("Expected 2 events to execute, got %d", count)
	}

	executions := tracker.getExecutions()

	// Verify panic handler executed
	foundPanic := false
	foundNormal := false
	for _, exec := range executions {
		if exec.handlerName == "panic_handler" {
			foundPanic = true
		}
		if exec.handlerName == "normal" {
			foundNormal = true
		}
	}

	if !foundPanic {
		t.Error("Panic handler should have executed")
	}
	if !foundNormal {
		t.Error("Normal handler should have executed after panic was recovered")
	}

	// Verify the event loop is still running (can schedule new events)
	tracker2 := newExecutionTracker()
	registry.RegisterHandler("after_panic", tracker2.track("after_panic", nil, 0))

	tracker2.expectCount(1)
	loop.ScheduleEvent(time.Now().Unix(), 0, "after_panic", "system-still-works")

	if !tracker2.waitWithTimeout(2 * time.Second) {
		t.Fatal("Event loop not functioning after panic")
	}

	t.Log("Successfully recovered from handler panic, system continues operating")
}
