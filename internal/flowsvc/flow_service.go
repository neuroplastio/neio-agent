package flowsvc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/neuroplastio/neuroplastio/internal/configsvc"
	"github.com/neuroplastio/neuroplastio/internal/hidsvc"
	"github.com/neuroplastio/neuroplastio/pkg/bus"
	"go.uber.org/zap"
)

type Service struct {
	config   *configsvc.Service
	log      *zap.Logger
	hid      *hidsvc.Service
	flowPath string

	mu         sync.Mutex
	ctx        context.Context
	flowCtx    context.Context
	flowCancel context.CancelFunc
	flow       *CompiledGraph
	bus        *FlowBus
	running    chan struct{}

	nodeRegistry *NodeRegistry

	state *FlowState
}

// func() Node
// func(cfg Config) Node
// func(cfg Config) (Node, error)
// func(cfg Config, hidSvc *hidsvc.Service) (Node, error)
type NodeCreator any

type (
	FlowEventType uint8
	FlowEventKey  struct {
		NodeID string
		Type   FlowEventType
	}
	FlowBus        = bus.Bus[FlowEventKey, FlowEvent]
	FlowPublisher  = bus.Publisher[FlowEvent]
	FlowSubscriber = bus.Subscriber[FlowEventKey, FlowEvent]
	FlowStream     struct {
		nodeID     string
		nodeIDs    []string
		subscriber FlowSubscriber
		publishers map[string]FlowPublisher
	}
)

func NewFlowStream(nodeID string, subscriber FlowSubscriber, publishers map[string]FlowPublisher) FlowStream {
	nodeIDs := make([]string, 0, len(publishers))
	for nodeID := range publishers {
		nodeIDs = append(nodeIDs, nodeID)
	}
	return FlowStream{
		nodeID:     nodeID,
		nodeIDs:    nodeIDs,
		subscriber: subscriber,
		publishers: publishers,
	}
}

func (f FlowStream) Publish(ctx context.Context, toNodeID string, msg FlowEvent) {
	msg.SourceNodeID = f.nodeID
	ctx, cancel := context.WithTimeout(ctx, 100*time.Microsecond)
	f.publishers[toNodeID](ctx, msg)
	cancel()
}

func (f FlowStream) Broadcast(ctx context.Context, msg FlowEvent) {
	for _, nodeID := range f.nodeIDs {
		f.Publish(ctx, nodeID, msg)
	}
}

func (f FlowStream) NodeIDs() []string {
	return f.nodeIDs
}

func (f FlowStream) Subscribe(ctx context.Context) <-chan bus.Message[FlowEventKey, FlowEvent] {
	return f.subscriber(ctx)
}

const (
	FlowEventDownstream FlowEventType = iota
	FlowEventUpstream
)

func New(log *zap.Logger, config *configsvc.Service, flowPath string, hidSvc *hidsvc.Service) *Service {
	state := NewState(log)
	actionRegistry := NewActionRegistry(state)
	return &Service{
		config:       config,
		log:          log,
		flowPath:     flowPath,
		hid:          hidSvc,
		bus:          bus.NewBus[FlowEventKey, FlowEvent](log),
		nodeRegistry: NewNodeRegistry(log, hidSvc, actionRegistry, state),
		state:        state,
	}
}

func (s *Service) Start(ctx context.Context) error {
	s.ctx = ctx
	s.running = make(chan struct{})
	select {
	case <-ctx.Done():
		return nil
	case <-s.hid.Ready():
	}
	select {
	case <-ctx.Done():
		return nil
	case <-s.config.Ready():
	}
	cfg, err := configsvc.Register(s.config, s.flowPath, FlowConfig{}, s.onConfigChange)
	if err != nil {
		return fmt.Errorf("failed to register config: %w", err)
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

	err = s.state.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start flow state: %w", err)
	}

	err = s.compile(cfg)
	if err != nil {
		return fmt.Errorf("failed to compile flow: %w", err)
	}
	<-s.running
	return nil
}

func (s *Service) onConfigChange(cfg FlowConfig, err error) {
	if err != nil {
		s.log.Error("failed to parse config", zap.Error(err))
		return
	}
	err = s.compile(cfg)
	if err != nil {
		s.log.Error("failed to compile flow", zap.Error(err))
	}
}

func (s *Service) compile(cfg FlowConfig) error {
	g := NewGraph()

	nodeIndex := make(map[string]Node)
	for _, node := range cfg.Nodes {
		n, err := s.nodeRegistry.New(node.Type, node.Config)
		if err != nil {
			return fmt.Errorf("failed to create node %s (%s): %w", node.ID, node.Type, err)
		}
		nodeIndex[node.ID] = n
		g.AddNode(node.ID, n)
	}

	for _, link := range cfg.Links {
		err := g.Connect(link.From, link.To)
		if err != nil {
			return fmt.Errorf("failed to connect %s -> %s: %w", link.From, link.To, err)
		}
	}

	compiled, err := NewCompiler(s.log, g).Compile()
	if err != nil {
		return fmt.Errorf("failed to compile graph: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.flowCancel != nil {
		s.flowCancel()
	}
	s.flowCtx, s.flowCancel = context.WithCancel(s.ctx)
	s.flow = compiled
	go func() {
		<-s.flowCtx.Done()
		s.log.Info("flow cancelled")
	}()
	go func() {
		// TODO: account for flow restarts
		defer close(s.running)
		err := s.flow.Start(s.flowCtx, s.bus)
		if err != nil {
			s.log.Error("flow failed", zap.Error(err))
		}
		s.log.Error("flow stopped")
	}()

	return nil
}
