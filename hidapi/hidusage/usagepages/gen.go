//go:generate go run -tags generate ./cmd/generate-pages ./pages_gen.go ./keycodes_gen.go
package usagepages

import "embed"

//go:embed data/*.md
var FS embed.FS
