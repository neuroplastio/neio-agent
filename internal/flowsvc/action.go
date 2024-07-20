package flowsvc

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/neuroplastio/neuroplastio/internal/flowsvc/actiondsl"
	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"github.com/neuroplastio/neuroplastio/pkg/registry"
	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hidusage/usagepages"
)

type HIDActionProvider struct {
	ActionRegistry *ActionRegistry
	State          *FlowState
}

type ActionRegistry struct {
	registry *registry.Registry[HIDUsageAction]

	declarations map[string]actiondsl.Declaration
	actionIDs    map[string]string
}

func (a *ActionRegistry) Register(id string, action HIDUsageAction) error {
	metadata := action.Metadata()
	decl, err := actiondsl.ParseDeclaration(metadata.Declaration)
	if err != nil {
		return fmt.Errorf("failed to parse declaration for action %q: %w", id, err)
	}
	if _, ok := a.actionIDs[decl.Action]; ok {
		return fmt.Errorf("action %q already registered", decl.Action)
	}
	if _, ok := a.declarations[id]; ok {
		return fmt.Errorf("action %q already registered", id)
	}

	a.declarations[id] = decl
	a.actionIDs[decl.Action] = id
	a.registry.Register(id, action)
	return nil
}

func (a *ActionRegistry) MustRegister(id string, action HIDUsageAction) {
	err := a.Register(id, action)
	if err != nil {
		panic(err)
	}
}

func (a *ActionRegistry) New(stmt actiondsl.Statement) (HIDUsageActionHandler, error) {
	if stmt.Action == "" {
		if len(stmt.Usages) > 0 {
			usages, err := ParseUsages(stmt.Usages)
			if err != nil {
				return nil, err
			}
			return newActionUsageHandler(usages), nil
		} else {
			return nil, fmt.Errorf("empty action statement")
		}
	}
	id, ok := a.actionIDs[stmt.Action]
	if !ok {
		return nil, fmt.Errorf("action %q not found", stmt.Action)
	}
	action, err := actiondsl.NewAction(a.declarations[id], stmt)
	if err != nil {
		return nil, fmt.Errorf("failed to create action %q: %w", stmt.Action, err)
	}
	act, err := a.registry.Get(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get action %q: %w", stmt.Action, err)
	}
	return act.Handler(action.Args(), &HIDActionProvider{
		ActionRegistry: a,
		State:          nil,
	})
}

func NewActionRegistry() *ActionRegistry {
	reg := &ActionRegistry{
		registry:     registry.NewRegistry[HIDUsageAction](),
		declarations: make(map[string]actiondsl.Declaration),
		actionIDs:    make(map[string]string),
	}
	reg.MustRegister("none", ActionNone{})
	reg.MustRegister("tap", ActionTap{})
	reg.MustRegister("tapHold", ActionTapHold{})
	reg.MustRegister("lock", ActionLock{})
	reg.MustRegister("set", ActionSet{})
	return reg
}

type UsageActivator func(usages []hidparse.Usage) (unset func())

type HIDUsageActionMetadata struct {
	DisplayName string
	Description string

	Declaration string
}

type HIDUsageAction interface {
	Metadata() HIDUsageActionMetadata
	Handler(args actiondsl.Arguments, provider *HIDActionProvider) (HIDUsageActionHandler, error)
}

type HIDUsageActionHandler interface {
	// Usages returns all usages that this action emits.
	Usages() []hidparse.Usage
	// Activate is called when the action is activated (i.e. key is pressed down).
	// It should return a function that deactivates the action (i.e. key is released).
	Activate(ctx context.Context, activator UsageActivator) func()
}

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
