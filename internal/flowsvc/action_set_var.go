package flowsvc

import (
	"context"
	"encoding/json"

	"github.com/neuroplastio/neuroplastio/internal/hidparse"
)

type setVarConfig struct {
	Name string `json:"name"`
	Value   any `json:"value"`
}

type SetAction struct {
	state StateList[any]

	value   any
}

func NewSetAction(data json.RawMessage, provider *HIDActionProvider) (HIDUsageAction, error) {
	var cfg setVarConfig
	err := json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &SetAction{
		state: NewStateList[any](provider.State, cfg.Name),
		value: cfg.Value,
	}, nil
}

func (a *SetAction) Usages() []hidparse.Usage {
	return []hidparse.Usage{}
}

func (a *SetAction) Activate(ctx context.Context, activator UsageActivator) func() {
	return a.state.Push(a.value)
}
