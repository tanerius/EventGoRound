package eventgoround

import (
	"fmt"
	"time"
)

const (
	eventQueuesCapacity                                       = 100000
	idleDispatcherSleepTime                     time.Duration = 5 * time.Millisecond
	registeringListenerWhileRunningErrorMessage               = "Tried to register listener while running event loop. Registering listeners is not thread safe therefore prohibited after starting event loop."
)

// Interface defining event type
type Event interface {
	Type() int
}

// Interface defining event handler type
type EventHandler interface {
	EventType() int
	HandleEvent(Event)
}

// The event manager
type EventManager struct {
	running        bool
	quit           chan struct{}
	eventQueue     chan Event
	eventListeners map[int][]EventHandler
}

// Ctor for a new event manaher
func NewEventManager(maxTypes int, eventQueueSize int) *EventManager {
	return &EventManager{
		running:        false,
		quit:           make(chan struct{}),
		eventQueue:     make(chan Event, 10000),
		eventListeners: make(map[int][]EventHandler),
	}
}

func (m *EventManager) Dispatch(event Event) error {
	select {
	case m.eventQueue <- event:
		return nil
	default:
		return fmt.Errorf("event manager failed to dispatch event %d", event.Type())
	}
}

// Register a handler for an event type
func (m *EventManager) RegisterHandler(_h EventHandler) {
	_id := _h.EventType()
	m.panicWhenEventLoopRunning()
	if _, ok := m.eventListeners[_id]; !ok {
		m.eventListeners[_id] = make([]EventHandler, 0, 10)
	}
	m.eventListeners[_id] = append(m.eventListeners[_id], _h)
}

func (m *EventManager) panicWhenEventLoopRunning() {
	if m.running {
		panic(registeringListenerWhileRunningErrorMessage)
	}
}
