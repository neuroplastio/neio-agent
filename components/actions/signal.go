package actions

import (
	"github.com/neuroplastio/neuroplastio/flowapi"
)

type Signal struct{}

func (a Signal) Metadata() flowapi.ActionDescriptor {
	return flowapi.ActionDescriptor{
		DisplayName: "Signal",
		Signature:   "signal(onActivate: Signal = null, onDeactivate: Signal = null)",
	}
}

func (a Signal) Handler(p flowapi.ActionProvider) (flowapi.ActionHandler, error) {
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

func NewSignalActionHandler(onActivate, onDeactivate flowapi.SignalHandler) flowapi.ActionHandler {
	return func(ac flowapi.ActionContext) flowapi.ActionFinalizer {
		if onActivate != nil {
			onActivate(ac.Context())
		}
		if onDeactivate == nil {
			return nil
		}
		return func(ac flowapi.ActionContext) {
			onDeactivate(ac.Context())
		}
	}
}
