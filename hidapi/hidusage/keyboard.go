package hidusage

import (
	"fmt"

	"github.com/neuroplastio/neuroplastio/hidapi/hiddesc"
	"github.com/neuroplastio/neuroplastio/hidapi/hidusage/usagepages"
)

type Key byte

func (k Key) String() string {
	return usagepages.KeyName(uint8(k))
}

type Keys []Key

func (k Keys) String() string {
	s := "["
	for i, key := range k {
		if i > 0 {
			s += ", "
		}
		s += key.String()
	}
	s += "]"
	return s
}

// KeyBits is a 240-bit long bitmap of pressed keys.
// Each bit represents a key.
type KeyBits [30]byte

func (k KeyBits) HidInput() {}

func (k KeyBits) IsPressed(key byte) bool {
	byteIndex := key / 8
	bitIndex := key % 8
	return k[byteIndex]&(1<<bitIndex) != 0
}

func (k KeyBits) PressedKeys() Keys {
	keys := make(Keys, 0, 8)
	for byteIndex, b := range k {
		for bitIndex := 0; bitIndex < 8; bitIndex++ {
			if b&(1<<bitIndex) != 0 {
				keys = append(keys, Key(byteIndex*8+bitIndex))
			}
		}
	}
	return keys
}

type KeyboardDriver struct {
	inputs     []hiddesc.DataItem
	startBytes []int
}

func NewKeyboardDriver(inputs []hiddesc.DataItem) (KeyboardDriver, error) {
	bitOffset := uint(0)
	driver := KeyboardDriver{
		inputs:     inputs,
		startBytes: make([]int, len(inputs)),
	}
	for i, dataItem := range inputs {
		if dataItem.UsagePage != usagepages.KeyboardKeypad {
			return KeyboardDriver{}, fmt.Errorf("unsupported usage page: %d", dataItem.UsagePage)
		}
		if dataItem.Flags.IsConstant() {
			// const - skip bits
			bitOffset += uint(dataItem.ReportSize) * uint(dataItem.ReportCount)
			driver.startBytes[i] = -1
			continue
		}
		if bitOffset%8 != 0 {
			return KeyboardDriver{}, fmt.Errorf("report descriptor is not byte-aligned. Offset: %d", bitOffset)
		}
		driver.startBytes[i] = int(bitOffset / 8)
		bitOffset += uint(dataItem.ReportSize) * uint(dataItem.ReportCount)
	}
	return driver, nil
}

func (k KeyboardDriver) Parse(report []byte) (KeyBits, error) {
	keys := KeyBits{}
	// TODO: configure report ID
	// report = report[1:]
	fmt.Println("report: ", report)
	for i, dataItem := range k.inputs {
		if k.startBytes[i] < 0 {
			continue
		}
		size := int(uint(dataItem.ReportSize)*uint(dataItem.ReportCount)+7) / 8
		data := report[k.startBytes[i] : k.startBytes[i]+size]
		switch {
		case dataItem.Flags.IsVariable():
			// always assume that it's byte-aligned
			byteStart := int(dataItem.UsageMinimum) / 8
			for i, b := range data {
				// Do not overwrite keys that are not pressed
				// This is to allow overlapping usage ranges
				if b != 0x00 {
					keys[byteStart+i] = b
				}
			}
		case dataItem.Flags.IsArray():
			for _, b := range data {
				if b == 0x00 {
					continue
				}
				byteIndex := b / 8
				bitIndex := b % 8
				keys[byteIndex] |= 1 << bitIndex
			}
		}
	}
	fmt.Println("keys: ", keys)
	return keys, nil
}
