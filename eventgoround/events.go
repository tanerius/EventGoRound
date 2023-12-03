package eventgoround

import "errors"

// An interface for the data sent with an event. This may be extended in the future
type EventData interface{}

// Struct defining event type
type Event struct {
	eventType int
	data      EventData
}

// Create a new event.
// Params: type int, data EventData interface
func NewEvent(_type int, _data EventData) *Event {
	return &Event{
		eventType: _type,
		data:      _data,
	}
}

// Returns the event type. This is an integer for now
func (e *Event) Type() int {
	return e.eventType
}

// A generic function used to return the event data into its correct type
func GetEventData[T EventData](_e *Event) (T, error) {
	var zero T
	if e, ok := _e.data.(T); ok {
		return e, nil
	}
	return zero, errors.New("invalid event type conversion")
}
