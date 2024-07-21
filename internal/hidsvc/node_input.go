package hidsvc

import (
	"context"
	"fmt"

	"github.com/neuroplastio/neuroplastio/flowapi"
	"github.com/neuroplastio/neuroplastio/hidapi"
	"github.com/neuroplastio/neuroplastio/hidapi/hiddesc"
	"go.uber.org/zap"
)

type InputNodeType struct {
	log *zap.Logger
	hid *Service
}

func (i InputNodeType) Descriptor() flowapi.NodeTypeDescriptor {
	return flowapi.NodeTypeDescriptor{
		DisplayName: "Input",

		UpstreamType:   flowapi.NodeLinkTypeNone,
		DownstreamType: flowapi.NodeLinkTypeMany,
	}
}

func (i InputNodeType) CreateNode(p flowapi.NodeProvider) (flowapi.Node, error) {
	return &InputNode{
		log: i.log,
		hid: i.hid,
	}, nil
}

type inputConfig struct {
	Addr Address `json:"addr"`
}

type InputNode struct {
	log     *zap.Logger
	hid     *Service
	dev     *InputDeviceHandle
	decoder *hidapi.ReportDecoder
	source  *hidapi.EventSource
	sink    *hidapi.EventSink
}

func (g *InputNode) Configure(c flowapi.NodeConfigurator) error {
	cfg := inputConfig{}
	if err := c.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	dev, err := g.hid.GetInputDeviceHandle(cfg.Addr)
	if err != nil {
		return fmt.Errorf("failed to get input device %s: %w", cfg.Addr, err)
	}

	desc, err := hiddesc.Decode(dev.InputDevice().BackendDevice.ReportDescriptor)
	if err != nil {
		return fmt.Errorf("failed to decode HID report descriptor: %w", err)
	}
	g.dev = dev
	g.decoder = hidapi.NewInputReportDecoder(desc)
	g.source = hidapi.NewEventSource(g.log, desc.GetInputDataItems())
	g.sink = hidapi.NewEventSink(g.log, desc.GetOutputDataItems())
	return nil
}

func (g *InputNode) Run(ctx context.Context, up flowapi.Stream, down flowapi.Stream) error {
	read := make(chan []byte)
	write := make(chan []byte)
	sub := down.Subscribe(ctx)
	go func() {
		defer close(write)
		for {
			select {
			case event := <-sub:
				reports := g.sink.OnEvent(event.HID)
				for _, report := range reports {
					write <- hidapi.EncodeReport(report).Bytes()
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
				report, ok := g.decoder.Decode(data)
				if !ok {
					g.log.Error("Failed to parse report")
					continue
				}
				event := g.source.OnReport(report)
				down.Broadcast(flowapi.Event{
					HID: event,
				})
			case <-ctx.Done():
				return
			}
		}
	}()
	return g.dev.Start(ctx, read, write)
}