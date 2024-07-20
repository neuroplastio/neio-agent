package hidparse

import (
	"fmt"

	"github.com/neuroplastio/neuroplastio/pkg/bits"
	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hiddesc"
)

type Report struct {
	ID     uint8
	Fields []bits.Bits
}

func (r Report) Clone() Report {
	fields := make([]bits.Bits, len(r.Fields))
	for i, field := range r.Fields {
		fields[i] = field.Clone()
	}
	return Report{
		ID:     r.ID,
		Fields: fields,
	}
}

func (r Report) String() string {
	return fmt.Sprintf("Report{ID: %d, Fields: %s}", r.ID, r.Fields)
}

func (r Report) FieldsStrings() []string {
	result := make([]string, len(r.Fields))
	for i, field := range r.Fields {
		result[i] = field.String()
	}
	return result
}

func (r Report) Equal(other Report) bool {
	if r.ID != other.ID {
		return false
	}
	if len(r.Fields) != len(other.Fields) {
		return false
	}
	for i, field := range r.Fields {
		if !field.Equal(other.Fields[i]) {
			return false
		}
	}
	return true
}

func ParseInputReport(desc hiddesc.ReportDescriptor, data []byte) (Report, bool) {
	reportID := uint8(0)
	hasReportID := desc.HasReportID()
	if hasReportID {
		reportID = data[0]
		data = data[1:]
	}
	items := desc.GetInputDataItems()[reportID]
	scanner := bits.NewScanner(data)
	report := Report{
		ID:     reportID,
		Fields: make([]bits.Bits, len(items)),
	}
	for i, id := range items {
		bits := scanner.Next(int(id.ReportSize) * int(id.ReportCount))
		if bits.Len() == 0 {
			return Report{}, false
		}
		report.Fields[i] = bits
	}
	return report, true
}

func ParseOutputReport(desc hiddesc.ReportDescriptor, data []byte) (Report, bool) {
	reportID := uint8(0)
	hasReportID := desc.HasReportID()
	if hasReportID {
		reportID = data[0]
		data = data[1:]
	}
	items := desc.GetOutputDataItems()[reportID]
	scanner := bits.NewScanner(data)
	report := Report{
		ID:     reportID,
		Fields: make([]bits.Bits, len(items)),
	}
	for i, id := range items {
		bits := scanner.Next(int(id.ReportSize) * int(id.ReportCount))
		if bits.Len() == 0 {
			return Report{}, false
		}
		report.Fields[i] = bits
	}
	return report, true
}

func EncodeReport(report Report) []byte {
	size := 0
	for _, field := range report.Fields {
		size += field.Len()
	}
	if report.ID != 0 {
		size++
	}
	// TODO: optimize allocations
	allBits := bits.Bits{}
	if report.ID != 0 {
		allBits = bits.New([]byte{report.ID}, 0)
	}
	for _, field := range report.Fields {
		allBits = bits.ConcatBits(allBits, field)
	}
	// TODO: warn when not byte-aligned
	return allBits.Bytes()
}

func GetAbsoluteFields(desc hiddesc.ReportDescriptor) map[uint8][]int {
	result := make(map[uint8][]int)
	reports := desc.GetInputReports()
	for _, report := range reports {
		fields := make([]int, 0, len(report.Items))
		for i, item := range report.Items {
			if !item.Flags.IsRelative() {
				fields = append(fields, i)
			}
		}
		if len(fields) > 0 {
			result[report.ID] = fields
		}
	}
	return nil
}
