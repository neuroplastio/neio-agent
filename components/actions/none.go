package actions

import "github.com/neuroplastio/neuroplastio/flowapi"

type None struct{}

func (a None) Metadata() flowapi.ActionDescriptor {
	return flowapi.ActionDescriptor{
		DisplayName: "None",
		Description: "No action",
		Signature:   "none()",
	}
}

func (a None) Handler(provider flowapi.ActionProvider) (flowapi.ActionHandler, error) {
	return NewActionNoneHandler(), nil
}

func NewActionNoneHandler() flowapi.ActionHandler {
	return func(ac flowapi.ActionContext) flowapi.ActionFinalizer {
		return nil
	}
}
