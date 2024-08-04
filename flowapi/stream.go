package flowapi

import (
	"context"

	"github.com/neuroplastio/neio-agent/hidapi"
)

type HIDEventType uint8

const (
	HIDEventTypeInput HIDEventType = iota
	HIDEventTypeOutput
	HIDEventTypeFeature
)

type Event struct {
	Type HIDEventType
	HID  *hidapi.Event
}

type Stream interface {
	Broadcast(event Event)
	Publish(nodeID string, event Event)
	Subscribe(ctx context.Context) <-chan Event
}
