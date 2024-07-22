package actions

import (
	"github.com/neuroplastio/neuroplastio/flowapi"
)

type Lock struct{}

func (a Lock) Descriptor() flowapi.ActionDescriptor {
	return flowapi.ActionDescriptor{
		DisplayName: "Lock",
		Description: "Locks a button until it's pressed again.",
		Signature:   "lock(action: Action)",
	}
}

func (a Lock) CreateHandler(p flowapi.ActionProvider) (flowapi.ActionHandler, error) {
	action, err := p.ActionArg("action")
	if err != nil {
		return nil, err
	}

	return NewActionLockHandler(action), nil
}

func NewActionLockHandler(action flowapi.ActionHandler) flowapi.ActionHandler {
	var deactivate flowapi.ActionFinalizer
	return func(ac flowapi.ActionContext) flowapi.ActionFinalizer {
		if deactivate != nil {
			deactivate(ac)
			deactivate = nil
		} else {
			deactivate = action(ac)
		}
		return nil
	}
}
