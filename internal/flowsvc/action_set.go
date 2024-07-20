package flowsvc

import (
	"context"

	"github.com/neuroplastio/neuroplastio/internal/flowsvc/actiondsl"
	"github.com/neuroplastio/neuroplastio/internal/hidparse"
)

type ActionSet struct{}

type actionSetHandler struct {
	state StateList[any]

	value any
}

func (a ActionSet) Metadata() HIDUsageActionMetadata {
	return HIDUsageActionMetadata{
		DisplayName: "Set",
		Description: "Sets variable to a value",
		Declaration: "set(name: string, value: any)",
	}
}

func (a ActionSet) Handler(args actiondsl.Arguments, provider *HIDActionProvider) (HIDUsageActionHandler, error) {
	return &actionSetHandler{
		state: NewStateList[any](provider.State, args.String("name")),
		value: args.Any("value"),
	}, nil
}

func (a *actionSetHandler) Usages() []hidparse.Usage {
	return []hidparse.Usage{}
}

func (a *actionSetHandler) Activate(ctx context.Context, activator UsageActivator) func() {
	return a.state.Push(a.value)
}
