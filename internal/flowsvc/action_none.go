package flowsvc

import (
	"context"
	"encoding/json"
)

type ActionNone struct {}

func NewActionNone(data json.RawMessage, provider *HIDActionProvider) (HIDUsageAction, error) {
	return &ActionNone{}, nil
}

func (a *ActionNone) Usages() []Usage {
	return []Usage{}
}

func (a *ActionNone) Activate(ctx context.Context, activator UsageActivator) func() {
	return func() {}
}
