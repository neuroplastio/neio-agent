package actions

import (
	"context"
	"time"

	"github.com/neuroplastio/neio-agent/flowapi"
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
	return func(ac flowapi.ActionContext) flowapi.ActionFinalizer {
		return ac.Async(func(async flowapi.AsyncActionContext) {
			fin := async.Action(action)
			<-async.After(tapDuration)
			async.Finish(fin)
		})
	}
}
