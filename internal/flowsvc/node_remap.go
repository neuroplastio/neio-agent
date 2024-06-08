package flowsvc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/neuroplastio/neuroplastio/internal/hidparse"
	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hiddesc"
	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hidusage/usagepages"
	"go.uber.org/atomic"
)

func NewRemapNode(data json.RawMessage, provider *NodeProvider) (Node, error) {
	remap := &Remap{
		activatedUsages: make(map[hidparse.Usage]int),
	}
	err := parseConfig(data, remap, provider.ActionRegistry)
	if err != nil {
		return nil, err
	}
	return remap, nil
}

type Remap struct {
	desc       []byte
	descParsed hiddesc.ReportDescriptor
	usageSets  map[uint8]map[int]hidparse.UsageSet
	usageItems map[hidparse.Usage]itemAddress
	actionMappings []actionMapping

	upstreamReports  map[uint8]hidparse.Report
	inflightReports  map[uint8]hidparse.Report
	downstreamReports  map[uint8]hidparse.Report
	processingReport *atomic.Bool
	activatedUsages map[hidparse.Usage]int
}

type actionMapping struct {
	trigger []hidparse.Usage
	action  HIDUsageAction

	triggeredBy []*itemAddress
	isTriggered bool
	clear func()
}

type itemAddress struct {
	reportID uint8
	index    int
}

func parseConfig(data json.RawMessage, remap *Remap, registry *ActionRegistry) error {
	stringMap := make(map[string]json.RawMessage) // map[trigger]action
	err := json.Unmarshal(data, &stringMap)
	if err != nil {
		return err
	}

	mappings := make([]actionMapping, 0, len(stringMap))
	for trigger, actionJSON := range stringMap {
		usageTrigger, err := ParseUsageCombo(trigger)
		if err != nil {
			return err
		}
		action, err := registry.NewFromJSON(actionJSON)
		if err != nil {
			return fmt.Errorf("failed to create action from JSON: %w", err)
		}
		mappings = append(mappings, actionMapping{
			trigger: usageTrigger,
			action:  action,

			triggeredBy: make([]*itemAddress, len(usageTrigger)),
		})
	}
	remap.actionMappings = mappings
	return nil
}

func (r *Remap) Start(ctx context.Context, up FlowStream, down FlowStream) error {
	in := up.Subscribe(ctx)
	r.upstreamReports = make(map[uint8]hidparse.Report)
	r.inflightReports = make(map[uint8]hidparse.Report)
	r.downstreamReports = make(map[uint8]hidparse.Report)
	r.processingReport = atomic.NewBool(false)
	sendCh := make(chan hidparse.Report)
	go func() {
		for {
			select {
			case report := <-sendCh:
				down.Broadcast(ctx, FlowEvent{HIDReport: report})
			case <-ctx.Done():
				return
			}
		}
	}()
	asyncActivateCh := make(chan activation)
	for {
		select {
		case event := <-in:
			report := event.Message.HIDReport
			r.inflightReports[report.ID] = report
			r.upstreamReports[report.ID], _ = r.stripRelativeValues(report)
			r.processingReport.Store(true)
			for _, report := range r.inflightReports {
				r.triggerMappings(ctx, report, asyncActivateCh)
			}
			r.processingReport.Store(false)
			r.applyCompareAndSend(sendCh)
		case a := <-asyncActivateCh:
			r.applyActivation(a.Usages, a.Delta)
			r.applyCompareAndSend(sendCh)
		case <-ctx.Done():
			return nil
		}
	}
}

type activation struct {
	Usages []hidparse.Usage
	Delta int
}

func (r *Remap) applyActivation(usages []hidparse.Usage, delta int) {
	for _, usage := range usages {
		changed := (delta > 0 && r.activatedUsages[usage] == 0) || (delta < 0 && r.activatedUsages[usage] == 1)
		r.activatedUsages[usage] += delta
		if changed {
			addr, ok := r.usageItems[usage]
			if !ok {
				continue
			}
			if _, ok := r.inflightReports[addr.reportID]; !ok {
				r.inflightReports[addr.reportID] = r.upstreamReports[addr.reportID].Clone()
			}
		}
	}
}

