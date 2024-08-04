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
	log *zap.Logger
	id  string
	hid *Service

	addr Address
	done chan struct{}
}

func (g *InputNode) Configure(c flowapi.NodeConfigurator) error {
	cfg := inputConfig{}
	if err := c.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	g.addr = cfg.Addr
	return nil
}

func (g *InputNode) handleDevice(ctx context.Context, down flowapi.Stream) {
	defer close(g.done)
	dev, err := g.hid.OpenInputDevice(g.addr)
	if err != nil {
		g.log.Error("Failed to open input device", zap.Error(err))
		return
	}
	descRaw, err := dev.GetReportDescriptor()
	if err != nil {
		dev.Close()
		g.log.Error("Failed to get report descriptor", zap.Error(err))
		return
	}
	desc, err := hiddesc.Decode(descRaw)
	if err != nil {
		dev.Close()
		g.log.Error("Failed to decode HID report descriptor", zap.Error(err))
		return
	}
	itemSet := hidapi.NewDataItemSet(desc)
	inputState := hidapi.NewReportState(g.log.Named("input"), itemSet.WithType(hiddesc.MainItemTypeInput))
	inputEvents, err := inputState.InitReports(dev.GetInputReport)
	if err != nil {
		dev.Close()
		g.log.Error("Failed to initialize input reports", zap.Error(err))
		return
	}
	featureState := hidapi.NewReportState(g.log.Named("feature"), itemSet.WithType(hiddesc.MainItemTypeFeature))
	featureEvents, err := inputState.InitReports(dev.GetFeatureReport)
	if err != nil {
		dev.Close()
		g.log.Error("Failed to initialize feature reports", zap.Error(err))
		return
	}
	outputState := hidapi.NewReportState(g.log.Named("output"), itemSet.WithType(hiddesc.MainItemTypeOutput))

	release, err := dev.Acquire()
	if err != nil {
		dev.Close()
		g.log.Error("Failed to acquire input device", zap.Error(err))
		return
	}
	g.log.Info("Input device acquired", zap.String("addr", g.addr.String()))
	downEvents := down.Subscribe(ctx)

	for _, event := range inputEvents {
		down.Broadcast(flowapi.Event{
			Type: flowapi.HIDEventTypeInput,
			HID:  event,
		})
	}

	for _, event := range featureEvents {
		down.Broadcast(flowapi.Event{
			Type: flowapi.HIDEventTypeFeature,
			HID:  event,
		})
	}

	go func() {
		// Input reports
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
				event := inputState.ApplyReport(buf[:n])
				if !event.IsEmpty() {
					g.log.Debug("event", zap.String("event", event.String()))
					down.Broadcast(flowapi.Event{
						Type: flowapi.HIDEventTypeInput,
						HID:  event,
					})
				}
			}
		}
	}()
	go func() {
		// Output and Feature reports
		for {
			select {
			case event := <-downEvents:
				switch event.Type {
				case flowapi.HIDEventTypeOutput:
					reports := outputState.ApplyEvent(event.HID)
					for _, report := range reports {
						_, err := dev.Write(report)
						if err != nil {
							g.log.Error("Failed to write output report", zap.Error(err))
						}
					}
				case flowapi.HIDEventTypeFeature:
					reports := featureState.ApplyEvent(event.HID)
					for _, report := range reports {
						_, err := dev.SetFeatureReport(report)
						if err != nil {
							g.log.Error("Failed to write feature report", zap.Error(err))
						}
					}
				default:
					g.log.Error("Unknown HID event type", zap.Any("type", event.Type))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	<-ctx.Done()
	release()
	g.log.Info("Input device released", zap.String("addr", g.addr.String()))
	dev.Close()
	g.log.Info("Input device closed", zap.String("addr", g.addr.String()))
}

func (g *InputNode) Run(ctx context.Context, _ flowapi.Stream, down flowapi.Stream) error {
	deviceEvents := g.hid.inputBus.Subscribe(ctx, InputBusKey{
		Type: InputConnected,
		Addr: g.addr,
	}, InputBusKey{
		Type: InputDisconnected,
		Addr: g.addr,
	})
	var deviceCtx context.Context
	var cancel context.CancelFunc
	isConnected := g.hid.IsInputConnected(g.addr)
	if isConnected {
		deviceCtx, cancel = context.WithCancel(ctx)
		g.done = make(chan struct{})
		go g.handleDevice(deviceCtx, down)
	}
	for {
		select {
		case ev := <-deviceEvents:
			switch ev.Key.Type {
			case InputConnected:
				if cancel != nil {
					break
				}
				g.log.Info("Input device connected", zap.String("addr", g.addr.String()))
				deviceCtx, cancel = context.WithCancel(ctx)
				g.done = make(chan struct{})
				go g.handleDevice(deviceCtx, down)
			case InputDisconnected:
				if cancel == nil {
					break
				}
				g.log.Info("Input device disconnected", zap.String("addr", g.addr.String()))
				cancel()
				<-g.done
				deviceCtx = nil
				cancel = nil
				g.done = nil
			}
		case <-ctx.Done():
			if cancel != nil {
				cancel()
				<-g.done
			}
			return nil
		}
	}
}
