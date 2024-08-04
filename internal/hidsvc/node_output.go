package hidsvc

import (
	"context"
	"fmt"

	"github.com/neuroplastio/neio-agent/flowapi"
	"github.com/neuroplastio/neio-agent/hidapi"
	"github.com/neuroplastio/neio-agent/hidapi/hiddesc"
	"go.uber.org/zap"
)

type OutputNodeType struct {
	log *zap.Logger
	hid *Service
}

func (o OutputNodeType) Descriptor() flowapi.NodeTypeDescriptor {
	return flowapi.NodeTypeDescriptor{
		DisplayName: "Output",

		UpstreamType:   flowapi.NodeLinkTypeMany,
		DownstreamType: flowapi.NodeLinkTypeNone,
	}
}

func (o OutputNodeType) CreateNode(p flowapi.NodeProvider) (flowapi.Node, error) {
	return &OutputNode{
		id:  p.Info().ID,
		log: o.log.With(zap.String("nodeId", p.Info().ID)),
		hid: o.hid,
	}, nil
}

type outputConfig struct {
	Addr       Address                `yaml:"addr"`
	Descriptor outputDescriptorConfig `yaml:"descriptor"`
}

type outputDescriptorConfig struct {
	Inputs []Address `yaml:"inputs"`
}

type OutputNode struct {
	id  string
	log *zap.Logger
	hid *Service

	addr       Address
	descriptor outputDescriptorConfig
}

func (o *OutputNode) Configure(c flowapi.NodeConfigurator) error {
	cfg := outputConfig{}
	if err := c.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	o.addr = cfg.Addr
	o.descriptor = cfg.Descriptor
	if len(o.descriptor.Inputs) == 0 {
		return fmt.Errorf("no input devices specified")
	}
	return nil
}

func (o *OutputNode) handleDevice(ctx context.Context, eventCh chan flowapi.Event) {
	desc, err := o.buildDescriptor()
	if err != nil {
		o.log.Error("Failed to build HID report descriptor", zap.Error(err))
		return
	}
	descRaw, err := hiddesc.Encode(desc)
	if err != nil {
		o.log.Error("Failed to encode HID report descriptor", zap.Error(err))
		return
	}
	dev, err := o.hid.OpenOutputDevice(o.addr, descRaw)
	if err != nil {
		o.log.Error("Failed to open output device", zap.Error(err))
		return
	}
	defer dev.Close()
	itemSet := hidapi.NewDataItemSet(desc)
	sink := hidapi.NewEventSink(o.log.Named("sink"), itemSet.WithType(hiddesc.MainItemTypeInput))

	for {
		select {
		case event := <-eventCh:
			reports := sink.OnEvent(event.HID)
			for _, report := range reports {
				_, err := dev.Write(hidapi.EncodeReport(report).Bytes())
				if err != nil {
					o.log.Error("Failed to write output report", zap.Error(err))
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (o *OutputNode) buildDescriptor() (hiddesc.ReportDescriptor, error) {
	desc := hiddesc.ReportDescriptor{}
	idMap := make(map[uint8]uint8)
	for _, addr := range o.descriptor.Inputs {
		inputDev, err := o.hid.OpenInputDevice(addr)
		if err != nil {
			// not connected
			continue
		}
		reportDesc, err := inputDev.GetReportDescriptor()
		_ = inputDev.Close()
		if err != nil {
			o.log.Error("Failed to get report descriptor", zap.Error(err), zap.String("addr", addr.String()))
			continue
		}
		inputDesc, err := hiddesc.Decode(reportDesc)
		if err != nil {
			o.log.Error("Failed to decode HID report descriptor", zap.Error(err))
			continue
		}
		inputSet := hidapi.NewDataItemSet(inputDesc)
		for _, rd := range inputSet.Reports() {
			id := rd.ID
			for {
				if _, ok := idMap[id]; !ok {
					idMap[rd.ID] = id
					break
				}
				id++
			}
		}
		inputDesc.Walk(func(item hiddesc.MainItem) bool {
			if item.DataItem != nil {
				item.DataItem.ReportID = idMap[item.DataItem.ReportID]
			}
			return true
		})
		desc.Collections = append(desc.Collections, inputDesc.Collections...)
	}
	if len(desc.Collections) == 0 {
		return desc, fmt.Errorf("no input devices connected")
	}
	return desc, nil
}

func (o *OutputNode) Run(ctx context.Context, up flowapi.Stream, _ flowapi.Stream) error {
	deviceEvents := o.hid.outputBus.Subscribe(ctx, OutputBusKey{
		Type: OutputConnected,
		Addr: o.addr,
	}, OutputBusKey{
		Type: OutputDisconnected,
		Addr: o.addr,
	})
	var deviceCtx context.Context
	var cancel context.CancelFunc
	events := up.Subscribe(ctx)
	eventCh := make(chan flowapi.Event)
	defer close(eventCh)
	inputDeviceKeys := make([]InputBusKey, len(o.descriptor.Inputs)*2)
	for _, addr := range o.descriptor.Inputs {
		inputDeviceKeys = append(inputDeviceKeys, InputBusKey{
			Type: InputConnected,
			Addr: addr,
		}, InputBusKey{
			Type: InputDisconnected,
			Addr: addr,
		})
	}
	inputDeviceEvents := o.hid.inputBus.Subscribe(ctx, inputDeviceKeys...)

	isConnected := o.hid.IsOutputConnected(o.addr)
	if isConnected {
		deviceCtx, cancel = context.WithCancel(ctx)
		go o.handleDevice(deviceCtx, eventCh)
	}
	go func() {
		for {
			select {
			case event := <-events:
				select {
				case eventCh <- event:
				default:
					o.log.Warn("Dropped event", zap.Any("event", event))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	for {
		select {
		case ev := <-inputDeviceEvents:
			if cancel == nil {
				break
			}
			o.log.Info("Restarting output device", zap.Any("event", ev))
			cancel()
			deviceCtx, cancel = context.WithCancel(ctx)
			go o.handleDevice(deviceCtx, eventCh)
		case ev := <-deviceEvents:
			switch ev.Key.Type {
			case OutputConnected:
				if cancel != nil {
					break
				}
				o.log.Info("Output device connected", zap.String("addr", o.addr.String()))
				deviceCtx, cancel = context.WithCancel(ctx)
				go o.handleDevice(deviceCtx, eventCh)
			case OutputDisconnected:
				if cancel == nil {
					break
				}
				o.log.Info("Output device disconnected", zap.String("addr", o.addr.String()))
				cancel()
				deviceCtx = nil
				cancel = nil
			}
		case <-ctx.Done():
			if cancel != nil {
				cancel()
			}
			return nil
		}
	}
}
