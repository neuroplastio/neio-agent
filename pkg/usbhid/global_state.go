package usbhid

type GlobalState struct {
	UsagePage uint16
	UsageID uint16
	LogicalMinimum int8
	LogicalMaximum int8
	PhysicalMinimum int8
	PhysicalMaximum int8
	UnitExponent int8
	Unit uint8
	ReportSize uint8
	ReportID uint8
	ReportCount uint8
}

