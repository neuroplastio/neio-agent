package nodes

import (
	"context"
	"fmt"

	"github.com/neuroplastio/neuroplastio/components/actions"
	"github.com/neuroplastio/neuroplastio/flowapi"
	"github.com/neuroplastio/neuroplastio/hidapi"
	"go.uber.org/zap"
)

type MuxType struct {
	log *zap.Logger
}

func (f MuxType) Descriptor() flowapi.NodeTypeDescriptor {
	return flowapi.NodeTypeDescriptor{
		DisplayName: "Mux",
		Description: `Mux (Multiplexer) routes HID events to one of the downstream nodes based on the current route.
To switch the route, use the "Switch" action`,
		UpstreamType:   flowapi.NodeLinkTypeMany,
		DownstreamType: flowapi.NodeLinkTypeMany,

		Actions: []flowapi.ActionDescriptor{
			{
				DisplayName: "Switch",
				Description: `When activated switches the route to the specified downstream node, and when deactivated switches it back.
Equivalent of calling "set" and "unset" signals on activation and deactivation.`,
				Signature: "switch(route: string)",
			},
		},
		Signals: []flowapi.SignalDescriptor{
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

func (r *Mux) actionSwitch(p flowapi.ActionProvider) (flowapi.ActionHandler, error) {
	set, err := r.signalSet(p)
	if err != nil {
		return nil, err
	}
	unset, err := r.signalUnset(p)
	if err != nil {
		return nil, err
	}
	return actions.NewSignalActionHandler(set, unset), nil
}

type muxReset struct{}
type muxSet struct {
	route string
}
type muxUnset struct {
	route string
}

func (r *Mux) signalReset(p flowapi.ActionProvider) (flowapi.SignalHandler, error) {
	return func(ctx context.Context) {
		r.signals <- muxReset{}
	}, nil
}

func (r *Mux) signalSet(p flowapi.ActionProvider) (flowapi.SignalHandler, error) {
	nodeID := p.Args().String("route")
	err := r.validateNode(nodeID)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) {
		r.signals <- muxSet{route: nodeID}
	}, nil
}

func (r *Mux) signalUnset(p flowapi.ActionProvider) (flowapi.SignalHandler, error) {
	nodeID := p.Args().String("route")
	err := r.validateNode(nodeID)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) {
		r.signals <- muxUnset{route: nodeID}
	}, nil
}

type Mux struct {
	id           string
	defaultRoute string
	log          *zap.Logger

	activatedUsages map[hidapi.Usage]string
	nodeIDs         []string
	signals         chan any
}

func (f MuxType) CreateNode(p flowapi.NodeProvider) (flowapi.Node, error) {
	node := &Mux{
		id:              p.Info().ID,
		log:             f.log.With(zap.String("nodeId", p.Info().ID)),
		activatedUsages: make(map[hidapi.Usage]string, 0),
		signals:         make(chan any),
		nodeIDs:         p.Info().Downstreams,
		defaultRoute:    p.Info().Downstreams[len(p.Info().Downstreams)-1],
	}

	p.RegisterSignal("reset", node.signalReset)
	p.RegisterSignal("set", node.signalSet)
	p.RegisterSignal("unset", node.signalUnset)
	p.RegisterAction("switch", node.actionSwitch)

	return node, nil
}

type muxConfig struct {
	Fallback string `json:"fallback"`
}

func (r *Mux) Configure(c flowapi.NodeConfigurator) error {
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

func (r *Mux) validateNode(nodeID string) error {
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

func (r *Mux) Run(ctx context.Context, up flowapi.Stream, down flowapi.Stream) error {
	routeList := make([]string, 0, len(r.nodeIDs))
	currentRoute := r.defaultRoute
	in := up.Subscribe(ctx)
	defer close(r.signals)
	for {
		changed := false
		select {
		case signal := <-r.signals:
			// TODO: improve this
			switch s := signal.(type) {
			case muxReset:
				changed = true
				currentRoute = r.defaultRoute
				routeList = routeList[:0]
			case muxSet:
				if s.route != currentRoute {
					changed = true
					routeList = append(routeList, s.route)
					currentRoute = s.route
				}
			case muxUnset:
				for i, route := range routeList {
					if route == s.route {
						changed = true
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
			if changed {
				r.log.Info("Route changed", zap.String("route", currentRoute))
			}
		case event := <-in:
			hidEvent := event.HID
			deactEvents := make(map[string]*hidapi.Event)
			for _, usage := range hidEvent.Usages() {
				if usage.Activate == nil {
					continue
				}
				// TODO: improve this part / reuse some parts from `bind.go`
				if *usage.Activate {
					if prev, ok := r.activatedUsages[usage.Usage]; ok && prev != currentRoute {
						ev, ok := deactEvents[prev]
						if !ok {
							ev = hidapi.NewEvent()
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
							ev = hidapi.NewEvent()
							deactEvents[prev] = ev
						}
						ev.Deactivate(usage.Usage)
						hidEvent.Suppress(usage.Usage)
					}
					delete(r.activatedUsages, usage.Usage)
				}
			}
			for route, ev := range deactEvents {
				down.Publish(route, flowapi.Event{
					HID: ev,
				})
			}
			if !hidEvent.IsEmpty() {
				down.Publish(currentRoute, flowapi.Event{
					HID: hidEvent,
				})
			}
		case <-ctx.Done():
			return nil
		}
	}
}
