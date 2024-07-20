package flowsvc

import (
	"context"

	"github.com/neuroplastio/neuroplastio/internal/flowsvc/actiondsl"
	"github.com/neuroplastio/neuroplastio/internal/hidparse"
)

type ActionNone struct{}

func (a ActionNone) Metadata() HIDUsageActionMetadata {
	return HIDUsageActionMetadata{
		DisplayName: "None",
		Description: "No action",
		Declaration: "none()",
	}
}

func (a ActionNone) Handler(args actiondsl.Arguments, provider *HIDActionProvider) (HIDUsageActionHandler, error) {
	return &actionNoneHandler{}, nil
}

type actionNoneHandler struct{}

func (a *actionNoneHandler) Usages() []hidparse.Usage {
	return []hidparse.Usage{}
}

func (a *actionNoneHandler) Activate(ctx context.Context, activator UsageActivator) func() {
	return func() {}
}
