package hidapi

import "sync"

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

func (h *Event) Usages() []UsageEvent {
	h.mu.Lock()
	usages := make([]UsageEvent, len(h.usages))
	copy(usages, h.usages)
	h.mu.Unlock()
	return usages
}
