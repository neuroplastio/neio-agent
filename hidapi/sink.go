package hidapi

import (
	"fmt"
	"slices"
	"time"

	"github.com/neuroplastio/neio-agent/hidapi/hiddesc"
	"github.com/neuroplastio/neio-agent/pkg/bits"
	"go.uber.org/zap"
)

type EventSink struct {
	log         *zap.Logger
	dataItems   DataItemSet
	usageSets   map[uint8]map[int]UsageSet
	usageValues map[uint8]map[int]UsageValues

	usageSetRanges   map[uint16][]usageRange
	usageValuesIndex map[Usage]itemAddress
	reports          map[uint8]Report

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

func NewEventSink(log *zap.Logger, dataItems DataItemSet) *EventSink {
	etr := &EventSink{
		log:                   log,
		dataItems:             dataItems,
		usageSets:             make(map[uint8]map[int]UsageSet),
		usageValues:           make(map[uint8]map[int]UsageValues),
		usageSetRanges:        make(map[uint16][]usageRange),
		usageValuesIndex:      make(map[Usage]itemAddress),
		reports:               make(map[uint8]Report),
		usageActivations:      make(map[uint8]map[Usage]int),
		activationMinInterval: time.Millisecond,
	}
	etr.initializeStates()
	return etr
}

func (t *EventSink) initializeStates() {
	for _, rd := range t.dataItems.Reports() {
		report := Report{
			ID:     rd.ID,
			Fields: make([]bits.Bits, len(rd.DataItems)),
		}
		for i, item := range rd.DataItems {
			// TODO: support empty dynamic arrays
			// TODO: support const values (from first HIDState)
			report.Fields[i] = bits.NewZeros(int(item.ReportCount * item.ReportSize))
			t.log.Debug("DataItem",
				zap.Uint8("reportId", rd.ID),
				zap.Any("usagePage", item.UsagePage),
				zap.Any("usageMinimum", item.UsageMinimum),
				zap.Any("usageMaximum", item.UsageMaximum),
			)
		}
		t.reports[rd.ID] = report
		t.usageSets[rd.ID] = NewUsageSets(rd.DataItems)
		for idx, usageSet := range t.usageSets[rd.ID] {
			t.log.Debug("Usage Set",
				zap.Uint8("reportId", rd.ID),
				zap.Int("itemIdx", idx),
				zap.String("page", fmt.Sprintf("%02x", usageSet.UsagePage())),
				zap.Any("min", usageSet.UsageMinimum()),
				zap.Any("max", usageSet.UsageMaximum()),
			)
			rang := usageRange{
				start:    usageSet.UsageMinimum(),
				end:      usageSet.UsageMaximum(),
				addr:     itemAddress{reportID: rd.ID, itemIdx: idx},
				dataItem: rd.DataItems[idx],
			}
			t.usageSetRanges[usageSet.UsagePage()] = append(t.usageSetRanges[usageSet.UsagePage()], rang)
		}

		t.usageValues[rd.ID] = NewUsageValuesItems(rd.DataItems)
		for idx, usageValue := range t.usageValues[rd.ID] {
			// TODO: handle overlapping usages
			for _, usage := range usageValue.Usages() {
				t.log.Debug("Usage Value",
					zap.Uint8("reportId", rd.ID),
					zap.Int("itemIdx", idx),
					zap.String("usage", usage.String()),
				)
				t.usageValuesIndex[usage] = itemAddress{
					reportID: rd.ID,
					itemIdx:  idx,
				}
			}
		}
		t.usageActivations[rd.ID] = make(map[Usage]int)
	}
	for usagePage, items := range t.usageSetRanges {
		slices.SortFunc(items, func(a, b usageRange) int {
			if a.start < b.start {
				return -1
			}
			if a.start > b.start {
				return 1
			}
			return 0
		})
		t.usageSetRanges[usagePage] = items
	}
	for usagePage, items := range t.usageSetRanges {
		for _, item := range items {
			t.log.Debug("Usage Set Range",
				zap.Uint16("page", usagePage),
				zap.Any("range", []uint16{item.start, item.end}),
				zap.Any("reportId", item.addr.reportID),
				zap.Any("itemIdx", item.addr.itemIdx),
			)
		}
	}
}

func (t *EventSink) usageRange(usage Usage) (usageRange, bool) {
	for _, rang := range t.usageSetRanges[usage.Page()] {
		if rang.Contains(usage.ID()) {
			return rang, true
		}
	}
	return usageRange{}, false
}

func (t *EventSink) OnEvent(e *Event) []Report {
	reports := make([]Report, 0, 1)
	reportMap := make(map[uint8]int)
	getReport := func(reportID uint8) Report {
		_, ok := reportMap[reportID]
		if !ok {
			reports = append(reports, t.reports[reportID])
			reportMap[reportID] = len(reports) - 1
		}
		return reports[reportMap[reportID]]
	}
	t.log.Debug("Event", zap.Any("usages", e.Usages()))
	for _, usageEvent := range e.Usages() {
		usage := usageEvent.Usage
		var (
			addr itemAddress
		)
		switch {
		case usageEvent.Activate != nil:
			rang, ok := t.usageRange(usage)
			if !ok {
				t.log.Warn("[ETR] Usage has no matching report",
					zap.String("usage", usage.String()),
				)
				continue
			}
			addr = rang.addr
		case usageEvent.Value != nil:
			a, ok := t.usageValuesIndex[usage]
			if !ok {
				t.log.Warn("[ETR] Usage has no matching report",
					zap.String("usage", usage.String()),
				)
				continue
			}
			addr = a
		default:
			t.log.Warn("[ETR] Usage event has no action")
			continue
		}
		report := getReport(addr.reportID)
		dataItem := t.dataItems.Report(addr.reportID)[addr.itemIdx]
		switch {
		case usageEvent.Activate != nil && *usageEvent.Activate:
			if dataItem.Flags.IsRelative() {
				t.usageSets[addr.reportID][addr.itemIdx].SetUsage(report.Fields[addr.itemIdx], usage)
			} else {
				t.usageActivations[addr.reportID][usage]++
				count := t.usageActivations[addr.reportID][usage]
				if count == 1 {
					t.usageSets[addr.reportID][addr.itemIdx].SetUsage(report.Fields[addr.itemIdx], usage)
					// TODO: configurable minInterval with 1ms by default
					// TODO: non-blocking rate limiting
					sinceLast := time.Since(t.lastActivation)
					if sinceLast < t.activationMinInterval {
						t.log.Info("Activation rate limit")
						time.Sleep(t.activationMinInterval - sinceLast)
					}
					t.lastActivation = time.Now()
				}
			}
		case usageEvent.Activate != nil && !*usageEvent.Activate:
			if dataItem.Flags.IsRelative() {
				t.usageSets[addr.reportID][addr.itemIdx].ClearUsage(report.Fields[addr.itemIdx], usage)
			} else {
				t.usageActivations[addr.reportID][usage]--
				count := t.usageActivations[addr.reportID][usage]
				if count <= 0 {
					t.usageSets[addr.reportID][addr.itemIdx].ClearUsage(report.Fields[addr.itemIdx], usage)
					delete(t.usageActivations[addr.reportID], usage)
					// TODO: configurable minInterval with 1ms by default
					// TODO: non-blocking rate limiting
					sinceLast := time.Since(t.lastActivation)
					if sinceLast < t.activationMinInterval {
						t.log.Info("Activation rate limit")
						time.Sleep(t.activationMinInterval - sinceLast)
					}
					t.lastActivation = time.Now()
				}
			}
		case usageEvent.Value != nil:
			t.usageValues[addr.reportID][addr.itemIdx].SetValue(report.Fields[addr.itemIdx], usage, *usageEvent.Value)
		}
	}

	for _, report := range reports {
		t.reports[report.ID] = t.stripRelativeValues(report.Clone())
	}

	return reports
}

func (t *EventSink) stripRelativeValues(report Report) Report {
	for i, item := range t.dataItems.Report(report.ID) {
		if item.Flags.IsRelative() {
			report.Fields[i].ClearAll()
		}
	}
	return report
}
