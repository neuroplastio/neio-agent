package hidapi

import (
	"fmt"

	"github.com/neuroplastio/neio-agent/pkg/bits"
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
