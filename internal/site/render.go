package site

import (
	"bytes"

	"github.com/yuin/goldmark"
)

// md is a single shared markdown instance. Goldmark is safe for concurrent use,
// and a personal blog has no need for per-call configuration.
var md = goldmark.New()

func markdownToHTML(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
