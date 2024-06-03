package flowsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"github.com/neuroplastio/neuroplastio/internal/hidsvc"
	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hiddesc"
)

type inputConfig struct {
	Addr hidsvc.Address `json:"addr"`
}

func NewInputNode(data json.RawMessage, provider *NodeProvider) (Node, error) {
	var cfg inputConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	dev, err := provider.HID.GetInputDeviceHandle(cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to get input device %s: %w", cfg.Addr, err)
	}
	return &InputNode{dev: dev}, nil
}

type InputNode struct {
	dev  *hidsvc.InputDeviceHandle
}

func (g *InputNode) Start(ctx context.Context, up FlowStream, down FlowStream) error {
	desc, err := hiddesc.NewDescriptorDecoder(bytes.NewBuffer(g.GetDescriptor())).Decode()
	if err != nil {
		return err
	}
	read := make(chan []byte)
	write := make(chan []byte)
	go func() {
		defer close(write)
		sub := down.Subscribe(ctx)
		for {
			select {
			case event := <-sub:
				write <- hidparse.EncodeReport(event.Message.HIDReport)
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
				report, ok := hidparse.ParseReport(*desc, data)
				if !ok {
					fmt.Println("Failed to parse report")
					continue
				}
				down.Broadcast(ctx, FlowEvent{HIDReport: report})
			case <-ctx.Done():
				return
			}
		}
	}()
	return g.dev.Start(ctx, read, write)
}

func (g *InputNode) GetDescriptor() []byte {
	return g.dev.InputDevice().BackendDevice.ReportDescriptor
}

func (d *InputNode) OriginSpec() OriginSpec {
	return OriginSpec{
		MinConnections: 1,
		MaxConnections: 1,
	}
}
