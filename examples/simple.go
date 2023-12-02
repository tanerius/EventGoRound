package main

import (
	"fmt"
	"time"

	"github.com/tanerius/EventGoRound/eventgoround"
)

// Declare a simple struct for event data
type SimpleEventData struct {
	EventName string
}

// Implement a handler
type SimpleEventHandler struct{}

// implements EventHandler[*SimpleEvent]
func (m *SimpleEventHandler) HandleEvent(e *eventgoround.Event) {
	data, err := eventgoround.GetEventData[*SimpleEventData](e)
	if err != nil {
		panic(err)
	} else {
		fmt.Println(data.EventName)
	}
}

func (m *SimpleEventHandler) Type() int {
	return 1
}

func main() {

	// register a manager for an event type
	manager := eventgoround.NewEventManager()

	// register a handler
	manager.RegisterListener(&SimpleEventHandler{})

	// Run the event manager
	go manager.Run()

	// Dispatch 100 cocurrent normal events
	for i := 0; i < 100; i++ {
		go func(x int) {
			eventdata := &SimpleEventData{
				EventName: fmt.Sprintf("simpleEvent_%d", x),
			}
			event := eventgoround.NewEvent(1, eventdata)
			manager.DispatchEvent(event)
		}(i)
	}

	// Dispatch 10 prio events
	for i := 0; i < 10; i++ {
		go func(x int) {
			eventdata := &SimpleEventData{
				EventName: fmt.Sprintf(" PRIO simpleEvent_%d", x),
			}
			event := eventgoround.NewEvent(1, eventdata)
			manager.DispatchPriorityEvent(event)
		}(i)
	}

	time.Sleep(3 * time.Second)
	manager.Stop()
}
