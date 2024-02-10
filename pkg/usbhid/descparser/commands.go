package descparser

import (
	"encoding/binary"
	"errors"
	"fmt"
)

func toUint16(payload []byte) (uint16, error) {
	if len(payload) > 2 {
		return 0, fmt.Errorf("uint16 payload too long")
	}
	if len(payload) == 0 {
		return 0, fmt.Errorf("uint16 payload is missing")
	}
	if len(payload) == 1 {
		return uint16(payload[0]), nil
	}
	return binary.LittleEndian.Uint16(payload), nil
}

func toInt16(payload []byte) (int16, error) {
	if len(payload) > 2 {
		return 0, fmt.Errorf("int16 payload too long")
	}
	if len(payload) == 0 {
		return 0, fmt.Errorf("int16 payload is missing")
	}
	if len(payload) == 1 {
		return int16(payload[0]), nil
	}
	return int16(binary.LittleEndian.Uint16(payload)), nil
}

func newDataItem(state *reportDescriptorState, flags DataFlags) *DataItem {
	return &DataItem{
		Flags:        flags,
		UsagePage:    state.global.usagePage,
		UsageIDs:     state.local.usage,
		UsageMinimum: state.local.usageMinimum,
		UsageMaximum: state.local.usageMaximum,
		ReportCount:  state.global.reportCount,
		ReportSize:   state.global.reportSize,
		ReportID:     state.global.reportID,

		DesignatorIndex:   state.local.designatorIndex,
		DesignatorMinimum: state.local.designatorMinimum,
		DesignatorMaximum: state.local.designatorMaximum,

		LogicalMinimum:  state.global.logicalMinimum,
		LogicalMaximum:  state.global.logicalMaximum,
		PhysicalMinimum: state.global.physicalMinimum,
		PhysicalMaximum: state.global.physicalMaximum,
		UnitExponent:    state.global.unitExponent,
		Unit:            state.global.unit,
	}
}

func cmdInput(state *reportDescriptorState, payload []byte) error {
	if state.collection == nil {
		return errors.New("input: no open collection")
	}
	if len(payload) != 1 {
		return fmt.Errorf("input: payload length is not 1")
	}
	state.collection.Items = append(state.collection.Items, MainItem{
		Type:     MainItemTypeInput,
		DataItem: newDataItem(state, DataFlags(payload[0])),
	})
	state.local = &localState{}
	return nil
}

func cmdOutput(state *reportDescriptorState, payload []byte) error {
	if state.collection == nil {
		return errors.New("output: no open collection")
	}
	if len(payload) != 1 {
		return fmt.Errorf("output: payload length is not 1")
	}
	state.collection.Items = append(state.collection.Items, MainItem{
		Type:     MainItemTypeOutput,
		DataItem: newDataItem(state, DataFlags(payload[0])),
	})
	state.local = &localState{}
	return nil
}

func cmdFeature(state *reportDescriptorState, payload []byte) error {
	if state.collection == nil {
		return errors.New("feature: no open collection")
	}
	if len(payload) != 1 {
		return fmt.Errorf("feature: payload length is not 1")
	}
	state.collection.Items = append(state.collection.Items, MainItem{
		Type:     MainItemTypeFeature,
		DataItem: newDataItem(state, DataFlags(payload[0])),
	})
	state.local = &localState{}
	return nil
}

func cmdCollection(state *reportDescriptorState, payload []byte) error {
	if len(payload) != 1 {
		return fmt.Errorf("collection: payload length is not 1")
	}
	// TODO: validate state
	c := Collection{
		Type:      CollectionType(payload[0]),
		UsagePage: state.global.usagePage,
		UsageID:   state.local.usage[0],
	}
	if state.collection != nil {
		state.collection.Items = append(state.collection.Items, MainItem{
			Type:       MainItemType(state.collection.Type),
			Collection: &c,
		})
		state.collectionStack = append(state.collectionStack, *state.collection)
	}
	state.collection = &c
	state.local = &localState{}
	return nil
}

func cmdEndCollection(state *reportDescriptorState, payload []byte) error {
	if len(payload) != 0 {
		return fmt.Errorf("end collection: payload length is not 0")
	}
	if state.collection == nil {
		return errors.New("end collection: no open collection")
	}
	if len(state.collectionStack) == 0 {
		state.collections = append(state.collections, *state.collection)
		state.collection = nil
	} else {
		c := state.collectionStack[len(state.collectionStack)-1]
		state.collectionStack = state.collectionStack[:len(state.collectionStack)-1]
		state.collection = &c
	}

	state.local = &localState{}
	return nil
}

func cmdUsagePage(state *reportDescriptorState, payload []byte) error {
	val, err := toUint16(payload)
	if err != nil {
		return fmt.Errorf("usage page: %w", err)
	}
	state.global.usagePage = val

	return nil
}

