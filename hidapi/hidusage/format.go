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

func Parse(str string) (usagepages.PageInfo, usagepages.UsageInfo, error) {
	parts := strings.Split(str, ".")
	if len(parts) == 1 {
		parts = []string{"kb", parts[0]}
	}
	if len(parts) != 2 {
		return usagepages.PageInfo{}, usagepages.UsageInfo{}, fmt.Errorf("invalid usage string: %s", str)
	}
	prefix := parts[0]
	pageInfo, err := ParsePage(prefix)
	if err != nil {
		return usagepages.PageInfo{}, usagepages.UsageInfo{}, err
	}
	if parser, ok := parsers[pageInfo.Code]; ok {
		id, err := parser(parts[1])
		if err != nil {
			return usagepages.PageInfo{}, usagepages.UsageInfo{}, err
		}
		// TODO: parsers and formatters inside PageInfo object
		return pageInfo, usagepages.UsageInfo{ID: id}, nil
	}
	usageInfo, ok := pageInfo.Usages.ByAlias(parts[1])
	if !ok {
		return usagepages.PageInfo{}, usagepages.UsageInfo{}, fmt.Errorf("unknown usage: %s", parts[1])
	}
	return pageInfo, usageInfo, nil
}

func ParsePage(str string) (usagepages.PageInfo, error) {
	var (
		pageInfo usagepages.PageInfo
		ok       bool
	)
	if strings.HasPrefix(str, "0x") {
		code, err := strconv.ParseUint(str[2:], 16, 16)
		if err != nil {
			return usagepages.PageInfo{}, fmt.Errorf("invalid usage page: %s", str)
		}
		pageInfo, ok = usagepages.GetPageInfoByCode(uint16(code))
		if !ok {
			pageInfo.Code = uint16(code)
			ok = true
		}
	} else {
		pageInfo, ok = usagepages.GetPageInfoByAlias(str)
	}
	if !ok {
		return usagepages.PageInfo{}, fmt.Errorf("unknown usage page: %s", str)
	}
	return pageInfo, nil
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
