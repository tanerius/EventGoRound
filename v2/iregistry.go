package eventgoround

// Registry of event handlers. It allows you to retrieve events by name.
type IEventRegistry interface {
	GetHandler(name string) (func(any), error)
}
