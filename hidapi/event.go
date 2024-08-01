package hidapi

import (
	"fmt"
	"strings"
	"sync"
)

type Event struct {
	mu       sync.Mutex
	usages   []UsageEvent
	usageMap map[Usage]int
}

func (h *Event) Clone() *Event {
	h.mu.Lock()
	clone := &Event{
		usageMap: make(map[Usage]int, len(h.usageMap)),
	}
	for _, usage := range h.usages {
		clone.addUsage(usage)
	}
	h.mu.Unlock()
	return clone
}

func NewEvent() *Event {
	return &Event{
		usageMap: make(map[Usage]int, 16),
	}
}

type UsageEvent struct {
	Usage    Usage
	Activate *bool
	Value    *int32
	Delta    *int32
}

func (u UsageEvent) String() string {
	if u.Activate != nil {
		if *u.Activate {
			return "+" + u.Usage.String()
		} else {
			return "-" + u.Usage.String()
		}
	} else if u.Delta != nil {
		if *u.Delta > 0 {
			return fmt.Sprintf("%s+=%d", u.Usage.String(), *u.Delta)
		} else {
			return fmt.Sprintf("%s-=%d", u.Usage.String(), -*u.Delta)
		}
	} else if u.Value != nil {
		return fmt.Sprintf("%s=%d", u.Usage.String(), *u.Value)
	}
	return "(empty)"
}

func (h *Event) IsEmpty() bool {
	h.mu.Lock()
	empty := len(h.usages) == 0
	h.mu.Unlock()
	return empty
}

func (h *Event) addUsage(diff UsageEvent) {
	if idx, ok := h.usageMap[diff.Usage]; ok {
		h.usages[idx] = diff
		return
	}
	h.usages = append(h.usages, diff)
	h.usageMap[diff.Usage] = len(h.usages) - 1
}

func (h *Event) removeUsage(usage Usage) {
	idx, ok := h.usageMap[usage]
	if !ok {
		return
	}
	last := len(h.usages) - 1
	if idx != last {
		h.usages[idx] = h.usages[last]
		h.usageMap[h.usages[idx].Usage] = idx
	}
	h.usages = h.usages[:last]
	delete(h.usageMap, usage)
}

func ptr[T any](v T) *T {
	return &v
}

func (h *Event) Suppress(usages ...Usage) {
	h.mu.Lock()
	for _, usage := range usages {
		h.removeUsage(usage)
	}
	h.mu.Unlock()
}

func (h *Event) Usage(usage Usage) (UsageEvent, bool) {
	h.mu.Lock()
	idx, ok := h.usageMap[usage]
	if !ok {
		h.mu.Unlock()
		return UsageEvent{}, false
	}
	usageEvent := h.usages[idx]
	h.mu.Unlock()
	return usageEvent, true
}

func (h *Event) AddUsage(usages ...UsageEvent) {
	h.mu.Lock()
	for _, usage := range usages {
		h.addUsage(usage)
	}
	h.mu.Unlock()
}

func (h *Event) Activate(usages ...Usage) {
	h.mu.Lock()
	for _, usage := range usages {
		event := UsageEvent{
			Usage:    usage,
			Activate: ptr(true),
		}
		h.addUsage(event)
	}
	h.mu.Unlock()
}

func (h *Event) Deactivate(usages ...Usage) {
	h.mu.Lock()
	for _, usage := range usages {
		diff := UsageEvent{
			Usage:    usage,
			Activate: ptr(false),
		}
		h.addUsage(diff)
	}
	h.mu.Unlock()
}

func (h *Event) SetValue(usage Usage, value int32) {
	h.mu.Lock()
	event := UsageEvent{
		Usage: usage,
		Value: ptr(value),
	}
	h.addUsage(event)
	h.mu.Unlock()
}

func (h *Event) SetDelta(usage Usage, delta int32) {
	h.mu.Lock()
	event := UsageEvent{
		Usage: usage,
		Delta: ptr(delta),
	}
	h.addUsage(event)
	h.mu.Unlock()
}

func (h *Event) Usages() []UsageEvent {
	h.mu.Lock()
	usages := make([]UsageEvent, len(h.usages))
	copy(usages, h.usages)
	h.mu.Unlock()
	return usages
}

func (h *Event) String() string {
	h.mu.Lock()
	var parts []string
	for _, usage := range h.usages {
		parts = append(parts, usage.String())
	}
	h.mu.Unlock()
	return strings.Join(parts, ", ")
}

func (h *Event) Clear() {
	h.mu.Lock()
	h.usages = h.usages[:0]
	clear(h.usageMap)
	h.mu.Unlock()
}
