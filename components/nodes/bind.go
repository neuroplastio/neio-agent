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
	trigger trigger
	handler flowapi.ActionHandler

	triggered bool
	finalizer flowapi.ActionFinalizer
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
			trigger: newUsageActivation(usages),
			handler: handler,
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
		case ev := <-in:
			event := ev.HID
			ac := actionPool.New(event)
			b.triggerMappings(ac)
			if !event.IsEmpty() {
				actionPool.Interrupt()
				sendCh <- event
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (b *Bind) triggerMappings(ac flowapi.ActionContext) {
	m := b.mappings
	for idx, mapping := range m {
		isTriggered := mapping.trigger.Check(ac)
		switch {
		case isTriggered && !mapping.triggered:
			m[idx].triggered = true
			m[idx].finalizer = mapping.handler(ac)
		case !isTriggered && mapping.triggered:
			if mapping.finalizer != nil {
				m[idx].finalizer(ac)
			}
			m[idx].triggered = false
			m[idx].finalizer = nil
		}
	}
}

type trigger interface {
	Check(ac flowapi.ActionContext) bool
}

func newUsageActivation(usages []hidapi.Usage) trigger {
	return &usageActivation{
		usages:   usages,
		counters: make(map[hidapi.Usage]int),
	}
}

type usageActivation struct {
	usages   []hidapi.Usage
	counters map[hidapi.Usage]int
}

func (u *usageActivation) Check(ac flowapi.ActionContext) bool {
	for _, usage := range u.usages {
		usageEvent, ok := ac.HIDEvent().Usage(usage)
		if !ok || usageEvent.Activate == nil {
			continue
		}
		if *usageEvent.Activate {
			u.counters[usage]++
		} else {
			u.counters[usage]--
			if u.counters[usage] <= 0 {
				delete(u.counters, usage)
			}
		}
	}
	if len(u.counters) == len(u.usages) {
		ac.HIDEvent().Suppress(u.usages...)
		return true
	}
	return false
}
