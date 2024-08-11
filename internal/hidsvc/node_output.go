package hidsvc

import (
	"context"
	"errors"
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

	addr    Address
	desc    hiddesc.ReportDescriptor
	descRaw []byte

	inputState   *hidapi.ReportState
	outputState  *hidapi.ReportState
	featureState *hidapi.ReportState
}

func (o *OutputNode) Configure(c flowapi.NodeConfigurator) error {
	cfg := outputConfig{}
	if err := c.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	o.addr = cfg.Addr
	desc, err := o.buildDescriptor(cfg.Descriptor)
	if err != nil {
		return fmt.Errorf("failed to build HID report descriptor: %w", err)
	}
	o.desc = desc
	o.descRaw, err = hiddesc.Encode(desc)
	if err != nil {
		return fmt.Errorf("failed to encode HID report descriptor: %w", err)
	}

	itemSet := hidapi.NewDataItemSet(o.desc)
	o.inputState = hidapi.NewReportState(o.log.Named("input"), itemSet.WithType(hiddesc.MainItemTypeInput))
	o.outputState = hidapi.NewReportState(o.log.Named("output"), itemSet.WithType(hiddesc.MainItemTypeOutput))
	o.featureState = hidapi.NewReportState(o.log.Named("feature"), itemSet.WithType(hiddesc.MainItemTypeFeature))

	return nil
}

type outDevHandler struct {
	inputState   *hidapi.ReportState
	outputState  *hidapi.ReportState
	featureState *hidapi.ReportState
}

func (o *outDevHandler) GetInputReport(reportID uint8) ([]byte, error) {
	return o.inputState.GetReport(reportID)
}

func (o *outDevHandler) GetOutputReport(reportID uint8) ([]byte, error) {
	return o.outputState.GetReport(reportID)
}

func (o *outDevHandler) GetFeatureReport(reportID uint8) ([]byte, error) {
	return o.featureState.GetReport(reportID)
}

func (o *outDevHandler) SetOutputReport(reportID uint8, data []byte) error {
	if data[0] != reportID {
		return fmt.Errorf("report ID mismatch")
	}
	o.outputState.ApplyReport(data)
	return nil
}

func (o *outDevHandler) SetFeatureReport(reportID uint8, data []byte) error {
	if data[0] != reportID {
		return fmt.Errorf("report ID mismatch")
	}
	o.featureState.ApplyReport(data)
	return nil
}

func (o *OutputNode) handleDevice(ctx context.Context, up flowapi.Stream, inputCh chan [][]byte) {
	handler := &outDevHandler{
		inputState:   o.inputState,
		outputState:  o.outputState,
		featureState: o.featureState,
	}
	dev, err := o.hid.OpenOutputDevice(o.addr, handler, o.descRaw)
	if err != nil {
		o.log.Error("Failed to open output device", zap.Error(err))
		return
	}
	defer dev.Close()

	go func() {
		// Output report reader
		buf := make([]byte, 1024)
		for {
			n, err := dev.Read(buf)
			if errors.Is(err, context.Canceled) {
				return
			}
			if err != nil {
				o.log.Error("Failed to read output report", zap.Error(err))
				return
			}
			o.log.Debug("Read output report", zap.Any("size", n), zap.Any("reportID", buf[0]))
			if ctx.Err() != nil {
				return
			}
			event := o.outputState.ApplyReport(buf[:n])
			if !event.IsEmpty() {
				up.Broadcast(flowapi.Event{
					Type: flowapi.HIDEventTypeOutput,
					HID:  event,
				})
			}
		}
	}()

	// Input And Feature Reports
	for {
		select {
		case reports := <-inputCh:
			for _, report := range reports {
				_, err := dev.Write(report)
				if err != nil {
					o.log.Error("Failed to write output report", zap.Error(err))
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (o *OutputNode) buildDescriptor(cfg outputDescriptorConfig) (hiddesc.ReportDescriptor, error) {
	desc := hiddesc.ReportDescriptor{}
	if len(cfg.Inputs) == 0 {
		return desc, fmt.Errorf("no input devices specified")
	}
	idMap := make(map[uint8]uint8)
	for _, addr := range cfg.Inputs {
		inputDescRaw, err := o.hid.GetReportDescriptor(addr)
		if err != nil {
			return desc, fmt.Errorf("failed to get report descriptor for input device %s: %w", addr, err)
		}
		inputDesc, err := hiddesc.Decode(inputDescRaw)
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
	reportsCh := make(chan [][]byte)
	defer close(reportsCh)

	isConnected := o.hid.IsOutputConnected(o.addr)
	if isConnected {
		deviceCtx, cancel = context.WithCancel(ctx)
		go o.handleDevice(deviceCtx, up, reportsCh)
	}
	go func() {
		for {
			select {
			case event := <-events:
				switch event.Type {
				case flowapi.HIDEventTypeInput:
					reports := o.inputState.ApplyEvent(event.HID)
					if len(reports) > 0 {
						select {
						case reportsCh <- reports:
						default:
							o.log.Warn("Dropped input reports")
						}
					}
				case flowapi.HIDEventTypeFeature:
					o.featureState.ApplyEvent(event.HID)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	for {
		select {
		case ev := <-deviceEvents:
			switch ev.Key.Type {
			case OutputConnected:
				if cancel != nil {
					break
				}
				o.log.Info("Output device connected", zap.String("addr", o.addr.String()))
				deviceCtx, cancel = context.WithCancel(ctx)
				go o.handleDevice(deviceCtx, up, reportsCh)
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
