package flowsvc

import (
	"context"
	"encoding/json"
	"fmt"
)

type Router struct {
	id    string
	state *FlowState
	defaultRoute string
}

type routerConfig struct {
	DefaultRoute string `json:"defaultRoute"`
}

func NewRouterNode(data json.RawMessage, provider *NodeProvider) (Node, error) {
	var cfg routerConfig
	err := json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &Router{
		state: provider.State,
		defaultRoute: cfg.DefaultRoute,
	}, nil
}

func (r *Router) Configure(descriptors map[string][]byte) ([]byte, error) {
	for _, desc := range descriptors {
		return desc, nil
	}
	return nil, fmt.Errorf("no descriptor found")
}

func (r *Router) Start(ctx context.Context, up FlowStream, down FlowStream) error {
	nodeIDs := down.NodeIDs()
	enumValues := make(map[string]int, len(nodeIDs))
	defaultRouteIdx := 0
	for i, id := range nodeIDs {
		enumValues[id] = i
		if id == r.defaultRoute {
			defaultRouteIdx = i
		}
	}
	valueSub, err := r.state.RegisterEnum(ctx, r.id, enumValues, defaultRouteIdx)
	if err != nil {
		return err
	}
	currentRoute := defaultRouteIdx
	in := up.Subscribe(ctx)
	valueCh := valueSub(ctx)
	for {
		select {
		case event := <-valueCh:
			fmt.Println("Router", r.id, "route changed to", *event.Message.Value.Int)
			currentRoute = *event.Message.Value.Int
		case event := <-in:
			fmt.Println("Router", r.id, "forwarding to", nodeIDs[currentRoute])
			down.Publish(ctx, nodeIDs[currentRoute], event.Message)
		case <-ctx.Done():
			return nil
		}
	}
}

func (r *Router) OriginSpec() OriginSpec {
	return OriginSpec{
		MinConnections: 1,
		MaxConnections: 255,
	}
}

func (r *Router) DestinationSpec() DestinationSpec {
	return DestinationSpec{
		MinConnections: 1,
		MaxConnections: 1,
	}
}
