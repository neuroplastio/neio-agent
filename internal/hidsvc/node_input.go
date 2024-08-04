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
	source := hidapi.NewEventSource(g.log.Named("source"), itemSet.WithType(hiddesc.MainItemTypeInput))
	err = source.InitReports(dev.GetInputReport)
	if err != nil {
		dev.Close()
		g.log.Error("Failed to initialize input reports", zap.Error(err))
		return
	}

	release, err := dev.Acquire()
	if err != nil {
		dev.Close()
		g.log.Error("Failed to acquire input device", zap.Error(err))
		return
	}
	g.log.Info("Input device acquired", zap.String("addr", g.addr.String()))

	go func() {
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
