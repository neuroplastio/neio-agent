package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/neuroplastio/neuroplastio/flowapi"
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
	return NewActionChainHandler(p.Context(), actions, delay+modDuration), nil
}

func NewActionChainHandler(ctx context.Context, actions []flowapi.ActionHandler, delay time.Duration) flowapi.ActionHandler {
	if len(actions) == 0 {
		return NewActionNoneHandler()
	}
	// TODO: this doesn't work well with shifted keys
	// TODO: optimize shift key handling
	sleeper := NewSleeper(ctx, delay)
	fmt.Printf("sleeper created %v\n", sleeper)
	action := func(ac flowapi.ActionContext) flowapi.ActionFinalizer {
		fin := actions[len(actions)-1](ac)
		if fin != nil {
			fmt.Printf("sleeper1 %v\n", sleeper)
			sleeper.do(func() {
				fin(ac)
			}, nil)
		}
		return nil
	}
	for i := len(actions) - 2; i >= 0; i-- {
		i := i
		prev := action
		action = func(ac flowapi.ActionContext) flowapi.ActionFinalizer {
			fin := actions[i](ac)
			fmt.Printf("sleeper2 %v\n", sleeper)
			sleeper.do(func() {
				if fin != nil {
					fin(ac)
				}
				prev(ac)
			}, nil)
			return nil
		}
	}
	return action
}
