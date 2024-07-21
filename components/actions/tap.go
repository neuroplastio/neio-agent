package actions

import (
	"time"

	"github.com/neuroplastio/neuroplastio/flowapi"
)

type Tap struct{}

func (a Tap) Metadata() flowapi.ActionDescriptor {
	return flowapi.ActionDescriptor{
		DisplayName: "Tap",
		Description: "Tap action",
		Signature:   "tap(action: Action, duration: Duration = 15ms)",
	}
}

func (a Tap) Handler(p flowapi.ActionProvider) (flowapi.ActionHandler, error) {
	action, err := p.ActionArg("action")
	if err != nil {
		return nil, err
	}
	return NewActionTapHandler(action, p.Args().Duration("duration")), nil
}

func NewActionTapHandler(action flowapi.ActionHandler, tapDuration time.Duration) flowapi.ActionHandler {
	return func(ac flowapi.ActionContext) flowapi.ActionFinalizer {
		deactivate := action(ac)
		go func() {
			timer := time.NewTimer(tapDuration)
			defer func() {
				if !timer.Stop() {
					<-timer.C
				}
			}()
			select {
			case <-timer.C:
				deactivate(ac)
			case <-ac.Context().Done():
			}
		}()
		return nil
	}
}
