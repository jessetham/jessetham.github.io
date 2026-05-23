package site

import (
	"bytes"
	"html/template"
	"io"
	"path/filepath"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

// md is a single shared markdown instance. Goldmark is safe for concurrent use,
// and a personal blog has no need for per-call configuration.
//
// Fenced code blocks are syntax-highlighted at build time with class-based
// output; the colours live in static/highlight.css (see cmd/genhl).
var md = goldmark.New(
	goldmark.WithExtensions(
		highlighting.NewHighlighting(
			highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
		),
	),
)

func markdownToHTML(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// parseTemplates parses the two template pairs (base+post, base+index) once.
// Callers reuse the returned *Template across many ExecuteTemplate calls.
func parseTemplates(templatesDir string) (postTmpl, indexTmpl *template.Template, err error) {
	postTmpl, err = template.ParseFiles(
		filepath.Join(templatesDir, "base.html"),
		filepath.Join(templatesDir, "post.html"),
	)
	if err != nil {
		return nil, nil, err
	}
	indexTmpl, err = template.ParseFiles(
		filepath.Join(templatesDir, "base.html"),
		filepath.Join(templatesDir, "index.html"),
	)
	if err != nil {
		return nil, nil, err
	}
	return postTmpl, indexTmpl, nil
}

func renderPost(w io.Writer, t *template.Template, p Post) error {
	return t.ExecuteTemplate(w, "base.html", p)
}

func renderIndex(w io.Writer, t *template.Template, posts []Post) error {
	return t.ExecuteTemplate(w, "base.html", posts)
}