func (r *Remap) triggerMappings(ctx context.Context, report hidparse.Report, asyncActivateCh chan activation) {
	am := r.actionMappings
	for mappingIdx, mapping := range r.actionMappings {
		for usageIdx, usage := range mapping.trigger {
			found := false
			for fieldIdx, field := range report.Fields {
				usageSet, ok := r.usageSets[report.ID][fieldIdx]
				if !ok {
					continue
				}
				if am[mappingIdx].triggeredBy[usageIdx] != nil && am[mappingIdx].triggeredBy[usageIdx].reportID != report.ID {
					continue
				}
				if usageSet.HasUsage(field, usage) {
					am[mappingIdx].triggeredBy[usageIdx] = &itemAddress{
						reportID: report.ID,
						index:    fieldIdx,
					}
					found = true
					break
				}
			}
			if !found && am[mappingIdx].triggeredBy[usageIdx] != nil && am[mappingIdx].triggeredBy[usageIdx].reportID == report.ID {
				am[mappingIdx].triggeredBy[usageIdx] = nil
			}
		}
		isTriggered := true
		for _, item := range am[mappingIdx].triggeredBy {
			if item == nil {
				isTriggered = false
				break
			}
		}
		if isTriggered && !mapping.isTriggered {
			am[mappingIdx].isTriggered = true
			am[mappingIdx].clear = mapping.action.Activate(ctx, func(usages []hidparse.Usage) func() {
				if r.processingReport.Load() {
					r.applyActivation(usages, 1)
				} else {
					asyncActivateCh <- activation{
						Usages: usages,
						Delta: 1,
					}
				}
				return func() {
					if r.processingReport.Load() {
						r.applyActivation(usages, -1)
					} else {
						asyncActivateCh <- activation{
							Usages: usages,
							Delta: -1,
						}
					}
				}
			})
		}
		if !isTriggered && mapping.isTriggered && mapping.clear != nil {
			am[mappingIdx].isTriggered = false
			am[mappingIdx].clear()
			am[mappingIdx].clear = nil
		}

		if isTriggered {
			// unset trigger usages
			for i, item := range mapping.triggeredBy {
				r.usageSets[item.reportID][item.index].ClearUsage(r.inflightReports[item.reportID].Fields[item.index], mapping.trigger[i])
			}
		}
	}
}

func (r *Remap) applyCompareAndSend(sendCh chan <- hidparse.Report) {
	for usage, count := range r.activatedUsages {
		if count == 0 {
			continue
		}
		addr, ok := r.usageItems[usage]
		if !ok {
			continue
		}
		report, ok := r.inflightReports[addr.reportID]
		if !ok {
			continue
		}
		r.usageSets[addr.reportID][addr.index].SetUsage(report.Fields[addr.index], usage)
	}
	for id, report := range r.inflightReports {
		if r.hasRelativeValues(report) {
			r.downstreamReports[report.ID] = report
			sendCh <- report
			continue
		}
		lastResult, ok := r.downstreamReports[id]
		if ok && lastResult.Equal(report) {
			continue
		}
		r.downstreamReports[report.ID] = report
		sendCh <- report
	}
	clear(r.inflightReports)
}

func (r *Remap) hasRelativeValues(report hidparse.Report) bool {
	desc, ok := r.descParsed.GetInputReport(report.ID)
	if !ok {
		return false
	}
	for i, item := range desc.Items {
		if item.Flags.IsRelative() {
			if !report.Fields[i].IsEmpty() {
				return true
			}
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
	r.usageItems = make(map[hidparse.Usage]itemAddress)

	for _, mapping := range r.actionMappings {
		usages := mapping.action.Usages()
		for _, usage := range usages {
			if _, ok := r.usageItems[usage]; ok {
				continue
			}
			found := false
			for reportID, reportUsageSets := range r.usageSets {
				for itemIdx, usageSet := range reportUsageSets {
					if usageSet.Contains(usage) {
						r.usageItems[usage] = itemAddress{
							reportID: reportID,
							index:    itemIdx,
						}
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}
	}

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
