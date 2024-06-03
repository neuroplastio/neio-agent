package flowsvc

import (
	"context"
	"encoding/json"
)

type usageActionConfig struct {
	Usages []Usage `json:"usages"`
}

type UsageAction struct {
	usages []Usage
}

func NewUsageAction(data json.RawMessage, provider *HIDActionProvider) (HIDUsageAction, error) {
	var cfg usageActionConfig
	err := json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &UsageAction{
		usages: cfg.Usages,
	}, nil
}

func (a *UsageAction) Usages() []Usage {
	return a.usages
}

func (a *UsageAction) Activate(ctx context.Context, activator UsageActivator) func() {
	return activator(a.Usages())
}
