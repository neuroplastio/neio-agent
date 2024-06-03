package flowsvc

import (
	"context"
	"time"
)

// ActionTap is not supposed to be used as a standalone action. It is a base action that is used to create more complex actions.
type ActionTap struct {
	action   HIDUsageAction
	duration time.Duration
}

func newTapAction(action HIDUsageAction, duration time.Duration) *ActionTap {
	return &ActionTap{
		action:   action,
		duration: duration,
	}
}

func (a *ActionTap) Usages() []Usage {
	return a.action.Usages()
}

func (a *ActionTap) Activate(ctx context.Context, activator UsageActivator) func() {
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
