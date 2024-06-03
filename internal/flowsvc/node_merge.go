package flowsvc

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hiddesc"
)

func NewMergeNode( data json.RawMessage, provider *NodeProvider) (Node, error) {
	return &MergeNode{
	}, nil
}

type MergeNode struct {
	desc         []byte
	inputMap     map[string]reportID
	hasReportIDs map[string]struct{}
	reportMap    map[reportID]reportInfo
}

func (m *MergeNode) Start(ctx context.Context, up FlowStream, down FlowStream) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	in := up.Subscribe(ctx)
	for {
		select {
		case msg := <-in:
			report := msg.Message.HIDReport
			report.ID = m.reportMap[reportID{inputID: msg.Message.SourceNodeID, reportID: report.ID}].outputReportID
			down.Broadcast(ctx, FlowEvent{HIDReport: report})
		case <-ctx.Done():
			return nil
		}
	}
}

type reportID struct {
	inputID  string
	reportID uint8
}

type reportInfo struct {
	id             reportID
	report         hiddesc.Report
	outputReportID uint8
}

func (m *MergeNode) OriginSpec() OriginSpec {
	return OriginSpec{
		MinConnections: 1,
		MaxConnections: 1,
	}
}

func (m *MergeNode) DestinationSpec() DestinationSpec {
	return DestinationSpec{
		MinConnections: 2,
		MaxConnections: 16,
	}
}

func (m *MergeNode) Configure(descriptors map[string][]byte) ([]byte, error) {
	merged := &hiddesc.ReportDescriptor{}
	m.inputMap = make(map[string]reportID)
	m.reportMap = make(map[reportID]reportInfo)
	m.hasReportIDs = make(map[string]struct{})
	outputID := uint8(1)
	for inputID, descB := range descriptors {
		desc, err := hiddesc.NewDescriptorDecoder(bytes.NewBuffer(descB)).Decode()
		if err != nil {
			return nil, err
		}
		for _, collection := range desc.Collections {
			ids := make(map[uint8]uint8)
			reports := collection.GetInputReport()
			for _, report := range reports {
				reportID := reportID{
					inputID:  inputID,
					reportID: report.ID,
				}
				if report.ID > 0 {
					m.hasReportIDs[inputID] = struct{}{}
				}
				m.inputMap[inputID] = reportID
				m.reportMap[reportID] = reportInfo{id: reportID, report: report, outputReportID: outputID}
				ids[report.ID] = outputID
				outputID++
			}
			collection = m.replaceIDs(ids, collection)
			merged.Collections = append(merged.Collections, collection)
		}
	}
	buf := bytes.NewBuffer(nil)
	err := hiddesc.NewDescriptorEncoder(buf, merged).Encode()
	if err != nil {
		return nil, err
	}
	m.desc = buf.Bytes()
	return m.desc, nil
}

func (m *MergeNode) replaceIDs(ids map[uint8]uint8, collection hiddesc.Collection) hiddesc.Collection {
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
			cc := m.replaceIDs(ids, *itemCopy.Collection)
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

