package pageparser

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark-meta"
)

type PageParser struct {
	md goldmark.Markdown
}

func NewPageParser() *PageParser {
	markdown := goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			meta.Meta,
		),
	)
	
	return &PageParser{
		md: markdown,
	}
}


