//go:build wireinject

// This file is the wire injector — it is excluded from regular builds.
// Run `wire gen` to regenerate wire_gen.go from this file.
package main

import "github.com/google/wire"

// InitializeEvent creates an Event. It will error if the Event is staffed with
// a grumpy greeter.
func InitializeEvent(phrase string) Event {
	wire.Build(NewEvent, NewGreeter, NewMessage)
	return Event{}
}
