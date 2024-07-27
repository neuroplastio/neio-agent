package actions

import (
	"context"
	"time"

	"github.com/neuroplastio/neio-agent/flowapi"
)

type SendString struct{}

func (a SendString) Descriptor() flowapi.ActionDescriptor {
	return flowapi.ActionDescriptor{
		DisplayName: "Send String",
		Description: "Send a string of ASCII characters in quick succession.",
		Signature:   "sendString(value: string, rightShift: boolean = false, delay: Duration = 4ms, modDuration: Duration = 1ms)",
	}
}

func (a SendString) CreateHandler(p flowapi.ActionProvider) (flowapi.ActionHandler, error) {
	value := p.Args().String("value")
	if value == "" {
		return NewActionNoneHandler(), nil
	}
	rightShift := p.Args().Boolean("rightShift")
	delay := p.Args().Duration("delay")
	modDuration := p.Args().Duration("modDuration")
	actions := make([]flowapi.ActionHandler, 0, len(value))
	for _, c := range value {
		charAction, err := NewCharActionHandler(p.Context(), c, rightShift, modDuration)
		if err != nil {
			return nil, err
		}
		actions = append(actions, charAction)
	}
	return NewActionChainHandler(p.Context(), actions, delay), nil
}

func NewActionChainHandler(ctx context.Context, actions []flowapi.ActionHandler, delay time.Duration) flowapi.ActionHandler {
	if len(actions) == 0 {
		return NewActionNoneHandler()
	}
	return func(ac flowapi.ActionContext) flowapi.ActionFinalizer {
		return ac.Async(func(async flowapi.AsyncActionContext) {
			for _, action := range actions {
				fin := async.Action(action)
				<-async.After(delay)
				async.Finish(fin)
				select {
				case <-async.Interrupt():
					return
				case <-async.After(delay):
				}
			}
		})
	}
}
