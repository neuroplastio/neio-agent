package hidnodes

import (
	"context"
	"fmt"

	"github.com/neuroplastio/neuroplastio/internal/flowsvc"
	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"github.com/neuroplastio/neuroplastio/pkg/hidevent"
	"go.uber.org/zap"
)

type Mux struct{}

func (f Mux) Metadata() flowsvc.NodeMetadata {
	return flowsvc.NodeMetadata{
		DisplayName: "Mux",
		Description: `Mux (Multiplexer) routes HID events to one of the downstream nodes based on the current route.
To switch the route, use the "Switch" action`,
		UpstreamType:   flowsvc.NodeTypeMany,
		DownstreamType: flowsvc.NodeTypeMany,

		Actions: []flowsvc.ActionMetadata{
			{
				DisplayName: "Switch",
				Description: `When activated switches the route to the specified downstream node, and when deactivated switches it back.
Equivalent of calling "set" and "unset" signals on activation and deactivation.`,
				Signature: "switch(route: string)",
			},
		},
		Signals: []flowsvc.SignalMetadata{
			{
				DisplayName: "Reset",
				Description: "Resets the state of mux to the default route",
				Signature:   "reset()",
			},
			{
				DisplayName: "Set",
				Description: "Sets the current route to the specified downstream node",
				Signature:   "set(route: string)",
			},
			{
				DisplayName: "Unset",
				Description: "Unsets specified route if it was previously set",
				Signature:   "unset(route: string)",
			},
		},
	}
}

func (r *MuxRunner) actionSwitch(p flowsvc.ActionProvider) (flowsvc.ActionHandler, error) {
	set, err := r.signalSet(p)
	if err != nil {
		return nil, err
	}
	unset, err := r.signalUnset(p)
	if err != nil {
		return nil, err
	}
	return flowsvc.NewSignalActionHandler(set, unset), nil
}

type muxReset struct{}
type muxSet struct {
	route string
}
type muxUnset struct {
	route string
}

func (r *MuxRunner) signalReset(p flowsvc.ActionProvider) (flowsvc.SignalHandler, error) {
	return func(ctx context.Context) {
		r.signals <- muxReset{}
	}, nil
}

func (r *MuxRunner) signalSet(p flowsvc.ActionProvider) (flowsvc.SignalHandler, error) {
	nodeID := p.Args().String("route")
	err := r.validateNode(nodeID)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) {
		r.signals <- muxSet{route: nodeID}
	}, nil
}

func (r *MuxRunner) signalUnset(p flowsvc.ActionProvider) (flowsvc.SignalHandler, error) {
	nodeID := p.Args().String("route")
	err := r.validateNode(nodeID)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) {
		r.signals <- muxUnset{route: nodeID}
	}, nil
}

type MuxRunner struct {
	id           string
	defaultRoute string
	log          *zap.Logger

	activatedUsages map[hidparse.Usage]string
	nodeIDs         []string
	signals         chan any
}

func (f Mux) Runner(p flowsvc.RunnerProvider) (flowsvc.NodeRunner, error) {
	runner := &MuxRunner{
		id:              p.Info().ID,
		log:             p.Log(),
		activatedUsages: make(map[hidparse.Usage]string, 0),
		signals:         make(chan any),
		nodeIDs:         p.Info().Downstreams,
		defaultRoute:    p.Info().Downstreams[len(p.Info().Downstreams)-1],
	}

	p.RegisterSignal("reset", runner.signalReset)
	p.RegisterSignal("set", runner.signalSet)
	p.RegisterSignal("unset", runner.signalUnset)
	p.RegisterAction("switch", runner.actionSwitch)

	return runner, nil
}

type muxConfig struct {
	Fallback string `json:"fallback"`
}

func (r *MuxRunner) Configure(c flowsvc.RunnerConfigurator) error {
	cfg := muxConfig{
		Fallback: r.defaultRoute,
	}
	if err := c.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	err := r.validateNode(cfg.Fallback)
	if err != nil {
		return err
	}
	r.defaultRoute = cfg.Fallback
	return nil
}

func (r *MuxRunner) validateNode(nodeID string) error {
	found := false
	for _, id := range r.nodeIDs {
		if id == nodeID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("node %s is not a downstream node", nodeID)
	}
	return nil
}

func (r *MuxRunner) Run(ctx context.Context, up flowsvc.FlowStream, down flowsvc.FlowStream) error {
	routeList := make([]string, 0, len(r.nodeIDs))
	currentRoute := r.defaultRoute
	in := up.Subscribe(ctx)
	defer close(r.signals)
	for {
		select {
		case signal := <-r.signals:
			// TODO: improve this
			switch s := signal.(type) {
			case muxReset:
				currentRoute = r.defaultRoute
				routeList = routeList[:0]
			case muxSet:
				routeList = append(routeList, s.route)
				currentRoute = s.route
			case muxUnset:
				for i, route := range routeList {
					if route == s.route {
						routeList = append(routeList[:i], routeList[i+1:]...)
						if currentRoute == s.route {
							if len(routeList) == 0 {
								currentRoute = r.defaultRoute
							} else {
								currentRoute = routeList[len(routeList)-1]
							}
						}
						break
					}
				}
			}
			r.log.Debug("[MUX] Current route", zap.String("route", currentRoute))
		case event := <-in:
			hidEvent := event.Message.HIDEvent
			deactEvents := make(map[string]*hidevent.HIDEvent)
			for _, usage := range hidEvent.Usages() {
				if usage.Activate == nil {
					continue
				}
				if *usage.Activate {
					if prev, ok := r.activatedUsages[usage.Usage]; ok && prev != currentRoute {
						ev, ok := deactEvents[prev]
						if !ok {
							ev = hidevent.NewHIDEvent()
							deactEvents[prev] = ev
						}
						ev.Deactivate(usage.Usage)
					}
					r.activatedUsages[usage.Usage] = currentRoute
				}
				if !*usage.Activate {
					if prev, ok := r.activatedUsages[usage.Usage]; ok && prev != currentRoute {
						ev, ok := deactEvents[prev]
						if !ok {
							ev = hidevent.NewHIDEvent()
							deactEvents[prev] = ev
						}
						ev.Deactivate(usage.Usage)
						hidEvent.Suppress(usage.Usage)
					}
					delete(r.activatedUsages, usage.Usage)
				}
			}
			for route, ev := range deactEvents {
				down.Publish(route, flowsvc.FlowEvent{
					HIDEvent: *ev,
				})
			}
			if !hidEvent.IsEmpty() {
				down.Publish(currentRoute, flowsvc.FlowEvent{
					HIDEvent: hidEvent,
				})
			}
		case <-ctx.Done():
			return nil
		}
	}
}
