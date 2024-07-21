package flowsvc

type ActionLock struct{}

func (a ActionLock) Metadata() ActionMetadata {
	return ActionMetadata{
		DisplayName: "Lock",
		Description: "Locks a button until it's pressed again.",
		Signature:   "lock(action: Action)",
	}
}

func (a ActionLock) Handler(p ActionProvider) (ActionHandler, error) {
	action, err := p.ActionArg("action")
	if err != nil {
		return nil, err
	}

	return NewActionLockHandler(action), nil
}

func NewActionLockHandler(action ActionHandler) ActionHandler {
	var deactivate ActionFinalizer
	return func(ac ActionContext) ActionFinalizer {
		if deactivate != nil {
			deactivate(ac)
			deactivate = nil
		} else {
			deactivate = action(ac)
		}
		return nil
	}
}
