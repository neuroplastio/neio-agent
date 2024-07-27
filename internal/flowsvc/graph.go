package flowsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/neuroplastio/neio-agent/flowapi"
	"github.com/neuroplastio/neio-agent/flowapi/flowdsl"
	"github.com/neuroplastio/neio-agent/hidapi"
	"go.uber.org/zap"
)

type GraphBuilder struct {
	log      *zap.Logger
	registry *GraphRegistry
	bus      *FlowBus

	nodeIDs []string

	edgesDown map[string][]string
	edgesUp   map[string][]string

	up   map[string]flowapi.Stream
	down map[string]flowapi.Stream

	errors []error
}

func NewGraphBuilder(log *zap.Logger, reg *Registry, bus *FlowBus) GraphBuilder {
	registry := &GraphRegistry{
		registry:     reg,
		nodeTypes:    make(map[string]string),
		signals:      make(map[string]map[string]flowapi.SignalCreator),
		actions:      make(map[string]map[string]flowapi.ActionCreator),
		declarations: make(map[string]map[string]flowdsl.Declaration),
	}
	return GraphBuilder{
		log:      log,
		registry: registry,
		bus:      bus,

		edgesDown: make(map[string][]string),
		edgesUp:   make(map[string][]string),
		up:        make(map[string]flowapi.Stream),
		down:      make(map[string]flowapi.Stream),
	}
}

func (g GraphBuilder) AddNode(typ string, id string, to []string) GraphBuilder {
	g.nodeIDs = append(g.nodeIDs, id)
	g.registry.nodeTypes[id] = typ
	for _, toID := range to {
		g.edgesDown[id] = append(g.edgesDown[id], toID)
		g.edgesUp[toID] = append(g.edgesUp[toID], id)
	}
	return g
}

