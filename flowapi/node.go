package flowapi

import (
	"context"

	"github.com/neuroplastio/neio-agent/flowapi/flowdsl"
)

type NodeLinkType int

const (
	NodeLinkTypeNone NodeLinkType = iota
	NodeLinkTypeOne
	NodeLinkTypeMany
)

type NodeTypeDescriptor struct {
	DisplayName string
	Description string

	UpstreamType   NodeLinkType
	DownstreamType NodeLinkType

	Actions []ActionDescriptor
	Signals []SignalDescriptor
}

type NodeGraphInfo struct {
	ID          string
	Descriptor  NodeTypeDescriptor
	Upstreams   []string
	Downstreams []string
}

type NodeType interface {
	Descriptor() NodeTypeDescriptor
	CreateNode(p NodeProvider) (Node, error)
}

type NodeProvider interface {
	Info() NodeGraphInfo
	RegisterAction(name string, creator ActionCreator)
	RegisterSignal(name string, creator SignalCreator)
}

type NodeConfigurator interface {
	Unmarshal(to any) error

	ActionHandler(stmt flowdsl.Statement) (ActionHandler, error)
	SignalHandler(stmt flowdsl.Statement) (SignalHandler, error)
}

type Node interface {
	Configure(c NodeConfigurator) error
	Run(ctx context.Context, up Stream, down Stream) error
}
