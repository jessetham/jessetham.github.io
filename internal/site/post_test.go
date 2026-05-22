package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSlugFromPath(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"content/posts/hello-world.md", "hello-world"},
		{"content/posts/Hello-World.md", "hello-world"},
		{"hello.md", "hello"},
		{"/abs/path/2026-05-22-Post.md", "2026-05-22-post"},
		{"content/posts/nested/dir/HelloAgain.md", "helloagain"},
	}
	for _, tc := range cases {
		got := slugFromPath(tc.in)
		if got != tc.want {
			t.Errorf("slugFromPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSortPosts(t *testing.T) {
	date := func(s string) time.Time {
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatalf("bad date in test: %v", err)
		}
		return d
	}
	posts := []Post{
		{Slug: "older", Date: date("2026-01-01")},
		{Slug: "tied-b", Date: date("2026-05-22")},
		{Slug: "newer", Date: date("2026-06-01")},
		{Slug: "tied-a", Date: date("2026-05-22")},
	}
	sortPosts(posts)
	got := make([]string, len(posts))
	for i, p := range posts {
		got[i] = p.Slug
	}
	want := []string{"newer", "tied-a", "tied-b", "older"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("sort order: got %v want %v", got, want)
	}
}

func TestLoadPost(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Hello-World.md")
	contents := "---\ntitle: Hello\ndate: 2026-05-22\n---\n# Heading\n\nBody.\n"
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	p, err := loadPost(path)
	if err != nil {
		t.Fatalf("loadPost: %v", err)
	}
	if p.Slug != "hello-world" {
		t.Errorf("slug: got %q want %q", p.Slug, "hello-world")
	}
	if p.Title != "Hello" {
		t.Errorf("title: got %q want %q", p.Title, "Hello")
	}
	if got := p.Date.Format("2006-01-02"); got != "2026-05-22" {
		t.Errorf("date: got %q want %q", got, "2026-05-22")
	}
	if !strings.Contains(string(p.Body), "<h1>Heading</h1>") {
		t.Errorf("body should contain <h1>Heading</h1>, got %q", p.Body)
	}
}

func TestLoadPost_BadFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no-fm.md")
	if err := os.WriteFile(path, []byte("just body, no frontmatter\n"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if _, err := loadPost(path); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadPosts_SortsAndIgnoresNonMarkdown(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, body string) {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	write("a.md", "---\ntitle: A\ndate: 2026-01-01\n---\nA\n")
	write("b.md", "---\ntitle: B\ndate: 2026-06-01\n---\nB\n")
	write("README.txt", "ignored")

	posts, err := loadPosts(dir)
	if err != nil {
		t.Fatalf("loadPosts: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("got %d posts, want 2", len(posts))
	}
	if posts[0].Slug != "b" || posts[1].Slug != "a" {
		t.Errorf("order: got %q,%q want b,a", posts[0].Slug, posts[1].Slug)
	}
}

func TestLoadPosts_DuplicateSlugError(t *testing.T) {
	dir := t.TempDir()
	write := func(rel string) {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		body := "---\ntitle: t\ndate: 2026-05-22\n---\nx\n"
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	write("2024/hello.md")
	write("2025/hello.md")

	_, err := loadPosts(dir)
	if err == nil {
		t.Fatal("expected duplicate-slug error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "hello") {
		t.Errorf("error should name the slug, got %q", msg)
	}
	if !strings.Contains(msg, "2024/hello.md") || !strings.Contains(msg, "2025/hello.md") {
		t.Errorf("error should list both paths, got %q", msg)
	}
}
