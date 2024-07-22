package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/neuroplastio/neuroplastio/flowapi"
)

type TapHold struct{}

func (a TapHold) Descriptor() flowapi.ActionDescriptor {
	return flowapi.ActionDescriptor{
		DisplayName: "Tap Hold",
		Description: "Tap and hold action",
		Signature:   "tapHold(onTap: Action, onHold: Action, delay: Duration = 250ms, tapDuration: Duration = 1ms)",
	}
}

func (a TapHold) CreateHandler(p flowapi.ActionProvider) (flowapi.ActionHandler, error) {
	onHold, err := p.ActionArg("onHold")
	if err != nil {
		return nil, fmt.Errorf("failed to create onHold action: %w", err)
	}

	onTap, err := p.ActionArg("onTap")
	if err != nil {
		return nil, fmt.Errorf("failed to create onTap action: %w", err)
	}

	return NewActionTapHoldHandler(p.Context(), onTap, onHold, p.Args().Duration("delay"), p.Args().Duration("tapDuration")), nil
}

func NewActionTapHoldHandler(ctx context.Context, onTap flowapi.ActionHandler, onHold flowapi.ActionHandler, delay time.Duration, tapDuration time.Duration) flowapi.ActionHandler {
	onTap = NewActionTapHandler(ctx, onTap, tapDuration)
	sleeper := NewSleeper(ctx, delay)
	return func(ac flowapi.ActionContext) flowapi.ActionFinalizer {
		var fin flowapi.ActionFinalizer
		sleeper.do(func() {
			fin = onHold(ac)
		}, func() {
			if fin == nil {
				onTap(ac)(ac)
			} else {
				fin(ac)
			}
		})
		return func(ac flowapi.ActionContext) {
			sleeper.cancel()
		}
	}
}
