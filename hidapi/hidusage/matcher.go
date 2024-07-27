package hidusage

import "strings"

type Matcher func(page uint16, id uint16) bool

func NewMatcher(patterns ...string) (Matcher, error) {
	if len(patterns) == 0 {
		return func(page, id uint16) bool {
			return true
		}, nil
	}
	funcs := make([]Matcher, len(patterns))
	for i, pattern := range patterns {
		matcher, err := newMatcher(pattern)
		if err != nil {
			return nil, err
		}
		funcs[i] = matcher
	}
	return func(page, id uint16) bool {
		for _, matcher := range funcs {
			if matcher(page, id) {
				return true
			}
		}
		return false
	}, nil
}

func newMatcher(pattern string) (Matcher, error) {
	parts := strings.Split(pattern, ".")
	if len(parts) == 1 {
		parts = []string{parts[0], "*"}
	}
	pagePattern := parts[0]
	idPattern := parts[1]
	if pagePattern == "*" {
		return func(page, id uint16) bool {
			return true
		}, nil
	}
	if idPattern == "*" {
		pageInfo, err := ParsePage(pagePattern)
		if err != nil {
			return nil, err
		}
		return func(page, id uint16) bool {
			return pageInfo.Code == page
		}, nil
	}
	pageInfo, usageInfo, err := Parse(pattern)
	if err != nil {
		return nil, err
	}
	return func(page, id uint16) bool {
		return pageInfo.Code == page && usageInfo.ID == id
	}, nil
}
