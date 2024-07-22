package nodes

import (
	"context"
	"fmt"

	"github.com/neuroplastio/neuroplastio/flowapi"
	"github.com/neuroplastio/neuroplastio/flowapi/flowdsl"
	"github.com/neuroplastio/neuroplastio/hidapi"
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
	finalize    flowapi.ActionFinalizer
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
	actionPool := flowapi.NewActionContextPool(ctx, b.log, sendCh)
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
		case ac := <-actionPool.Flush():
			event := ac.HIDEvent()
			b.triggerMappings(ac)
			if !event.IsEmpty() {
				sendCh <- event
			}
		case ev := <-in:
			event := ev.HID
			ac := actionPool.New(event)
			if actionPool.TryCapture(ac) {
				continue
			}
			b.triggerMappings(ac)
			if !event.IsEmpty() {
				sendCh <- event
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (b *Bind) triggerMappings(ac flowapi.ActionContext) {
	am := b.mappings
	event := ac.HIDEvent()
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
			am[mappingIdx].finalize = mapping.handler(ac)
		}
		if !isTriggered && mapping.isTriggered {
			am[mappingIdx].isTriggered = false
			if mapping.finalize != nil {
				am[mappingIdx].finalize(ac)
				am[mappingIdx].finalize = nil
			}
		}
	}
}
