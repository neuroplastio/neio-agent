package hidnodes

import (
	"context"
	"encoding/json"
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

func (r Bind) Runner(info flowsvc.NodeInfo, config json.RawMessage, provider flowsvc.NodeRunnerProvider) (flowsvc.NodeRunner, error) {
	b := &BindRunner{
		log: provider.Log(),
	}
	err := parseConfig(config, b, provider.Actions())
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return b, nil
}

type BindRunner struct {
	log          *zap.Logger
	bindMappings []bindItem

	event atomic.Pointer[hidevent.HIDEvent]
}

type bindItem struct {
	trigger   []hidparse.Usage
	handler   flowsvc.HIDUsageActionHandler
	triggered map[int]struct{}

	isTriggered bool
	clear       func()
}

func parseConfig(data json.RawMessage, bind *BindRunner, registry *flowsvc.ActionRegistry) error {
	stringMap := make(map[string]string)
	err := json.Unmarshal(data, &stringMap)
	if err != nil {
		return err
	}

	mappings := make([]bindItem, 0, len(stringMap))
	for trigger, stmtString := range stringMap {
		usageStrings, err := actiondsl.ParseUsages(trigger)
		if err != nil {
			return err
		}
		usages, err := flowsvc.ParseUsages(usageStrings)
		if err != nil {
			return err
		}
		stmt, err := actiondsl.ParseStatement(stmtString)
		if err != nil {
			return fmt.Errorf("failed to parse action statement: %w", err)
		}
		handler, err := registry.New(stmt)
		if err != nil {
			return fmt.Errorf("failed to create action handler: %w", err)
		}
		mappings = append(mappings, bindItem{
			trigger:   usages,
			handler:   handler,
			triggered: make(map[int]struct{}),
		})
	}
	bind.bindMappings = mappings
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
			b.event.Store(&hidEvent)
			b.triggerMappings(ctx, sendCh)
			b.event.Store(nil)
			sendCh <- hidEvent
		case <-ctx.Done():
			return nil
		}
	}
}

func (b *BindRunner) triggerMappings(ctx context.Context, sendCh chan<- hidevent.HIDEvent) {
	event := b.event.Load()
	am := b.bindMappings
	for mappingIdx, mapping := range b.bindMappings {
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
			am[mappingIdx].clear = mapping.handler.Activate(ctx, func(usages []hidparse.Usage) func() {
				event := b.event.Load()
				var send bool
				if event == nil {
					event = hidevent.NewHIDEvent()
					send = true
				}
				b.log.Debug("Activating usage", zap.String("usages", usages[0].String()))
				event.Activate(usages...)
				if send {
					sendCh <- *event
				}
				return func() {
					event := b.event.Load()
					var send bool
					if event == nil {
						event = hidevent.NewHIDEvent()
						send = true
					}
					event.Deactivate(usages...)
					b.log.Debug("Deactivating usage", zap.String("usages", usages[0].String()))
					if send {
						sendCh <- *event
					}
				}
			})
		}
		if !isTriggered && mapping.isTriggered && mapping.clear != nil {
			am[mappingIdx].isTriggered = false
			am[mappingIdx].clear()
			am[mappingIdx].clear = nil
		}
	}
}
