package flowsvc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/neuroplastio/neio-agent/flowapi"
	"github.com/neuroplastio/neio-agent/internal/configsvc"
	"github.com/neuroplastio/neio-agent/pkg/bus"
	"go.uber.org/zap"
)

type Service struct {
	config   *configsvc.Service
	log      *zap.Logger
	flowPath string

	mu           sync.Mutex
	ctx          context.Context
	graphCtx     context.Context
	graphCancel  context.CancelFunc
	graph        *Graph
	bus          *FlowBus
	graphHash    uint64
	graphRunning chan struct{}

	registry *Registry
}

type (
	FlowEventType uint8
	FlowEventKey  struct {
		NodeID string
		Type   FlowEventType
	}
	FlowBus        = bus.Bus[FlowEventKey, flowapi.Event]
	FlowPublisher  = bus.Publisher[flowapi.Event]
	FlowSubscriber = bus.MessageSubscriber[flowapi.Event]
	flowStream     struct {
		ctx        context.Context
		nodeID     string
		nodeIDs    []string
		subscriber FlowSubscriber
		publishers map[string]FlowPublisher
	}
)

const (
	FlowEventDownstream FlowEventType = iota
	FlowEventUpstream
)

func newFlowStream(ctx context.Context, nodeID string, subscriber FlowSubscriber, publishers map[string]FlowPublisher) flowStream {
	nodeIDs := make([]string, 0, len(publishers))
	for nodeID := range publishers {
		nodeIDs = append(nodeIDs, nodeID)
	}
	return flowStream{
		ctx:        ctx,
		nodeID:     nodeID,
		nodeIDs:    nodeIDs,
		subscriber: subscriber,
		publishers: publishers,
	}
}

func (f flowStream) Publish(toNodeID string, msg flowapi.Event) {
	// TODO: configurable timeout
	ctx, cancel := context.WithTimeout(f.ctx, 100*time.Microsecond)
	f.publishers[toNodeID](ctx, msg)
	cancel()
}

func (f flowStream) Broadcast(msg flowapi.Event) {
	for _, nodeID := range f.nodeIDs {
		f.Publish(nodeID, msg)
	}
}

func (f flowStream) Subscribe(ctx context.Context) <-chan flowapi.Event {
	return f.subscriber(ctx)
}

func New(
	log *zap.Logger,
	config *configsvc.Service,
	flowPath string,
	registry *Registry,
) *Service {
	return &Service{
		config:   config,
		log:      log,
		flowPath: flowPath,
		bus:      bus.NewBus[FlowEventKey, flowapi.Event](log),
		registry: registry,
	}
}

func (s *Service) Start(ctx context.Context) error {
	s.ctx = ctx
	select {
	case <-ctx.Done():
		return nil
	case <-s.config.Ready():
	}
	cfg, err := configsvc.Register(s.config, s.flowPath, FlowConfig{}, s.onConfigChange)
	if err != nil {
		return fmt.Errorf("failed to register flow config: %w", err)
	}
	err = s.bus.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start flow bus: %w", err)
	}
	select {
	case <-ctx.Done():
		return nil
	case <-s.bus.Ready():
	}
	err = s.startGraph(cfg)
	if err != nil {
		return fmt.Errorf("failed to compile flow: %w", err)
	}
	<-s.ctx.Done()
	<-s.graphRunning
	return nil
}

func (s *Service) onConfigChange(cfg FlowConfig, err error) {
	if err != nil {
		s.log.Error("failed to parse config", zap.Error(err))
		return
	}
	treeHash := cfg.treeHash()
	if treeHash != s.graphHash {
		s.log.Info("Configuration updated", zap.Uint64("hash", treeHash), zap.Uint64("old", s.graphHash))
		err = s.restartGraph(cfg)
		if err != nil {
			s.log.Error("invalid graph configuration", zap.Error(err))
		}
		return
	}
	for _, node := range cfg.Nodes {
		err = s.graph.Configure(node.ID, node.Config)
		if err != nil {
			s.log.Error("failed to configure node", zap.String("node", node.ID), zap.Error(err))
		}
	}
}

func (s *Service) restartGraph(cfg FlowConfig) error {
	graph, graphCtx, graphCancel, err := s.buildGraph(cfg)
	if err != nil {
		return fmt.Errorf("failed to build graph: %w", err)
	}
	s.graphCancel()
	<-s.graphRunning
	s.graphHash = cfg.treeHash()
	s.graphRunning = make(chan struct{})
	s.graph = graph
	s.graphCtx = graphCtx
	s.graphCancel = graphCancel
	go func() {
		s.graph.Run()
		s.log.Info("flow stopped")
		s.graphCancel()
		s.graph = nil
		s.graphCtx = nil
		s.graphCancel = nil
		close(s.graphRunning)
	}()

	return nil
}

func (s *Service) startGraph(cfg FlowConfig) error {
	graph, graphCtx, graphCancel, err := s.buildGraph(cfg)
	if err != nil {
		return fmt.Errorf("failed to build graph: %w", err)
	}
	s.graphHash = cfg.treeHash()
	s.graphRunning = make(chan struct{})
	s.graph = graph
	s.graphCtx = graphCtx
	s.graphCancel = graphCancel
	go func() {
		s.graph.Run()
		s.log.Info("flow stopped")
		s.graphCancel()
		s.graph = nil
		s.graphCtx = nil
		s.graphCancel = nil
		close(s.graphRunning)
	}()

	return nil
}

func (s *Service) buildGraph(cfg FlowConfig) (*Graph, context.Context, context.CancelFunc, error) {
	b := NewGraphBuilder(s.log, s.registry, s.bus)

	for _, node := range cfg.Nodes {
		b = b.AddNode(node.Type, node.ID, node.To)
	}
	if err := b.Validate(); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to validate graph: %w", err)
	}
	graphCtx, graphCancel := context.WithCancel(s.ctx)
	graph, err := b.Build(graphCtx)
	if err != nil {
		graphCancel()
		return nil, nil, nil, fmt.Errorf("failed to build graph: %w", err)
	}
	for _, node := range cfg.Nodes {
		err := graph.Configure(node.ID, node.Config)
		if err != nil {
			graphCancel()
			return nil, nil, nil, fmt.Errorf("failed to configure node %s: %w", node.ID, err)
		}
	}
	return graph, graphCtx, graphCancel, nil
}
