package hidapi

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/neuroplastio/neio-agent/hidapi/hiddesc"
	"github.com/neuroplastio/neio-agent/pkg/bits"
	"go.uber.org/zap"
)

type ReportState struct {
	log       *zap.Logger
	decoder   *ReportDecoder
	dataItems DataItemSet

	usageSets   map[uint8]map[int]UsageSet
	usageValues map[uint8]map[int]UsageValues

	usageSetRanges   map[uint16][]usageRange
	usageSetMap      map[uint16]map[uint16]itemAddress
	usageValuesIndex map[Usage]itemAddress

	mu                    sync.RWMutex
	reports               map[uint8]Report
	usageActivations      map[uint8]map[Usage]int
	lastActivation        time.Time
	activationMinInterval time.Duration
}

type usageRange struct {
	start uint16
	end   uint16

	addr     itemAddress
	dataItem hiddesc.DataItem
}

func (u usageRange) Contains(usageID uint16) bool {
	return usageID >= u.start && usageID <= u.end
}

type itemAddress struct {
	reportID uint8
	itemIdx  int
}

func NewReportState(log *zap.Logger, dataItems DataItemSet) *ReportState {
	rte := &ReportState{
		log:       log,
		decoder:   NewReportDecoder(dataItems),
		dataItems: dataItems,

		usageSets:        make(map[uint8]map[int]UsageSet),
		usageValues:      make(map[uint8]map[int]UsageValues),
		usageSetRanges:   make(map[uint16][]usageRange),
		usageSetMap:      make(map[uint16]map[uint16]itemAddress),
		usageValuesIndex: make(map[Usage]itemAddress),

		reports:               make(map[uint8]Report),
		usageActivations:      make(map[uint8]map[Usage]int),
		activationMinInterval: 500 * time.Microsecond,
	}
	rte.initializeStates()
	return rte
}

func (r *ReportState) initializeStates() {
	for _, rd := range r.dataItems.Reports() {
		report := Report{
			ID:     rd.ID,
			Fields: make([]bits.Bits, len(rd.DataItems)),
		}
		for i, item := range rd.DataItems {
			report.Fields[i] = bits.NewZeros(int(item.ReportCount * item.ReportSize))
		}
		r.mu.Lock()
		r.reports[rd.ID] = report
		r.mu.Unlock()
		r.usageSets[rd.ID] = NewUsageSets(rd.DataItems)
		r.usageValues[rd.ID] = NewUsageValuesItems(rd.DataItems)

		for idx, usageSet := range r.usageSets[rd.ID] {
			if unordered, ok := usageSet.(UnorderedUsageSet); ok {
				var usageIDs []string
				for _, usageID := range unordered.UsageIDs() {
					usageIDs = append(usageIDs, NewUsage(unordered.UsagePage(), usageID).String())
				}
				r.log.Debug("Unordered Usage Set",
					zap.Uint8("reportId", rd.ID),
					zap.Int("itemIdx", idx),
					zap.String("page", fmt.Sprintf("%02x", usageSet.UsagePage())),
					zap.Any("usages", usageIDs),
				)
				if _, ok := r.usageSetMap[usageSet.UsagePage()]; !ok {
					r.usageSetMap[usageSet.UsagePage()] = make(map[uint16]itemAddress)
				}
				for _, usageID := range unordered.UsageIDs() {
					r.usageSetMap[usageSet.UsagePage()][usageID] = itemAddress{reportID: rd.ID, itemIdx: idx}
				}
				continue
			}
			if ordered, ok := usageSet.(OrderedUsageSet); ok {
				rang := usageRange{
					start:    ordered.UsageMinimum(),
					end:      ordered.UsageMaximum(),
					addr:     itemAddress{reportID: rd.ID, itemIdx: idx},
					dataItem: rd.DataItems[idx],
				}
				r.usageSetRanges[usageSet.UsagePage()] = append(r.usageSetRanges[usageSet.UsagePage()], rang)
				continue
			}
			r.log.Error("Unknown Usage Set type")
		}
		for idx, usageValue := range r.usageValues[rd.ID] {
			// TODO: handle overlapping usages
			for _, usage := range usageValue.Usages() {
				r.log.Debug("Usage Value",
					zap.Uint8("reportId", rd.ID),
					zap.Int("itemIdx", idx),
					zap.String("usage", usage.String()),
				)
				r.usageValuesIndex[usage] = itemAddress{
					reportID: rd.ID,
					itemIdx:  idx,
				}
			}
		}
		r.usageActivations[rd.ID] = make(map[Usage]int)

	}
	for usagePage, items := range r.usageSetRanges {
		slices.SortFunc(items, func(a, b usageRange) int {
			if a.dataItem.ReportSize < b.dataItem.ReportSize {
				return -1
			}
			if a.dataItem.ReportSize > b.dataItem.ReportSize {
				return 1
			}
			if a.start < b.start {
				return -1
			}
			if a.start > b.start {
				return 1
			}
			return 0
		})
		r.usageSetRanges[usagePage] = items
	}
}

