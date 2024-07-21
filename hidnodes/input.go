package hidnodes

import (
	"context"
	"fmt"

	"github.com/neuroplastio/neuroplastio/internal/flowsvc"
	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"github.com/neuroplastio/neuroplastio/internal/hidsvc"
	"github.com/neuroplastio/neuroplastio/pkg/hidevent"
	"go.uber.org/zap"
)

type Input struct{}

func (i Input) Descriptor() flowsvc.NodeTypeDescriptor {
	return flowsvc.NodeTypeDescriptor{
		DisplayName: "Input",

		UpstreamType:   flowsvc.NodeLinkTypeNone,
		DownstreamType: flowsvc.NodeLinkTypeMany,
	}
}

func (i Input) CreateNode(p flowsvc.RunnerProvider) (flowsvc.Node, error) {
	return &InputRunner{
		log: p.Log(),
	}, nil
}

type inputConfig struct {
	Addr hidsvc.Address `json:"addr"`
}

type InputRunner struct {
	log  *zap.Logger
	dev  *hidsvc.InputDeviceHandle
	desc flowsvc.HIDReportDescriptor
	rte  *hidevent.RTETranscoder
	etr  *hidevent.ETRTranscoder
}

func (g *InputRunner) Configure(c flowsvc.NodeConfigurator) error {
	cfg := inputConfig{}
	if err := c.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	dev, err := c.HID().GetInputDeviceHandle(cfg.Addr)
	if err != nil {
		return fmt.Errorf("failed to get input device %s: %w", cfg.Addr, err)
	}
	desc, err := flowsvc.NewHIDReportDescriptorFromRaw(dev.InputDevice().BackendDevice.ReportDescriptor)
	if err != nil {
		return fmt.Errorf("failed to create HID report descriptor: %w", err)
	}
	g.dev = dev
	g.desc = desc
	g.rte = hidevent.NewRTE(g.log, desc.Parsed().GetInputDataItems())
	g.etr = hidevent.NewETR(g.log, desc.Parsed().GetOutputDataItems())
	return nil
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
				reports := g.etr.OnEvent(*event.Message.HIDEvent)
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
				down.Broadcast(flowsvc.FlowEvent{HIDEvent: &event})
			case <-ctx.Done():
				return
			}
		}
	}()
	return g.dev.Start(ctx, read, write)
}
