package eventgoround

import (
	"sync"
)

// Event represents a scheduled event with a handler function
type Event struct {
	Timestamp int64       `json:"timestamp"`
	Duration  int64       `json:"duration"`
	Payload   interface{} `json:"payload"`
	Handler   string      `json:"handler"`
	handler   func(any)   `json:"-"`
}

func (e Event) Addhandler(h func(any)) {
	e.handler = h
}

// EventStorage provides thread-safe storage for events organized by timestamp
type eventStorage struct {
	mu     sync.RWMutex
	events map[int64][]Event // Map of timestamp to slice of events
}

// NewEventStorage creates a new thread-safe event storage
func newEventStorage() *eventStorage {
	return &eventStorage{
		events: make(map[int64][]Event),
	}
}

// Add adds an event to the storage for the given timestamp + duration
func (es *eventStorage) add(event Event) {
	es.mu.Lock()
	defer es.mu.Unlock()
	timestamp := event.Duration + event.Timestamp

	es.events[timestamp] = append(es.events[timestamp], event)
}

// GetAndRemove retrieves all events for a given timestamp and removes them from storage
func (es *eventStorage) getAndRemove(timestamp int64) []Event {
	es.mu.Lock()
	defer es.mu.Unlock()

	events := es.events[timestamp]
	delete(es.events, timestamp)
	return events
}

// GetTimestampsUpTo returns all timestamps that are less than or equal to the given time, sorted
func (es *eventStorage) getTimestampsUpTo(currentTime int64) []int64 {
	es.mu.RLock()
	defer es.mu.RUnlock()

	timestamps := make([]int64, 0)
	for ts := range es.events {
		if ts <= currentTime {
			timestamps = append(timestamps, ts)
		}
	}

	// Sort timestamps to process in chronological order
	for i := 0; i < len(timestamps)-1; i++ {
		for j := i + 1; j < len(timestamps); j++ {
			if timestamps[i] > timestamps[j] {
				timestamps[i], timestamps[j] = timestamps[j], timestamps[i]
			}
		}
	}

	return timestamps
}

// HasPastEvents checks if there are any events with timestamps in the past
func (es *eventStorage) hasPastEvents(currentTime int64) bool {
	es.mu.RLock()
	defer es.mu.RUnlock()

	for ts := range es.events {
		if ts < currentTime {
			return true
		}
	}
	return false
}
