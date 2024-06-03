package hidusage

type UsageType string

// Usage Types (Controls)
const (
	UsageTypeLC  UsageType = "LC"
	UsageTypeOOC UsageType = "OOC"
	UsageTypeMC  UsageType = "MC"
	UsageTypeOSC UsageType = "OSC"
	UsageTypeRTC UsageType = "RTC"
)

// Usage Types (Data)
const (
	UsageTypeSel UsageType = "Sel"
	UsageTypeSV  UsageType = "SV"
	UsageTypeSF  UsageType = "SF"
	UsageTypeDV  UsageType = "DV"
	UsageTypeDF  UsageType = "DF"
)

// Usage Types (Collection)
const (
	UsageTypeNAry UsageType = "NAry"
	UsageTypeCA   UsageType = "CA"
	UsageTypeCL   UsageType = "CL"
	UsageTypeCP   UsageType = "CP"
	UsageTypeUS   UsageType = "US"
	UssageTypeUM  UsageType = "UM"
)

type Usage struct {
	ID    uint16
	Types []UsageType
	Name  string
}

type Page interface {
	ID() uint16
	Name() string
	// TODO: use DataFlags to determine usage type for mixed usage types (MC/DV)
	GetUsageType(usageID uint16) (UsageType, bool)
}

