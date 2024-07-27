package actions

import "github.com/neuroplastio/neio-agent/flowapi"

type None struct{}

func (a None) Descriptor() flowapi.ActionDescriptor {
	return flowapi.ActionDescriptor{
		DisplayName: "None",
		Description: "No action",
		Signature:   "none()",
	}
}

func (a None) CreateHandler(provider flowapi.ActionProvider) (flowapi.ActionHandler, error) {
	return NewActionNoneHandler(), nil
}

func NewActionNoneHandler() flowapi.ActionHandler {
	return func(ac flowapi.ActionContext) flowapi.ActionFinalizer {
		return nil
	}
}
