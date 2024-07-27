package hidapi

import (
	"github.com/neuroplastio/neio-agent/hidapi/hidusage"
)

type Usage uint32

func (u Usage) Page() uint16 {
	return uint16(u >> 16)
}

func (u Usage) ID() uint16 {
	return uint16(u)
}

func (u Usage) String() string {
	return hidusage.Format(u.Page(), u.ID())
}

func NewUsage(page, id uint16) Usage {
	return Usage(uint32(page)<<16 | uint32(id))
}

func ParseUsages(str []string) ([]Usage, error) {
	usages := make([]Usage, 0, len(str))
	for _, part := range str {
		usage, err := ParseUsage(part)
		if err != nil {
			return nil, err
		}
		usages = append(usages, usage)
	}

	return usages, nil
}

func ParseUsage(str string) (Usage, error) {
	pageInfo, usageInfo, err := hidusage.Parse(str)
	if err != nil {
		return 0, err
	}
	return NewUsage(pageInfo.Code, usageInfo.ID), nil
}
