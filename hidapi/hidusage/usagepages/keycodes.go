//go:build !generate

package usagepages

import (
	"fmt"
	"strconv"
	"strings"
)

var keyCodeMap = map[string]uint8{}

func init() {
	for code, name := range keyNameMap {
		keyCodeMap[name] = code
	}
}

func KeyCode(name string) (uint8, error) {
	if strings.HasPrefix(name, "Code") {
		code, err := strconv.ParseUint(name[4:], 16, 8)
		if err != nil {
			return 0, fmt.Errorf("invalid key code: %s", name)
		}
		return uint8(code), nil
	}
	code, ok := keyCodeMap[name]
	if !ok {
		return 0, fmt.Errorf("invalid key name: %s", name)
	}
	return code, nil
}

func KeyName(code uint8) string {
	name, ok := keyNameMap[code]
	if !ok {
		return fmt.Sprintf("Code%02x", code)
	}
	return name
}
