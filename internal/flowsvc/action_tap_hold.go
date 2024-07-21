package flowsvc

import (
	"fmt"
	"time"
)

type ActionTapHold struct{}

func (a ActionTapHold) Metadata() ActionMetadata {
	return ActionMetadata{
		DisplayName: "Tap Hold",
		Description: "Tap and hold action",
		Signature:   "tapHold(onTap: Action, onHold: Action, delay: Duration = 250ms, tapDuration: Duration = 10ms)",
	}
}

func (a ActionTapHold) Handler(p ActionProvider) (ActionHandler, error) {
	onHold, err := p.ActionArg("onHold")
	if err != nil {
		return nil, fmt.Errorf("failed to create onHold action: %w", err)
	}

	onTap, err := p.ActionArg("onTap")
	if err != nil {
		return nil, fmt.Errorf("failed to create onTap action: %w", err)
	}

	return NewActionTapHoldHandler(onTap, onHold, p.Args().Duration("delay"), p.Args().Duration("tapDuration")), nil
}

func NewActionTapHoldHandler(onTap ActionHandler, onHold ActionHandler, delay time.Duration, tapDuration time.Duration) ActionHandler {
	return func(ac ActionContext) ActionFinalizer {
		deactivateCh := make(chan struct{})
		timer := time.NewTimer(delay)
		go func() {
			defer func() {
				if !timer.Stop() {
					<-timer.C
				}
			}()
			var deactivateHold ActionFinalizer
			for {
				select {
				case <-timer.C:
					// start holding
					deactivateHold = onHold(ac)
				case <-deactivateCh:
					if deactivateHold == nil {
						// tap
						onTap(ac)(ac)
					} else {
						// finish holding
						deactivateHold(ac)
					}
					return
				case <-ac.Context().Done():
					return
				}
			}
		}()
		return func(ac ActionContext) {
			close(deactivateCh)
		}
	}
}
