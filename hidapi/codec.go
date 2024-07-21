package hidapi

import (
	"github.com/neuroplastio/neuroplastio/hidapi/hiddesc"
	"github.com/neuroplastio/neuroplastio/pkg/bits"
)

type ReportDecoder struct {
	dataItems map[uint8][]hiddesc.DataItem
}

func NewInputReportDecoder(desc hiddesc.ReportDescriptor) *ReportDecoder {
	return &ReportDecoder{dataItems: desc.GetInputDataItems()}
}

func NewOutputReportDecoder(desc hiddesc.ReportDescriptor) *ReportDecoder {
	return &ReportDecoder{dataItems: desc.GetOutputDataItems()}
}

func (r *ReportDecoder) Decode(data []byte) (Report, bool) {
	reportID := uint8(0)
	hasReportID := len(r.dataItems) > 1
	if hasReportID {
		reportID = data[0]
		data = data[1:]
	}
	items := r.dataItems[reportID]
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

func EncodeReport(report Report) bits.Bits {
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
	return allBits
}
