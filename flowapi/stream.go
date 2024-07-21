package flowapi

import (
	"context"

	"github.com/neuroplastio/neuroplastio/hidapi"
)

type Event struct {
	SourceNodeID string
	HID          *hidapi.Event
}

type Stream interface {
	Broadcast(event Event)
	Publish(nodeID string, event Event)
	Subscribe(ctx context.Context) <-chan Event
}