func (g GraphBuilder) entryNodeIDs() []string {
	ids := make([]string, 0, len(g.nodeIDs))
	for _, id := range g.nodeIDs {
		if len(g.edgesUp[id]) == 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

func (g GraphBuilder) Validate() error {
	if len(g.errors) > 0 {
		return fmt.Errorf("errors: %v", g.errors)
	}
	if len(g.nodeIDs) == 0 {
		return fmt.Errorf("no nodes")
	}

	for _, id := range g.nodeIDs {
		node, err := g.registry.NewNode(id)
		if err != nil {
			return fmt.Errorf("failed to get node %s: %w", id, err)
		}
		meta := node.Descriptor()
		switch meta.UpstreamType {
		case flowapi.NodeLinkTypeMany:
			if len(g.edgesUp[id]) == 0 {
				fmt.Println(g.edgesDown[id])
				return fmt.Errorf("node %s has no upstream nodes", id)
			}
		case flowapi.NodeLinkTypeOne:
			if len(g.edgesUp[id]) == 0 {
				return fmt.Errorf("node %s has no upstream nodes", id)
			}
			if len(g.edgesUp[id]) > 1 {
				return fmt.Errorf("node %s shold only have one upstream node", id)
			}
		case flowapi.NodeLinkTypeNone:
			if len(g.edgesUp[id]) > 0 {
				return fmt.Errorf("node %s should not have upstream nodes", id)
			}
		}
		switch meta.DownstreamType {
		case flowapi.NodeLinkTypeMany:
			if len(g.edgesDown[id]) == 0 {
				return fmt.Errorf("node %s has no downstream nodes", id)
			}
		case flowapi.NodeLinkTypeOne:
			if len(g.edgesDown[id]) == 0 {
				return fmt.Errorf("node %s has no downstream nodes", id)
			}
			if len(g.edgesDown[id]) > 1 {
				return fmt.Errorf("node %s should only have one downstream node", id)
			}
		case flowapi.NodeLinkTypeNone:
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

func (g GraphBuilder) Build(ctx context.Context) (*Graph, error) {
	up := make(map[string]flowapi.Stream, len(g.nodeIDs))
	down := make(map[string]flowapi.Stream, len(g.nodeIDs))
	for _, id := range g.nodeIDs {
		up[id] = g.createStream(ctx, id, g.edgesUp[id], true)
		down[id] = g.createStream(ctx, id, g.edgesDown[id], false)
	}
	graph := &Graph{
		log:      g.log,
		registry: g.registry,
		baseCtx:  ctx,
		nodeIDs:  g.nodeIDs,
		makeNode: g.createNode,
		up:       up,
		down:     down,
		configs:  make(map[string]json.RawMessage),
	}
	err := graph.initRunners()
	if err != nil {
		return nil, fmt.Errorf("failed to init runners: %w", err)
	}
	return graph, nil
}

func (g GraphBuilder) createNode(id string) (flowapi.Node, error) {
	nodeType, err := g.registry.NewNode(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get node type %s: %w", id, err)
	}
	provider := &nodeProvider{
		graphInfo: flowapi.NodeGraphInfo{
			ID:          id,
			Descriptor:  nodeType.Descriptor(),
			Downstreams: g.edgesDown[id],
			Upstreams:   g.edgesUp[id],
		},
		registry: g.registry,
	}
	node, err := nodeType.CreateNode(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create node %s: %w", id, err)
	}
	return node, nil
}

type nodeProvider struct {
	graphInfo flowapi.NodeGraphInfo
	registry  *GraphRegistry

	errors []error
}

func (r *nodeProvider) Info() flowapi.NodeGraphInfo {
	return r.graphInfo
}

func (r *nodeProvider) RegisterAction(name string, creator flowapi.ActionCreator) {
	err := r.registry.RegisterAction(r.graphInfo.ID, name, creator)
	if err != nil {
		r.errors = append(r.errors, err)
	}
}

func (r *nodeProvider) RegisterSignal(name string, creator flowapi.SignalCreator) {
	err := r.registry.RegisterSignal(r.graphInfo.ID, name, creator)
	if err != nil {
		r.errors = append(r.errors, err)
	}
}

// GraphRegistry allows to register NodeRunner signals and actions
type GraphRegistry struct {
	registry  *Registry
	nodeTypes map[string]string

	declarations map[string]map[string]flowdsl.Declaration
	actions      map[string]map[string]flowapi.ActionCreator
	signals      map[string]map[string]flowapi.SignalCreator
}

func (r *GraphRegistry) NewNode(id string) (flowapi.NodeType, error) {
	return r.registry.GetNode(r.nodeTypes[id])
}

func (r *GraphRegistry) RegisterAction(nodeID string, name string, creator flowapi.ActionCreator) error {
	typ, ok := r.nodeTypes[nodeID]
	if !ok {
		return fmt.Errorf("node %s not found", nodeID)
	}
	reg, err := r.registry.getNodeRegistration(typ)
	if err != nil {
		return fmt.Errorf("failed to get node %s: %w", nodeID, err)
	}
	decl, ok := reg.actions[name]
	if !ok {
		return fmt.Errorf("action %s is not declared in node type %s", name, typ)
	}
	if _, ok := r.declarations[nodeID][name]; ok {
		return fmt.Errorf("identifier %s is already registered in node %s", name, nodeID)
	}
	if r.actions[nodeID] == nil {
		r.actions[nodeID] = make(map[string]flowapi.ActionCreator)
	}
	if r.declarations[nodeID] == nil {
		r.declarations[nodeID] = make(map[string]flowdsl.Declaration)
	}
	r.actions[nodeID][name] = creator
	r.declarations[nodeID][name] = decl
	return nil
}

func (r *GraphRegistry) RegisterSignal(nodeID string, name string, creator flowapi.SignalCreator) error {
	typ, ok := r.nodeTypes[nodeID]
	if !ok {
		return fmt.Errorf("node %s not found", nodeID)
	}
	reg, err := r.registry.getNodeRegistration(typ)
	if err != nil {
		return fmt.Errorf("failed to get node %s: %w", nodeID, err)
	}
	decl, ok := reg.signals[name]
	if !ok {
		return fmt.Errorf("signal %s is not declared in node type %s", name, typ)
	}
	if _, ok := r.declarations[nodeID][name]; ok {
		return fmt.Errorf("identifier %s is already registered in node %s", name, nodeID)
	}
	if r.signals[nodeID] == nil {
		r.signals[nodeID] = make(map[string]flowapi.SignalCreator)
	}
	if r.declarations[nodeID] == nil {
		r.declarations[nodeID] = make(map[string]flowdsl.Declaration)
	}
	r.signals[nodeID][name] = creator
	r.declarations[nodeID][name] = decl
	return nil
}

type Graph struct {
	log      *zap.Logger
	registry *GraphRegistry

	baseCtx  context.Context
	makeNode func(id string) (flowapi.Node, error)
	nodeIDs  []string
	up       map[string]flowapi.Stream
	down     map[string]flowapi.Stream

	configs map[string]json.RawMessage
	runners map[string]*nodeRunner
}

type nodeConfigurator struct {
	ctx      context.Context
	config   json.RawMessage
	registry *GraphRegistry
}

func (r nodeConfigurator) Unmarshal(to any) error {
	return json.Unmarshal(r.config, to)
}

func (r nodeConfigurator) ActionHandler(stmt flowdsl.Statement) (flowapi.ActionHandler, error) {
	return r.registry.ActionHandler(r.ctx, stmt)
}

func (r nodeConfigurator) SignalHandler(stmt flowdsl.Statement) (flowapi.SignalHandler, error) {
	return r.registry.SignalHandler(r.ctx, stmt)
}

func (g *Graph) Configure(nodeID string, config json.RawMessage) error {
	runner, ok := g.runners[nodeID]
	if !ok {
		return fmt.Errorf("node %s not found", nodeID)
	}
	configurator := &nodeConfigurator{
		ctx:      runner.ctx,
		config:   config,
		registry: g.registry,
	}
	oldConfig, ok := g.configs[nodeID]
	if ok {
		if bytes.Equal(oldConfig, config) {
			return nil
		}
		g.configs[nodeID] = config
		newNode, err := g.makeNode(nodeID)
		if err != nil {
			return fmt.Errorf("failed to create node %s: %w", nodeID, err)
		}
		newCtx, newCancel := context.WithCancel(g.baseCtx)
		configurator.ctx = newCtx
		err = newNode.Configure(configurator)
		if err != nil {
			return fmt.Errorf("failed to configure node %s: %w", nodeID, err)
		}
		g.log.Debug("Replacing node", zap.String("node", nodeID))
		runner.replaceNode(newCtx, newCancel, newNode)
		return nil
	}
	g.configs[nodeID] = config
	return runner.node.Configure(configurator)
}

func (g *GraphBuilder) createStream(ctx context.Context, nodeID string, nodes []string, reverse bool) flowapi.Stream {
	if len(nodes) == 0 {
		return newFlowStream(ctx, nodeID, nil, nil)
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
	sub := g.bus.CreateMessageSubscriber(subKeys...)
	pub := make(map[string]FlowPublisher, len(pubKeys))
	for _, key := range pubKeys {
		pub[key.NodeID] = g.bus.CreatePublisher(key)
	}
	return newFlowStream(ctx, nodeID, sub, pub)
}

func (g *Graph) initRunners() error {
	g.runners = make(map[string]*nodeRunner, len(g.nodeIDs))
	for _, id := range g.nodeIDs {
		node, err := g.makeNode(id)
		if err != nil {
			return fmt.Errorf("failed to create node %s: %w", id, err)
		}
		runner := newNodeRunner(g.baseCtx, g.log, node, g.up[id], g.down[id])
		g.runners[id] = runner
	}
	return nil
}

func (g *Graph) Run() {
	for _, id := range g.nodeIDs {
		g.runners[id].start()
	}
	for _, g := range g.runners {
		<-g.running
	}
}

func (g *GraphRegistry) NewUsageActionHandler(stmt flowdsl.UsageStatement) (flowapi.ActionHandler, error) {
	usages, err := hidapi.ParseUsages(stmt.Usages)
	if err != nil {
		return nil, err
	}
	return flowapi.NewActionUsageHandler(usages...), nil
}

func (g *GraphRegistry) ActionHandler(ctx context.Context, stmt flowdsl.Statement) (flowapi.ActionHandler, error) {
	switch {
	case stmt.Usage != nil:
		return g.NewUsageActionHandler(*stmt.Usage)
	case stmt.Expr != nil:
		ident, err := g.parseIdentifier(stmt.Expr.Identifier)
		if err != nil {
			return nil, err
		}
		var (
			creator flowapi.ActionCreator
			decl    flowdsl.Declaration
		)
		if ident.NodeID == nil {
			reg, err := g.registry.getActionRegistration(ident.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to get action %q: %w", ident.Name, err)
			}
			creator = reg.action.CreateHandler
			decl = reg.declaration
		} else {
			c, ok := g.actions[*ident.NodeID][ident.Name]
			if !ok {
				return nil, fmt.Errorf("action %q not found in node %q", ident.Name, *ident.NodeID)
			}
			creator = c
			decl = g.declarations[*ident.NodeID][ident.Name]
		}
		args, err := flowapi.NewArguments(decl.Parameters, stmt.Expr.Arguments)
		if err != nil {
			return nil, fmt.Errorf("failed to create signal %q: %w", ident.Name, err)
		}
		return creator(g.newActionProvider(ctx, args))
	default:
		return nil, fmt.Errorf("invalid action statement: %v", stmt)
	}
}

func (g *GraphRegistry) SignalHandler(ctx context.Context, stmt flowdsl.Statement) (flowapi.SignalHandler, error) {
	if stmt.Expr == nil {
		return nil, fmt.Errorf("invalid signal statement")
	}
	ident, err := g.parseIdentifier(stmt.Expr.Identifier)
	if err != nil {
		return nil, err
	}
	if ident.NodeID == nil {
		return nil, fmt.Errorf("invalid signal identifier")
	}
	creator, ok := g.signals[*ident.NodeID][ident.Name]
	if !ok {
		return nil, fmt.Errorf("signal %q not found in node %q", ident.Name, *ident.NodeID)
	}
	args, err := flowapi.NewArguments(g.declarations[*ident.NodeID][ident.Name].Parameters, stmt.Expr.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to create signal %q: %w", ident.Name, err)
	}
	return creator(g.newActionProvider(ctx, args))
}

func (g *GraphRegistry) parseIdentifier(ident string) (componentIdentifier, error) {
	parts := strings.Split(ident, ".")
	if strings.HasPrefix(ident, "$") {
		if len(parts) != 2 {
			return componentIdentifier{}, fmt.Errorf("invalid identifier: %s", ident)
		}
		nodeID := parts[0][1:]
		return componentIdentifier{
			Name:   parts[1],
			NodeID: &nodeID,
		}, nil
	}
	if len(parts) != 1 {
		return componentIdentifier{}, fmt.Errorf("invalid identifier")
	}
	return componentIdentifier{
		Name: ident,
	}, nil
}

type componentIdentifier struct {
	Name   string
	NodeID *string
}

func (g *GraphRegistry) newActionProvider(ctx context.Context, args flowapi.Arguments) flowapi.ActionProvider {
	return &actionProvider{
		ctx:      ctx,
		args:     args,
		registry: g,
	}
}

type actionProvider struct {
	ctx      context.Context
	args     flowapi.Arguments
	registry *GraphRegistry
}

func (a *actionProvider) Context() context.Context {
	return a.ctx
}

func (a *actionProvider) Args() flowapi.Arguments {
	return a.args
}

func (a *actionProvider) ActionArg(argName string) (flowapi.ActionHandler, error) {
	stmt := a.args.StatementOrNil(argName)
	if stmt == nil {
		return nil, nil
	}
	return a.registry.ActionHandler(a.ctx, *stmt)
}

func (a *actionProvider) SignalArg(argName string) (flowapi.SignalHandler, error) {
	stmt := a.args.StatementOrNil(argName)
	if stmt == nil {
		return nil, nil
	}
	return a.registry.SignalHandler(a.ctx, *stmt)
}

type nodeRunner struct {
	log *zap.Logger

	node       flowapi.Node
	upstream   flowapi.Stream
	downstream flowapi.Stream

	baseCtx context.Context

	ctx     context.Context
	cancel  context.CancelFunc
	running chan struct{}
}

func newNodeRunner(ctx context.Context, log *zap.Logger, node flowapi.Node, up flowapi.Stream, down flowapi.Stream) *nodeRunner {
	runnerCtx, cancel := context.WithCancel(ctx)
	return &nodeRunner{
		log:        log,
		node:       node,
		baseCtx:    ctx,
		ctx:        runnerCtx,
		cancel:     cancel,
		upstream:   up,
		downstream: down,
	}
}

func (n *nodeRunner) start() {
	n.running = make(chan struct{})
	go func() {
		defer func() {
			n.cancel()
			close(n.running)
			if r := recover(); r != nil {
				n.log.Error("Node panic", zap.Any("panic", r))
			}
		}()
		n.log.Debug("Starting node")
		err := n.node.Run(n.ctx, n.upstream, n.downstream)
		if err != nil {
			n.log.Error("Node failed", zap.Error(err))
		}
	}()
}

func (n *nodeRunner) replaceNode(newCtx context.Context, newCancel context.CancelFunc, node flowapi.Node) {
	n.log.Debug("Replacing node")
	n.cancel()
	<-n.running
	n.node = node
	n.ctx, n.cancel = newCtx, newCancel
	n.running = make(chan struct{})
	n.start()
}
