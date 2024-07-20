package hidnodes

import (
	"context"
	"encoding/json"

	"github.com/neuroplastio/neuroplastio/internal/flowsvc"
	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"github.com/neuroplastio/neuroplastio/pkg/hidevent"
	"go.uber.org/zap"
)

type Mux struct{}

func (f Mux) Metadata() flowsvc.NodeMetadata {
	return flowsvc.NodeMetadata{
		DisplayName: "Mux",

		UpstreamType:   flowsvc.NodeTypeMany,
		DownstreamType: flowsvc.NodeTypeMany,
	}
}

type MuxRunner struct {
	id           string
	defaultRoute string
	log          *zap.Logger

	activatedUsages map[hidparse.Usage]int
	valueSub        flowsvc.FlowStateSubscriber
	nodeIDs         map[int]string
	defaultRouteIdx int
}

func (f Mux) Runner(info flowsvc.NodeInfo, config json.RawMessage, provider flowsvc.NodeRunnerProvider) (flowsvc.NodeRunner, error) {
	cfg := &muxConfig{}
	if len(info.Downstreams) > 0 {
		cfg.Fallback = info.Downstreams[0]
	}
	if err := json.Unmarshal(config, cfg); err != nil {
		return nil, err
	}
	nodeIDs := make(map[int]string, len(info.Downstreams))
	enumValues := make(map[string]int, len(info.Downstreams))
	defaultRouteIdx := 0
	for i, down := range info.Downstreams {
		enumValues[down] = i
		nodeIDs[i] = down
		if down == cfg.Fallback {
			defaultRouteIdx = i
		}
	}
	valueSub, err := provider.State().RegisterEnum(info.ID, enumValues, defaultRouteIdx)
	if err != nil {
		return nil, err
	}
	return &MuxRunner{
		id:           info.ID,
		defaultRoute: cfg.Fallback,
		log:          provider.Log(),

		activatedUsages: make(map[hidparse.Usage]int, 0),
		valueSub:        valueSub,
		nodeIDs:         nodeIDs,
		defaultRouteIdx: defaultRouteIdx,
	}, nil
}

type muxConfig struct {
	Fallback string `json:"fallback"`
}

func (r *MuxRunner) Run(ctx context.Context, up flowsvc.FlowStream, down flowsvc.FlowStream) error {
	currentRoute := r.defaultRouteIdx
	in := up.Subscribe(ctx)
	valueCh := r.valueSub(ctx)
	for {
		select {
		case event := <-valueCh:
			currentRoute = *event.Message.Value.Int
		case event := <-in:
			hidEvent := event.Message.HIDEvent
			deactEvents := make(map[int]*hidevent.HIDEvent)
			for _, usage := range hidEvent.Usages() {
				if usage.Activate == nil {
					continue
				}
				if *usage.Activate {
					if prev, ok := r.activatedUsages[usage.Usage]; ok && prev != currentRoute {
						ev, ok := deactEvents[prev]
						if !ok {
							ev = hidevent.NewHIDEvent()
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
							ev = hidevent.NewHIDEvent()
							deactEvents[prev] = ev
						}
						ev.Deactivate(usage.Usage)
						hidEvent.Suppress(usage.Usage)
					}
					delete(r.activatedUsages, usage.Usage)
				}
			}
			for route, ev := range deactEvents {
				down.Publish(r.nodeIDs[route], flowsvc.FlowEvent{
					HIDEvent: *ev,
				})
			}
			if !hidEvent.IsEmpty() {
				down.Publish(r.nodeIDs[currentRoute], flowsvc.FlowEvent{
					HIDEvent: hidEvent,
				})
			}
		case <-ctx.Done():
			return nil
		}
	}
}
