package hidparse

import (
	"fmt"

	"github.com/neuroplastio/neuroplastio/pkg/bits"
	"github.com/neuroplastio/neuroplastio/pkg/usbhid/hiddesc"
)

type Usage uint32

func (u Usage) Page() uint16 {
	return uint16(u >> 16)
}

func (u Usage) ID() uint16 {
	return uint16(u)
}

func (u Usage) String() string {
	return fmt.Sprintf("0x%02x/0x%02x", u.Page(), u.ID())
}

func NewUsage(page, id uint16) Usage {
	return Usage(uint32(page)<<16 | uint32(id))
}

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
	Usages(bits bits.Bits) []Usage
	UsagePage() uint16
	UsageMinimum() uint16
	UsageMaximum() uint16
	HasUsage(bits bits.Bits, usage Usage) bool
	ReplaceUsage(bits bits.Bits, from, to Usage) bool
	SetUsage(bits bits.Bits, usage Usage) bool
	ClearUsage(bits bits.Bits, usage Usage) bool
	Contains(usage Usage) bool
}

type UsageFlags struct {
	usagePage uint16
	minimum   uint16
	maximum   uint16
}

func NewUsageFlags(usagePage uint16, minimum, maximum uint16) UsageFlags {
	return UsageFlags{
		usagePage: usagePage,
		minimum:   minimum,
		maximum:   maximum,
	}
}

func (u UsageFlags) UsagePage() uint16 {
	return u.usagePage
}

func (u UsageFlags) UsageMinimum() uint16 {
	return u.minimum
}

func (u UsageFlags) UsageMaximum() uint16 {
	return u.maximum
}

func (u UsageFlags) Usages(bits bits.Bits) []Usage {
	var usages []Usage
	bits.Each(func(i int, set bool) bool {
		if set {
			usages = append(usages, NewUsage(u.usagePage, uint16(i)+u.minimum))
		}
		return true
	})
	return usages
}

func (u UsageFlags) HasUsage(bits bits.Bits, usage Usage) bool {
	if !u.Contains(usage) {
		return false
	}
	return bits.IsSet(int(usage.ID() - u.minimum))
}

func (u UsageFlags) ReplaceUsage(bits bits.Bits, from, to Usage) bool {
	if !u.Contains(from) || !u.Contains(to) {
		return false
	}
	wasSet := bits.Clear(int(from.ID() - u.minimum))
	if wasSet {
		bits.Set(int(to.ID() - u.minimum))
	}
	return wasSet
}

func (u UsageFlags) SetUsage(bits bits.Bits, usage Usage) bool {
	if !u.Contains(usage) {
		return false
	}
	return bits.Set(int(usage.ID() - u.minimum))
}

func (u UsageFlags) ClearUsage(bits bits.Bits, usage Usage) bool {
	if !u.Contains(usage) {
		return false
	}
	return bits.Clear(int(usage.ID() - u.minimum))
}

func (u UsageFlags) Contains(usage Usage) bool {
	if u.usagePage != usage.Page() {
		return false
	}
	return usage.ID() >= u.minimum && usage.ID() <= u.maximum
}

type UsageSelector struct {
	size      int
	usagePage uint16
	minimum   uint16
	maximum   uint16
}

func NewUsageSelector(size int, usagePage, minimum, maximum uint16) UsageSelector {
	return UsageSelector{
		size:      size,
		usagePage: usagePage,
		minimum:   minimum,
		maximum:   maximum,
	}
}

func (u UsageSelector) UsagePage() uint16 {
	return u.usagePage
}

func (u UsageSelector) UsageMinimum() uint16 {
	return u.minimum
}

func (u UsageSelector) UsageMaximum() uint16 {
	return u.maximum
}

func (u UsageSelector) Usages(bits bits.Bits) []Usage {
	var usages []Usage
	switch u.size {
	case 8:
		bits.EachUint8(func(i int, val uint8) bool {
			if val == 0 {
				return true
			}
			usages = append(usages, NewUsage(u.usagePage, uint16(val)))
			return true
		})
	case 16:
		bits.EachUint16(func(i int, val uint16) bool {
			if val == 0 {
				return true
			}
			usages = append(usages, NewUsage(u.usagePage, val))
			return true
		})
	}
	return usages
}

