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
		Signature:   "tapHold(onTap: Action, onHold: Action, delay: Duration = 250ms, tapDuration: Duration = 1ms, interrupt: boolean = true)",
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

	return NewActionTapHoldHandler(p.Context(), onTap, onHold, p.Args().Duration("delay"), p.Args().Duration("tapDuration"), p.Args().Boolean("interrupt")), nil
}

func NewActionTapHoldHandler(ctx context.Context, onTap flowapi.ActionHandler, onHold flowapi.ActionHandler, delay time.Duration, tapDuration time.Duration, interrupt bool) flowapi.ActionHandler {
	return func(ac flowapi.ActionContext) flowapi.ActionFinalizer {
		return ac.Async(func(async flowapi.AsyncActionContext) {
			select {
			case <-async.Interrupt():
				async.OnFinish(async.Action(onHold))
			case <-async.After(delay):
				async.OnFinish(async.Action(onHold))
			case <-async.Finished():
				fin := async.Action(onTap)
				<-async.After(tapDuration)
				async.Finish(fin)
			}
		})
	}
}
