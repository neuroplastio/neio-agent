package flowsvc

import (
	"github.com/neuroplastio/neuroplastio/internal/hidsvc"
	"github.com/neuroplastio/neuroplastio/pkg/registry"
	"go.uber.org/zap"
)

type NodeProvider struct {
	HID            *hidsvc.Service
	Log            *zap.Logger
	ActionRegistry *ActionRegistry
	State          *FlowState
}

func NewNodeProvider(hid *hidsvc.Service, log *zap.Logger, actionRegistry *ActionRegistry, state *FlowState) *NodeProvider {
	return &NodeProvider{
		HID:            hid,
		Log:            log,
		ActionRegistry: actionRegistry,
		State: state,
	}
}

type NodeRegistry = registry.Registry[Node, *NodeProvider]

func NewNodeRegistry(log *zap.Logger, hid *hidsvc.Service, actionRegistry *ActionRegistry, state *FlowState) *NodeRegistry {
	reg := registry.NewRegistry[Node, *NodeProvider](NewNodeProvider(hid, log, actionRegistry, state))
	reg.Register("input", NewInputNode)
	reg.Register("output", NewOutputNode)
	reg.Register("merge", NewMergeNode)
	reg.Register("remap", NewRemapNode)
	reg.Register("router", NewRouterNode)
	return reg
}
