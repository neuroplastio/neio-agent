package hidapi

import (
	"github.com/neuroplastio/neio-agent/pkg/bits"
)

type ReportDecoder struct {
	dataItems DataItemSet
}

func NewReportDecoder(dataItems DataItemSet) *ReportDecoder {
	return &ReportDecoder{dataItems: dataItems}
}

func (r *ReportDecoder) Decode(data []byte) (Report, bool) {
	reportID := uint8(0)
	if r.dataItems.HasReportID() {
		reportID = data[0]
		data = data[1:]
	}
	items := r.dataItems.Report(reportID)
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
	// TODO: optimize allocations
	allBits := bits.Bits{}
	if report.ID != 0 {
		allBits = bits.New([]byte{report.ID}, 0)
	}
	for _, field := range report.Fields {
		allBits = bits.ConcatBits(allBits, field)
	}
	return allBits
}
