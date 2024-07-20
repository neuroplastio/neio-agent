package flowsvc

import (
	"context"
	"time"

	"github.com/neuroplastio/neuroplastio/internal/flowsvc/actiondsl"
	"github.com/neuroplastio/neuroplastio/internal/hidparse"
)

type ActionTap struct{}

func (a ActionTap) Metadata() HIDUsageActionMetadata {
	return HIDUsageActionMetadata{
		DisplayName: "Tap",
		Description: "Tap action",
		Declaration: "tap(action: Action, duration: Duration = 15ms)",
	}
}

func (a ActionTap) Handler(args actiondsl.Arguments, provider *HIDActionProvider) (HIDUsageActionHandler, error) {
	action, err := provider.ActionRegistry.New(args.Action("action"))
	if err != nil {
		return nil, err
	}
	return newTapActionHandler(action, args.Duration("duration")), nil
}

// actionTapHandler is not supposed to be used as a standalone action. It is a base action that is used to create more complex actions.
type actionTapHandler struct {
	action   HIDUsageActionHandler
	duration time.Duration
}

func newTapActionHandler(action HIDUsageActionHandler, duration time.Duration) *actionTapHandler {
	return &actionTapHandler{
		action:   action,
		duration: duration,
	}
}

func (a *actionTapHandler) Usages() []hidparse.Usage {
	return a.action.Usages()
}

func (a *actionTapHandler) Activate(ctx context.Context, activator UsageActivator) func() {
	deactivate := a.action.Activate(ctx, activator)
	go func() {
		timer := time.NewTimer(a.duration)
		defer func() {
			if !timer.Stop() {
				<-timer.C
			}
		}()
		select {
		case <-timer.C:
			deactivate()
		case <-ctx.Done():
		}
	}()
	return func() {}
}
