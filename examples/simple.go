package main

import (
	"fmt"
	"time"

	"github.com/tanerius/EventGoRound/eventgoround"
)

// Declare a simple struct
type SimpleEvent struct {
	EventName string
}

// Implement Id() string from Event
func (e *SimpleEvent) Id() string {
	return e.EventName
}

// Implement a handler
type SimpleEventHandler struct{}

// implements EventHandler[*SimpleEvent]
func (m *SimpleEventHandler) HandleEvent(e *SimpleEvent) {
	fmt.Printf("%s : event %p ; handler %p\n", e.EventName, e, m)
}

func newSimpleEventHandler() eventgoround.EventHandler[*SimpleEvent] {
	return &SimpleEventHandler{}
}

func main() {
	// register a manager for an event type
	manager := eventgoround.NewEventManager[*SimpleEvent]()

	// register a handler
	manager.RegisterHandler(newSimpleEventHandler())

	// Run the event manager
	go manager.Run()

	// Dispatch 10 cocurrent events
	for i := 0; i < 10; i++ {
		go func(x int) {
			event := &SimpleEvent{
				EventName: fmt.Sprintf("simpleEvent_%d", x),
			}
			manager.Dispatch(event)
		}(i)
	}

	time.Sleep(3 * time.Second)
	manager.Stop()
}
