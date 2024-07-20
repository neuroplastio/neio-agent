package flowsvc

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/cespare/xxhash/v2"
	"github.com/neuroplastio/neuroplastio/internal/hidsvc"
	"github.com/neuroplastio/neuroplastio/pkg/hidevent"
	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hiddesc"
	"go.uber.org/zap"
)

type NodeType int

const (
	NodeTypeNone NodeType = iota
	NodeTypeOne
	NodeTypeMany
)

type NodeMetadata struct {
	DisplayName string
	Description string

	UpstreamType   NodeType
	DownstreamType NodeType
}

type NodeInfo struct {
	ID          string
	Metadata    NodeMetadata
	Upstreams   []string
	Downstreams []string
}

type HIDReportDescriptor struct {
	id     uint64
	raw    []byte
	parsed hiddesc.ReportDescriptor
}

func (h HIDReportDescriptor) ID() uint64 {
	return h.id
}

func (h HIDReportDescriptor) Raw() []byte {
	return h.raw
}

func (h HIDReportDescriptor) Parsed() hiddesc.ReportDescriptor {
	return h.parsed
}

func NewHIDReportDescriptorFromRaw(data []byte) (HIDReportDescriptor, error) {
	id := xxhash.Sum64(data)
	desc, err := hiddesc.NewDescriptorDecoder(bytes.NewBuffer(data)).Decode()
	if err != nil {
		return HIDReportDescriptor{}, err
	}
	return HIDReportDescriptor{
		id:     id,
		raw:    data,
		parsed: desc,
	}, nil
}

func NewHIDReportDescriptor(desc hiddesc.ReportDescriptor) (HIDReportDescriptor, error) {
	buffer := bytes.NewBuffer(nil)
	err := hiddesc.NewDescriptorEncoder(buffer, desc).Encode()
	if err != nil {
		return HIDReportDescriptor{}, err
	}
	id := xxhash.Sum64(buffer.Bytes())
	return HIDReportDescriptor{
		id:     id,
		raw:    buffer.Bytes(),
		parsed: desc,
	}, nil
}

type Node interface {
	Metadata() NodeMetadata
	Runner(info NodeInfo, config json.RawMessage, provider NodeRunnerProvider) (NodeRunner, error)
}

type NodeRunnerProvider interface {
	HID() *hidsvc.Service
	Log() *zap.Logger
	Actions() *ActionRegistry
	State() *FlowState
}

type NodeRunner interface {
	Run(ctx context.Context, up FlowStream, down FlowStream) error
}

type FlowEvent struct {
	HIDEvent hidevent.HIDEvent
}
