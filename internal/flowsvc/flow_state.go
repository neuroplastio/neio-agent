package flowsvc

import (
	"container/list"
	"context"
	"fmt"

	"github.com/neuroplastio/neuroplastio/pkg/bus"
	"github.com/puzpuzpuz/xsync/v3"
	"go.uber.org/zap"
)

func NewState(log *zap.Logger) *FlowState {
	return &FlowState{
		variables: xsync.NewMapOf[string, VariableValue](),
		enums:     xsync.NewMapOf[string, map[string]int](),
		lists:     xsync.NewMapOf[string, *list.List](),
		log:       log,
		bus:       bus.NewBus[FlowStateEventKey, FlowStateEvent](log),
	}
}

type (
	FlowStateEvent struct {
		Type  FlowStateEventType
		Value VariableValue
	}
	FlowStateEventKey struct {
		Name string
	}
	FlowStateBus        = bus.Bus[FlowStateEventKey, FlowStateEvent]
	FlowStatePublisher  = bus.Publisher[FlowStateEvent]
	FlowStateSubscriber = bus.Subscriber[FlowStateEventKey, FlowStateEvent]
	FlowStateEventType  uint8
)

const (
	FlowStateEventRegistered FlowStateEventType = iota
	FlowStateEventDeregistered
	FlowStateEventChanged
)

type FlowState struct {
	variables *xsync.MapOf[string, VariableValue]
	enums     *xsync.MapOf[string, map[string]int]
	lists     *xsync.MapOf[string, *list.List]

	log *zap.Logger
	bus *FlowStateBus
}

func (f *FlowState) Start(ctx context.Context) error {
	err := f.bus.Start(ctx)
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return nil
	case <-f.bus.Ready():
	}

	return nil
}

type VariableValue struct {
	Int *int
}

func (v VariableValue) Equal(other VariableValue) bool {
	if v.Int != nil && other.Int != nil {
		return *v.Int == *other.Int
	}
	return false
}

func (f *FlowState) SetVariable(ctx context.Context, name string, value VariableValue) error {
	previous, loaded := f.variables.LoadAndStore(name, value)
	changed := false
	if loaded {
		changed = !previous.Equal(value)
	} else {
		changed = true
	}
	if changed {
		f.bus.Publish(ctx, FlowStateEventKey{Name: name}, FlowStateEvent{
			Type:  FlowStateEventChanged,
			Value: value,
		})
	}
	return nil
}

func (f *FlowState) RegisterEnum(name string, valueMap map[string]int, initialValue int) (FlowStateSubscriber, error) {
	_, loaded := f.enums.LoadOrStore(name, valueMap)
	if loaded {
		return nil, fmt.Errorf("enum %s already registered", name)
	}

	sub := f.bus.CreateSubscriber(FlowStateEventKey{Name: name})
	return sub, nil
}

func (f *FlowState) SetEnumValue(ctx context.Context, name string, value string) error {
	valueMap, ok := f.enums.Load(name)
	if !ok {
		return fmt.Errorf("enum %s not registered", name)
	}
	valueInt, ok := valueMap[value]
	if !ok {
		return fmt.Errorf("enum %s has no value %s", name, value)
	}
	return f.SetVariable(ctx, name, VariableValue{Int: &valueInt})
}

func (f *FlowState) GetEnumValue(name string) (int, error) {
	value, ok := f.variables.Load(name)
	if !ok {
		return 0, fmt.Errorf("variable %s not found", name)
	}
	if value.Int == nil {
		return 0, fmt.Errorf("variable %s is not an int", name)
	}
	return *value.Int, nil
}

func (f *FlowState) listPush(name string, val any) func() {
	var el *list.Element
	// TODO: emit changed event
	f.lists.Compute(name, func(l *list.List, loaded bool) (newValue *list.List, delete bool) {
		if !loaded {
			l = list.New()
		}
		el = l.PushFront(val)
		return l, false
	})
	return func() {
		// TODO: emit changed event if list.Front changed
		f.lists.Compute(name, func(l *list.List, loaded bool) (newValue *list.List, delete bool) {
			l.Remove(el)
			return l, false
		})
	}
}

func NewStateList[T any](f *FlowState, name string) StateList[T] {
	return StateList[T]{
		f:    f,
		name: name,
	}
}

type StateList[T any] struct {
	f    *FlowState
	name string
}

func (s StateList[T]) Push(value T) func() {
	return s.f.listPush(s.name, value)
}

type EnumDefinition[T comparable] struct {
	values []T
}
