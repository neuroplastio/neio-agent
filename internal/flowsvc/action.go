package flowsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"github.com/neuroplastio/neuroplastio/pkg/registry"
	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hidusage/usagepages"
)

type HIDActionProvider struct {
	ActionRegistry *ActionRegistry
	State          *FlowState
}

type ActionRegistry struct {
	registry *registry.Registry[HIDUsageAction, *HIDActionProvider]
}

func (a *ActionRegistry) NewFromJSON(data json.RawMessage) (HIDUsageAction, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty action config")
	}
	if string(data) == "null" || string(data) == `""` {
		return a.registry.New("none", nil)
	}
	if data[0] == '"' {
		var str string
		err := json.Unmarshal(data, &str)
		if err != nil {
			return nil, err
		}
		return a.NewFromString(str)
	}
	var actionStringMap map[string]json.RawMessage
	err := json.Unmarshal(data, &actionStringMap)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal action JSON: %w", err)
	}
	if len(actionStringMap) != 1 {
		return nil, fmt.Errorf("invalid action config: %s", data)
	}
	for actionType, actionConfig := range actionStringMap {
		action, err := a.registry.New(actionType, actionConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create action %s: %w", actionType, err)
		}
		return action, nil
	}
	return nil, fmt.Errorf("no action type found in %s", data)
}

func (a *ActionRegistry) NewFromString(str string) (HIDUsageAction, error) {
	usages, err := ParseUsageCombo(str)
	if err != nil {
		return nil, err
	}
	// TODO: support DSL
	return &UsageAction{
		usages: usages,
	}, nil
}

func NewActionRegistry(state *FlowState) *ActionRegistry {
	fmt.Println(state)
	provider := &HIDActionProvider{
		State: state,
	}
	reg := registry.NewRegistry[HIDUsageAction, *HIDActionProvider](provider)
	provider.ActionRegistry = &ActionRegistry{registry: reg}
	reg.Register("none", NewActionNone)
	reg.Register("usage", NewUsageAction) // *
	reg.Register("tapHold", NewTapHoldAction) // tapHold(onTap, onHold[, delay])
	reg.Register("lock", NewLockAction) // lock(*)
	reg.Register("set", NewSetAction) // set(name, value)
	return provider.ActionRegistry
}

type UsageActivator func(usages []hidparse.Usage) (unset func())

type HIDUsageAction interface {
	// Usages returns all usages that this action emits.
	Usages() []hidparse.Usage
	// Activate is called when the action is activated (i.e. key is pressed down).
	// It should return a function that deactivates the action (i.e. key is released).
	Activate(ctx context.Context, activator UsageActivator) func()
}


func ParseUsageCombo(str string) ([]hidparse.Usage, error) {
	parts := strings.Split(str, "+")
	usages := make([]hidparse.Usage, 0, len(parts))
	for _, part := range parts {
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
