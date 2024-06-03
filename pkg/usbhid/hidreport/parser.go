package hidreport

import (
	"encoding/binary"
	"fmt"

	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hiddesc"
)

// Parser handles data of a single ReportDescriptor.
// Signle ReportDescriptor may contain multiple reports.
type Parser struct {
	desc hiddesc.ReportDescriptor
}

func NewParser(desc hiddesc.ReportDescriptor) *Parser {
	return &Parser{
		desc: desc,
	}
}

func (p Parser) GetIndexes() map[uint8]ReportFieldIndex {
	reports := p.desc.GetInputReports()
	indexes := make(map[uint8]ReportFieldIndex, len(reports))
	for _, report := range reports {
		indexes[report.ID] = ReportFieldIndex{
			ReportID:  report.ID,
			DataItems: report.Items,
		}
	}
	return indexes
}

// ReportFieldIndex keeps the information about DataItems' offsets in a report.
// A data item may contain multiple fields and may have multiple usages.
// ReportFieldIndex indexes data based on the Usage (combination of UsagePage and UsageID).
type ReportFieldIndex struct {
	ReportID  uint8
	DataItems []hiddesc.DataItem

	selectorOffsets map[uint8][]uint8 // map[usagePage][]offset
}

type IndexedReport struct {
	index ReportFieldIndex
	data []byte
}

type ReportField struct {
	Selector     *ReportFieldSelector
	DynamicFlags *ReportFieldDynamicFlags
	DynamicValue *ReportFieldDynamicValue
}

func (r ReportField) String() string {
	if r.Selector != nil {
		return r.Selector.String()
	}
	if r.DynamicFlags != nil {
		return r.DynamicFlags.String()
	}
	if r.DynamicValue != nil {
		return r.DynamicValue.String()
	}
	return ""
}

type ReportFieldSelector struct {
	UsagePage uint16
	UsageIDs  []uint16
}

func (r ReportFieldSelector) String() string {
	return fmt.Sprintf("s%d: %v", r.UsagePage, r.UsageIDs)
}

type ReportFieldDynamicFlags struct {
	UsagePage uint16
	UsageMin  uint16
	UsageMax  uint16
	Data      []byte
}

func (r ReportFieldDynamicFlags) String() string {
	str := fmt.Sprintf("f%d: ", r.UsagePage)
	for _, b := range r.Data {
		str += fmt.Sprintf("%08b ", b)
	}
	return str
}

func (r ReportFieldDynamicFlags) IsSet(usageID uint16) bool {
	if usageID < r.UsageMin || usageID > r.UsageMax {
		return false
	}
	bitOffset := int(usageID - r.UsageMin)
	return r.Data[bitOffset/8]&(1<<(bitOffset%8)) != 0
}

type ReportFieldDynamicValue struct {
	UsagePage uint16
	Signed    bool
	Values    map[uint16]int
}

func (r ReportFieldDynamicValue) String() string {
	str := fmt.Sprintf("v: ")
	for usageID, value := range r.Values {
		str += fmt.Sprintf("%d:%d ", usageID, value)
	}
	return str
}

func (r ReportFieldIndex) GetFields(report []byte) []ReportField {
	bitOffset := 0
	fields := make([]ReportField, 0, len(r.DataItems))
	for _, item := range r.DataItems {
		byteOffset := bitOffset / 8
		switch {
		case item.Flags.IsArray() && (item.ReportSize == 8 || item.ReportSize == 16):
			// Selector (array of usage IDs)
			selector := &ReportFieldSelector{
				UsagePage: item.UsagePage,
				UsageIDs:  make([]uint16, item.ReportCount),
			}
			for i := 0; i < int(item.ReportCount); i++ {
				switch item.ReportSize {
				case 8:
					selector.UsageIDs[i] = uint16(report[byteOffset+i])
				case 16:
					selector.UsageIDs[i] = binary.LittleEndian.Uint16(report[byteOffset+i*2 : byteOffset+i*2+2])
				}
			}
			fields = append(fields, ReportField{
				Selector: selector,
			})
		case item.Flags.IsVariable() && item.ReportSize == 1:
			// ReportDatumDynamicFlags
			size := int(uint(item.ReportSize)*uint(item.ReportCount)+7) / 8
			dynamicFlags := &ReportFieldDynamicFlags{
				UsagePage: item.UsagePage,
				UsageMin:  item.UsageMinimum,
				UsageMax:  item.UsageMaximum,
				Data:      report[byteOffset : byteOffset+size],
			}
			fields = append(fields, ReportField{
				DynamicFlags: dynamicFlags,
			})
		case item.Flags.IsVariable() && (item.ReportSize == 8 || item.ReportSize == 16 || item.ReportSize == 32):
			// Dynamic Value
			dynamicValue := &ReportFieldDynamicValue{
				UsagePage: item.UsagePage,
				Signed:    item.LogicalMinimum < 0,
				Values:    make(map[uint16]int, item.ReportCount),
			}
			for i := 0; i < int(item.ReportCount); i++ {
				if i > len(item.UsageIDs)-1 {
					// No more usages for this item, likely incorrect descriptor
					break
				}
				usageID := item.UsageIDs[i]
				switch item.ReportSize {
				case 8:
					if dynamicValue.Signed {
						dynamicValue.Values[usageID] = int(int8(report[byteOffset+i]))
					} else {
						dynamicValue.Values[usageID] = int(report[byteOffset+i])
					}
				case 16:
					if dynamicValue.Signed {
						dynamicValue.Values[usageID] = int(int16(binary.LittleEndian.Uint16(report[byteOffset+i*2 : byteOffset+i*2+2])))
					} else {
						dynamicValue.Values[usageID] = int(binary.LittleEndian.Uint16(report[byteOffset+i*2 : byteOffset+i*2+2]))
					}
				case 32:
					if dynamicValue.Signed {
						dynamicValue.Values[usageID] = int(int32(binary.LittleEndian.Uint32(report[byteOffset+i*4 : byteOffset+i*4+4])))
					} else {
						dynamicValue.Values[usageID] = int(binary.LittleEndian.Uint32(report[byteOffset+i*4 : byteOffset+i*4+4]))
					}
				}
				fields = append(fields, ReportField{
					DynamicValue: dynamicValue,
				})
			}
		}
		bitOffset += int(item.ReportSize) * int(item.ReportCount)
	}
	return fields
}
