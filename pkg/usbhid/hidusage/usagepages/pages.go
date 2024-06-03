package usagepages

import "embed"

//go:generate go run ./cmd/generate-pages ./pages_gen.go ./keycodes_gen.go

//go:embed *.md
var FS embed.FS

// GetPageName returns the name of the page with the given code.
func GetPageName(code uint16) string {
	return pageNameMap[code]
}
