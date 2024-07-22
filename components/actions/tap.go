package actions

import (
	"context"
	"time"

	"github.com/neuroplastio/neuroplastio/flowapi"
)

type Tap struct{}

func (a Tap) Descriptor() flowapi.ActionDescriptor {
	return flowapi.ActionDescriptor{
		DisplayName: "Tap",
		Description: "Tap action",
		Signature:   "tap(action: Action, duration: Duration = 2ms)",
	}
}

func (a Tap) CreateHandler(p flowapi.ActionProvider) (flowapi.ActionHandler, error) {
	action, err := p.ActionArg("action")
	if err != nil {
		return nil, err
	}
	return NewActionTapHandler(p.Context(), action, p.Args().Duration("duration")), nil
}

func NewActionTapHandler(ctx context.Context, action flowapi.ActionHandler, tapDuration time.Duration) flowapi.ActionHandler {
	sleeper := NewSleeper(ctx, tapDuration)
	return func(ac flowapi.ActionContext) flowapi.ActionFinalizer {
		deactivate := action(ac)
		sleeper.do(func() {
			deactivate(ac)
		}, nil)
		return nil
	}
}
