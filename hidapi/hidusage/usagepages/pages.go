//go:build !generate

package usagepages

import (
	"fmt"
	"strconv"

	"github.com/iancoleman/strcase"
)

var pageAliasMap = map[string]uint16{}

func init() {
	for code, page := range pages {
		pageAliasMap[page.Alias] = code
	}
}

func GetPageInfoByAlias(alias string) (PageInfo, bool) {
	code, ok := pageAliasMap[alias]
	if !ok {
		return PageInfo{}, false
	}
	return pages[code], ok
}

func GetPageInfoByCode(code uint16) (PageInfo, bool) {
	page, ok := pages[code]
	return page, ok
}

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
	UsageTypeUM   UsageType = "UM"
)

type UsageInfo struct {
	ID    uint16
	Name  string
	Alias string
	Types []UsageType
}

type PageInfo struct {
	Code   uint16
	Name   string
	Alias  string
	Usages UsageCollection
}

type UsageCollection interface {
	Get(id uint16) (UsageInfo, bool)
	ByAlias(alias string) (UsageInfo, bool)
}

type ordinalUsageCollection struct {
	namePrefix string
	types      []UsageType
}

func (o ordinalUsageCollection) Get(id uint16) (UsageInfo, bool) {
	return UsageInfo{
		ID:    id,
		Name:  fmt.Sprintf("%s %d", o.namePrefix, id),
		Alias: strconv.FormatInt(int64(id), 10),
		Types: o.types,
	}, true
}

func (o ordinalUsageCollection) ByAlias(alias string) (UsageInfo, bool) {
	code, err := strconv.ParseInt(alias, 10, 16)
	if err != nil {
		return UsageInfo{}, false
	}
	return UsageInfo{
		ID:    uint16(code),
		Name:  fmt.Sprintf("%s %d", o.namePrefix, code),
		Alias: alias,
		Types: o.types,
	}, true

}

func newOrdinalUsageCollection(namePrefix string, types ...UsageType) ordinalUsageCollection {
	return ordinalUsageCollection{
		namePrefix: namePrefix,
		types:      types,
	}
}

func newUsageTable() usageTable {
	return usageTable{
		usages:   make(map[uint16]UsageInfo),
		aliasMap: make(map[string]UsageInfo),
	}
}

type usageTable struct {
	usages   map[uint16]UsageInfo
	aliasMap map[string]UsageInfo
}

func (u usageTable) Get(id uint16) (UsageInfo, bool) {
	usage, ok := u.usages[id]
	if ok {
		return usage, true
	}
	return UsageInfo{}, false
}

func (u usageTable) ByAlias(alias string) (UsageInfo, bool) {
	usage, ok := u.aliasMap[alias]
	return usage, ok
}

func (u usageTable) usage(id uint16, name string, types ...UsageType) usageTable {
	alias := strcase.ToCamel(name)
	u.usages[id] = UsageInfo{
		ID:    id,
		Name:  name,
		Alias: alias,
		Types: types,
	}
	u.aliasMap[alias] = u.usages[id]
	return u
}
