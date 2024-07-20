package flowsvc

import (
	"context"

	"github.com/neuroplastio/neuroplastio/internal/flowsvc/actiondsl"
	"github.com/neuroplastio/neuroplastio/internal/hidparse"
)

type ActionLock struct{}

func (a ActionLock) Metadata() HIDUsageActionMetadata {
	return HIDUsageActionMetadata{
		DisplayName: "Lock",
		Description: "Locks a button until it's pressed again.",
		Declaration: "lock(action: Action)",
	}
}

type actionLockHandler struct {
	action HIDUsageActionHandler

	deactivate func()
}

func (a ActionLock) Handler(args actiondsl.Arguments, provider *HIDActionProvider) (HIDUsageActionHandler, error) {
	action, err := provider.ActionRegistry.New(args.Action("action"))
	if err != nil {
		return nil, err
	}

	return &actionLockHandler{
		action: action,
	}, nil
}

func (a *actionLockHandler) Usages() []hidparse.Usage {
	return a.action.Usages()
}

func (a *actionLockHandler) Activate(ctx context.Context, activator UsageActivator) func() {
	if a.deactivate != nil {
		a.deactivate()
		a.deactivate = nil
	} else {
		a.deactivate = a.action.Activate(ctx, activator)
	}
	return func() {}
}