func (r *ReportState) InitReports(reportGetter func(reportID uint8) ([]byte, error)) ([]*Event, error) {
	var events []*Event
	for _, rd := range r.dataItems.Reports() {
		reportData, err := reportGetter(rd.ID)
		if err != nil {
			r.log.Warn("failed to get report", zap.Error(err))
			continue
		}
		event := r.ApplyReport(reportData)
		if !event.IsEmpty() {
			events = append(events, event)
		}
	}
	return events, nil
}

func (r *ReportState) ApplyReport(reportData []byte) *Event {
	report, ok := r.decoder.Decode(reportData)
	if !ok {
		r.log.Error("failed to decode report")
		return nil
	}
	dataItems := r.dataItems.Report(report.ID)
	if len(report.Fields) != len(dataItems) {
		r.log.Error("report field count mismatch")
		return NewEvent()
	}
	r.mu.RLock()
	lastReport := r.reports[report.ID]
	r.mu.RUnlock()

	event := NewEvent()
	for i, item := range dataItems {
		reportField := report.Fields[i]
		lastReportField := lastReport.Fields[i]
		usageSet, ok := r.usageSets[report.ID][i]
		if ok {
			if reportField.Equal(lastReportField) {
				continue
			}
			activated, deactivated := UsageSetDiff(usageSet, lastReport.Fields[i], report.Fields[i])
			event.Activate(activated...)
			event.Deactivate(deactivated...)
			continue
		}
		values, ok := r.usageValues[report.ID][i]
		if ok {
			usages := values.Usages()
			for _, usage := range usages {
				if item.Flags.IsRelative() {
					if reportField.Equal(lastReportField) {
						continue
					}
					t0 := values.GetValue(lastReport.Fields[i], usage)
					t1 := values.GetValue(report.Fields[i], usage)
					if t0 == t1 {
						continue
					}
					event.SetDelta(usage, t1-t0)
				} else {
					event.SetValue(usage, values.GetValue(report.Fields[i], usage))
				}
			}
			continue
		}
	}

	r.mu.Lock()
	r.reports[report.ID] = r.stripRelativeValues(report)
	r.mu.Unlock()

	return event
}

func (r *ReportState) getUsageSet(usage Usage) (usageRange, bool) {
	for _, rang := range r.usageSetRanges[usage.Page()] {
		if rang.Contains(usage.ID()) {
			return rang, true
		}
	}
	return usageRange{}, false
}

func (r *ReportState) GetReport(reportID uint8) ([]byte, error) {
	r.mu.RLock()
	report, ok := r.reports[reportID]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("report ID %d not found", reportID)
	}
	return EncodeReport(report).Bytes(), nil
}

