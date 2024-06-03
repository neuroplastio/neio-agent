package flowsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"github.com/neuroplastio/neuroplastio/internal/hidsvc"
	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hiddesc"
)

type outputConfig struct {
	Addr hidsvc.Address `json:"addr"`
}

type OutputNode struct {
	dev        *hidsvc.OutputDeviceHandle
	desc       []byte
	descParsed hiddesc.ReportDescriptor
}

func NewOutputNode(data json.RawMessage, provider *NodeProvider) (Node, error) {
	var cfg outputConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	dev, err := provider.HID.GetOutputDeviceHandle(cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to get output device %s: %w", cfg.Addr, err)
	}
	return &OutputNode{dev: dev}, nil
}

func (o *OutputNode) Start(ctx context.Context, up FlowStream, down FlowStream) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	read := make(chan []byte)
	defer close(read)
	write := make(chan []byte)
	defer close(write)
	go func() {
		sub := up.Subscribe(ctx)
		for {
			select {
			case event := <-sub:
				data := hidparse.EncodeReport(event.Message.HIDReport)
				write <-data
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case data := <-read:
				report, ok := hidparse.ParseReport(o.descParsed, data)
				if !ok {
					fmt.Println("Failed to parse report")
				}
				up.Broadcast(ctx, FlowEvent{HIDReport: report})
			case <-ctx.Done():
				return
			}
		}
	}()
	return o.dev.Start(ctx, o.desc, read, write)
}

func (o *OutputNode) DestinationSpec() DestinationSpec {
	return DestinationSpec{
		MinConnections: 1,
		MaxConnections: 10,
	}
}

func (o *OutputNode) Configure(descriptors map[string][]byte) ([]byte, error) {
	for _, desc := range descriptors {
		o.desc = desc
		parsed, err := hiddesc.NewDescriptorDecoder(bytes.NewBuffer(desc)).Decode()
		if err != nil {
			return nil, err
		}
		o.descParsed = *parsed
		return desc, nil
	}
	return nil, errors.New("No dependencies")
}

func (o *OutputNode) GetDescriptor() []byte {
	return o.desc
}
