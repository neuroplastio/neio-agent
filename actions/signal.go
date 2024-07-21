package actions

import "github.com/neuroplastio/neuroplastio/internal/flowsvc"

type ActionSignal struct{}

func (a ActionSignal) Metadata() flowsvc.ActionDescriptor {
	return flowsvc.ActionDescriptor{
		DisplayName: "Signal",
		Signature:   "signal(onActivate: Signal = null, onDeactivate: Signal = null)",
	}
}

func (a ActionSignal) Handler(p flowsvc.ActionProvider) (flowsvc.ActionHandler, error) {
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

func NewSignalActionHandler(onActivate, onDeactivate flowsvc.SignalHandler) flowsvc.ActionHandler {
	return func(ac flowsvc.ActionContext) flowsvc.ActionFinalizer {
		if onActivate != nil {
			onActivate(ac.Context())
		}
		if onDeactivate == nil {
			return nil
		}
		return func(ac flowsvc.ActionContext) {
			onDeactivate(ac.Context())
		}
	}
}
