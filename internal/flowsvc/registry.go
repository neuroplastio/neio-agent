package flowsvc

import (
	"fmt"

	"github.com/neuroplastio/neio-agent/flowapi"
	"github.com/neuroplastio/neio-agent/flowapi/flowdsl"
	"github.com/neuroplastio/neio-agent/pkg/registry"
)

type Registry struct {
	nodes   *registry.Registry[nodeRegistration]
	actions *registry.Registry[actionRegistration]
}

func (a *Registry) RegisterAction(action flowapi.Action) error {
	metadata := action.Descriptor()
	decl, err := flowdsl.ParseDeclaration(metadata.Signature)
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

func (a *Registry) MustRegisterAction(action flowapi.Action) {
	err := a.RegisterAction(action)
	if err != nil {
		panic(err)
	}
}

func (a *Registry) RegisterNodeType(typ string, node flowapi.NodeType) error {
	if a.nodes.Has(typ) {
		return fmt.Errorf("node already registered: %s", typ)
	}
	registration := nodeRegistration{
		node:         node,
		actions:      make(map[string]flowdsl.Declaration),
		signals:      make(map[string]flowdsl.Declaration),
		declarations: make(map[string]flowdsl.Declaration),
	}
	metadata := node.Descriptor()
	for _, action := range metadata.Actions {
		decl, err := flowdsl.ParseDeclaration(action.Signature)
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
		decl, err := flowdsl.ParseDeclaration(signal.Signature)
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

func (a *Registry) MustRegisterNodeType(typ string, node flowapi.NodeType) {
	err := a.RegisterNodeType(typ, node)
	if err != nil {
		panic(err)
	}
}

func (a *Registry) GetNode(typ string) (flowapi.NodeType, error) {
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
	node         flowapi.NodeType
	actions      map[string]flowdsl.Declaration
	signals      map[string]flowdsl.Declaration
	declarations map[string]flowdsl.Declaration
}

type actionRegistration struct {
	action      flowapi.Action
	declaration flowdsl.Declaration
}

func NewRegistry() *Registry {
	return &Registry{
		nodes:   registry.NewRegistry[nodeRegistration](),
		actions: registry.NewRegistry[actionRegistration](),
	}
}
