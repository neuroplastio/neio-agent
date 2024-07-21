package flowsvc

import (
	"time"
)

type ActionTap struct{}

func (a ActionTap) Metadata() ActionMetadata {
	return ActionMetadata{
		DisplayName: "Tap",
		Description: "Tap action",
		Signature:   "tap(action: Action, duration: Duration = 15ms)",
	}
}

func (a ActionTap) Handler(p ActionProvider) (ActionHandler, error) {
	action, err := p.ActionArg("action")
	if err != nil {
		return nil, err
	}
	return NewActionTapHandler(action, p.Args().Duration("duration")), nil
}

func NewActionTapHandler(action ActionHandler, tapDuration time.Duration) ActionHandler {
	return func(ac ActionContext) ActionFinalizer {
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
