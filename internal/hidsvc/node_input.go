package hidsvc

import (
	"context"
	"fmt"

	"github.com/neuroplastio/neio-agent/flowapi"
	"github.com/neuroplastio/neio-agent/hidapi"
	"github.com/neuroplastio/neio-agent/hidapi/hiddesc"
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
		id:  p.Info().ID,
		log: i.log.With(zap.String("nodeId", p.Info().ID)),
		hid: i.hid,
	}, nil
}

type inputConfig struct {
	Addr Address `yaml:"addr"`
}

type InputNode struct {
	log     *zap.Logger
	id      string
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
	dev, err := g.hid.GetInputDeviceHandle(cfg.Addr, g.id)
	if err != nil {
		return fmt.Errorf("failed to get input device %s: %w", cfg.Addr, err)
	}

	desc, err := hiddesc.Decode(dev.InputDevice().BackendDevice.ReportDescriptor)
	if err != nil {
		return fmt.Errorf("failed to decode HID report descriptor: %w", err)
	}
	g.dev = dev
	itemSet := hidapi.NewDataItemSet(desc)
	g.decoder = hidapi.NewReportDecoder(itemSet.WithType(hiddesc.MainItemTypeInput))
	g.source = hidapi.NewEventSource(g.log.Named("source"), itemSet.WithType(hiddesc.MainItemTypeInput))
	g.sink = hidapi.NewEventSink(g.log.Named("sink"), itemSet.WithType(hiddesc.MainItemTypeOutput))
	return nil
}

func (g *InputNode) Run(ctx context.Context, up flowapi.Stream, down flowapi.Stream) error {
	read := make(chan []byte)
	write := make(chan []byte)
	sub := down.Subscribe(ctx)
	// TODO: query GetInputReport for each reportID and send through the pipeline
	//   (and simplify how we open and close the device)
	go func() {
		<-ctx.Done()
		release()
		g.log.Info("Input device released", zap.String("addr", g.addr.String()))
		dev.Close()
		g.log.Info("Input device closed", zap.String("addr", g.addr.String()))
	}()

	buf := make([]byte, 2048) // TODO: calculate from the descriptor (only for standard input devices)
	for {
		n, err := dev.Read(buf)
		if err != nil {
			g.log.Error("Failed to read from device, releasing", zap.Error(err))
			return
		}
		if ctx.Err() != nil {
			return
		}
		if n > 0 {
			event := source.OnReport(buf[:n])
			if !event.IsEmpty() {
				g.log.Debug("event", zap.String("event", event.String()))
				down.Broadcast(flowapi.Event{
					HID: event,
				})
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
				if !event.IsEmpty() {
					down.Broadcast(flowapi.Event{
						HID: event,
					})
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return g.dev.Start(ctx, read, write)
}