func cmdLogicalMinimum(state *reportDescriptorState, payload []byte) error {
	val, err := toInt16(payload)
	if err != nil {
		return fmt.Errorf("logical minimum: %w", err)
	}
	state.global.logicalMinimum = val
	return nil
}

func cmdLogicalMaximum(state *reportDescriptorState, payload []byte) error {
	val, err := toInt16(payload)
	if err != nil {
		return fmt.Errorf("logical maximum: %w", err)
	}
	state.global.logicalMaximum = val
	return nil
}

func cmdPhysicalMinimum(state *reportDescriptorState, payload []byte) error {
	val, err := toInt16(payload)
	if err != nil {
		return fmt.Errorf("physical minimum: %w", err)
	}
	state.global.physicalMinimum = val
	return nil
}

func cmdPhysicalMaximum(state *reportDescriptorState, payload []byte) error {
	val, err := toInt16(payload)
	if err != nil {
		return fmt.Errorf("physical maximum: %w", err)
	}
	state.global.physicalMaximum = val
	return nil
}

func cmdUnitExponent(state *reportDescriptorState, payload []byte) error {
	if len(payload) != 1 {
		return fmt.Errorf("unit exponent: payload length is not 1")
	}
	state.global.unitExponent = payload[0]
	return nil
}

func cmdUnit(state *reportDescriptorState, payload []byte) error {
	if len(payload) != 1 {
		return fmt.Errorf("unit: payload length is not 1")
	}
	state.global.unit = payload[0]
	return nil
}

func cmdReportSize(state *reportDescriptorState, payload []byte) error {
	if len(payload) != 1 {
		return fmt.Errorf("report size: payload length is not 1")
	}
	state.global.reportSize = payload[0]
	return nil
}

func cmdReportID(state *reportDescriptorState, payload []byte) error {
	if len(payload) != 1 {
		return fmt.Errorf("report id: payload length is not 1")
	}
	state.global.reportID = payload[0]
	return nil
}

func cmdReportCount(state *reportDescriptorState, payload []byte) error {
	if len(payload) != 1 {
		return fmt.Errorf("report count: payload length is not 1")
	}
	state.global.reportCount = payload[0]
	return nil
}

func cmdPush(state *reportDescriptorState, payload []byte) error {
	state.globalStack = append(state.globalStack, *state.global)
	return nil
}

func cmdPop(state *reportDescriptorState, payload []byte) error {
	if len(state.globalStack) == 0 {
		return errors.New("pop: stack is empty")
	}
	*state.global = state.globalStack[len(state.globalStack)-1]
	state.globalStack = state.globalStack[:len(state.globalStack)-1]
	return nil
}

func cmdUsage(state *reportDescriptorState, payload []byte) error {
	val, err := toUint16(payload)
	if err != nil {
		return fmt.Errorf("usage: %w", err)
	}
	state.local.usage = append(state.local.usage, val)
	return nil
}

func cmdDelimiter(state *reportDescriptorState, payload []byte) error {
	return errors.New("not implemented")
}

func cmdUsageMinimum(state *reportDescriptorState, payload []byte) error {
	val, err := toUint16(payload)
	if err != nil {
		return fmt.Errorf("usage minimum: %w", err)
	}
	state.local.usageMinimum = val
	return nil
}

func cmdUsageMaximum(state *reportDescriptorState, payload []byte) error {
	val, err := toUint16(payload)
	if err != nil {
		return fmt.Errorf("usage maximum: %w", err)
	}
	state.local.usageMaximum = val
	return nil
}

func cmdDesignatorIndex(state *reportDescriptorState, payload []byte) error {
	if len(payload) != 1 {
		return fmt.Errorf("designator index: payload length is not 1")
	}
	state.local.designatorIndex = payload[0]
	return nil
}

func cmdDesignatorMinimum(state *reportDescriptorState, payload []byte) error {
	if len(payload) != 1 {
		return fmt.Errorf("designator minimum: payload length is not 1")
	}
	state.local.designatorMinimum = payload[0]
	return nil
}

func cmdDesignatorMaximum(state *reportDescriptorState, payload []byte) error {
	if len(payload) != 1 {
		return fmt.Errorf("designator maximum: payload length is not 1")
	}
	state.local.designatorMaximum = payload[0]
	return nil
}

func cmdStringIndex(state *reportDescriptorState, payload []byte) error {
	if len(payload) != 1 {
		return fmt.Errorf("string index: payload length is not 1")
	}
	state.local.stringIndex = payload[0]
	return nil
}

func cmdStringMinimum(state *reportDescriptorState, payload []byte) error {
	if len(payload) != 1 {
		return fmt.Errorf("string minimum: payload length is not 1")
	}
	state.local.stringMinimum = payload[0]
	return nil
}

func cmdStringMaximum(state *reportDescriptorState, payload []byte) error {
	if len(payload) != 1 {
		return fmt.Errorf("string maximum: payload length is not 1")
	}
	state.local.stringMaximum = payload[0]
	return nil
}
