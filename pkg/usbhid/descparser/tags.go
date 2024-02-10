package descparser

// Main items: xxxx 00 xx
// Global items: xxxx 01 xx
// Local items: xxxx 10 xx
// Size is determined by last two bits.
// These constants only contain first 6 bits. Size is determined by last two bits.
const (
	TagInput         = 0x80 // 1000 0001 + DataFlags
	TagOutput        = 0x90 // 1001 0001 + DataFlags
	TagFeature       = 0xB0 // 1011 0001 + DataFlags
	TagCollection    = 0xA0 // 1010 0001 + CollectionType
	TagEndCollection = 0xC0 // 1100 0000

	TagUsagePage       = 0x04 // 0000 01xx + UsagePage
	TagLogicalMinimum  = 0x14 // 0001 01xx + int
	TagLogicalMaximum  = 0x24 // 0010 01xx + int
	TagPhysicalMinimum = 0x34 // 0011 01xx + int
	TagPhysicalMaximum = 0x44 // 0100 01xx + int
	TagUnitExponent    = 0x54 // 0101 01xx + int
	TagUnit            = 0x64 // 0110 01xx + int
	TagReportSize      = 0x74 // 0111 01xx + int
	TagReportID        = 0x84 // 1000 01xx + int
	TagReportCount     = 0x94 // 1001 01xx + int
	TagPush            = 0xA4 // 1010 0100
	TagPop             = 0xB4 // 1011 0100

	TagUsage             = 0x08 // 0000 1001 + UsageID
	TagUsageMinimum      = 0x18 // 0001 10xx + int
	TagUsageMaximum      = 0x28 // 0010 10xx + int
	TagDesignatorIndex   = 0x38 // 0011 10xx + int
	TagDesignatorMinimum = 0x48 // 0100 10xx + int
	TagDesignatorMaximum = 0x58 // 0101 10xx + int
	TagStringIndex       = 0x68 // 0110 10xx + int
	TagStringMinimum     = 0x78 // 0111 10xx + int
	TagStringMaximum     = 0x88 // 1000 10xx + int
	TagDelimiter         = 0xA8 // 1010 1001 + 0/1
)

type Tag uint8

type TagItemSize uint8

const (
	TagItemSize0 TagItemSize = iota
	TagItemSize8
	TagItemSize16
	TagItemSize32
)

func (t Tag) PayloadSize() TagItemSize {
	return TagItemSize(t & 0x03)
}

type TagItemType uint8

const (
	TagItemTypeMain TagItemType = iota
	TagItemTypeGlobal
	TagItemTypeLocal
)

func (t Tag) ItemType() TagItemType {
	return TagItemType(t & 0x0C)
}

func (t Tag) TagPrefix() Tag {
	return t & 0xFC
}
