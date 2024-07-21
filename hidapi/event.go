package hidapi

type Event struct {
	usages   []UsageEvent
	usageMap map[Usage]int
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
	return len(h.usages) == 0
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
}

func ptr[T any](v T) *T {
	return &v
}

func (h *Event) Suppress(usages ...Usage) {
	for _, usage := range usages {
		h.removeUsage(usage)
	}
}

func (h *Event) Usage(usage Usage) (UsageEvent, bool) {
	idx, ok := h.usageMap[usage]
	if !ok {
		return UsageEvent{}, false
	}
	return h.usages[idx], true
}

func (h *Event) Activate(usages ...Usage) {
	for _, usage := range usages {
		event := UsageEvent{
			Usage:    usage,
			Activate: ptr(true),
		}
		h.addUsage(event)
	}
}

func (h *Event) Deactivate(usages ...Usage) {
	for _, usage := range usages {
		diff := UsageEvent{
			Usage:    usage,
			Activate: ptr(false),
		}
		h.addUsage(diff)
	}
}

func (h *Event) SetValue(usage Usage, value int32) {
	event := UsageEvent{
		Usage: usage,
		Value: ptr(value),
	}
	h.addUsage(event)
}

func (h *Event) Usages() []UsageEvent {
	return h.usages
}
