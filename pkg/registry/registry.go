package registry

import (
	"fmt"
)

type Component interface {
	any
}

type Registry[C Component] struct {
	components map[string]C
}

func NewRegistry[C Component]() *Registry[C] {
	return &Registry[C]{
		components: make(map[string]C),
	}
}

func (r *Registry[C]) Register(id string, component C) error {
	if _, ok := r.components[id]; ok {
		return fmt.Errorf("component already registered: %s", id)
	}
	r.components[id] = component
	return nil
}

func (r *Registry[C]) Has(id string) bool {
	_, ok := r.components[id]
	return ok
}

func (r *Registry[C]) Get(id string) (C, error) {
	component, ok := r.components[id]
	if !ok {
		return component, fmt.Errorf("component not found: %s", id)
	}
	return component, nil
}

func (r *Registry[C]) Components() map[string]C {
	components := make(map[string]C, len(r.components))
	for id, component := range r.components {
		components[id] = component
	}
	return components
}
