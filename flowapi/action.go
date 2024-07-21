package flowapi

import (
	"context"

	"github.com/neuroplastio/neuroplastio/hidapi"
)

type ActionDescriptor struct {
	DisplayName string
	Description string

	Signature string
}

type SignalDescriptor struct {
	DisplayName string
	Description string

	Signature string
}

type Action interface {
	Metadata() ActionDescriptor
	Handler(provider ActionProvider) (ActionHandler, error)
}

type ActionContext interface {
	Context() context.Context
	HIDEvent(modifier func(e *hidapi.Event))
}

type ActionFinalizer func(ac ActionContext)
type ActionHandler func(ac ActionContext) ActionFinalizer
type SignalHandler func(ctx context.Context)

type ActionProvider interface {
	Args() Arguments
	ActionArg(argName string) (ActionHandler, error)
	SignalArg(argName string) (SignalHandler, error)
}

type ActionCreator func(p ActionProvider) (ActionHandler, error)
type SignalCreator func(p ActionProvider) (SignalHandler, error)
