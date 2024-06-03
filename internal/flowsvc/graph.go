package flowsvc

import (
	"context"
	"errors"
	"fmt"

	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Graph struct {
	nodeMap    map[string]Node
	origMap    map[string]OriginNode
	entryNodes map[string]DescriptorSource
	destMap    map[string]DestinationNode
	origEdges  map[string][]string // map[origID][]destID
	destEdges  map[string][]string // map[destID][]origID
}

func NewGraph() *Graph {
	return &Graph{
		nodeMap:    make(map[string]Node),
		origMap:    make(map[string]OriginNode),
		entryNodes: make(map[string]DescriptorSource),
		destMap:    make(map[string]DestinationNode),
		origEdges:  make(map[string][]string),
		destEdges:  make(map[string][]string),
	}
}

func (g *Graph) AddNode(id string, n Node) {
	g.nodeMap[id] = n
	if orig, ok := n.(OriginNode); ok {
		g.origMap[id] = orig
	}
	if dest, ok := n.(DestinationNode); ok {
		g.destMap[id] = dest
	}
	if entry, ok := n.(DescriptorSource); ok {
		g.entryNodes[id] = entry
	}
}

func (g *Graph) Connect(origID, destID string) error {
	if _, ok := g.origMap[origID]; !ok {
		return errors.New("Origin node not found")
	}
	if _, ok := g.destMap[destID]; !ok {
		return errors.New("Destination node not found")
	}
	if _, ok := g.origEdges[origID]; !ok {
		g.origEdges[origID] = make([]string, 0)
	}
	g.origEdges[origID] = append(g.origEdges[origID], destID)
	if _, ok := g.destEdges[destID]; !ok {
		g.destEdges[destID] = make([]string, 0)
	}
	g.destEdges[destID] = append(g.destEdges[destID], origID)
	return nil
}

type Node interface {
	Start(ctx context.Context, up FlowStream, down FlowStream) error
}

type hidReport struct {
	SourceID string
	Data     []byte
}

type FlowEvent struct {
	SourceNodeID string
	HIDReport    hidparse.Report
}

type DescriptorSource interface {
	OriginNode
	GetDescriptor() []byte
}

type OriginNode interface {
	Node
	OriginSpec() OriginSpec
}

type OriginSpec struct {
	MinConnections int
	MaxConnections int
}

type DestinationNode interface {
	Node
	DestinationSpec() DestinationSpec
	Configure(dependencies map[string][]byte) ([]byte, error)
}

type DestinationSpec struct {
	MinConnections int
	MaxConnections int
}

func (g *Graph) Validate() error {
	if len(g.entryNodes) == 0 {
		return errors.New("No entry nodes")
	}
	for origID, destIDs := range g.origEdges {
		orig := g.origMap[origID]
		if len(destIDs) < orig.OriginSpec().MinConnections {
			return fmt.Errorf("Origin node %q does not meet minimum connections", origID)
		}
		if len(destIDs) > orig.OriginSpec().MaxConnections {
			return fmt.Errorf("Origin node %q exceeds maximum connections", origID)
		}
	}
	for destID, origIDs := range g.destEdges {
		dest := g.destMap[destID]
		if len(origIDs) < dest.DestinationSpec().MinConnections {
			return fmt.Errorf("Destination node %q does not meet minimum connections", destID)
		}
		if len(origIDs) > dest.DestinationSpec().MaxConnections {
			return fmt.Errorf("Destination node %q exceeds maximum connections", destID)
		}
	}

	return nil
}

// ExitNodes returns a list of DestinationNodes that have no downstream connections.
func (g *Graph) ExitNodes() []string {
	var exits []string
	for destID := range g.destEdges {
		if len(g.origEdges[destID]) == 0 {
			exits = append(exits, destID)
		}
	}
	return exits
}

type MutationFunc func(sourceID string, data []byte)

type Compiler struct {
	log   *zap.Logger
	graph *Graph
	cg    *CompiledGraph
}

func NewCompiler(logger *zap.Logger, g *Graph) *Compiler {
	return &Compiler{
		log:   logger,
		graph: g,
	}
}

func (c *Compiler) Compile() (*CompiledGraph, error) {
	c.cg = NewCompiledGraph(c.log, c.graph)
	if err := c.graph.Validate(); err != nil {
		return nil, fmt.Errorf("failed to compile graph: %w", err)
	}
	exitNodes := c.graph.ExitNodes()
	if len(exitNodes) == 0 {
		return nil, errors.New("No exit nodes")
	}

	for _, exitID := range exitNodes {
		if err := c.compileNode(exitID); err != nil {
			return nil, fmt.Errorf("failed to compile exit node: %w", err)
		}
	}

	return c.cg, nil
}

// compileNode identifies all node dependencies and attempts to compile them.
// node compilation consists of creting a descriptor object that represents HID
// reports that node inputs and/or outputs.
func (c *Compiler) compileNode(id string) error {
	n := c.graph.nodeMap[id]
	if _, ok := c.cg.nodeDescriptors[id]; ok {
		// Node already compiled
		return nil
	}
	if src, ok := n.(DescriptorSource); ok {
		// reached descriptor source
		c.cg.nodeDescriptors[id] = src.GetDescriptor()
		return nil
	}
	if dest, ok := n.(DestinationNode); ok {
		dependencies := c.graph.destEdges[id]
		for _, origID := range dependencies {
			if err := c.compileNode(origID); err != nil {
				return fmt.Errorf("failed to compile origin node %q: %w", origID, err)
			}
		}
		descriptors := make(map[string][]byte, len(dependencies))
		for _, origID := range dependencies {
			descriptors[origID] = c.cg.nodeDescriptors[origID]
		}
		desc, err := dest.Configure(descriptors)
		if err != nil {
			return fmt.Errorf("failed to compile destination node %q: %w", id, err)
		}
		c.cg.nodeDescriptors[id] = desc
	}
	return nil
}

type CompiledGraph struct {
	log             *zap.Logger
	graph           *Graph
	nodeDescriptors map[string][]byte
}

func NewCompiledGraph(logger *zap.Logger, g *Graph) *CompiledGraph {
	return &CompiledGraph{
		log:             logger,
		graph:           g,
		nodeDescriptors: make(map[string][]byte),
	}
}

func (c *CompiledGraph) NodeDescriptors() map[string][]byte {
	return c.nodeDescriptors
}

func (c *CompiledGraph) Start(ctx context.Context, bus *FlowBus) error {
	g, groupCtx := errgroup.WithContext(ctx)
	for id := range c.graph.nodeMap {
		if err := c.start(groupCtx, g, bus, id); err != nil {
			return fmt.Errorf("failed to start node %q: %w", id, err)
		}
	}
	return g.Wait()
}

func (c *CompiledGraph) createStream(ctx context.Context, bus *FlowBus, nodeID string, nodes []string, reverse bool) FlowStream {
	if len(nodes) == 0 {
		c.log.Debug("Created empty stream", zap.Any("node", nodeID))
		return NewFlowStream(nodeID, nil, nil)
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
	sub := bus.CreateSubscriber(subKeys...)
	pub := make(map[string]FlowPublisher, len(pubKeys))
	for _, key := range pubKeys {
		pub[key.NodeID] = bus.CreatePublisher(key)
	}
	c.log.Debug("Created stream", zap.Any("node", nodeID), zap.Any("sub", subKeys), zap.Any("pub", pubKeys))
	return NewFlowStream(nodeID, sub, pub)
}

func (c *CompiledGraph) start(ctx context.Context, g *errgroup.Group, bus *FlowBus, nodeID string) error {
	upstream := c.createStream(ctx, bus, nodeID, c.graph.destEdges[nodeID], true)
	downstream := c.createStream(ctx, bus, nodeID, c.graph.origEdges[nodeID], false)
	g.Go(func() error {
		err := c.graph.nodeMap[nodeID].Start(ctx, upstream, downstream)
		if err != nil {
			return fmt.Errorf("failed to start node %q: %w", nodeID, err)
		}
		return nil
	})
	return nil
}
