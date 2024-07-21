package flowsvc

type ActionNone struct{}

func (a ActionNone) Metadata() ActionMetadata {
	return ActionMetadata{
		DisplayName: "None",
		Description: "No action",
		Signature:   "none()",
	}
}

func (a ActionNone) Handler(provider ActionProvider) (ActionHandler, error) {
	return NewActionNoneHandler(), nil
}

func NewActionNoneHandler() ActionHandler {
	return func(ac ActionContext) ActionFinalizer {
		return nil
	}
}
