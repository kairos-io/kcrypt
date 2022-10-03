package bus

import (
	"github.com/mudler/go-pluggable"
)

const (
	// Package events.

	// EventChallenge is issued before installation begins to gather information about how the device should be provisioned.
	EventDiscoveryPassword pluggable.EventType = "discovery.password"
)

// AllEvents is a convenience list of all the events streamed from the bus.
var AllEvents = []pluggable.EventType{
	EventDiscoveryPassword,
}

// IsEventDefined checks wether an event is defined in the bus.
// It accepts strings or EventType, returns a boolean indicating that
// the event was defined among the events emitted by the bus.
func IsEventDefined(i interface{}) bool {
	checkEvent := func(e pluggable.EventType) bool {
		for _, ee := range AllEvents {
			if ee == e {
				return true
			}
		}

		return false
	}

	switch f := i.(type) {
	case string:
		return checkEvent(pluggable.EventType(f))
	case pluggable.EventType:
		return checkEvent(f)
	default:
		return false
	}
}

func EventError(err error) pluggable.EventResponse {
	return pluggable.EventResponse{Error: err.Error()}
}
