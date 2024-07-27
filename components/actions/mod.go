package actions

import (
	"context"
	"time"

	"github.com/neuroplastio/neio-agent/flowapi"
	"github.com/neuroplastio/neio-agent/hidapi"
)

type Mod struct{}

func (a Mod) Descriptor() flowapi.ActionDescriptor {
	return flowapi.ActionDescriptor{
		DisplayName: "Modifier",
		Description: "Perform an action with active modifier key (or any other HID usage)",
		Signature:   "mod(modifier: Usage, action: Action, delay: Duration = 1ms)",
	}
}

func (a Mod) CreateHandler(p flowapi.ActionProvider) (flowapi.ActionHandler, error) {
	modifier, err := p.Args().Usages("modifier")
	if err != nil {
		return nil, err
	}
	action, err := p.ActionArg("action")
	if err != nil {
		return nil, err
	}
	return NewModHandler(p.Context(), modifier, action, p.Args().Duration("delay")), nil
}

func NewModHandler(ctx context.Context, modifier []hidapi.Usage, action flowapi.ActionHandler, duration time.Duration) flowapi.ActionHandler {
	return func(ac flowapi.ActionContext) flowapi.ActionFinalizer {
		ac.HIDEvent().Activate(modifier...)
		return ac.Async(func(async flowapi.AsyncActionContext) {
			<-async.After(duration)
			fin := async.Action(action)
			async.OnFinish(func(ac flowapi.ActionContext) {
				if fin != nil {
					fin(ac)
				}
				ac.HIDEvent().Deactivate(modifier...)
			})
		})
	}
}
