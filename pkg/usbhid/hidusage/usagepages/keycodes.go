package usagepages

import (
	"fmt"
	"strings"
)

func KeyName(code uint8) string {
	name, ok := keyNameMap[code]
	if !ok {
		return fmt.Sprintf("0x%x", code)
	}
	return name
}

var keyNameReverseMap = map[string]uint8{}

func init() {
	for code, name := range keyNameMap {
		keyNameReverseMap[name] = code
	}
}

func KeyCode(name string) uint8 {
	code, ok := keyNameReverseMap[name]
	if !ok {
		return 0
	}
	return code
}

func KeyCodesRegexp() string {
	items := make([]string, 0, len(keyNameMap))
	for _, name := range keyNameMap {
		n := strings.TrimPrefix(name, "Key")
		items = append(items, n)
	}

	return "(" + strings.Join(items, "|") + ")"
}
