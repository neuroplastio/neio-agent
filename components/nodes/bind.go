package nodes

import (
	"context"
	"fmt"

	"github.com/neuroplastio/neuroplastio/flowapi"
	"github.com/neuroplastio/neuroplastio/flowapi/flowdsl"
	"github.com/neuroplastio/neuroplastio/hidapi"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type BindType struct {
	log *zap.Logger
}

func (r BindType) Descriptor() flowapi.NodeTypeDescriptor {
	return flowapi.NodeTypeDescriptor{
		DisplayName: "Bind",

		UpstreamType:   flowapi.NodeLinkTypeMany,
		DownstreamType: flowapi.NodeLinkTypeOne,
	}
}

func (r BindType) CreateNode(p flowapi.NodeProvider) (flowapi.Node, error) {
	b := &Bind{
		log: r.log.With(zap.String("nodeId", p.Info().ID)),
	}
	return b, nil
}

type Bind struct {
	log      *zap.Logger
	mappings []bindItem
}

type bindItem struct {
	trigger   []hidapi.Usage
	handler   flowapi.ActionHandler
	triggered map[int]struct{}

	isTriggered bool
	clear       flowapi.ActionFinalizer
}

func (b *Bind) Configure(c flowapi.NodeConfigurator) error {
	var items flowdsl.JSONExpressionItems
	err := c.Unmarshal(&items)
	if err != nil {
		return err
	}

	for _, item := range items {
		usages, err := hidapi.ParseUsages(item.Usage.Usages)
		if err != nil {
			return err
		}
		handler, err := c.ActionHandler(item.Statement)
		if err != nil {
			return fmt.Errorf("failed to create action handler for %s %s: %w", item.UsageString, item.StatementString, err)
		}
		b.mappings = append(b.mappings, bindItem{
			trigger:   usages,
			handler:   handler,
			triggered: make(map[int]struct{}),
		})
	}
	return nil
}

func (b *Bind) Run(ctx context.Context, up flowapi.Stream, down flowapi.Stream) error {
	in := up.Subscribe(ctx)
	sendCh := make(chan *hidapi.Event)
	go func() {
		for {
			select {
			case event := <-sendCh:
				down.Broadcast(flowapi.Event{
					HID: event,
				})
			case <-ctx.Done():
				return
			}
		}
	}()
	for {
		select {
		case event := <-in:
			b.triggerMappings(ctx, event.HID, sendCh)
			if event.HID.IsEmpty() {
				continue
			}
			sendCh <- event.HID
		case <-ctx.Done():
			return nil
		}
	}
}

type actionContext struct {
	ctx   context.Context
	event *atomic.Pointer[hidapi.Event]

	sendCh chan<- *hidapi.Event
}

func (a *actionContext) Context() context.Context {
	return a.ctx
}

func (a *actionContext) HIDEvent(fn func(e *hidapi.Event)) {
	event := a.event.Load()
	send := false
	if event == nil {
		send = true
		event = hidapi.NewEvent()
		a.event.Store(event)
	}
	fn(event)
	if send && !event.IsEmpty() {
		a.sendCh <- event
	}
}

func (b *Bind) triggerMappings(ctx context.Context, event *hidapi.Event, sendCh chan<- *hidapi.Event) {
	am := b.mappings
	ac := &actionContext{
		ctx:    ctx,
		event:  atomic.NewPointer(event),
		sendCh: sendCh,
	}
	defer ac.event.Store(nil)
	for mappingIdx, mapping := range b.mappings {
		for usageIdx, usage := range mapping.trigger {
			usageEvent, ok := event.Usage(usage)
			if !ok || usageEvent.Activate == nil {
				continue
			}
			event.Suppress(usage)
			if *usageEvent.Activate {
				am[mappingIdx].triggered[usageIdx] = struct{}{}
			} else {
				delete(am[mappingIdx].triggered, usageIdx)
			}
		}
		isTriggered := len(am[mappingIdx].triggered) == len(am[mappingIdx].trigger)
		if isTriggered && !mapping.isTriggered {
			am[mappingIdx].isTriggered = true
			am[mappingIdx].clear = mapping.handler(ac)
		}
		if !isTriggered && mapping.isTriggered {
			am[mappingIdx].isTriggered = false
			if mapping.clear != nil {
				am[mappingIdx].clear(ac)
				am[mappingIdx].clear = nil
			}
		}
	}
}
