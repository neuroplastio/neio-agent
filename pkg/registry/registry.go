package registry

import (
	"encoding/json"
	"fmt"
)

type Component interface {
	any
}

type Provider interface {
	any
}

type ComponentCreator[C Component, P Provider] func(config json.RawMessage, provider P) (C, error)

type Registry[C Component, P Provider] struct {
	components map[string]ComponentCreator[C, P]
	provider   P
}

func NewRegistry[C Component, P Provider](provider P) *Registry[C, P] {
	return &Registry[C, P]{
		provider:   provider,
		components: make(map[string]ComponentCreator[C, P]),
	}
}

func (r *Registry[C, P]) Register(id string, creator ComponentCreator[C, P]) {
	if _, ok := r.components[id]; ok {
		panic("component already registered")
	}
	r.components[id] = creator
}

func (r *Registry[C, P]) New(id string, config json.RawMessage) (C, error) {
	creator, ok := r.components[id]
	if !ok {
		var component C
		return component, fmt.Errorf("component not found: %s", id)
	}
	return creator(config, r.provider)
}

