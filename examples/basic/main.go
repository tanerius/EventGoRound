package main

import (
	"fmt"
	"log"
	"time"

	eventgoround "github.com/tanerius/EventGoRound/v2"
)

// SimpleRegistry implements the IEventRegistry interface
type SimpleRegistry struct {
	handlers map[string]func(any)
}

func NewSimpleRegistry() *SimpleRegistry {
	return &SimpleRegistry{
		handlers: make(map[string]func(any)),
	}
}

func (r *SimpleRegistry) RegisterHandler(name string, handler func(any)) {
	r.handlers[name] = handler
}

func (r *SimpleRegistry) GetHandler(name string) (func(any), error) {
	handler, exists := r.handlers[name]
	if !exists {
		return nil, fmt.Errorf("handler not found: %s", name)
	}
	return handler, nil
}

func main() {
	// Create a registry and register event handlers
	registry := NewSimpleRegistry()

	registry.RegisterHandler("greet", func(payload any) {
		name := payload.(string)
		fmt.Printf("Hello, %s!\n", name)
	})

	registry.RegisterHandler("calculate", func(payload any) {
		numbers := payload.([]int)
		sum := 0
		for _, n := range numbers {
			sum += n
		}
		fmt.Printf("Sum: %d\n", sum)
	})

	// Create event loop with 100ms tick interval
	eventLoop := eventgoround.NewEventLoop(100*time.Millisecond, registry)
	eventLoop.Start()

	// Schedule an immediate event
	now := time.Now().UnixMilli()
	eventLoop.ScheduleEvent(now, 0, "greet", "World")

	// Schedule an event 1 second in the future
	future := time.Now().Add(1 * time.Second).UnixMilli()
	eventLoop.ScheduleEvent(future, 0, "calculate", []int{1, 2, 3, 4, 5})

	// Let events execute
	time.Sleep(2 * time.Second)

	// Stop the event loop
	eventLoop.Stop()

	log.Println("Event loop stopped")
}
