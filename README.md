# Event Go Round

[![Go Reference](https://pkg.go.dev/badge/github.com/tanerius/EventGoRound/v2.svg)](https://pkg.go.dev/github.com/tanerius/EventGoRound/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/tanerius/EventGoRound)](https://goreportcard.com/report/github.com/tanerius/EventGoRound)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Event Go Round is a simple, thread-safe event scheduling and dispatch system for Go, originally designed for game backends. It allows you to schedule events to be executed at specific times with flexible event handlers.

## Features

- **Thread-safe**: Event dispatching is concurrent-safe and can run in multiple goroutines
- **Time-based scheduling**: Schedule events for immediate, past, or future execution
- **Flexible event handlers**: Implement your own event registry to handle events however you need
- **Panic recovery**: Automatically recovers from panics in event handlers
- **Pause/Resume support**: Control event loop execution dynamically
- **Decoupled design**: Event handlers implement the simple `IEventRegistry` interface

## Installation

```bash
go get github.com/tanerius/EventGoRound/v2
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"
    eventgoround "github.com/tanerius/EventGoRound/v2"
)

// Implement IEventRegistry
type MyRegistry struct {
    handlers map[string]func(any)
}

func (r *MyRegistry) GetHandler(name string) (func(any), error) {
    if handler, exists := r.handlers[name]; exists {
        return handler, nil
    }
    return nil, fmt.Errorf("handler not found: %s", name)
}

func main() {
    // Create registry and register handlers
    registry := &MyRegistry{
        handlers: map[string]func(any){
            "greet": func(payload any) {
                fmt.Printf("Hello, %s!\n", payload.(string))
            },
        },
    }

    // Create and start event loop
    eventLoop := eventgoround.NewEventLoop(100*time.Millisecond, registry)
    eventLoop.Start()

    // Schedule an event
    eventLoop.ScheduleEvent(time.Now().UnixMilli(), 0, "greet", "World")

    time.Sleep(1 * time.Second)
    eventLoop.Stop()
}
```

## API Overview

### EventLoop

The main component for managing event scheduling and execution.

```go
// Create a new event loop with specified tick interval
func NewEventLoop(tickInterval time.Duration, registry IEventRegistry) *EventLoop

// Start the event loop
func (el *EventLoop) Start()

// Stop the event loop
func (el *EventLoop) Stop()

// Schedule an event
func (el *EventLoop) ScheduleEvent(timestamp int64, duration int64, handlerName string, payload any)

// Pause/Resume event processing
func (el *EventLoop) Pause()
func (el *EventLoop) Resume()

// Check if catching up on past events
func (el *EventLoop) IsCatchingUp() bool
```

### Event

Represents a scheduled event with timing and handler information.

```go
type Event struct {
    Timestamp int64       // Unix timestamp in milliseconds
    Duration  int64       // Duration for the event
    Payload   interface{} // Event data
    Handler   string      // Name of the handler function
}
```

### IEventRegistry

Interface that your event handler registry must implement.

```go
type IEventRegistry interface {
    GetHandler(name string) (func(any), error)
}
```

## Examples

See the [examples](examples/) directory for more detailed usage examples:

- [Basic Example](examples/basic/main.go) - Simple event scheduling and handling

## Use Cases

Event Go Round is perfect for:

- Game server event scheduling
- Time-based task execution
- Background job processing
- Event-driven architectures
- Scheduled notifications and reminders

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
