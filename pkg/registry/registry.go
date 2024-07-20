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

func (r *Registry[C]) Register(id string, component C) {
	if _, ok := r.components[id]; ok {
		panic("component already registered")
	}
	r.components[id] = component
}

func (r *Registry[C]) Get(id string) (C, error) {
	component, ok := r.components[id]
	if !ok {
		return component, fmt.Errorf("component not found: %s", id)
	}
	return component, nil
}
