package hidnodes

import (
	"context"
	"fmt"

	"github.com/neuroplastio/neuroplastio/internal/flowsvc"
	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"github.com/neuroplastio/neuroplastio/internal/hidsvc"
	"github.com/neuroplastio/neuroplastio/pkg/hidevent"
	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hiddesc"
	"go.uber.org/zap"
)

type Output struct{}

func (o Output) Descriptor() flowsvc.NodeTypeDescriptor {
	return flowsvc.NodeTypeDescriptor{
		DisplayName: "Output",

		UpstreamType:   flowsvc.NodeLinkTypeMany,
		DownstreamType: flowsvc.NodeLinkTypeNone,
	}
}

func (o Output) CreateNode(p flowsvc.RunnerProvider) (flowsvc.Node, error) {
	return &OutputRunner{
		log: p.Log(),
	}, nil
}

type outputConfig struct {
	Addr           hidsvc.Address   `json:"addr"`
	DescriptorFrom []hidsvc.Address `json:"descriptorFrom"`
}

type OutputRunner struct {
	log *zap.Logger

	dev  *hidsvc.OutputDeviceHandle
	desc flowsvc.HIDReportDescriptor
	rte  *hidevent.RTETranscoder
	etr  *hidevent.ETRTranscoder
}

func (o *OutputRunner) Configure(c flowsvc.NodeConfigurator) error {
	cfg := outputConfig{}
	if err := c.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	dev, err := c.HID().GetOutputDeviceHandle(cfg.Addr)
	if err != nil {
		return fmt.Errorf("failed to get output device %s: %w", cfg.Addr, err)
	}
	merged := hiddesc.ReportDescriptor{}
	outputID := uint8(1)
	for _, addr := range cfg.DescriptorFrom {
		inputDev, err := c.HID().GetInputDevice(addr)
		if err != nil {
			return fmt.Errorf("failed to get input device %s: %w", addr, err)
		}
		inputDesc, err := flowsvc.NewHIDReportDescriptorFromRaw(inputDev.BackendDevice.ReportDescriptor)
		if err != nil {
			return fmt.Errorf("failed to parse input descriptor: %w", err)
		}
		for _, collection := range inputDesc.Parsed().Collections {
			ids := make(map[uint8]uint8)
			reports := collection.GetInputReport()
			for _, report := range reports {
				ids[report.ID] = outputID
				outputID++
			}
			collection = o.replaceIDs(ids, collection)
			merged.Collections = append(merged.Collections, collection)
		}
	}
	desc, err := flowsvc.NewHIDReportDescriptor(merged)
	if err != nil {
		return err
	}
	o.dev = dev
	o.desc = desc
	o.rte = hidevent.NewRTE(o.log, desc.Parsed().GetOutputDataItems())
	o.etr = hidevent.NewETR(o.log, desc.Parsed().GetInputDataItems())
	return nil
}

func (o *OutputRunner) replaceIDs(ids map[uint8]uint8, collection hiddesc.Collection) hiddesc.Collection {
	collectionCopy := hiddesc.Collection{
		UsagePage: collection.UsagePage,
		UsageID:   collection.UsageID,
		Type:      collection.Type,
	}
	for _, item := range collection.Items {
		itemCopy := hiddesc.MainItem{
			Type: item.Type,
		}
		if item.Collection != nil {
			itemCopy.Collection = &(*item.Collection)
		}
		if item.DataItem != nil {
			itemCopy.DataItem = &(*item.DataItem)
		}
		if itemCopy.Collection != nil {
			cc := o.replaceIDs(ids, *itemCopy.Collection)
			itemCopy.Collection = &cc
		}
		if itemCopy.DataItem != nil {
			if newID, ok := ids[itemCopy.DataItem.ReportID]; ok {
				itemCopy.DataItem.ReportID = newID
			}
		}
		collectionCopy.Items = append(collectionCopy.Items, itemCopy)
	}
	return collectionCopy
}

func (o *OutputRunner) Run(ctx context.Context, up flowsvc.FlowStream, down flowsvc.FlowStream) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	read := make(chan []byte)
	defer close(read)
	write := make(chan []byte)
	defer close(write)
	sub := up.Subscribe(ctx)
	go func() {
		for {
			select {
			case event := <-sub:
				reports := o.etr.OnEvent(*event.Message.HIDEvent)
				for _, report := range reports {
					write <- hidparse.EncodeReport(report)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case data := <-read:
				report, ok := hidparse.ParseOutputReport(o.desc.Parsed(), data)
				if !ok {
					fmt.Println("Failed to parse output report")
					continue
				}
				event := o.rte.OnReport(report)
				up.Broadcast(flowsvc.FlowEvent{HIDEvent: &event})
			case <-ctx.Done():
				return
			}
		}
	}()
	return o.dev.Start(ctx, o.desc.Raw(), read, write)
}
