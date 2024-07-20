package hidnodes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/neuroplastio/neuroplastio/internal/flowsvc"
	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"github.com/neuroplastio/neuroplastio/internal/hidsvc"
	"github.com/neuroplastio/neuroplastio/pkg/hidevent"
)

type Input struct{}

func (i Input) Metadata() flowsvc.NodeMetadata {
	return flowsvc.NodeMetadata{
		DisplayName: "Input",

		UpstreamType:   flowsvc.NodeTypeNone,
		DownstreamType: flowsvc.NodeTypeMany,
	}
}

func (i Input) Runner(info flowsvc.NodeInfo, config json.RawMessage, provider flowsvc.NodeRunnerProvider) (flowsvc.NodeRunner, error) {
	cfg := &inputConfig{}
	if err := json.Unmarshal(config, cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	dev, err := provider.HID().GetInputDeviceHandle(cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to get input device %s: %w", cfg.Addr, err)
	}
	desc, err := flowsvc.NewHIDReportDescriptorFromRaw(dev.InputDevice().BackendDevice.ReportDescriptor)
	if err != nil {
		return nil, fmt.Errorf("failed to create HID report descriptor: %w", err)
	}
	return &InputRunner{
		dev:  dev,
		desc: desc,
		rte:  hidevent.NewRTE(provider.Log(), desc.Parsed().GetInputDataItems()),
		etr:  hidevent.NewETR(provider.Log(), desc.Parsed().GetOutputDataItems()),
	}, nil

}

type inputConfig struct {
	Addr hidsvc.Address `json:"addr"`
}

type InputRunner struct {
	dev  *hidsvc.InputDeviceHandle
	desc flowsvc.HIDReportDescriptor
	rte  *hidevent.RTETranscoder
	etr  *hidevent.ETRTranscoder
}

func (g *InputRunner) Run(ctx context.Context, up flowsvc.FlowStream, down flowsvc.FlowStream) error {
	read := make(chan []byte)
	write := make(chan []byte)
	sub := down.Subscribe(ctx)
	go func() {
		defer close(write)
		for {
			select {
			case event := <-sub:
				reports := g.etr.OnEvent(event.Message.HIDEvent)
				for _, report := range reports {
					write <- hidparse.EncodeReport(report)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		defer close(read)
		for {
			select {
			case data := <-read:
				report, ok := hidparse.ParseInputReport(g.desc.Parsed(), data)
				if !ok {
					fmt.Println("Failed to parse report")
					continue
				}
				event := g.rte.OnReport(report)
				down.Broadcast(flowsvc.FlowEvent{HIDEvent: event})
			case <-ctx.Done():
				return
			}
		}
	}()
	return g.dev.Start(ctx, read, write)
}
