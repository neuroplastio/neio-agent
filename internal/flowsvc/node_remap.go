package flowsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hiddesc"
	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hidusage/usagepages"
)

func NewRemapNode(data json.RawMessage, provider *NodeProvider) (Node, error) {
	// TODO: parse remap config into action-driven mappings
	return &Remap{
	}, nil
}

type Remap struct {
	desc       []byte
	descParsed hiddesc.ReportDescriptor
	cfg        NodeConfigRemap
	usageSets  map[uint8]map[int]hidparse.UsageSet

	lastReports  map[uint8]hidparse.Report
	lastResults  map[uint8]hidparse.Report
	mappings     []mapping
	clearMapping map[int]struct{}

	state *FlowState

}

func (r *Remap) Start(ctx context.Context, up FlowStream, down FlowStream) error {
	in := up.Subscribe(ctx)
	r.lastReports = make(map[uint8]hidparse.Report)
	r.lastResults = make(map[uint8]hidparse.Report)
	for {
		select {
		case event := <-in:
			reports := r.processReport(ctx, event.Message.HIDReport)
			for _, report := range reports {
				down.Broadcast(ctx, FlowEvent{HIDReport: report})
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (r *Remap) triggerMappings(report hidparse.Report) {
	for mappingIdx, mapping := range r.mappings {
		for usageIdx, usage := range mapping.from {
			found := false
			for fieldIdx, field := range report.Fields {
				if _, ok := r.usageSets[report.ID][fieldIdx]; !ok {
					continue
				}
				if r.mappings[mappingIdx].fromItems[usageIdx] != nil && r.mappings[mappingIdx].fromItems[usageIdx].reportID != report.ID {
					continue
				}
				if r.usageSets[report.ID][fieldIdx].HasUsage(field, usage.ID()) {
					r.mappings[mappingIdx].fromItems[usageIdx] = &itemAddress{
						reportID: report.ID,
						index:    fieldIdx,
					}
					found = true
					break
				}
			}
			if !found && r.mappings[mappingIdx].fromItems[usageIdx] != nil && r.mappings[mappingIdx].fromItems[usageIdx].reportID == report.ID {
				r.mappings[mappingIdx].fromItems[usageIdx] = nil
			}
		}
		isTriggered := true
		for _, item := range r.mappings[mappingIdx].fromItems {
			if item == nil {
				isTriggered = false
				break
			}
		}
		if mapping.isTriggered && !isTriggered {
			r.clearMapping[mappingIdx] = struct{}{}
		}
		r.mappings[mappingIdx].isTriggered = isTriggered
	}
}

func (r *Remap) processReport(ctx context.Context, report hidparse.Report) []hidparse.Report {
	r.triggerMappings(report)

	r.lastReports[report.ID] = report

	reports := make(map[uint8]hidparse.Report, len(r.lastReports))
	reports[report.ID] = report
	for id, lastResult := range r.lastResults {
		if id == report.ID {
			continue
		}
		reports[id] = lastResult.Clone()
	}

	for i, mapping := range r.mappings {
		if mapping.isTriggered {
			for i, item := range mapping.fromItems {
				r.usageSets[item.reportID][item.index].ClearUsage(reports[item.reportID].Fields[item.index], mapping.from[i].ID())
			}
			mapping.action.Activate(func(usageIDs []uint16) func() {
				for _, usageID := range usageIDs {
					r.usageSets[item.reportID][item.index].SetUsage(reports[item.reportID].Fields[item.index], mapping.to[i].ID())
				}
			})
			for i, item := range mapping.toItems {
				r.usageSets[item.reportID][item.index].SetUsage(reports[item.reportID].Fields[item.index], mapping.to[i].ID())
			}
			for name, values := range mapping.setVar {
				r.state.SetEnumValue(ctx, name, values[0])
			}
		} else if _, ok := r.clearMapping[i]; ok {
			for i, item := range mapping.toItems {
				r.usageSets[item.reportID][item.index].ClearUsage(reports[item.reportID].Fields[item.index], mapping.to[i].ID())
			}
			for name, values := range mapping.setVar {
				r.state.SetEnumValue(ctx, name, values[1])
			}
			delete(r.clearMapping, i)
		}
	}

	reportsToSend := make([]hidparse.Report, 0, len(reports))
	for _, rep := range reports {
		if rep.ID == report.ID && r.hasRelativeValues(rep.ID) {
			stripped, changed := r.stripRelativeValues(rep)
			r.lastResults[rep.ID] = stripped
			if changed {
				reportsToSend = append(reportsToSend, rep)
			}
			continue
		}
		lastResult, ok := r.lastResults[rep.ID]
		if ok && lastResult.Equal(rep) {
			continue
		}
		r.lastResults[rep.ID] = rep
		reportsToSend = append(reportsToSend, rep)
	}

	return reportsToSend
}

func (r *Remap) hasRelativeValues(id uint8) bool {
	report, ok := r.descParsed.GetInputReport(id)
	if !ok {
		return false
	}
	for _, item := range report.Items {
		if item.Flags.IsRelative() {
			return true
		}
	}

	return false
}

func (r *Remap) stripRelativeValues(report hidparse.Report) (hidparse.Report, bool) {
	result := report.Clone()
	desc, ok := r.descParsed.GetInputReport(report.ID)
	if !ok {
		return result, false
	}
	changed := false
	for i, item := range desc.Items {
		if item.Flags.IsRelative() {
			changed = changed || result.Fields[i].ClearAll()
		}
	}
	return result, changed
}

func (r *Remap) getReportIDForUsage(usagePage, usageID uint16) (uint8, int, bool) {
	reports := r.descParsed.GetInputReports()
	for _, report := range reports {
		for i, item := range report.Items {
			if item.UsagePage != usagePage {
				continue
			}
			if len(item.UsageIDs) == 1 && item.UsageIDs[0] == usageID {
				return report.ID, i, true
			}
			if item.UsageMaximum != 0 {
				if usageID >= item.UsageMinimum && usageID <= item.UsageMaximum {
					return report.ID, i, true
				}
			}
		}
	}
	return 0, 0, false
}

func ParseUsageCombo(str string) ([]Usage, error) {
	parts := strings.Split(str, "+")
	usages := make([]Usage, 0, len(parts))
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

func ParseUsage(str string) (Usage, error) {
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
		return NewUsage(usagepages.KeyboardKeypad, uint16(code)), nil
	case "btn":
		code, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("invalid button code: %s", parts[1])
		}
		return NewUsage(usagepages.Button, uint16(code)), nil
	default:
		return 0, fmt.Errorf("invalid usage prefix: %s", prefix)
	}
}

type usageMapItem struct {
	from []Usage
	to   []Usage
}

type itemAddress struct {
	reportID uint8
	index    int
}

type mapping struct {
	from []Usage

	action HIDUsageAction

	toItems     []itemAddress
	fromItems   []*itemAddress
	isTriggered bool
}

func (r *Remap) Configure(descriptors map[string][]byte) ([]byte, error) {
	for _, desc := range descriptors {
		r.desc = desc
	}
	desc, err := hiddesc.NewDescriptorDecoder(bytes.NewBuffer(r.desc)).Decode()
	if err != nil {
		return nil, err
	}
	r.descParsed = *desc
	r.usageSets = hidparse.GetUsageSets(*desc, hidparse.Or(
		hidparse.UsagePageFilter(usagepages.KeyboardKeypad),
		hidparse.UsagePageFilter(usagepages.Button),
	))
	mappings := make([]mapping, 0, len(r.cfg))
	for from, to := range r.cfg {
		fromUsages, _, err := ParseUsageCombo(from)
		if err != nil {
			return nil, err
		}
		toUsages, setVar, err := ParseUsageCombo(to)
		if err != nil {
			return nil, err
		}
		toItems := make([]itemAddress, 0, len(toUsages))
		for _, usage := range toUsages {
			reportID, index, ok := r.getReportIDForUsage(usage.Page(), usage.ID())
			if !ok {
				return nil, fmt.Errorf("unable to find report for usage: %d", usage)
			}
			toItems = append(toItems, itemAddress{
				reportID: reportID,
				index:    index,
			})
		}
		mappings = append(mappings, mapping{
			from:      fromUsages,
			to:        toUsages,
			toItems:   toItems,
			setVar:    setVar,
			fromItems: make([]*itemAddress, len(fromUsages)),
		})
	}
	r.mappings = mappings
	r.clearMapping = make(map[int]struct{})

	return r.desc, nil
}

func (r *Remap) OriginSpec() OriginSpec {
	return OriginSpec{
		MinConnections: 1,
		MaxConnections: 1,
	}
}

func (r *Remap) DestinationSpec() DestinationSpec {
	return DestinationSpec{
		MinConnections: 1,
		MaxConnections: 1,
	}
}
