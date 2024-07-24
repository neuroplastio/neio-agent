package hiddesc

type ReportDescriptor struct {
	// Top-level Application Collections
	Collections []Collection
}

func (r ReportDescriptor) MaxReportSize() int {
	maxSize := 0
	for _, collection := range r.Collections {
		maxSize += collection.MaxReportSize()
	}
	return maxSize
}

type CollectionType uint8

const (
	CollectionTypePhysical CollectionType = iota
	CollectionTypeApplication
	CollectionTypeLogical
	CollectionTypeReport
	CollectionTypeNamedArray
	CollectionTypeUsageSwitch
	CollectionTypeUsageModifier
)

// A Collection item identifies a relationship between two or more data (Input,
// Output, or Feature.) For example, a mouse could be described as a collection of
// two to four data (x, y, button 1, button 2).
// All Main items between the Collection item and the End Collection item are
// included in the collection. Collections may contain other nested collections.
// Collection items may be nested, and they are always optional, except for the top-level
// application collection.
type Collection struct {
	Type      CollectionType
	UsagePage uint16
	UsageID   uint16
	// Items contains ordered list of Main Items, including nested collections.
	Items []MainItem
}

func (c Collection) MaxReportSize() int {
	size := 0
	for _, item := range c.Items {
		if item.DataItem != nil {
			size += int(item.DataItem.ReportSize)
		}
		if item.Collection != nil {
			size += item.Collection.MaxReportSize()
		}
	}
	return size
}

type DataFlags uint32

const (
	DataFlagConstant      DataFlags = 1 << iota // 0 = Data is variable, 1 = Data is constant
	DataFlagVariable                            // 0 = Array, 1 = Variable
	DataFlagRelative                            // 0 = Absolute, 1 = Relative
	DataFlagWrap                                // 0 = No wrap, 1 = Wrap
	DataFlagNonLinear                           // 0 = Linear, 1 = Non-linear
	DataFlagNoPreferred                         // 0 = Preferred state, 1 = No preferred
	DataFlagNullState                           // 0 = No null position, 1 = Null state
	DataFlagVolatile                            // 0 = Non-volatile, 1 = Volatile, not applicable to Input
	DataFlagBufferedBytes                       // 0 = Bit field, 1 = Buffered bytes
)

func (d DataFlags) IsConstant() bool {
	return d&DataFlagConstant != 0
}

func (d DataFlags) IsVariable() bool {
	return d&DataFlagVariable != 0
}

func (d DataFlags) IsArray() bool {
	return !d.IsVariable()
}

func (d DataFlags) IsRelative() bool {
	return d&DataFlagRelative != 0
}

func (d DataFlags) IsWrap() bool {
	return d&DataFlagWrap != 0
}

func (d DataFlags) IsNonLinear() bool {
	return d&DataFlagNonLinear != 0
}

func (d DataFlags) IsNoPreferred() bool {
	return d&DataFlagNoPreferred != 0
}

func (d DataFlags) IsNullState() bool {
	return d&DataFlagNullState != 0
}

func (d DataFlags) IsVolatile() bool {
	return d&DataFlagVolatile != 0
}

func (d DataFlags) IsBufferedBytes() bool {
	return d&DataFlagBufferedBytes != 0
}

// MainItemType is not a part of the spec, but an internal abstraction.
// Input, output and feature items carry mostly the same information.
// Collection is also included here.
type MainItemType uint8

const (
	MainItemTypeInput MainItemType = iota
	MainItemTypeOutput
	MainItemTypeFeature
	MainItemTypeCollection
)

// MainItem is a oneOf type.
// Avoiding pointers to avoid extra GC pressure.
type MainItem struct {
	Type       MainItemType
	DataItem   *DataItem
	Collection *Collection
}

// An Input item describes information about the data provided by one or more
// physical controls. An application can use this information to interpret the data
// provided by the device. All data fields defined in a single item share an
// identical data format.
// The number of data fields in an item can be determined by examining the
// Report Size and Report Count values. For example an item with a Report
// Size of 8 bits and a Report Count of 3 has three 8-bit data fields.
type DataItem struct {
	Flags        DataFlags
	UsagePage    uint16
	UsageIDs     []uint16
	UsageMinimum uint16
	UsageMaximum uint16
	ReportCount  uint32
	ReportSize   uint32
	ReportID     uint8

	DesignatorIndex   uint8
	DesignatorMinimum uint8
	DesignatorMaximum uint8

	LogicalMinimum  int32
	LogicalMaximum  int32
	PhysicalMinimum int32
	PhysicalMaximum int32
	UnitExponent    uint32
	Unit            uint32
}

// NewUsage creates a new Usage from a UsagePage and UsageID.
func NewUsage(usagePage, usageID uint16) Usage {
	return Usage(uint32(usagePage)<<16 | uint32(usageID))
}

// Usage is a combination of UsagePage and UsageID.
type Usage uint32

func (u Usage) Page() uint16 {
	return uint16(u >> 16)
}

func (u Usage) UsageID() uint16 {
	return uint16(u)
}

func (r ReportDescriptor) Clone() ReportDescriptor {
	collections := make([]Collection, len(r.Collections))
	for i, c := range r.Collections {
		items := make([]MainItem, len(c.Items))
		for j, item := range c.Items {
			if item.DataItem != nil {
				dataItem := *item.DataItem
				items[j] = MainItem{
					Type:     item.Type,
					DataItem: &dataItem,
				}
			}
			if item.Collection != nil {
				collection := item.Collection.Clone()
				items[j] = MainItem{
					Type:       item.Type,
					Collection: &collection,
				}
			}
		}
		collections[i] = Collection{
			Type:  c.Type,
			Items: items,
		}
	}
	return ReportDescriptor{
		Collections: collections,
	}
}

func (c Collection) Clone() Collection {
	items := make([]MainItem, len(c.Items))
	for i, item := range c.Items {
		if item.DataItem != nil {
			dataItem := *item.DataItem
			items[i] = MainItem{
				Type:     item.Type,
				DataItem: &dataItem,
			}
		}
		if item.Collection != nil {
			collection := item.Collection.Clone()
			items[i] = MainItem{
				Type:       item.Type,
				Collection: &collection,
			}
		}
	}
	return Collection{
		Type:  c.Type,
		Items: items,
	}

}

func (c Collection) Walk(fn func(item MainItem) bool) {
	for _, item := range c.Items {
		if !fn(item) {
			return
		}
		if item.Collection != nil {
			item.Collection.Walk(fn)
		}
	}
}

func (r ReportDescriptor) Walk(fn func(item MainItem) bool) {
	for _, c := range r.Collections {
		c.Walk(fn)
	}
}
