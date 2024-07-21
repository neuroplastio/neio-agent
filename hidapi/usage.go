package hidapi

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/neuroplastio/neuroplastio/hidapi/hidusage/usagepages"
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

func ParseUsages(str []string) ([]Usage, error) {
	usages := make([]Usage, 0, len(str))
	for _, part := range str {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("empty usage")
		}
		usage, err := ParseUsage(part)
		if err != nil {
			return nil, err
		}
		usages = append(usages, usage)
	}

	return usages, nil
}

func ParseUsage(str string) (Usage, error) {
	parts := strings.Split(str, ".")
	if len(parts) == 1 {
		parts = []string{"key", parts[0]}
	}
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid usage: %s", str)
	}
	prefix := parts[0]
	switch prefix {
	case "key":
		code := usagepages.KeyCode("Key" + parts[1])
		if code == 0 {
			return 0, fmt.Errorf("invalid key code: %s", parts[1])
		}
		return NewUsage(usagepages.KeyboardKeypad, uint16(code)), nil
	case "btn":
		code, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("invalid button code: %s", parts[1])
		}
		return NewUsage(usagepages.Button, uint16(code)), nil
	default:
		return 0, fmt.Errorf("invalid usage prefix: %s", prefix)
	}
}
