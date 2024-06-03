package flowsvc

import (
	"context"
	"encoding/json"
)

type lockConfig struct {
	Action HIDUsageActionConfig `json:"action"`
}

type LockAction struct {
	action HIDUsageAction

	deactivate func()
}

func NewLockAction(data json.RawMessage, provider *HIDActionProvider) (HIDUsageAction, error) {
	var cfg lockConfig
	err := json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	action, err := provider.ActionRegistry.New(cfg.Action.Type, cfg.Action.Config)
	if err != nil {
		return nil, err
	}

	return &LockAction{
		action: action,
	}, nil
}

func (a *LockAction) Usages() []Usage {
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
