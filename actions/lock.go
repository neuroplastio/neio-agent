package actions

import "github.com/neuroplastio/neuroplastio/internal/flowsvc"

type ActionLock struct{}

func (a ActionLock) Metadata() flowsvc.ActionDescriptor {
	return flowsvc.ActionDescriptor{
		DisplayName: "Lock",
		Description: "Locks a button until it's pressed again.",
		Signature:   "lock(action: Action)",
	}
}

func (a ActionLock) Handler(p flowsvc.ActionProvider) (flowsvc.ActionHandler, error) {
	action, err := p.ActionArg("action")
	if err != nil {
		return nil, err
	}

	return NewActionLockHandler(action), nil
}

func NewActionLockHandler(action flowsvc.ActionHandler) flowsvc.ActionHandler {
	var deactivate flowsvc.ActionFinalizer
	return func(ac flowsvc.ActionContext) flowsvc.ActionFinalizer {
		if deactivate != nil {
			deactivate(ac)
			deactivate = nil
		} else {
			deactivate = action(ac)
		}
		return nil
	}
}
