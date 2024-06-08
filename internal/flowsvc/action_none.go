package flowsvc

import (
	"context"
	"encoding/json"

	"github.com/neuroplastio/neuroplastio/internal/hidparse"
)

type ActionNone struct {}

func NewActionNone(data json.RawMessage, provider *HIDActionProvider) (HIDUsageAction, error) {
	return &ActionNone{}, nil
}

func (a *ActionNone) Usages() []hidparse.Usage {
	return []hidparse.Usage{}
}

func (a *ActionNone) Activate(ctx context.Context, activator UsageActivator) func() {
	return func() {}
}
