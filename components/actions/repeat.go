package actions

import (
	"context"
	"time"

	"github.com/neuroplastio/neio-agent/flowapi"
)

type Repeat struct{}

func (a Repeat) Descriptor() flowapi.ActionDescriptor {
	return flowapi.ActionDescriptor{
		DisplayName: "Repeat",
		Signature:   "repeat(action: Action, delay: Duration = 150ms, interval: Duration = 50ms, tapDuration: Duration = 1ms)",
	}
}

func (a Repeat) CreateHandler(p flowapi.ActionProvider) (flowapi.ActionHandler, error) {
	action, err := p.ActionArg("action")
	if err != nil {
		return nil, err
	}
	delay := p.Args().Duration("delay")
	interval := p.Args().Duration("interval")
	tapDuration := p.Args().Duration("tapDuration")
	return NewRepeatActionHandler(p.Context(), action, delay, interval, tapDuration), nil
}

func NewRepeatActionHandler(ctx context.Context, action flowapi.ActionHandler, delay, interval, tapDuration time.Duration) flowapi.ActionHandler {
	half := interval / 2
	return func(ac flowapi.ActionContext) flowapi.ActionFinalizer {
		fin := action(ac)
		return ac.Async(func(async flowapi.AsyncActionContext) {
			select {
			case <-async.After(delay - half):
			case <-async.Finished():
				async.Finish(fin)
				return
			}
			for {
				async.Finish(fin)
				select {
				case <-time.After(half):
				case <-async.Finished():
					return
				}
				fin = async.Action(action)
				select {
				case <-time.After(half):
				case <-async.Finished():
					async.Finish(fin)
					return
				}
			}
		})
	}
}
