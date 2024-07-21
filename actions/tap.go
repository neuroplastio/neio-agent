package actions

import (
	"time"

	"github.com/neuroplastio/neuroplastio/internal/flowsvc"
)

type ActionTap struct{}

func (a ActionTap) Metadata() flowsvc.ActionDescriptor {
	return flowsvc.ActionDescriptor{
		DisplayName: "Tap",
		Description: "Tap action",
		Signature:   "tap(action: Action, duration: Duration = 15ms)",
	}
}

func (a ActionTap) Handler(p flowsvc.ActionProvider) (flowsvc.ActionHandler, error) {
	action, err := p.ActionArg("action")
	if err != nil {
		return nil, err
	}
	return NewActionTapHandler(action, p.Args().Duration("duration")), nil
}

func NewActionTapHandler(action flowsvc.ActionHandler, tapDuration time.Duration) flowsvc.ActionHandler {
	return func(ac flowsvc.ActionContext) flowsvc.ActionFinalizer {
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
