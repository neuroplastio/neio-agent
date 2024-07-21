package flowsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/neuroplastio/neuroplastio/internal/flowsvc/actiondsl"
	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"github.com/neuroplastio/neuroplastio/internal/hidsvc"
	"github.com/neuroplastio/neuroplastio/pkg/hidevent"
	"go.uber.org/zap"
)

type GraphBuilder struct {
	log      *zap.Logger
	registry *GraphRegistry
	hid      *hidsvc.Service
	bus      *FlowBus

	nodeIDs []string

	edgesDown map[string][]string
	edgesUp   map[string][]string

	up   map[string]FlowStream
	down map[string]FlowStream

	errors []error
}

func NewGraphBuilder(log *zap.Logger, reg *Registry, bus *FlowBus, hid *hidsvc.Service) GraphBuilder {
	registry := &GraphRegistry{
		registry:     reg,
		nodeTypes:    make(map[string]string),
		signals:      make(map[string]map[string]SignalCreator),
		actions:      make(map[string]map[string]ActionCreator),
		declarations: make(map[string]map[string]actiondsl.Declaration),
	}
	return GraphBuilder{
		log:      log,
		registry: registry,
		hid:      hid,
		bus:      bus,

		edgesDown: make(map[string][]string),
		edgesUp:   make(map[string][]string),
		up:        make(map[string]FlowStream),
		down:      make(map[string]FlowStream),
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
		case NodeLinkTypeMany:
			if len(g.edgesUp[id]) == 0 {
				fmt.Println(g.edgesDown[id])
				return fmt.Errorf("node %s has no upstream nodes", id)
			}
		case NodeLinkTypeOne:
			if len(g.edgesUp[id]) == 0 {
				return fmt.Errorf("node %s has no upstream nodes", id)
			}
			if len(g.edgesUp[id]) > 1 {
				return fmt.Errorf("node %s shold only have one upstream node", id)
			}
		case NodeLinkTypeNone:
			if len(g.edgesUp[id]) > 0 {
				return fmt.Errorf("node %s should not have upstream nodes", id)
			}
		}
		switch meta.DownstreamType {
		case NodeLinkTypeMany:
			if len(g.edgesDown[id]) == 0 {
				return fmt.Errorf("node %s has no downstream nodes", id)
			}
		case NodeLinkTypeOne:
			if len(g.edgesDown[id]) == 0 {
				return fmt.Errorf("node %s has no downstream nodes", id)
			}
			if len(g.edgesDown[id]) > 1 {
				return fmt.Errorf("node %s should only have one downstream node", id)
			}
		case NodeLinkTypeNone:
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
	up := make(map[string]FlowStream, len(g.nodeIDs))
	down := make(map[string]FlowStream, len(g.nodeIDs))
	for _, id := range g.nodeIDs {
		up[id] = g.createStream(ctx, id, g.edgesUp[id], true)
		down[id] = g.createStream(ctx, id, g.edgesDown[id], false)
	}
	graph := &Graph{
		log:      g.log,
		registry: g.registry,
		hid:      g.hid,
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

func (g GraphBuilder) createNode(id string) (Node, error) {
	nodeType, err := g.registry.NewNode(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get node type %s: %w", id, err)
	}
	provider := &nodeProvider{
		log: g.log,
		graphInfo: NodeGraphInfo{
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
	log       *zap.Logger
	graphInfo NodeGraphInfo
	registry  *GraphRegistry

	errors []error
}

func (r *nodeProvider) Log() *zap.Logger {
	return r.log
}

func (r *nodeProvider) Info() NodeGraphInfo {
	return r.graphInfo
}

func (r *nodeProvider) RegisterAction(name string, creator ActionCreator) {
	err := r.registry.RegisterAction(r.graphInfo.ID, name, creator)
	if err != nil {
		r.errors = append(r.errors, err)
	}
}

func (r *nodeProvider) RegisterSignal(name string, creator SignalCreator) {
	err := r.registry.RegisterSignal(r.graphInfo.ID, name, creator)
	if err != nil {
		r.errors = append(r.errors, err)
	}
}

// GraphRegistry allows to register NodeRunner signals and actions
type GraphRegistry struct {
	registry  *Registry
	nodeTypes map[string]string

	declarations map[string]map[string]actiondsl.Declaration
	actions      map[string]map[string]ActionCreator
	signals      map[string]map[string]SignalCreator
}

func (r *GraphRegistry) NewNode(id string) (NodeType, error) {
	return r.registry.GetNode(r.nodeTypes[id])
}

func (r *GraphRegistry) RegisterAction(nodeID string, name string, creator ActionCreator) error {
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
		r.actions[nodeID] = make(map[string]ActionCreator)
	}
	if r.declarations[nodeID] == nil {
		r.declarations[nodeID] = make(map[string]actiondsl.Declaration)
	}
	r.actions[nodeID][name] = creator
	r.declarations[nodeID][name] = decl
	return nil
}

func (r *GraphRegistry) RegisterSignal(nodeID string, name string, creator SignalCreator) error {
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
		r.signals[nodeID] = make(map[string]SignalCreator)
	}
	if r.declarations[nodeID] == nil {
		r.declarations[nodeID] = make(map[string]actiondsl.Declaration)
	}
	r.signals[nodeID][name] = creator
	r.declarations[nodeID][name] = decl
	return nil
}

type Graph struct {
	log      *zap.Logger
	registry *GraphRegistry
	hid      *hidsvc.Service

	baseCtx  context.Context
	makeNode func(id string) (Node, error)
	nodeIDs  []string
	up       map[string]FlowStream
	down     map[string]FlowStream

	configs map[string]json.RawMessage
	runners map[string]*nodeRunner
}

type nodeConfigurator struct {
	ctx      context.Context
	config   json.RawMessage
	registry *GraphRegistry
	hid      *hidsvc.Service
}

func (r nodeConfigurator) Unmarshal(to any) error {
	return json.Unmarshal(r.config, to)
}

func (r nodeConfigurator) Registry() *GraphRegistry {
	return r.registry
}

func (r nodeConfigurator) HID() *hidsvc.Service {
	return r.hid
}

func (r nodeConfigurator) ActionHandler(stmt actiondsl.Statement) (ActionHandler, error) {
	return r.registry.ActionHandler(stmt)
}

func (r nodeConfigurator) SignalHandler(stmt actiondsl.Statement) (SignalHandler, error) {
	return r.registry.SignalHandler(stmt)
}

func (g *Graph) Configure(nodeID string, config json.RawMessage) error {
	runner, ok := g.runners[nodeID]
	if !ok {
		return fmt.Errorf("node %s not found", nodeID)
	}
	configurator := &nodeConfigurator{
		ctx:      g.baseCtx,
		config:   config,
		registry: g.registry,
		hid:      g.hid,
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
		err = newNode.Configure(configurator)
		if err != nil {
			return fmt.Errorf("failed to configure node %s: %w", nodeID, err)
		}
		g.log.Debug("Replacing node", zap.String("node", nodeID))
		runner.replaceNode(newNode)
		return nil
	}
	g.configs[nodeID] = config
	return runner.node.Configure(configurator)
}

func (g *GraphBuilder) createStream(ctx context.Context, nodeID string, nodes []string, reverse bool) FlowStream {
	if len(nodes) == 0 {
		return NewFlowStream(ctx, nodeID, nil, nil)
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
	return NewFlowStream(ctx, nodeID, sub, pub)
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

func NewActionUsageHandler(usages []hidparse.Usage) ActionHandler {
	return func(ac ActionContext) ActionFinalizer {
		ac.HIDEvent(func(e *hidevent.HIDEvent) {
			e.Activate(usages...)
		})
		return func(ac ActionContext) {
			ac.HIDEvent(func(e *hidevent.HIDEvent) {
				e.Deactivate(usages...)
			})
		}
	}
}

func (g *GraphRegistry) NewUsageActionHandler(stmt actiondsl.UsageStatement) (ActionHandler, error) {
	usages, err := ParseUsages(stmt.Usages)
	if err != nil {
		return nil, err
	}
	return NewActionUsageHandler(usages), nil
}

func (g *GraphRegistry) ActionHandler(stmt actiondsl.Statement) (ActionHandler, error) {
	switch {
	case stmt.Usage != nil:
		return g.NewUsageActionHandler(*stmt.Usage)
	case stmt.Expr != nil:
		ident, err := g.parseIdentifier(stmt.Expr.Identifier)
		if err != nil {
			return nil, err
		}
		var (
			creator ActionCreator
			decl    actiondsl.Declaration
		)
		if ident.NodeID == nil {
			reg, err := g.registry.getActionRegistration(ident.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to get action %q: %w", ident.Name, err)
			}
			creator = reg.action.Handler
			decl = reg.declaration
		} else {
			c, ok := g.actions[*ident.NodeID][ident.Name]
			if !ok {
				return nil, fmt.Errorf("action %q not found in node %q", ident.Name, *ident.NodeID)
			}
			creator = c
			decl = g.declarations[*ident.NodeID][ident.Name]
		}
		call, err := actiondsl.NewDeclarationCall(decl, *stmt.Expr)
		if err != nil {
			return nil, fmt.Errorf("failed to create signal %q: %w", ident.Name, err)
		}
		return creator(g.newActionProvider(call.Args()))
	default:
		return nil, fmt.Errorf("invalid action statement: %v", stmt)
	}
}

func (g *GraphRegistry) SignalHandler(stmt actiondsl.Statement) (SignalHandler, error) {
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
	call, err := actiondsl.NewDeclarationCall(g.declarations[*ident.NodeID][ident.Name], *stmt.Expr)
	if err != nil {
		return nil, fmt.Errorf("failed to create signal %q: %w", ident.Name, err)
	}
	return creator(g.newActionProvider(call.Args()))
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

func (g *GraphRegistry) newActionProvider(args actiondsl.Arguments) ActionProvider {
	return &actionProvider{
		args:     args,
		registry: g,
	}
}

type actionProvider struct {
	args     actiondsl.Arguments
	registry *GraphRegistry
}

func (a *actionProvider) Args() actiondsl.Arguments {
	return a.args
}

func (a *actionProvider) ActionArg(argName string) (ActionHandler, error) {
	stmt := a.args.StatementOrNil(argName)
	if stmt == nil {
		return nil, nil
	}
	return a.registry.ActionHandler(*stmt)
}

func (a *actionProvider) SignalArg(argName string) (SignalHandler, error) {
	stmt := a.args.StatementOrNil(argName)
	if stmt == nil {
		return nil, nil
	}
	return a.registry.SignalHandler(*stmt)
}

type nodeRunner struct {
	log *zap.Logger

	node       Node
	upstream   FlowStream
	downstream FlowStream

	baseCtx context.Context

	ctx     context.Context
	cancel  context.CancelFunc
	running chan struct{}
}

func newNodeRunner(ctx context.Context, log *zap.Logger, node Node, up FlowStream, down FlowStream) *nodeRunner {
	return &nodeRunner{
		log:        log,
		node:       node,
		baseCtx:    ctx,
		upstream:   up,
		downstream: down,
	}
}

func (n *nodeRunner) start() {
	n.ctx, n.cancel = context.WithCancel(n.baseCtx)
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

func (n *nodeRunner) replaceNode(node Node) {
	n.log.Debug("Replacing node")
	n.cancel()
	<-n.running
	n.node = node
	n.ctx, n.cancel = context.WithCancel(n.baseCtx)
	n.running = make(chan struct{})
	n.start()
}
