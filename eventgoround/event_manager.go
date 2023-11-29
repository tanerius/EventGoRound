package eventgoround

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	eventQueuesCapacity                                       = 10000
	idleDispatcherSleepTime                     time.Duration = 5 * time.Millisecond
	registeringListenerWhileRunningErrorMessage               = "Tried to register listener while running event loop. Registering listeners is not thread safe therefore prohibited after starting event loop."
)

// Interface defining event type
type Event interface {
	Id() string
}

// event handler type
type EventHandler[T Event] interface {
	HandleEvent(T)
}

// A manager definition useful for grouping
type IManager interface {
	Run()
	Stop()
}

// The event manager
type EventManager[T Event] struct {
	eqc            int
	running        bool
	terminated     bool
	eventQueue     chan T
	eventListeners []EventHandler[T]
}

// Ctor for a new event manaher
// the first argument is the eventQueuesCapacity which if not given defaults to 100000
func NewEventManager[T Event](args ...int) *EventManager[T] {
	_eqc := eventQueuesCapacity

	if len(args) > 0 {
		_eqc = args[0]
	}

	return &EventManager[T]{
		eqc:            _eqc,
		running:        false,
		terminated:     false,
		eventQueue:     make(chan T, _eqc),
		eventListeners: make([]EventHandler[T], 0),
	}
}

// Dispatch an event
func (m *EventManager[T]) Dispatch(event T) error {
	select {
	case m.eventQueue <- event:
		return nil
	default:
		return fmt.Errorf("event manager failed to dispatch event %s", event.Id())
	}
}

// Subscribe a handler for an event type
func (m *EventManager[T]) Subscribe(_h EventHandler[T]) {
	m.panicWhenEventLoopRunning()

	m.eventListeners = append(m.eventListeners, _h)

}

// Run the event loop
func (m *EventManager[T]) Run() {
	if m.running {
		log.Fatalf("event manager %T already running", m)
		return
	}

	defer func() {
		m.eventQueue = make(chan T, m.eqc)
		m.running = false
	}()

	m.running = true

	for {
		select {

		case e, ok := <-m.eventQueue:
			if !ok {
				return
			}
			for _, handler := range m.eventListeners {
				handler.HandleEvent(e)
			}
		default:
			time.Sleep(idleDispatcherSleepTime)
		}
	}
}

func (m *EventManager[T]) Stop() {
	if !m.running {
		return
	}
	close(m.eventQueue)
	time.Sleep(idleDispatcherSleepTime)
}

func (m *EventManager[T]) panicWhenEventLoopRunning() {
	if m.running {
		panic(registeringListenerWhileRunningErrorMessage)
	}
}
