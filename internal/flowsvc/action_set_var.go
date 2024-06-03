package flowsvc

import (
	"context"
	"encoding/json"
)

type setVarConfig struct {
	Name string `json:"name"`
	Value   any `json:"value"`
}

type SetVarAction struct {
	state *FlowState

	name string
	value   any
}

func NewSetAction(data json.RawMessage, provider *HIDActionProvider) (HIDUsageAction, error) {
	var cfg setVarConfig
	err := json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &SetVarAction{
		state: provider.State,
		name: cfg.Name,
		value: cfg.Value,
	}, nil
}

func (a *SetVarAction) Usages() []Usage {
	return []Usage{}
}

func (a *SetVarAction) Activate(ctx context.Context, activator UsageActivator) func() {
	// TODO: untyped state API
	pop := a.state.PushValue(a.name, a.value)
	return pop
}

