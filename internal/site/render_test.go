package site

import (
	"strings"
	"testing"
)

func TestMarkdownToHTML_HighlightsFencedCode(t *testing.T) {
	src := "```go\nfunc main() {}\n```\n"
	out, err := markdownToHTML([]byte(src))
	if err != nil {
		t.Fatalf("markdownToHTML: %v", err)
	}
	html := string(out)
	if !strings.Contains(html, `class="chroma"`) {
		t.Errorf("fenced code block not highlighted, got:\n%s", html)
	}
	// Class-based output must not bake colours into inline style attributes.
	if strings.Contains(html, "style=") {
		t.Errorf("expected class-based highlighting, found inline styles:\n%s", html)
	}
}

func TestMarkdownToHTML_InlineCodeNotHighlighted(t *testing.T) {
	out, err := markdownToHTML([]byte("a `value` here\n"))
	if err != nil {
		t.Fatalf("markdownToHTML: %v", err)
	}
	if html := string(out); strings.Contains(html, "chroma") {
		t.Errorf("inline code should not be highlighted, got:\n%s", html)
	}
}
