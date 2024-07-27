package hidusage

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/neuroplastio/neio-agent/hidapi/hidusage/usagepages"
)

var (
	formatters = map[uint16]func(id uint16) string{
		usagepages.KeyboardKeypad: FormatKeyboardKeypad,
	}
	parsers = map[uint16]func(string) (uint16, error){
		usagepages.KeyboardKeypad: ParseKeyboardKeypad,
	}
)

func Format(page, id uint16) string {
	pageInfo, ok := usagepages.GetPageInfoByCode(page)
	if !ok {
		return fmt.Sprintf("0x%02x.0x%02x", page, id)
	}
	if formatter, ok := formatters[page]; ok {
		return formatter(id)
	}
	usageInfo, ok := pageInfo.Usages.Get(id)
	if !ok {
		return fmt.Sprintf("%s.0x%02x", pageInfo.Alias, id)
	}
	return fmt.Sprintf("%s.%s", pageInfo.Alias, usageInfo.Alias)
}

func Parse(str string) (uint16, uint16, error) {
	parts := strings.Split(str, ".")
	if len(parts) == 1 {
		parts = []string{"kb", parts[0]}
	}
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid usage string: %s", str)
	}
	prefix := parts[0]
	var (
		pageInfo usagepages.PageInfo
		ok       bool
	)
	if strings.HasPrefix(prefix, "0x") {
		code, err := strconv.ParseUint(prefix[2:], 16, 16)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid usage page: %s", prefix)
		}
		pageInfo, ok = usagepages.GetPageInfoByCode(uint16(code))
		if !ok {
			pageInfo.Code = uint16(code)
			ok = true
		}
	} else {
		pageInfo, ok = usagepages.GetPageInfoByAlias(prefix)
	}
	if !ok {
		return 0, 0, fmt.Errorf("unknown usage page: %s", prefix)
	}
	if parser, ok := parsers[pageInfo.Code]; ok {
		id, err := parser(parts[1])
		if err != nil {
			return 0, 0, err
		}
		return pageInfo.Code, id, nil
	}
	usageInfo, ok := pageInfo.Usages.ByAlias(parts[1])
	if !ok {
		return 0, 0, fmt.Errorf("unknown usage: %s", parts[1])
	}
	return pageInfo.Code, usageInfo.ID, nil
}

func FormatKeyboardKeypad(id uint16) string {
	return usagepages.KeyName(uint8(id))
}

func ParseKeyboardKeypad(str string) (uint16, error) {
	code, err := usagepages.KeyCode(str)
	if err != nil {
		return 0, err
	}
	return uint16(code), nil
}
