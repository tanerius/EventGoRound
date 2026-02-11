# Event Go Round

Event Go Around is a simple event system in golang planned for use to dispatch events for game backends. The usage of generics guarantees that events can be of any type. It was designed and written when i started working on a gaming backend as my first serious golang project. Initially this event system was written as a part of the backend, but then making it its own package seemed like a good idea since it can be used for all sorts of things requiring event based synchronization.

## Features

I tried to keep this event system simple and useful for my use cases, the requirements of which were:

- Thread safe - dispatching an event is thread safe and can run concurrently
- Decoupled - How you design your events is completely up to you and is not enforced by Event Go Round. Only the event handlers need to implement the `IEventRegistry` interface.

## Examples

The ```examples``` folder illustrates some use cases.
