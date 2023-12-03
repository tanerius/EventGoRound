package eventgoround

import (
	"log"
	"time"
)

const (
	eventQueuesCapacity                                       = 10000
	idleDispatcherSleepTime                     time.Duration = 5 * time.Millisecond
	registeringListenerWhileRunningErrorMessage               = "Tried to register listener while running event loop. Registering listeners is not thread safe therefore prohibited after starting event loop."
)

type eventHandler interface {
	handle()
}

type genericHandler struct {
	event          *Event
	eventListeners []Listener
}

func (handler *genericHandler) handle() {
	for _, listener := range handler.eventListeners {
		listener.HandleEvent(handler.event)
	}
}

// An interface for event listeners
type Listener interface {
	Type() int
	HandleEvent(*Event)
}

// Defines the event manager
type EventManager struct {
	running         bool
	eqc             int
	eventsPrioQueue chan eventHandler
	eventsQueue     chan eventHandler

	genericListeners map[int][]Listener
}

// Ctor for a new event manager
// the first argument is the event queue capapcity which if not given defaults to 100000
func NewEventManager(args ...int) *EventManager {
	queueSize := eventQueuesCapacity

	if len(args) > 0 {
		queueSize = args[0]
	}

	return &EventManager{
		running:          false,
		eqc:              queueSize,
		eventsPrioQueue:  make(chan eventHandler, queueSize),
		eventsQueue:      make(chan eventHandler, queueSize),
		genericListeners: make(map[int][]Listener),
	}
}

// Runs the main loop of the event manager
func (dispatcher *EventManager) Run() {
	if dispatcher.running {
		log.Fatalf("event manager %T already running", dispatcher)
		return
	}

	defer func() {
		dispatcher.eventsPrioQueue = make(chan eventHandler, dispatcher.eqc)
		dispatcher.eventsQueue = make(chan eventHandler, dispatcher.eqc)
		dispatcher.running = false
	}()

	dispatcher.running = true

	for {
		select {
		case handler, ok := <-dispatcher.eventsPrioQueue:
			if !ok {
				return
			}
			handler.handle()

		case handler, ok := <-dispatcher.eventsQueue:
			if ok {
				handler.handle()
			}

		default:
			time.Sleep(idleDispatcherSleepTime)
		}
	}
}

// Dispatches an event to the regular queue
func (dispatcher *EventManager) DispatchEvent(event *Event) {
	handler := &genericHandler{
		event:          event,
		eventListeners: dispatcher.genericListeners[event.eventType],
	}

	dispatcher.eventsQueue <- handler
}

// Dispatches an event to the priority queue
func (dispatcher *EventManager) DispatchPriorityEvent(event *Event) {
	handler := &genericHandler{
		event:          event,
		eventListeners: dispatcher.genericListeners[event.eventType],
	}

	dispatcher.eventsPrioQueue <- handler
}

// Register a new event listener (that must implement Listener interface)
func (dispatcher *EventManager) RegisterListener(listener Listener) {
	if dispatcher.running {
		panic(registeringListenerWhileRunningErrorMessage)
	}

	if _, ok := dispatcher.genericListeners[listener.Type()]; !ok {
		dispatcher.genericListeners[listener.Type()] = make([]Listener, 0)
	}

	dispatcher.genericListeners[listener.Type()] = append(dispatcher.genericListeners[listener.Type()], listener)
}

// Stops the main loop of the event manager. This will discard any pending calls in the event queues
func (dispatcher *EventManager) Stop() {
	if !dispatcher.running {
		return
	}
	close(dispatcher.eventsPrioQueue)
	close(dispatcher.eventsQueue)
	time.Sleep(idleDispatcherSleepTime)
}
