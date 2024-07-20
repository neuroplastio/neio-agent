package hidevent

import (
	"fmt"
	"slices"

	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"github.com/neuroplastio/neuroplastio/pkg/bits"
	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hiddesc"
	"go.uber.org/zap"
)

type ETRTranscoder struct {
	log         *zap.Logger
	dataItems   map[uint8][]hiddesc.DataItem
	usageSets   map[uint8]map[int]hidparse.UsageSet
	usageValues map[uint8]map[int]hidparse.UsageValues

	usageSetRanges   map[uint16][]usageRange
	usageValuesIndex map[hidparse.Usage]itemAddress
	reports          map[uint8]hidparse.Report

	usageActivations map[uint8]map[hidparse.Usage]int
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

func NewETR(log *zap.Logger, dataItems map[uint8][]hiddesc.DataItem) *ETRTranscoder {
	etr := &ETRTranscoder{
		log:              log,
		dataItems:        dataItems,
		usageSets:        make(map[uint8]map[int]hidparse.UsageSet),
		usageValues:      make(map[uint8]map[int]hidparse.UsageValues),
		usageSetRanges:   make(map[uint16][]usageRange),
		usageValuesIndex: make(map[hidparse.Usage]itemAddress),
		reports:          make(map[uint8]hidparse.Report),
		usageActivations: make(map[uint8]map[hidparse.Usage]int),
	}
	etr.initializeStates()
	return etr
}

func (t *ETRTranscoder) initializeStates() {
	for reportID, items := range t.dataItems {
		report := hidparse.Report{
			ID:     reportID,
			Fields: make([]bits.Bits, len(items)),
		}
		for i, item := range items {
			// TODO: support empty dynamic arrays
			// TODO: support const values (from first HIDState)
			report.Fields[i] = bits.NewZeros(int(item.ReportCount * item.ReportSize))
			t.log.Debug("[ETR] DataItem",
				zap.Uint8("reportId", reportID),
				zap.Any("usagePage", item.UsagePage),
				zap.Any("usageMinimum", item.UsageMinimum),
				zap.Any("usageMaximum", item.UsageMaximum),
			)
		}
		t.reports[reportID] = report
		t.usageSets[reportID] = hidparse.NewUsageSets(items)
		for idx, usageSet := range t.usageSets[reportID] {
			t.log.Debug("[ETR] Usage Set",
				zap.Uint8("reportId", reportID),
				zap.Int("itemIdx", idx),
				zap.String("page", fmt.Sprintf("%02x", usageSet.UsagePage())),
				zap.Any("min", usageSet.UsageMinimum()),
				zap.Any("max", usageSet.UsageMaximum()),
			)
			rang := usageRange{
				start:    usageSet.UsageMinimum(),
				end:      usageSet.UsageMaximum(),
				addr:     itemAddress{reportID: reportID, itemIdx: idx},
				dataItem: items[idx],
			}
			t.usageSetRanges[usageSet.UsagePage()] = append(t.usageSetRanges[usageSet.UsagePage()], rang)
		}

		t.usageValues[reportID] = hidparse.NewUsageValuesItems(items)
		for idx, usageValue := range t.usageValues[reportID] {
			// TODO: handle overlapping usages
			for _, usage := range usageValue.Usages() {
				t.log.Debug("[ETR] Usage Value",
					zap.Uint8("reportId", reportID),
					zap.Int("itemIdx", idx),
					zap.String("usage", usage.String()),
				)
				t.usageValuesIndex[usage] = itemAddress{
					reportID: reportID,
					itemIdx:  idx,
				}
			}
		}
		t.usageActivations[reportID] = make(map[hidparse.Usage]int)
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
			t.log.Debug("[ETR] Usage Set Range",
				zap.Uint16("page", usagePage),
				zap.Any("range", []uint16{item.start, item.end}),
				zap.Any("reportId", item.addr.reportID),
				zap.Any("itemIdx", item.addr.itemIdx),
			)
		}
	}
}

func (t *ETRTranscoder) usageRange(usage hidparse.Usage) (usageRange, bool) {
	for _, rang := range t.usageSetRanges[usage.Page()] {
		if rang.Contains(usage.ID()) {
			return rang, true
		}
	}
	return usageRange{}, false
}

func (t *ETRTranscoder) OnEvent(e HIDEvent) []hidparse.Report {
	reports := make([]hidparse.Report, 0, 1)
	reportMap := make(map[uint8]int)
	getReport := func(reportID uint8) hidparse.Report {
		_, ok := reportMap[reportID]
		if !ok {
			reports = append(reports, t.reports[reportID])
			reportMap[reportID] = len(reports) - 1
		}
		return reports[reportMap[reportID]]
	}
	if t.log.Level() <= zap.DebugLevel {
		t.log.Debug("[ETR] Event",
			zap.Any("usages", e.usages),
		)
	}
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
		dataItem := t.dataItems[addr.reportID][addr.itemIdx]
		switch {
		case usageEvent.Activate != nil && *usageEvent.Activate:
			if dataItem.Flags.IsRelative() {
				t.usageSets[addr.reportID][addr.itemIdx].SetUsage(report.Fields[addr.itemIdx], usage)
			} else {
				t.usageActivations[addr.reportID][usage]++
				count := t.usageActivations[addr.reportID][usage]
				t.log.Debug("[ETR] Usage activation count", zap.Int("count", count), zap.String("usage", usage.String()))
				if count == 1 {
					t.usageSets[addr.reportID][addr.itemIdx].SetUsage(report.Fields[addr.itemIdx], usage)
				}
			}
		case usageEvent.Activate != nil && !*usageEvent.Activate:
			if dataItem.Flags.IsRelative() {
				t.usageSets[addr.reportID][addr.itemIdx].ClearUsage(report.Fields[addr.itemIdx], usage)
			} else {
				t.usageActivations[addr.reportID][usage]--
				count := t.usageActivations[addr.reportID][usage]
				t.log.Debug("[ETR] Usage activation count", zap.Int("count", count), zap.String("usage", usage.String()))
				if count == 0 {
					t.log.Debug("[ETR] Before clearing", zap.Any("fields", report.FieldsStrings()))
					t.usageSets[addr.reportID][addr.itemIdx].ClearUsage(report.Fields[addr.itemIdx], usage)
					t.log.Debug("[ETR] After clearing", zap.Any("fields", report.FieldsStrings()))
					delete(t.usageActivations[addr.reportID], usage)
				}
			}
		case usageEvent.Value != nil:
			t.usageValues[addr.reportID][addr.itemIdx].SetValue(report.Fields[addr.itemIdx], usage, *usageEvent.Value)
		}
	}

	for _, report := range reports {
		t.reports[report.ID] = t.stripRelativeValues(report.Clone())
		if t.log.Level() <= zap.DebugLevel {
			t.log.Debug("[ETR] Report",
				zap.Uint8("id", report.ID),
				zap.Any("fields", report.FieldsStrings()),
			)
		}
	}

	return reports
}

func (t *ETRTranscoder) stripRelativeValues(report hidparse.Report) hidparse.Report {
	for i, item := range t.dataItems[report.ID] {
		if item.Flags.IsRelative() {
			report.Fields[i].ClearAll()
		}
	}
	return report
}
