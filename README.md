# Event Go Round

Event Go Around is a simple event system in golang planned for use to dispatch events for game backends. The usage of generics guarantees that events can be of any type. It was designed and written when i started working on a gaming backend as my first serious golang project. Initially this event system was written as a part of the backend, but then making it its own package seemed like a good idea since it can be used for all sorts of things requiring event based synchronization.

## Features

I tried to keep this event system simple and useful for my use cases, the requirements of which were:

- Thread safe - dispatching an event is thread safe and can run concurrently
- Decoupled - How you design your events is completely up to you and is not enforced by Event Go Round. Only the event handlers need to implement the `Listener` interface.
- Simplicity - in order to avoid code generators but still be able to group main aspects of an `EventManager` the `GetEventData[T]` generic function provides a simple way to retreive an event data into its correct type.
  
An important **note** to keep in mind is that stopping a manager will clear its current event queue but will **not** detach the listeners.

## Todo

Things I would probably want to do are:

- Enable running managers to accept new subscriptions to events. This will also make the unsubscribe useful.
- Generalise subscriptions using generics, which would enable a subscription to add to the correct handler slice.

## Examples

The ```examples``` folder illustrates some use cases.
