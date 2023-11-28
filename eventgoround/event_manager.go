package eventgoround

import (
	"errors"
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
	EType() int
}

// Interface defining event handler type
type EventHandler interface {
	EventType() int
	HandleEvent(Event)
}

// The event manager
type EventManager struct {
	eqc            int
	maxTypes       int
	running        bool
	eventQueue     chan Event
	eventListeners map[int][]EventHandler
}

// Ctor for a new event manaher
// maxTypes must be given signaling the maximum different event types
// the second argument is the eventQueuesCapacity which if not given defaults to 100000
func NewEventManager(maxTypes int, args ...int) *EventManager {
	_eqc := eventQueuesCapacity

	if len(args) > 0 {
		_eqc = args[0]
	}

	return &EventManager{
		eqc:            _eqc,
		maxTypes:       maxTypes,
		running:        false,
		eventQueue:     make(chan Event, _eqc),
		eventListeners: make(map[int][]EventHandler, maxTypes),
	}
}

// Dispatch an event
func (m *EventManager) Dispatch(event Event) error {
	select {
	case m.eventQueue <- event:
		return nil
	default:
		return fmt.Errorf("event manager failed to dispatch event %d", event.EType())
	}
}

// Register a handler for an event type
func (m *EventManager) RegisterHandler(_h EventHandler) error {
	m.panicWhenEventLoopRunning()

	if _h.EventType() >= m.maxTypes {
		return errors.New("events maxlimit reached")
	}

	_id := _h.EventType()

	if _, ok := m.eventListeners[_id]; !ok {
		m.eventListeners[_id] = make([]EventHandler, 0, 10)
	}
	m.eventListeners[_id] = append(m.eventListeners[_id], _h)
	return nil
}

// Run the event loop
func (m *EventManager) Run() {
	defer func() {
		m.running = false
	}()
	m.running = true

	for {
		select {

		case e := <-m.eventQueue:
			if listeners, ok := m.eventListeners[e.EType()]; ok {
				for _, handler := range listeners {
					handler.HandleEvent(e)
				}
			}
		default:
			time.Sleep(idleDispatcherSleepTime)
		}
	}
}

func (m *EventManager) Stop() {
	if !m.running {
		return
	}

	close(m.eventQueue)
	time.Sleep(idleDispatcherSleepTime)
	m.eventListeners = make(map[int][]EventHandler, m.maxTypes)
	m.eventQueue = make(chan Event, m.eqc)
	m.running = false
}

func (m *EventManager) panicWhenEventLoopRunning() {
	if m.running {
		panic(registeringListenerWhileRunningErrorMessage)
	}
}
