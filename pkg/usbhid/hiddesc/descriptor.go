package hiddesc

type ReportDescriptor struct {
	// Top-level Application Collections
	Collections []Collection
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
// TODO: when setting up data item, it is possible to specify multiple usages and other properties. Keep track of that.
type DataItem struct {
	Flags        DataFlags
	UsagePage    uint16
	UsageIDs     []uint16
	UsageMinimum uint16
	UsageMaximum uint16
	ReportCount  uint8
	ReportSize   uint8
	ReportID     uint8

	DesignatorIndex   uint8
	DesignatorMinimum uint8
	DesignatorMaximum uint8

	LogicalMinimum  int16
	LogicalMaximum  int16
	PhysicalMinimum int16
	PhysicalMaximum int16
	UnitExponent    uint8
	Unit            uint8
}

type Control struct {
	Flags       DataFlags // same as in DataItem
	UsagePage   uint16    // same as in DataItem
	ReportIndex uint8     // index of this control in the DataItem
	ReportSize  uint8     // same as in DataItem
	ReportID    uint8     // same as in DataItem

	UsageID      uint16
	UsageMinimum uint16
	UsageMaximum uint16

	DesignatorIndex   uint8
	DesignatorMinimum uint8
	DesignatorMaximum uint8

	LogicalMinimum  int16
	LogicalMaximum  int16
	PhysicalMinimum int16
	PhysicalMaximum int16
	UnitExponent    uint16
	Unit            uint16
}
