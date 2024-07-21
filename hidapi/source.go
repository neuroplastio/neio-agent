package hidapi

import (
	"github.com/neuroplastio/neuroplastio/hidapi/hiddesc"
	"github.com/neuroplastio/neuroplastio/pkg/bits"
	"go.uber.org/zap"
)

type EventSource struct {
	log         *zap.Logger
	dataItems   map[uint8][]hiddesc.DataItem
	usageSets   map[uint8]map[int]UsageSet
	usageValues map[uint8]map[int]UsageValues

	reports map[uint8]Report
}

func NewEventSource(log *zap.Logger, dataItems map[uint8][]hiddesc.DataItem) *EventSource {
	rte := &EventSource{
		log:         log,
		dataItems:   dataItems,
		usageSets:   make(map[uint8]map[int]UsageSet),
		usageValues: make(map[uint8]map[int]UsageValues),
		reports:     make(map[uint8]Report),
	}
	rte.initializeStates()
	return rte
}

func (r *EventSource) initializeStates() {
	for reportID, items := range r.dataItems {
		report := Report{
			ID:     reportID,
			Fields: make([]bits.Bits, len(items)),
		}
		for i, item := range items {
			// TODO: support empty dynamic arrays
			// TODO: support correct const values (when handling first report)
			report.Fields[i] = bits.NewZeros(int(item.ReportCount * item.ReportSize))
		}
		r.reports[reportID] = report
		r.usageSets[reportID] = NewUsageSets(items)
		r.usageValues[reportID] = NewUsageValuesItems(items)
	}
}

func (r *EventSource) OnReport(report Report) *Event {
	lastReport := r.reports[report.ID]
	if len(report.Fields) != len(r.dataItems[report.ID]) {
		r.log.Error("report field count mismatch")
		return NewEvent()
	}

	event := NewEvent()
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
			activated, deactivated := UsageSetDiff(usageSet, lastReport.Fields[i], report.Fields[i])
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

	return event
}

func (r *EventSource) stripRelativeValues(report Report) Report {
	for i, item := range r.dataItems[report.ID] {
		if item.Flags.IsRelative() {
			report.Fields[i].ClearAll()
		}
	}
	return report
}
