package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoTemplatesDir resolves to the repo's templates/ directory.
// build_test.go lives at internal/site/, so ../../templates is the repo root.
func repoTemplatesDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "templates"))
	if err != nil {
		t.Fatalf("abs templates path: %v", err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("templates dir %s not found (run tests from repo): %v", abs, err)
	}
	return abs
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestBuild_EndToEnd(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	static := filepath.Join(root, "static")
	out := filepath.Join(root, "public")

	writeFile(t, filepath.Join(content, "posts", "hello-world.md"),
		"---\ntitle: Hello, world\ndate: 2026-05-22\n---\nGreetings.\n")
	writeFile(t, filepath.Join(content, "posts", "second-post.md"),
		"---\ntitle: Second Post\ndate: 2026-06-01\n---\nMore words.\n")
	writeFile(t, filepath.Join(static, "style.css"), "body { color: rebeccapurple; }\n")

	cfg := Config{
		ContentDir:   content,
		TemplatesDir: repoTemplatesDir(t),
		StaticDir:    static,
		OutDir:       out,
	}
	if err := Build(cfg); err != nil {
		t.Fatalf("Build: %v", err)
	}

	indexBytes, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	index := string(indexBytes)
	for _, want := range []string{"Hello, world", "Second Post", `href="/posts/hello-world/"`, `href="/posts/second-post/"`} {
		if !strings.Contains(index, want) {
			t.Errorf("index missing %q", want)
		}
	}

	for slug, title := range map[string]string{
		"hello-world": "Hello, world",
		"second-post": "Second Post",
	} {
		p := filepath.Join(out, "posts", slug, "index.html")
		bs, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("read %s: %v", p, err)
			continue
		}
		if !strings.Contains(string(bs), title) {
			t.Errorf("post %s missing title %q", slug, title)
		}
	}

	srcCSS, _ := os.ReadFile(filepath.Join(static, "style.css"))
	dstCSS, err := os.ReadFile(filepath.Join(out, "style.css"))
	if err != nil {
		t.Fatalf("read out style.css: %v", err)
	}
	if string(srcCSS) != string(dstCSS) {
		t.Errorf("static file not copied verbatim")
	}
}

func TestBuild_DuplicateSlugFails(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	out := filepath.Join(root, "public")
	body := "---\ntitle: t\ndate: 2026-05-22\n---\nx\n"
	writeFile(t, filepath.Join(content, "posts", "2024", "hello.md"), body)
	writeFile(t, filepath.Join(content, "posts", "2025", "hello.md"), body)

	cfg := Config{
		ContentDir:   content,
		TemplatesDir: repoTemplatesDir(t),
		OutDir:       out,
	}
	err := Build(cfg)
	if err == nil {
		t.Fatal("expected duplicate slug error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "hello") {
		t.Errorf("error should name slug, got %q", msg)
	}
	if !strings.Contains(msg, "2024/hello.md") || !strings.Contains(msg, "2025/hello.md") {
		t.Errorf("error should list both paths, got %q", msg)
	}
}
