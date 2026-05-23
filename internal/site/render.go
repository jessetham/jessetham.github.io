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

// pageData is the single view-model passed to base.html for every page. Post
// is non-nil on post pages; Posts is populated on the index.
type pageData struct {
	BaseURL     string
	SiteTitle   string
	Author      string
	Title       string
	Description string
	Canonical   string
	OGType      string
	Post        *Post
	Posts       []Post
	JSONLD      template.HTML
}

func newPostPage(cfg Config, p Post) pageData {
	canonical := postURL(cfg, p)
	author := map[string]any{"@type": "Person", "name": siteAuthor}
	ld := map[string]any{
		"@context":      "https://schema.org",
		"@type":         "BlogPosting",
		"headline":      p.Title,
		"datePublished": p.Date.Format("2006-01-02"),
		"url":           canonical,
		"author":        author,
	}
	if p.Description != "" {
		ld["description"] = p.Description
	}
	return pageData{
		BaseURL:     cfg.BaseURL,
		SiteTitle:   siteTitle,
		Author:      siteAuthor,
		Title:       p.Title + " — " + siteTitle,
		Description: p.Description,
		Canonical:   canonical,
		OGType:      "article",
		Post:        &p,
		JSONLD:      jsonLDScript(ld),
	}
}

func newIndexPage(cfg Config, posts []Post) pageData {
	ld := map[string]any{
		"@context": "https://schema.org",
		"@type":    "WebSite",
		"name":     siteTitle,
		"url":      cfg.BaseURL + "/",
	}
	return pageData{
		BaseURL:     cfg.BaseURL,
		SiteTitle:   siteTitle,
		Author:      siteAuthor,
		Title:       siteTitle,
		Description: siteDescription,
		Canonical:   cfg.BaseURL + "/",
		OGType:      "website",
		Posts:       posts,
		JSONLD:      jsonLDScript(ld),
	}
}

func renderPost(w io.Writer, t *template.Template, cfg Config, p Post) error {
	return t.ExecuteTemplate(w, "base.html", newPostPage(cfg, p))
}

func renderIndex(w io.Writer, t *template.Template, cfg Config, posts []Post) error {
	return t.ExecuteTemplate(w, "base.html", newIndexPage(cfg, posts))
}
