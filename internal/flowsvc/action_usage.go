package flowsvc

import (
	"context"
	"encoding/json"

	"github.com/neuroplastio/neuroplastio/internal/hidparse"
)

type usageActionConfig struct {
	Usages []hidparse.Usage `json:"usages"`
}

type UsageAction struct {
	usages []hidparse.Usage
}

func NewUsageAction(data json.RawMessage, provider *HIDActionProvider) (HIDUsageAction, error) {
	if data[0] == '"' {
		var str string
		err := json.Unmarshal(data, &str)
		if err != nil {
			return nil, err
		}
		usages, err := ParseUsageCombo(str)
		if err != nil {
			return nil, err
		}
		return &UsageAction{
			usages: usages,
		}, nil
	}
	var cfg usageActionConfig
	err := json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &UsageAction{
		usages: cfg.Usages,
	}, nil
}

func (a *UsageAction) Usages() []hidparse.Usage {
	return a.usages
}

func (a *UsageAction) Activate(ctx context.Context, activator UsageActivator) func() {
	return activator(a.Usages())
}
