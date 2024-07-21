package flowsvc

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/neuroplastio/neuroplastio/internal/flowsvc/actiondsl"
	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"github.com/neuroplastio/neuroplastio/pkg/hidevent"
	"github.com/neuroplastio/neuroplastio/pkg/registry"
	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hidusage/usagepages"
)

type Registry struct {
	nodes   *registry.Registry[nodeRegistration]
	actions *registry.Registry[actionRegistration]
}

func (a *Registry) RegisterAction(action Action) error {
	metadata := action.Metadata()
	decl, err := actiondsl.ParseDeclaration(metadata.Signature)
	if err != nil {
		return fmt.Errorf("failed to parse declaration for action %s: %w", metadata.Signature, err)
	}
	err = a.actions.Register(decl.Identifier, actionRegistration{
		action:      action,
		declaration: decl,
	})
	if err != nil {
		return fmt.Errorf("failed to register action %s: %w", decl.Identifier, err)
	}
	return nil
}

func (a *Registry) MustRegisterAction(action Action) {
	err := a.RegisterAction(action)
	if err != nil {
		panic(err)
	}
}

func (a *Registry) RegisterNode(typ string, node Node) error {
	if a.nodes.Has(typ) {
		return fmt.Errorf("node already registered: %s", typ)
	}
	registration := nodeRegistration{
		node:         node,
		actions:      make(map[string]actiondsl.Declaration),
		signals:      make(map[string]actiondsl.Declaration),
		declarations: make(map[string]actiondsl.Declaration),
	}
	metadata := node.Metadata()
	for _, action := range metadata.Actions {
		decl, err := actiondsl.ParseDeclaration(action.Signature)
		if err != nil {
			return fmt.Errorf("failed to parse declaration for action %s: %w", action.Signature, err)
		}
		if _, ok := registration.declarations[decl.Identifier]; ok {
			return fmt.Errorf("identifier %s already registered for node %s", decl.Identifier, typ)
		}
		registration.actions[decl.Identifier] = decl
		registration.declarations[decl.Identifier] = decl
	}
	for _, signal := range metadata.Signals {
		decl, err := actiondsl.ParseDeclaration(signal.Signature)
		if err != nil {
			return fmt.Errorf("failed to parse declaration for signal %s: %w", signal.Signature, err)
		}
		if _, ok := registration.declarations[decl.Identifier]; ok {
			return fmt.Errorf("identifier %s already registered for node %s", decl.Identifier, typ)
		}
		registration.signals[decl.Identifier] = decl
		registration.declarations[decl.Identifier] = decl
	}
	return a.nodes.Register(typ, registration)
}

func (a *Registry) MustRegisterNode(typ string, node Node) {
	err := a.RegisterNode(typ, node)
	if err != nil {
		panic(err)
	}
}

func (a *Registry) GetNode(typ string) (Node, error) {
	reg, err := a.nodes.Get(typ)
	if err != nil {
		return nil, fmt.Errorf("failed to get node %s: %w", typ, err)
	}
	return reg.node, nil
}

func (a *Registry) getNodeRegistration(typ string) (nodeRegistration, error) {
	reg, err := a.nodes.Get(typ)
	if err != nil {
		return nodeRegistration{}, fmt.Errorf("failed to get node %s: %w", typ, err)
	}
	return reg, nil
}

func (a *Registry) getActionRegistration(name string) (actionRegistration, error) {
	reg, err := a.actions.Get(name)
	if err != nil {
		return actionRegistration{}, fmt.Errorf("failed to get action %s: %w", name, err)
	}
	return reg, nil
}

type nodeRegistration struct {
	node         Node
	actions      map[string]actiondsl.Declaration
	signals      map[string]actiondsl.Declaration
	declarations map[string]actiondsl.Declaration
}

type actionRegistration struct {
	action      Action
	declaration actiondsl.Declaration
}

func NewRegistry() *Registry {
	reg := &Registry{
		nodes:   registry.NewRegistry[nodeRegistration](),
		actions: registry.NewRegistry[actionRegistration](),
	}
	reg.MustRegisterAction(ActionNone{})
	reg.MustRegisterAction(ActionTap{})
	reg.MustRegisterAction(ActionTapHold{})
	reg.MustRegisterAction(ActionLock{})
	reg.MustRegisterAction(ActionSignal{})
	return reg
}

type ActionMetadata struct {
	DisplayName string
	Description string

	Signature string
}

type SignalMetadata struct {
	DisplayName string
	Description string

	Signature string
}

type Action interface {
	Metadata() ActionMetadata
	Handler(provider ActionProvider) (ActionHandler, error)
}

type ActionContext interface {
	Context() context.Context
	HIDEvent(modifier func(e *hidevent.HIDEvent))
}

type ActionFinalizer func(ac ActionContext)
type ActionHandler func(ac ActionContext) ActionFinalizer

type SignalHandler func(ctx context.Context)

func ParseUsages(str []string) ([]hidparse.Usage, error) {
	usages := make([]hidparse.Usage, 0, len(str))
	for _, part := range str {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("empty usage")
		}
		usage, err := ParseUsage(part)
		if err != nil {
			return nil, err
		}
		usages = append(usages, usage)
	}

	return usages, nil
}

func ParseUsage(str string) (hidparse.Usage, error) {
	parts := strings.Split(str, ".")
	if len(parts) == 1 {
		parts = []string{"key", parts[0]}
	}
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid usage: %s", str)
	}
	prefix := parts[0]
	switch prefix {
	case "key":
		code := usagepages.KeyCode("Key" + parts[1])
		if code == 0 {
			return 0, fmt.Errorf("invalid key code: %s", parts[1])
		}
		return hidparse.NewUsage(usagepages.KeyboardKeypad, uint16(code)), nil
	case "btn":
		code, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("invalid button code: %s", parts[1])
		}
		return hidparse.NewUsage(usagepages.Button, uint16(code)), nil
	default:
		return 0, fmt.Errorf("invalid usage prefix: %s", prefix)
	}
}
