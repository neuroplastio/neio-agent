package hidsvc

import (
	"context"
	"fmt"

	"github.com/neuroplastio/neuroplastio/flowapi"
	"github.com/neuroplastio/neuroplastio/hidapi"
	"github.com/neuroplastio/neuroplastio/hidapi/hiddesc"
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
	Addr           Address   `json:"addr"`
	DescriptorFrom []Address `json:"descriptorFrom"`
}

type OutputNode struct {
	id  string
	log *zap.Logger
	hid *Service

	dev     *OutputDeviceHandle
	descRaw []byte
	decoder *hidapi.ReportDecoder
	source  *hidapi.EventSource
	sink    *hidapi.EventSink
}

func (o *OutputNode) Configure(c flowapi.NodeConfigurator) error {
	cfg := outputConfig{}
	if err := c.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	dev, err := o.hid.GetOutputDeviceHandle(cfg.Addr, o.id)
	if err != nil {
		return fmt.Errorf("failed to get output device %s: %w", cfg.Addr, err)
	}
	desc := hiddesc.ReportDescriptor{}
	outputID := uint8(1)
	for _, addr := range cfg.DescriptorFrom {
		inputDev, err := o.hid.GetInputDevice(addr)
		if err != nil {
			return fmt.Errorf("failed to get input device %s: %w", addr, err)
		}
		inputDesc, err := hiddesc.Decode(inputDev.BackendDevice.ReportDescriptor)
		if err != nil {
			return fmt.Errorf("failed to decode HID report descriptor: %w", err)
		}
		for _, collection := range inputDesc.Collections {
			ids := make(map[uint8]uint8)
			reports := collection.GetInputReport()
			for _, report := range reports {
				ids[report.ID] = outputID
				outputID++
			}
			collection = o.replaceIDs(ids, collection)
			desc.Collections = append(desc.Collections, collection)
		}
	}
	descRaw, err := hiddesc.Encode(desc)
	if err != nil {
		return fmt.Errorf("failed to encode HID report descriptor: %w", err)
	}
	o.descRaw = descRaw
	o.dev = dev
	o.decoder = hidapi.NewOutputReportDecoder(desc)
	o.source = hidapi.NewEventSource(o.log.Named("source"), desc.GetOutputDataItems())
	o.sink = hidapi.NewEventSink(o.log.Named("sink"), desc.GetInputDataItems())
	return nil
}

func (o *OutputNode) replaceIDs(ids map[uint8]uint8, collection hiddesc.Collection) hiddesc.Collection {
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

func (o *OutputNode) Run(ctx context.Context, up flowapi.Stream, down flowapi.Stream) error {
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
				reports := o.sink.OnEvent(event.HID)
				for _, report := range reports {
					write <- hidapi.EncodeReport(report).Bytes()
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
				report, ok := o.decoder.Decode(data)
				if !ok {
					o.log.Error("Failed to parse output report")
					continue
				}
				event := o.source.OnReport(report)
				up.Broadcast(flowapi.Event{
					HID: event,
				})
			case <-ctx.Done():
				return
			}
		}
	}()
	return o.dev.Start(ctx, o.descRaw, read, write)
}
