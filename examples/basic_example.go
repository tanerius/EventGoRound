package main

import (
	"fmt"

	egr "github.com/tanerius/EventGoRound/eventgoround"
)

// lets define 2 events
// foo with id 1
// and
// bar with id 2

type FooEvent struct {
	StringData string
}

func (e *FooEvent) EType() int {
	return 1
}

type BarEvent struct {
	IntData int
}

func (e *BarEvent) EType() int {
	return 2
}

// Now lets define a foo and bar handlers

type FooHandler struct{}

func (e *FooHandler) EventType() int {
	return 1 // same as the Foo event type
}

func (h *FooHandler) HandleEvent(event *FooEvent) {
	fmt.Printf("FooHandler Data : %s", event.StringData)
}

type BarHandler struct{}

func (e *BarHandler) EventType() int {
	return 1 // same as the Bar event type
}

func (h *BarHandler) HandleEvent(event *BarEvent) {
	fmt.Printf("BarHandler Data : %d", event.IntData)
}

func Test(handler egr.EventHandler) {
	fmt.Println(handler.EventType())
}

func main() {
	x := &FooHandler{}

	Test(x)

}
