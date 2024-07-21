package hidevent

import (
	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"github.com/neuroplastio/neuroplastio/pkg/bits"
	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hiddesc"
	"go.uber.org/zap"
)

type HIDEvent struct {
	usages   []UsageEvent
	usageMap map[hidparse.Usage]int
}

func NewHIDEvent() *HIDEvent {
	return &HIDEvent{
		usageMap: make(map[hidparse.Usage]int, 16),
	}
}

type UsageEvent struct {
	Usage    hidparse.Usage
	Activate *bool
	Value    *int32
}

func (h *HIDEvent) IsEmpty() bool {
	return len(h.usages) == 0
}

func (h *HIDEvent) addUsage(diff UsageEvent) {
	if idx, ok := h.usageMap[diff.Usage]; ok {
		h.usages[idx] = diff
		return
	}
	h.usages = append(h.usages, diff)
	h.usageMap[diff.Usage] = len(h.usages) - 1
}

func (h *HIDEvent) removeUsage(usage hidparse.Usage) {
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

func (h *HIDEvent) Suppress(usages ...hidparse.Usage) {
	for _, usage := range usages {
		h.removeUsage(usage)
	}
}

func (h *HIDEvent) Usage(usage hidparse.Usage) (UsageEvent, bool) {
	idx, ok := h.usageMap[usage]
	if !ok {
		return UsageEvent{}, false
	}
	return h.usages[idx], true
}

func (h *HIDEvent) Activate(usages ...hidparse.Usage) {
	for _, usage := range usages {
		event := UsageEvent{
			Usage:    usage,
			Activate: ptr(true),
		}
		h.addUsage(event)
	}
}

func (h *HIDEvent) Deactivate(usages ...hidparse.Usage) {
	for _, usage := range usages {
		diff := UsageEvent{
			Usage:    usage,
			Activate: ptr(false),
		}
		h.addUsage(diff)
	}
}

func (h *HIDEvent) SetValue(usage hidparse.Usage, value int32) {
	event := UsageEvent{
		Usage: usage,
		Value: ptr(value),
	}
	h.addUsage(event)
}

func (h *HIDEvent) Usages() []UsageEvent {
	return h.usages
}

type RTETranscoder struct {
	log         *zap.Logger
	dataItems   map[uint8][]hiddesc.DataItem
	usageSets   map[uint8]map[int]hidparse.UsageSet
	usageValues map[uint8]map[int]hidparse.UsageValues

	reports map[uint8]hidparse.Report
}

func NewRTE(log *zap.Logger, dataItems map[uint8][]hiddesc.DataItem) *RTETranscoder {
	rte := &RTETranscoder{
		log:         log,
		dataItems:   dataItems,
		usageSets:   make(map[uint8]map[int]hidparse.UsageSet),
		usageValues: make(map[uint8]map[int]hidparse.UsageValues),
		reports:     make(map[uint8]hidparse.Report),
	}
	rte.initializeStates()
	return rte
}

// initializeStates creates empty report objects and empty initial states
func (r *RTETranscoder) initializeStates() {
	for reportID, items := range r.dataItems {
		report := hidparse.Report{
			ID:     reportID,
			Fields: make([]bits.Bits, len(items)),
		}
		for i, item := range items {
			// TODO: support empty dynamic arrays
			// TODO: support correct const values (when handling first report, put them into HIDState)
			report.Fields[i] = bits.NewZeros(int(item.ReportCount * item.ReportSize))
		}
		r.reports[reportID] = report
		r.usageSets[reportID] = hidparse.NewUsageSets(items)
		r.usageValues[reportID] = hidparse.NewUsageValuesItems(items)
	}
}

func (r *RTETranscoder) OnReport(report hidparse.Report) HIDEvent {
	lastReport := r.reports[report.ID]
	if len(report.Fields) != len(r.dataItems[report.ID]) {
		r.log.Error("report field count mismatch")
		return *NewHIDEvent()
	}

	event := NewHIDEvent()
	for i, item := range r.dataItems[report.ID] {
		if item.Flags.IsConstant() {
			continue
		}
		reportField := report.Fields[i]
		lastReportField := lastReport.Fields[i]
		if reportField.Equal(lastReportField) {
			continue
		}
		usageSet, ok := r.usageSets[report.ID][i]
		if ok {
			activated, deactivated := hidparse.CompareUsages(usageSet, lastReport.Fields[i], report.Fields[i])
			event.Activate(activated...)
			event.Deactivate(deactivated...)
			continue
		}
		values, ok := r.usageValues[report.ID][i]
		if ok {
			usages := values.Usages()
			for _, usage := range usages {
				t0 := values.GetValue(lastReport.Fields[i], usage)
				t1 := values.GetValue(report.Fields[i], usage)
				if t0 == t1 {
					continue
				}
				event.SetValue(usage, t1)
			}
			continue
		}
	}

	r.reports[report.ID] = r.stripRelativeValues(report)

	return *event
}

func (r *RTETranscoder) stripRelativeValues(report hidparse.Report) hidparse.Report {
	for i, item := range r.dataItems[report.ID] {
		if item.Flags.IsRelative() {
			report.Fields[i].ClearAll()
		}
	}
	return report
}
