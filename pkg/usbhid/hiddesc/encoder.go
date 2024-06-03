package hiddesc

import (
	"encoding/binary"
	"io"
)

type Encoder struct {
	desc   *ReportDescriptor
	w      io.Writer
	global *globalState
	local  *localState
}

func NewDescriptorEncoder(w io.Writer, desc *ReportDescriptor) *Encoder {
	return &Encoder{
		desc:   desc,
		w:      w,
		global: &globalState{},
		local:  &localState{},
	}
}

func (e *Encoder) Encode() error {
	for _, collection := range e.desc.Collections {
		if err := e.encodeCollection(collection); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeCollection(collection Collection) error {
	if err := e.encodeUsagePage(collection.UsagePage); err != nil {
		return err
	}
	if err := e.encodeUsageID(collection.UsageID); err != nil {
		return err
	}
	if err := e.encodeTag8(TagCollection, uint8(collection.Type)); err != nil {
		return err
	}
	e.local = &localState{}
	for _, item := range collection.Items {
		if err := e.encodeMainItem(item); err != nil {
			return err
		}
	}
	if err := e.encodeTag(TagEndCollection); err != nil {
		return err
	}
	e.local = &localState{}
	return nil
}

func (e *Encoder) encodeMainItem(item MainItem) error {
	if item.Collection != nil {
		return e.encodeCollection(*item.Collection)
	}
	if item.DataItem != nil {
		if err := e.encodeUsagePage(item.DataItem.UsagePage); err != nil {
			return err
		}
			if err := e.encodeUsages(item.DataItem.UsageIDs); err != nil {
				return err
			}
		if item.DataItem.UsageMinimum != e.local.usageMinimum || item.DataItem.UsageMaximum != e.local.usageMaximum{
			if err := e.encodeTag16(TagUsageMinimum, item.DataItem.UsageMinimum); err != nil {
				return err
			}
			if err := e.encodeTag16(TagUsageMaximum, item.DataItem.UsageMaximum); err != nil {
				return err
			}
			e.local.usageMinimum = item.DataItem.UsageMinimum
			e.local.usageMaximum = item.DataItem.UsageMaximum
		}
		if item.DataItem.DesignatorIndex != e.local.designatorIndex {
			if err := e.encodeTag8(TagDesignatorIndex, item.DataItem.DesignatorIndex); err != nil {
				return err
			}
			e.local.designatorIndex = item.DataItem.DesignatorIndex
		}
		if item.DataItem.DesignatorMinimum != e.local.designatorMinimum || item.DataItem.DesignatorMaximum != e.local.designatorMaximum {
			if err := e.encodeTag8(TagDesignatorMinimum, item.DataItem.DesignatorMinimum); err != nil {
				return err
			}
			if err := e.encodeTag8(TagDesignatorMaximum, item.DataItem.DesignatorMaximum); err != nil {
				return err
			}
			e.local.designatorMinimum = item.DataItem.DesignatorMinimum
			e.local.designatorMaximum = item.DataItem.DesignatorMaximum
		}

		if item.DataItem.LogicalMinimum != e.global.logicalMinimum || item.DataItem.LogicalMaximum != e.global.logicalMaximum {
			if err := e.encodeTagi32(TagLogicalMinimum, item.DataItem.LogicalMinimum); err != nil {
				return err
			}
			if err := e.encodeTagi32(TagLogicalMaximum, item.DataItem.LogicalMaximum); err != nil {
				return err
			}
			e.global.logicalMinimum = item.DataItem.LogicalMinimum
			e.global.logicalMaximum = item.DataItem.LogicalMaximum
		}
		if item.DataItem.PhysicalMinimum != e.global.physicalMinimum || item.DataItem.PhysicalMaximum != e.global.physicalMaximum{
			if err := e.encodeTagi32(TagPhysicalMinimum, item.DataItem.PhysicalMinimum); err != nil {
				return err
			}
			if err := e.encodeTagi32(TagPhysicalMaximum, item.DataItem.PhysicalMaximum); err != nil {
				return err
			}
			e.global.physicalMinimum = item.DataItem.PhysicalMinimum
			e.global.physicalMaximum = item.DataItem.PhysicalMaximum
		}
		if item.DataItem.UnitExponent != e.global.unitExponent {
			if err := e.encodeTag32(TagUnitExponent, item.DataItem.UnitExponent); err != nil {
				return err
			}
			e.global.unitExponent = item.DataItem.UnitExponent
		}
		if item.DataItem.Unit != e.global.unit {
			if err := e.encodeTag32(TagUnit, item.DataItem.Unit); err != nil {
				return err
			}
			e.global.unit = item.DataItem.Unit
		}
		if item.DataItem.ReportID != e.global.reportID {
			if err := e.encodeTag8(TagReportID, item.DataItem.ReportID); err != nil {
				return err
			}
			e.global.reportID = item.DataItem.ReportID
		}
		if item.DataItem.ReportCount != e.global.reportCount {
			if err := e.encodeTag32(TagReportCount, item.DataItem.ReportCount); err != nil {
				return err
			}
			e.global.reportCount = item.DataItem.ReportCount
		}
		if item.DataItem.ReportSize != e.global.reportSize {
			if err := e.encodeTag32(TagReportSize, item.DataItem.ReportSize); err != nil {
				return err
			}
			e.global.reportSize = item.DataItem.ReportSize
		}
		switch item.Type {
		case MainItemTypeInput:
			if err := e.encodeTag32(TagInput, uint32(item.DataItem.Flags)); err != nil {
				return err
			}
		case MainItemTypeOutput:
			if err := e.encodeTag32(TagOutput, uint32(item.DataItem.Flags)); err != nil {
				return err
			}
		case MainItemTypeFeature:
			if err := e.encodeTag32(TagFeature, uint32(item.DataItem.Flags)); err != nil {
				return err
			}
		}
		e.local = &localState{}
	}
	return nil
}

func (e *Encoder) encodeUsagePage(usagePage uint16) error {
	if usagePage == e.global.usagePage {
		return nil
	}
	if err := e.encodeTag16(TagUsagePage, usagePage); err != nil {
		return err
	}
	e.global.usagePage = usagePage
	return nil
}

func (e *Encoder) encodeUsages(usageIDs []uint16) error {
	for _, usageID := range usageIDs {
		if err := e.encodeUsageID(usageID); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeUsageID(usageID uint16) error {
	if usageID == 0 {
		return nil
	}
	if len(e.local.usage) > 0 && e.local.usage[len(e.local.usage)-1] == usageID {
		return nil
	}
	if err := e.encodeTag16(TagUsage, usageID); err != nil {
		return err
	}
	e.local.usage = []uint16{usageID}
	return nil
}

func (e *Encoder) encodeTag(tag Tag) error {
	_, err := e.w.Write([]byte{byte(tag.WithItemSize(TagItemSize0))})
	return err
}

func (e *Encoder) encodeTag8(tag Tag, value uint8) error {
	_, err := e.w.Write([]byte{byte(tag.WithItemSize(TagItemSize8)), value})
	return err
}

func (e *Encoder) encodeTag16(tag Tag, value uint16) error {
	// check if value fits into one byte
	if value < 0x100 {
		return e.encodeTag8(tag, uint8(value))
	}
	_, err := e.w.Write([]byte{byte(tag.WithItemSize(TagItemSize16)), byte(value), byte(value >> 8)})
	return err
}

func (e *Encoder) encodeTagi32(tag Tag, value int32) error {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(value))
	size := TagItemSize32
	pad := byte(0)
	if value < 0 {
		pad = 0xff
	}
	switch {
	case data[1] == pad && data[2] ==pad  && data[3] == pad:
		size = TagItemSize8
		data = data[:1]
	case data[2] == pad && data[3] == pad:
		size = TagItemSize16
		data = data[:2]
	}
	data = append([]byte{byte(tag.WithItemSize(size))}, data...)
	_, err := e.w.Write(data)
	return err
}

func (e *Encoder) encodeTag32(tag Tag, value uint32) error {
	// check if value fits into one byte
	if value < 0x100 {
		return e.encodeTag8(tag, uint8(value))
	}
	// check if value fits into two bytes
	if value < 0x10000 {
		return e.encodeTag16(tag, uint16(value))
	}
	_, err := e.w.Write([]byte{byte(tag.WithItemSize(TagItemSize32)), byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24)})
	return err
}
