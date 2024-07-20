package flowsvc

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
)

type GraphBuilder struct {
	nodes     map[string]Node
	nodeOrder []string

	edgesDown map[string][]string
	edgesUp   map[string][]string
}

func NewGraphBuilder() GraphBuilder {
	return GraphBuilder{
		nodes:     make(map[string]Node),
		edgesDown: make(map[string][]string),
		edgesUp:   make(map[string][]string),
	}
}

func (g GraphBuilder) AddNode(id string, node Node, to []string) GraphBuilder {
	g.nodeOrder = append(g.nodeOrder, id)
	g.nodes[id] = node
	for _, toID := range to {
		g.edgesDown[id] = append(g.edgesDown[id], toID)
		g.edgesUp[toID] = append(g.edgesUp[toID], id)
	}
	return g
}

func (g GraphBuilder) entryNodeIDs() []string {
	ids := make([]string, 0, len(g.nodes))
	for _, id := range g.nodeOrder {
		if len(g.edgesUp[id]) == 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

func (g GraphBuilder) Validate() error {
	if len(g.nodes) == 0 {
		return fmt.Errorf("no nodes")
	}

	for _, id := range g.nodeOrder {
		node := g.nodes[id]
		meta := node.Metadata()
		switch meta.UpstreamType {
		case NodeTypeMany:
			if len(g.edgesUp[id]) == 0 {
				fmt.Println(g.edgesDown[id])
				return fmt.Errorf("node %s has no upstream nodes", id)
			}
		case NodeTypeOne:
			if len(g.edgesUp[id]) == 0 {
				return fmt.Errorf("node %s has no upstream nodes", id)
			}
			if len(g.edgesUp[id]) > 1 {
				return fmt.Errorf("node %s shold only have one upstream node", id)
			}
		case NodeTypeNone:
			if len(g.edgesUp[id]) > 0 {
				return fmt.Errorf("node %s should not have upstream nodes", id)
			}
		}
		switch meta.DownstreamType {
		case NodeTypeMany:
			if len(g.edgesDown[id]) == 0 {
				return fmt.Errorf("node %s has no downstream nodes", id)
			}
		case NodeTypeOne:
			if len(g.edgesDown[id]) == 0 {
				return fmt.Errorf("node %s has no downstream nodes", id)
			}
			if len(g.edgesDown[id]) > 1 {
				return fmt.Errorf("node %s should only have one downstream node", id)
			}
		case NodeTypeNone:
			if len(g.edgesDown[id]) > 0 {
				return fmt.Errorf("node %s should not have downstream nodes", id)
			}
		}
	}

	entryNodeIDs := g.entryNodeIDs()
	if len(entryNodeIDs) == 0 {
		return fmt.Errorf("no entry nodes")
	}

	for _, id := range entryNodeIDs {
		if err := g.validateCycles(id, make(map[string]struct{})); err != nil {
			return fmt.Errorf("cycle detected: %w", err)
		}
	}

	return nil
}

func (g GraphBuilder) validateCycles(id string, visited map[string]struct{}) error {
	if _, ok := visited[id]; ok {
		return fmt.Errorf("cycle detected: %s", id)
	}
	visited[id] = struct{}{}
	for _, downstreamID := range g.edgesDown[id] {
		if err := g.validateCycles(downstreamID, visited); err != nil {
			return err
		}
	}
	delete(visited, id)
	return nil
}

func (g GraphBuilder) Build(configs map[string]json.RawMessage, p NodeRunnerProvider, bus *FlowBus) (*GraphV2, error) {
	runners := make(map[string]NodeRunner, len(g.nodes))
	for _, id := range g.nodeOrder {
		node := g.nodes[id]
		info := NodeInfo{
			ID:          id,
			Metadata:    node.Metadata(),
			Downstreams: g.edgesDown[id],
			Upstreams:   g.edgesUp[id],
		}
		runner, err := node.Runner(info, configs[id], p)
		if err != nil {
			return nil, fmt.Errorf("failed to create runner for node %s: %w", id, err)
		}
		runners[id] = runner
	}
	return &GraphV2{
		log:       p.Log(),
		bus:       bus,
		runners:   runners,
		edgesUp:   g.edgesUp,
		edgesDown: g.edgesDown,
		states:    make(map[string]*runnerState),
	}, nil
}

type GraphV2 struct {
	log         *zap.Logger
	bus         *FlowBus
	runners     map[string]NodeRunner
	baseContext context.Context
	edgesUp     map[string][]string
	edgesDown   map[string][]string

	states map[string]*runnerState
}

func (g *GraphV2) createStream(nodeID string, nodes []string, reverse bool) FlowStream {
	if len(nodes) == 0 {
		g.log.Debug("Created empty stream", zap.Any("node", nodeID))
		return NewFlowStream(g.baseContext, nodeID, nil, nil)
	}
	t1 := FlowEventUpstream
	t2 := FlowEventDownstream
	if reverse {
		t1 = FlowEventDownstream
		t2 = FlowEventUpstream
	}
	subKeys := make([]FlowEventKey, 0, len(nodes))
	subKeys = append(subKeys, FlowEventKey{
		NodeID: nodeID,
		Type:   t1,
	})
	pubKeys := make([]FlowEventKey, 0, len(nodes))
	for _, id := range nodes {
		pubKeys = append(pubKeys, FlowEventKey{
			NodeID: id,
			Type:   t2,
		})
	}
	sub := g.bus.CreateSubscriber(subKeys...)
	pub := make(map[string]FlowPublisher, len(pubKeys))
	for _, key := range pubKeys {
		pub[key.NodeID] = g.bus.CreatePublisher(key)
	}
	g.log.Debug("Created stream", zap.Any("node", nodeID), zap.Any("sub", subKeys), zap.Any("pub", pubKeys))
	return NewFlowStream(g.baseContext, nodeID, sub, pub)
}

func (g *GraphV2) Run(ctx context.Context) error {
	g.baseContext = ctx
	for id := range g.runners {
		g.startRunner(id)
	}
	// TODO: node error handling
	<-ctx.Done()
	return nil
}

type runnerState struct {
	upstream   FlowStream
	downstream FlowStream

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

func (g *GraphV2) startRunner(id string) {
	ctx, cancel := context.WithCancel(g.baseContext)
	state := &runnerState{
		ctx:        ctx,
		cancel:     cancel,
		upstream:   g.createStream(id, g.edgesUp[id], true),
		downstream: g.createStream(id, g.edgesDown[id], false),
		done:       make(chan struct{}),
	}
	g.states[id] = state
	go func() {
		defer close(state.done)
		g.log.Info("Starting runner", zap.String("node", id))
		err := g.runners[id].Run(state.ctx, state.upstream, state.downstream)
		if err != nil {
			// TODO: node failure handling
			g.log.Error("Runner failed", zap.Error(err))
		}
	}()
}
