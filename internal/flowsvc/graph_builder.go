package flowsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/neuroplastio/neuroplastio/internal/flowsvc/actiondsl"
	"github.com/neuroplastio/neuroplastio/internal/hidsvc"
	"go.uber.org/zap"
)

type GraphBuilder struct {
	log      *zap.Logger
	registry *Registry
	hid      *hidsvc.Service

	nodes     map[string]Node
	nodeTypes map[string]string
	nodeOrder []string

	edgesDown map[string][]string
	edgesUp   map[string][]string

	errors []error
}

func NewGraphBuilder(log *zap.Logger, registry *Registry, hid *hidsvc.Service) GraphBuilder {
	return GraphBuilder{
		log:      log,
		registry: registry,
		hid:      hid,

		nodes:     make(map[string]Node),
		nodeTypes: make(map[string]string),
		edgesDown: make(map[string][]string),
		edgesUp:   make(map[string][]string),
	}
}

func (g GraphBuilder) AddNode(typ string, id string, to []string) GraphBuilder {
	node, err := g.registry.GetNode(typ)
	if err != nil {
		g.errors = append(g.errors, fmt.Errorf("failed to create node %s: %w", id, err))
		return g
	}
	g.nodeOrder = append(g.nodeOrder, id)
	g.nodes[id] = node
	g.nodeTypes[id] = typ
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
	if len(g.errors) > 0 {
		return fmt.Errorf("errors: %v", g.errors)
	}
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

func (g GraphBuilder) Build() (*GraphV2, error) {
	registry := &GraphRegistry{
		registry:     g.registry,
		nodeTypes:    g.nodeTypes,
		signals:      make(map[string]map[string]SignalCreator),
		actions:      make(map[string]map[string]ActionCreator),
		declarations: make(map[string]map[string]actiondsl.Declaration),
	}
	runners := make(map[string]NodeRunner, len(g.nodes))
	for _, id := range g.nodeOrder {
		node := g.nodes[id]
		info := NodeInfo{
			ID:          id,
			Type:        g.nodeTypes[id],
			Metadata:    node.Metadata(),
			Downstreams: g.edgesDown[id],
			Upstreams:   g.edgesUp[id],
		}
		provider := &runnerProvider{
			log:      g.log,
			node:     info,
			registry: registry,
		}
		runner, err := node.Runner(provider)
		if err != nil {
			return nil, fmt.Errorf("failed to create runner for node %s: %w", id, err)
		}
		runners[id] = runner
	}
	return &GraphV2{
		log:       g.log,
		registry:  registry,
		hid:       g.hid,
		runners:   runners,
		edgesUp:   g.edgesUp,
		edgesDown: g.edgesDown,
		states:    make(map[string]*runnerState),
	}, nil
}

type runnerProvider struct {
	log      *zap.Logger
	node     NodeInfo
	registry *GraphRegistry

	errors []error
}

func (r *runnerProvider) Log() *zap.Logger {
	return r.log
}

func (r *runnerProvider) Info() NodeInfo {
	return r.node
}

func (r *runnerProvider) RegisterAction(name string, creator ActionCreator) {
	err := r.registry.RegisterAction(r.node.ID, name, creator)
	if err != nil {
		r.errors = append(r.errors, err)
	}
}

func (r *runnerProvider) RegisterSignal(name string, creator SignalCreator) {
	err := r.registry.RegisterSignal(r.node.ID, name, creator)
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

type GraphV2 struct {
	log         *zap.Logger
	registry    *GraphRegistry
	hid         *hidsvc.Service
	runners     map[string]NodeRunner
	baseContext context.Context
	edgesUp     map[string][]string
	edgesDown   map[string][]string

	states map[string]*runnerState
}

type runnerConfigurator struct {
	config   json.RawMessage
	registry *GraphRegistry
	hid      *hidsvc.Service
}

func (r runnerConfigurator) Unmarshal(to any) error {
	return json.Unmarshal(r.config, to)
}

func (r runnerConfigurator) Registry() *GraphRegistry {
	return r.registry
}

func (r runnerConfigurator) HID() *hidsvc.Service {
	return r.hid
}

func (r runnerConfigurator) ActionHandler(stmt actiondsl.Statement) (ActionHandler, error) {
	return r.registry.ActionHandler(stmt)
}

func (r runnerConfigurator) SignalHandler(stmt actiondsl.Statement) (SignalHandler, error) {
	return r.registry.SignalHandler(stmt)
}

func (g *GraphV2) Configure(nodeID string, config json.RawMessage) error {
	runner, ok := g.runners[nodeID]
	if !ok {
		return fmt.Errorf("node %s not found", nodeID)
	}
	configurator := &runnerConfigurator{
		config:   config,
		registry: g.registry,
		hid:      g.hid,
	}
	return runner.Configure(configurator)
}

func (g *GraphV2) createStream(bus *FlowBus, nodeID string, nodes []string, reverse bool) FlowStream {
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
	sub := bus.CreateSubscriber(subKeys...)
	pub := make(map[string]FlowPublisher, len(pubKeys))
	for _, key := range pubKeys {
		pub[key.NodeID] = bus.CreatePublisher(key)
	}
	g.log.Debug("Created stream", zap.Any("node", nodeID), zap.Any("sub", subKeys), zap.Any("pub", pubKeys))
	return NewFlowStream(g.baseContext, nodeID, sub, pub)
}

func (g *GraphV2) Run(ctx context.Context, bus *FlowBus) error {
	g.baseContext = ctx
	for id := range g.runners {
		g.startRunner(id, bus)
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

func (g *GraphV2) startRunner(id string, bus *FlowBus) {
	ctx, cancel := context.WithCancel(g.baseContext)
	state := &runnerState{
		ctx:        ctx,
		cancel:     cancel,
		upstream:   g.createStream(bus, id, g.edgesUp[id], true),
		downstream: g.createStream(bus, id, g.edgesDown[id], false),
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
