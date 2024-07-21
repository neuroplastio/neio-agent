package flowsvc

import (
	"bytes"
	"context"

	"github.com/cespare/xxhash/v2"
	"github.com/neuroplastio/neuroplastio/internal/flowsvc/actiondsl"
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

	Actions []ActionMetadata
	Signals []SignalMetadata
}

type NodeInfo struct {
	ID          string
	Type        string
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
	Runner(p RunnerProvider) (NodeRunner, error)
}

type RunnerProvider interface {
	Log() *zap.Logger
	Info() NodeInfo
	RegisterAction(id string, creator ActionCreator)
	RegisterSignal(id string, creator SignalCreator)
}

type RunnerConfigurator interface {
	Unmarshal(to any) error

	HID() *hidsvc.Service

	ActionHandler(stmt actiondsl.Statement) (ActionHandler, error)
	SignalHandler(stmt actiondsl.Statement) (SignalHandler, error)
}

type NodeRunner interface {
	Configure(c RunnerConfigurator) error
	Run(ctx context.Context, up FlowStream, down FlowStream) error
}

type ActionProvider interface {
	Args() actiondsl.Arguments
	ActionArg(argName string) (ActionHandler, error)
	SignalArg(argName string) (SignalHandler, error)
}

type ActionCreator func(p ActionProvider) (ActionHandler, error)
type SignalCreator func(p ActionProvider) (SignalHandler, error)

type FlowEvent struct {
	HIDEvent hidevent.HIDEvent
}
