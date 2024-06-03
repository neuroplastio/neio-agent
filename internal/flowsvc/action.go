package flowsvc

import (
	"context"

	"github.com/neuroplastio/neuroplastio/pkg/registry"
)

type HIDActionProvider struct {
	ActionRegistry *ActionRegistry
	State          *FlowState
}

type ActionRegistry = registry.Registry[HIDUsageAction, *HIDActionProvider]

func NewActionRegistry(state *FlowState) *ActionRegistry {
	provider := &HIDActionProvider{
		State: state,
	}
	reg := registry.NewRegistry[HIDUsageAction, *HIDActionProvider](provider)
	provider.ActionRegistry = reg
	reg.Register("none", NewActionNone)
	reg.Register("usage", NewUsageAction)
	reg.Register("tapHold", NewTapHoldAction)
	reg.Register("lock", NewLockAction)
	reg.Register("set", NewSetAction)
	return reg
}

type UsageActivator func(usages []Usage) (unset func())

type HIDUsageAction interface {
	// Usages returns all usages that this action emits.
	Usages() []Usage
	// Activate is called when the action is activated (i.e. key is pressed down).
	// It should return a function that deactivates the action (i.e. key is released).
	Activate(ctx context.Context, activator UsageActivator) func()
}

type Usage uint32

func (u Usage) Page() uint16 {
	return uint16(u >> 16)
}

func (u Usage) ID() uint16 {
	return uint16(u)
}

func NewUsage(page, id uint16) Usage {
	return Usage(uint32(page)<<16 | uint32(id))
}