func (u UsageSelector) HasUsage(bits bits.Bits, usage Usage) bool {
	if !u.Contains(usage) {
		return false
	}
	has := false
	switch u.size {
	case 8:
		bits.EachUint8(func(i int, val uint8) bool {
			if uint16(val) == usage.ID() {
				has = true
				return false
			}
			return true
		})
	case 16:
		bits.EachUint16(func(i int, val uint16) bool {
			if val == usage.ID() {
				has = true
				return false
			}
			return true
		})
	}
	return has
}

func (u UsageSelector) ReplaceUsage(bits bits.Bits, from, to Usage) bool {
	if !u.Contains(from) || !u.Contains(to) {
		return false
	}
	wasSet := false
	switch u.size {
	case 8:
		bits.EachUint8(func(i int, val uint8) bool {
			if uint16(val) == from.ID() {
				wasSet = true
				bits.SetUint8(i, uint8(to.ID()))
				return false
			}
			return true
		})
	case 16:
		bits.EachUint16(func(i int, val uint16) bool {
			if val == from.ID() {
				wasSet = true
				bits.SetUint16(i, to.ID())
				return false
			}
			return true
		})
	}
	return wasSet
}

func (u UsageSelector) SetUsage(bits bits.Bits, usage Usage) bool {
	if !u.Contains(usage) {
		return false
	}
	switch u.size {
	case 8:
		bits.EachUint8(func(i int, val uint8) bool {
			if val == uint8(usage.ID()) {
				return false
			}
			if val == 0 {
				bits.SetUint8(i, uint8(usage.ID()))
				return false
			}
			return true
		})
	case 16:
		bits.EachUint16(func(i int, val uint16) bool {
			if val == usage.ID() {
				return false
			}
			if val == 0 {
				bits.SetUint16(i, usage.ID())
				return false
			}
			return true
		})
	}
	return true
}

func (u UsageSelector) ClearUsage(bits bits.Bits, usage Usage) bool {
	if !u.Contains(usage) {
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
				bits.SetUint8(i, 0)
				return true
			}
			if uint16(val) == usage.ID() {
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
				bits.SetUint16(i, 0)
				return true
			}
			if val == usage.ID() {
				bits.SetUint16(i, 0)
				cleared = true
			}
			return true
		})
	}
	return true
}

func (u UsageSelector) Contains(usage Usage) bool {
	if u.usagePage != usage.Page() {
		return false
	}
	return usage.ID() >= u.minimum && usage.ID() <= u.maximum
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
				sets[i] = NewUsageSelector(int(item.ReportSize), item.UsagePage, item.UsageMinimum, item.UsageMaximum)
			case item.Flags.IsVariable() && item.ReportSize == 1:
				sets[i] = NewUsageFlags(item.UsagePage, item.UsageMinimum, item.UsageMaximum)
			}
		}
		if len(sets) > 0 {
			usageSets[report.ID] = sets
		}
	}

	return usageSets
}

func NewUsageSets(dataItems []hiddesc.DataItem) map[int]UsageSet {
	sets := make(map[int]UsageSet)
	for i, item := range dataItems {
		if item.UsageMaximum == 0 || item.Flags.IsConstant() {
			// not a usage-set data item
			continue
		}
		switch {
		case item.Flags.IsArray() && (item.ReportSize == 8 || item.ReportSize == 16):
			sets[i] = NewUsageSelector(int(item.ReportSize), item.UsagePage, item.UsageMinimum, item.UsageMaximum)
		case item.Flags.IsVariable() && item.ReportSize == 1:
			sets[i] = NewUsageFlags(item.UsagePage, item.UsageMinimum, item.UsageMaximum)
		}
	}
	return sets
}

func CompareUsages(usageSet UsageSet, t0, t1 bits.Bits) (activated, deactivated []Usage) {
	usagesT0 := usageSet.Usages(t0)
	usagesT1 := usageSet.Usages(t1)
	for _, usage := range usagesT0 {
		if !usageSet.HasUsage(t1, usage) {
			deactivated = append(deactivated, usage)
		}
	}
	for _, usage := range usagesT1 {
		if !usageSet.HasUsage(t0, usage) {
			activated = append(activated, usage)
		}
	}
	return
}
