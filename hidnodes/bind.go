package hidnodes

import (
	"context"
	"fmt"

	"github.com/neuroplastio/neuroplastio/internal/flowsvc"
	"github.com/neuroplastio/neuroplastio/internal/flowsvc/actiondsl"
	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"github.com/neuroplastio/neuroplastio/pkg/hidevent"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type Bind struct{}

func (r Bind) Metadata() flowsvc.NodeMetadata {
	return flowsvc.NodeMetadata{
		DisplayName: "Bind",

		UpstreamType:   flowsvc.NodeTypeMany,
		DownstreamType: flowsvc.NodeTypeOne,
	}
}

func (r Bind) Runner(p flowsvc.RunnerProvider) (flowsvc.NodeRunner, error) {
	b := &BindRunner{
		log: p.Log(),
	}
	return b, nil
}

type BindRunner struct {
	log      *zap.Logger
	mappings []bindItem

	event atomic.Pointer[hidevent.HIDEvent]
}

type bindItem struct {
	trigger   []hidparse.Usage
	handler   flowsvc.ActionHandler
	triggered map[int]struct{}

	isTriggered bool
	clear       flowsvc.ActionFinalizer
}

func (b *BindRunner) Configure(c flowsvc.RunnerConfigurator) error {
	var items actiondsl.JSONExpressionItems
	err := c.Unmarshal(&items)
	if err != nil {
		return err
	}

	for _, item := range items {
		usages, err := flowsvc.ParseUsages(item.Usage.Usages)
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

func (b *BindRunner) Run(ctx context.Context, up flowsvc.FlowStream, down flowsvc.FlowStream) error {
	in := up.Subscribe(ctx)
	sendCh := make(chan hidevent.HIDEvent)
	go func() {
		for {
			select {
			case event := <-sendCh:
				down.Broadcast(flowsvc.FlowEvent{HIDEvent: event})
			case <-ctx.Done():
				return
			}
		}
	}()
	for {
		select {
		case event := <-in:
			hidEvent := event.Message.HIDEvent
			b.triggerMappings(ctx, &hidEvent, sendCh)
			sendCh <- hidEvent
		case <-ctx.Done():
			return nil
		}
	}
}

type actionContext struct {
	ctx   context.Context
	event *atomic.Pointer[hidevent.HIDEvent]

	sendCh chan<- hidevent.HIDEvent
}

func (a *actionContext) Context() context.Context {
	return a.ctx
}

func (a *actionContext) HIDEvent(fn func(e *hidevent.HIDEvent)) {
	event := a.event.Load()
	send := false
	if event == nil {
		send = true
		event = hidevent.NewHIDEvent()
		a.event.Store(event)
	}
	fn(event)
	if send && !event.IsEmpty() {
		a.sendCh <- *event
	}
}

func (b *BindRunner) triggerMappings(ctx context.Context, event *hidevent.HIDEvent, sendCh chan<- hidevent.HIDEvent) {
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
