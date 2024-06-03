package hidparse

import (
	"github.com/neuroplastio/neuroplastio/pkg/bits"
	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hiddesc"
)

type DescriptorQueryier struct {
	reports []hiddesc.Report
}

func NewDescriptorQueryier(reports []hiddesc.Report) *DescriptorQueryier {
	return &DescriptorQueryier{reports: reports}
}

type QueryResult struct {
	ReportID  uint8
	ItemIndex int
	DataItem  hiddesc.DataItem
}

func (d *DescriptorQueryier) FindByUsagePage(page uint16) []QueryResult {
	var results []QueryResult
	for _, report := range d.reports {
		for i, item := range report.Items {
			if item.UsagePage == page {
				results = append(results, QueryResult{ReportID: report.ID, ItemIndex: i, DataItem: item})
			}
		}
	}
	return results
}

type UsageSet interface {
	HasUsage(bits bits.Bits, usageID uint16) bool
	ReplaceUsage(bits bits.Bits, from, to uint16) bool
	SetUsage(bits bits.Bits, usageID uint16) bool
	ClearUsage(bits bits.Bits, usageID uint16) bool
}

type UsageFlags struct {
	minimum uint16
	maximum uint16
}

func NewUsageFlags(minimum, maximum uint16) UsageFlags {
	return UsageFlags{
		minimum: minimum,
		maximum: maximum,
	}
}

func (u UsageFlags) HasUsage(bits bits.Bits, usageID uint16) bool {
	if usageID < u.minimum || usageID > u.maximum {
		return false
	}
	return bits.IsSet(int(usageID - u.minimum))
}

func (u UsageFlags) ReplaceUsage(bits bits.Bits, from, to uint16) bool {
	if from < u.minimum || from > u.maximum || to < u.minimum || to > u.maximum {
		return false
	}
	wasSet := bits.Clear(int(from - u.minimum))
	if wasSet {
		bits.Set(int(to - u.minimum))
	}
	return wasSet
}

func (u UsageFlags) SetUsage(bits bits.Bits, usageID uint16) bool {
	if usageID < u.minimum || usageID > u.maximum {
		return false
	}
	return bits.Set(int(usageID - u.minimum))
}

func (u UsageFlags) ClearUsage(bits bits.Bits, usageID uint16) bool {
	if usageID < u.minimum || usageID > u.maximum {
		return false
	}
	return bits.Clear(int(usageID - u.minimum))
}

type UsageSelector struct {
	size    int
	minimum uint16
	maximum uint16
}

func NewUsageSelector(size int, minimum, maximum uint16) UsageSelector {
	return UsageSelector{
		size:    size,
		minimum: minimum,
		maximum: maximum,
	}
}

func (u UsageSelector) HasUsage(bits bits.Bits, usageID uint16) bool {
	if usageID < u.minimum || usageID > u.maximum {
		return false
	}
	has := false
	switch u.size {
	case 8:
		bits.EachUint8(func(i int, val uint8) bool {
			if uint16(val) == usageID {
				has = true
				return false
			}
			return true
		})
	case 16:
		bits.EachUint16(func(i int, val uint16) bool {
			if val == usageID {
				has = true
				return false
			}
			return true
		})
	}
	return has
}

func (u UsageSelector) ReplaceUsage(bits bits.Bits, from, to uint16) bool {
	if from < u.minimum || from > u.maximum || to < u.minimum || to > u.maximum {
		return false
	}
	wasSet := false
	switch u.size {
	case 8:
		bits.EachUint8(func(i int, val uint8) bool {
			if uint16(val) == from {
				wasSet = true
				bits.SetUint8(i, uint8(to))
				return false
			}
			return true
		})
	case 16:
		bits.EachUint16(func(i int, val uint16) bool {
			if val == from {
				wasSet = true
				bits.SetUint16(i, to)
				return false
			}
			return true
		})
	}
	return wasSet
}

func (u UsageSelector) SetUsage(bits bits.Bits, usageID uint16) bool {
	if usageID < u.minimum || usageID > u.maximum {
		return false
	}
	switch u.size {
	case 8:
		bits.EachUint8(func(i int, val uint8) bool {
			if val == uint8(usageID) {
				return false
			}
			if val == 0 {
				bits.SetUint8(i, uint8(usageID))
				return false
			}
			return true
		})
	case 16:
		bits.EachUint16(func(i int, val uint16) bool {
			if val == usageID {
				return false
			}
			if val == 0 {
				bits.SetUint16(i, usageID)
				return false
			}
			return true
		})
	}
	return true
}

func (u UsageSelector) ClearUsage(bits bits.Bits, usageID uint16) bool {
	if usageID < u.minimum || usageID > u.maximum {
		return false
	}
	switch u.size {
	case 8:
		cleared := false
		bits.EachUint8(func(i int, val uint8) bool {
			if val == 0 {
				return false
			}
			if cleared {
				bits.SetUint8(i-1, val)
				return true
			}
			if uint16(val) == usageID {
				bits.SetUint8(i, 0)
				cleared = true
			}
			return true
		})
	case 16:
		cleared := false
		bits.EachUint16(func(i int, val uint16) bool {
			if val == 0 {
				return false
			}
			if cleared {
				bits.SetUint16(i-1, val)
				return true
			}
			if val == usageID {
				bits.SetUint16(i, 0)
				cleared = true
			}
			return true
		})
	}
	return true
}

type Filter func(hiddesc.DataItem) bool

func And(filters ...Filter) Filter {
	return func(item hiddesc.DataItem) bool {
		for _, filter := range filters {
			if !filter(item) {
				return false
			}
		}
		return true
	}
}

func UsagePageFilter(page uint16) Filter {
	return func(item hiddesc.DataItem) bool {
		return item.UsagePage == page
	}
}

func Or(filters ...Filter) Filter {
	return func(item hiddesc.DataItem) bool {
		for _, filter := range filters {
			if filter(item) {
				return true
			}
		}
		return false
	}
}

func GetUsageSets(desc hiddesc.ReportDescriptor, filter Filter) map[uint8]map[int]UsageSet {
	usageSets := make(map[uint8]map[int]UsageSet)
	reports := desc.GetInputReports()
	for _, report := range reports {
		sets := make(map[int]UsageSet)
		for i, item := range report.Items {
			if filter != nil && !filter(item) {
				continue
			}
			switch {
			case item.Flags.IsArray() && (item.ReportSize == 8 || item.ReportSize == 16):
				sets[i] = NewUsageSelector(int(item.ReportSize), item.UsageMinimum, item.UsageMaximum)
			case item.Flags.IsVariable() && item.ReportSize == 1:
				sets[i] = NewUsageFlags(item.UsageMinimum, item.UsageMaximum)
			}
		}
		if len(sets) > 0 {
			usageSets[report.ID] = sets
		}
	}

	return usageSets
}