func (r *ReportState) ApplyEvent(e *Event) [][]byte {
	reports := make([]Report, 0, 1)
	reportMap := make(map[uint8]int)
	getReport := func(reportID uint8) Report {
		_, ok := reportMap[reportID]
		if !ok {
			r.mu.RLock()
			reports = append(reports, r.reports[reportID])
			r.mu.RUnlock()
			reportMap[reportID] = len(reports) - 1
		}
		return reports[reportMap[reportID]]
	}
	r.log.Debug("Event", zap.String("event", e.String()))
	for _, usageEvent := range e.Usages() {
		usage := usageEvent.Usage
		var (
			addr itemAddress
		)
		switch {
		case usageEvent.Activate != nil:
			if a, ok := r.usageSetMap[usage.Page()][usage.ID()]; ok {
				addr = a
				break
			}
			if rang, ok := r.getUsageSet(usage); ok {
				addr = rang.addr
				break
			}
			r.log.Warn("Usage has no matching report",
				zap.String("usage", usage.String()),
			)
			continue
		case usageEvent.Delta != nil || usageEvent.Value != nil:
			a, ok := r.usageValuesIndex[usage]
			if !ok {
				r.log.Warn("Usage has no matching report",
					zap.String("usage", usage.String()),
				)
				continue
			}
			addr = a
		default:
			r.log.Warn("Usage event has no action")
			continue
		}
		report := getReport(addr.reportID)
		dataItem := r.dataItems.Report(addr.reportID)[addr.itemIdx]
		switch {
		case usageEvent.Activate != nil && *usageEvent.Activate:
			if dataItem.Flags.IsRelative() {
				r.usageSets[addr.reportID][addr.itemIdx].SetUsage(report.Fields[addr.itemIdx], usage)
			} else {
				r.usageActivations[addr.reportID][usage]++
				count := r.usageActivations[addr.reportID][usage]
				if count == 1 {
					r.usageSets[addr.reportID][addr.itemIdx].SetUsage(report.Fields[addr.itemIdx], usage)
					// TODO: configurable minInterval with 1ms by default
					// TODO: non-blocking rate limiting
					sinceLast := time.Since(r.lastActivation)
					if sinceLast < r.activationMinInterval {
						r.log.Warn("Activation rate limit", zap.Duration("sinceLast", sinceLast))
						time.Sleep(r.activationMinInterval - sinceLast)
					}
					r.lastActivation = time.Now()
				}
			}
		case usageEvent.Activate != nil && !*usageEvent.Activate:
			if dataItem.Flags.IsRelative() {
				r.usageSets[addr.reportID][addr.itemIdx].ClearUsage(report.Fields[addr.itemIdx], usage)
			} else {
				r.usageActivations[addr.reportID][usage]--
				count := r.usageActivations[addr.reportID][usage]
				if count <= 0 {
					r.usageSets[addr.reportID][addr.itemIdx].ClearUsage(report.Fields[addr.itemIdx], usage)
					delete(r.usageActivations[addr.reportID], usage)
					// TODO: configurable minInterval with 1ms by default
					// TODO: non-blocking rate limiting
					sinceLast := time.Since(r.lastActivation)
					if sinceLast < r.activationMinInterval {
						r.log.Warn("Activation rate limit", zap.Duration("sinceLast", sinceLast))
						time.Sleep(r.activationMinInterval - sinceLast)
					}
					r.lastActivation = time.Now()
				}
			}
		case usageEvent.Delta != nil:
			current := r.usageValues[addr.reportID][addr.itemIdx].GetValue(report.Fields[addr.itemIdx], usage)
			r.usageValues[addr.reportID][addr.itemIdx].SetValue(report.Fields[addr.itemIdx], usage, current+*usageEvent.Delta)
		case usageEvent.Value != nil:
			r.usageValues[addr.reportID][addr.itemIdx].SetValue(report.Fields[addr.itemIdx], usage, *usageEvent.Value)
		}
	}

	encoded := make([][]byte, len(reports))
	for i, report := range reports {
		r.mu.Lock()
		r.reports[report.ID] = r.stripRelativeValues(report.Clone())
		r.mu.Unlock()
		encoded[i] = EncodeReport(report).Bytes()
	}

	return encoded
}

func (r *ReportState) stripRelativeValues(report Report) Report {
	for i, item := range r.dataItems.Report(report.ID) {
		if item.Flags.IsRelative() {
			report.Fields[i].ClearAll()
		}
	}
	return report
}
