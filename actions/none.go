package actions

import "github.com/neuroplastio/neuroplastio/internal/flowsvc"

type ActionNone struct{}

func (a ActionNone) Metadata() flowsvc.ActionDescriptor {
	return flowsvc.ActionDescriptor{
		DisplayName: "None",
		Description: "No action",
		Signature:   "none()",
	}
}

func (a ActionNone) Handler(provider flowsvc.ActionProvider) (flowsvc.ActionHandler, error) {
	return NewActionNoneHandler(), nil
}

func NewActionNoneHandler() flowsvc.ActionHandler {
	return func(ac flowsvc.ActionContext) flowsvc.ActionFinalizer {
		return nil
	}
}
