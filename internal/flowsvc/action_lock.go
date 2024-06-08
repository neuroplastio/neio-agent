package flowsvc

import (
	"context"
	"encoding/json"

	"github.com/neuroplastio/neuroplastio/internal/hidparse"
)

type LockAction struct {
	action HIDUsageAction

	deactivate func()
}

func NewLockAction(data json.RawMessage, provider *HIDActionProvider) (HIDUsageAction, error) {
	action, err := provider.ActionRegistry.NewFromJSON(data)
	if err != nil {
		return nil, err
	}

	return &LockAction{
		action: action,
	}, nil
}

func (a *LockAction) Usages() []hidparse.Usage {
	return a.action.Usages()
}

func (a *LockAction) Activate(ctx context.Context, activator UsageActivator) func() {
	if a.deactivate != nil {
		a.deactivate()
		a.deactivate = nil
	} else {
		a.deactivate = a.action.Activate(ctx, activator)
	}
	return func() {}
}
