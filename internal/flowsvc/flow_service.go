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

	mu          sync.Mutex
	ctx         context.Context
	graphCtx    context.Context
	graphCancel context.CancelFunc
	graph       *GraphV2
	bus         *FlowBus
	running     chan struct{}

	registry *Registry
}

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
		ctx        context.Context
		nodeID     string
		nodeIDs    []string
		subscriber FlowSubscriber
		publishers map[string]FlowPublisher
	}
)

func NewFlowStream(ctx context.Context, nodeID string, subscriber FlowSubscriber, publishers map[string]FlowPublisher) FlowStream {
	nodeIDs := make([]string, 0, len(publishers))
	for nodeID := range publishers {
		nodeIDs = append(nodeIDs, nodeID)
	}
	return FlowStream{
		ctx:        ctx,
		nodeID:     nodeID,
		nodeIDs:    nodeIDs,
		subscriber: subscriber,
		publishers: publishers,
	}
}

func (f FlowStream) Publish(toNodeID string, msg FlowEvent) {
	ctx, cancel := context.WithTimeout(f.ctx, 100*time.Microsecond)
	f.publishers[toNodeID](ctx, msg)
	cancel()
}

func (f FlowStream) Broadcast(msg FlowEvent) {
	for _, nodeID := range f.nodeIDs {
		f.Publish(nodeID, msg)
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

func New(
	log *zap.Logger,
	config *configsvc.Service,
	flowPath string,
	hidSvc *hidsvc.Service,
	registry *Registry,
) *Service {
	return &Service{
		config:   config,
		log:      log,
		flowPath: flowPath,
		hid:      hidSvc,
		bus:      bus.NewBus[FlowEventKey, FlowEvent](log),
		registry: registry,
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

	err = s.startGraph(cfg)
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
}

func (s *Service) startGraph(cfg FlowConfig) error {
	b := NewGraphBuilder(s.log, s.registry, s.hid)

	for _, node := range cfg.Nodes {
		b = b.AddNode(node.Type, node.ID, node.To)
	}
	if err := b.Validate(); err != nil {
		return fmt.Errorf("failed to validate graph: %w", err)
	}
	graph, err := b.Build()
	if err != nil {
		return fmt.Errorf("failed to build graph: %w", err)
	}
	for _, node := range cfg.Nodes {
		err := graph.Configure(node.ID, node.Config)
		if err != nil {
			return fmt.Errorf("failed to configure node %s: %w", node.ID, err)
		}
	}
	s.graphCtx, s.graphCancel = context.WithCancel(s.ctx)
	s.graph = graph
	go func() {
		<-s.graphCtx.Done()
		s.log.Info("flow cancelled")
		close(s.running)
	}()
	go func() {
		err := s.graph.Run(s.graphCtx, s.bus)
		if err != nil {
			s.log.Error("flow failed", zap.Error(err))
		}
		s.log.Error("flow stopped")
	}()

	return nil
}
