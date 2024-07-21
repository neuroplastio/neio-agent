package flowsvc

type ActionSignal struct{}

func (a ActionSignal) Metadata() ActionMetadata {
	return ActionMetadata{
		DisplayName: "Signal",
		Signature:   "signal(onActivate: Signal = null, onDeactivate: Signal = null)",
	}
}

func (a ActionSignal) Handler(p ActionProvider) (ActionHandler, error) {
	onActivate, err := p.SignalArg("onActivate")
	if err != nil {
		return nil, err
	}
	onDeactivate, err := p.SignalArg("onDeactivate")
	if err != nil {
		return nil, err
	}
	if onActivate == nil && onDeactivate == nil {
		return NewActionNoneHandler(), nil
	}
	return NewSignalActionHandler(onActivate, onDeactivate), nil
}

func NewSignalActionHandler(onActivate, onDeactivate SignalHandler) ActionHandler {
	return func(ac ActionContext) ActionFinalizer {
		if onActivate != nil {
			onActivate(ac.Context())
		}
		if onDeactivate == nil {
			return nil
		}
		return func(ac ActionContext) {
			onDeactivate(ac.Context())
		}
	}
}
