package site

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeBaseURL(t *testing.T) {
	cases := map[string]string{
		"":                          defaultBaseURL,
		"https://jtham.dev":         "https://jtham.dev",
		"https://jtham.dev/":        "https://jtham.dev",
		"https://example.com/blog/": "https://example.com/blog",
	}
	for in, want := range cases {
		if got := normalizeBaseURL(in); got != want {
			t.Errorf("normalizeBaseURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDeriveDescription(t *testing.T) {
	cases := []struct {
		name string
		in   template.HTML
		want string
	}{
		{"first paragraph only", "<p>Hello there.</p><p>Second.</p>", "Hello there."},
		{"strips inline tags", `<p>See <a href="/x">my link</a> and <code>code</code>.</p>`, "See my link and code."},
		{"unescapes entities", "<p>Tom &amp; Jerry &lt;3</p>", "Tom & Jerry <3"},
		{"collapses whitespace", "<p>a\n\n  b   c</p>", "a b c"},
		{"falls back without paragraph", "<h1>Title</h1>", "Title"},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := deriveDescription(tc.in); got != tc.want {
				t.Errorf("deriveDescription(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestDeriveDescription_Truncates(t *testing.T) {
	long := strings.Repeat("word ", 60) // 300 chars
	got := deriveDescription(template.HTML("<p>" + long + "</p>"))
	r := []rune(got)
	if len(r) > descriptionMaxLen+1 { // +1 for the ellipsis rune
		t.Errorf("description not truncated: %d runes", len(r))
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncated description should end with ellipsis, got %q", got)
	}
	if strings.HasSuffix(strings.TrimSuffix(got, "…"), " ") {
		t.Errorf("should trim trailing space before ellipsis, got %q", got)
	}
}

func TestBuild_SEOArtifacts(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	out := filepath.Join(root, "public")

	writeFile(t, filepath.Join(content, "posts", "hello-world.md"),
		"---\ntitle: Hello, world\ndate: 2026-05-22\n---\nThe quick brown fox jumps.\n")

	cfg := Config{
		ContentDir:   content,
		TemplatesDir: repoTemplatesDir(t),
		OutDir:       out,
		BaseURL:      "https://jtham.dev/",
	}
	if err := Build(cfg); err != nil {
		t.Fatalf("Build: %v", err)
	}

	read := func(rel string) string {
		t.Helper()
		bs, err := os.ReadFile(filepath.Join(out, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		return string(bs)
	}

	sitemap := read("sitemap.xml")
	for _, want := range []string{
		"<loc>https://jtham.dev/</loc>",
		"<loc>https://jtham.dev/posts/hello-world/</loc>",
		"<lastmod>2026-05-22</lastmod>",
	} {
		if !strings.Contains(sitemap, want) {
			t.Errorf("sitemap.xml missing %q", want)
		}
	}

	robots := read("robots.txt")
	if !strings.Contains(robots, "Sitemap: https://jtham.dev/sitemap.xml") {
		t.Errorf("robots.txt missing sitemap line, got: %s", robots)
	}

	feed := read("feed.xml")
	for _, want := range []string{
		`<rss version="2.0">`,
		"<link>https://jtham.dev/posts/hello-world/</link>",
		"The quick brown fox jumps.",
	} {
		if !strings.Contains(feed, want) {
			t.Errorf("feed.xml missing %q", want)
		}
	}

	post := read(filepath.Join("posts", "hello-world", "index.html"))
	for _, want := range []string{
		`<title>Hello, world — Jesse Tham</title>`,
		`<link rel="canonical" href="https://jtham.dev/posts/hello-world/">`,
		`<meta name="description" content="The quick brown fox jumps.">`,
		`<meta property="og:type" content="article">`,
		`<meta property="article:published_time" content="2026-05-22">`,
		`"@type":"BlogPosting"`,
	} {
		if !strings.Contains(post, want) {
			t.Errorf("post page missing %q", want)
		}
	}

	index := read("index.html")
	for _, want := range []string{
		`<link rel="canonical" href="https://jtham.dev/">`,
		`<meta property="og:type" content="website">`,
		`"@type":"WebSite"`,
	} {
		if !strings.Contains(index, want) {
			t.Errorf("index page missing %q", want)
		}
	}
}
