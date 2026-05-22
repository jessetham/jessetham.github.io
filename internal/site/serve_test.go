package site

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestShouldWatch(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		outDir string
		want   bool
	}{
		{"normal markdown", "content/posts/hello.md", "public", true},
		{"nested template", "templates/post.html", "public", true},
		{"static css", "static/style.css", "public", true},
		{"hidden file at root", ".DS_Store", "public", false},
		{"hidden file nested", "content/.draft.md", "public", false},
		{"hidden dir", "content/.cache/x.md", "public", false},
		{"path inside outDir", "public/index.html", "public", false},
		{"outDir itself", "public", "public", false},
		{"outDir nested deep", "public/posts/hello/index.html", "public", false},
		{"prefix match isn't substring match", "publicity/x.md", "public", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldWatch(tt.path, tt.outDir); got != tt.want {
				t.Errorf("shouldWatch(%q, %q) = %v, want %v", tt.path, tt.outDir, got, tt.want)
			}
		})
	}
}

func TestDebounce_SingleEventFiresOnce(t *testing.T) {
	in := make(chan fsnotify.Event, 1)
	var fired atomic.Int32
	done := make(chan struct{})
	go func() {
		debounce(in, 20*time.Millisecond, func() { fired.Add(1) })
		close(done)
	}()

	in <- fsnotify.Event{}
	time.Sleep(80 * time.Millisecond)
	close(in)
	<-done

	if got := fired.Load(); got != 1 {
		t.Errorf("fired = %d, want 1", got)
	}
}

func TestDebounce_BurstFiresOnce(t *testing.T) {
	in := make(chan fsnotify.Event, 16)
	var fired atomic.Int32
	done := make(chan struct{})
	go func() {
		debounce(in, 30*time.Millisecond, func() { fired.Add(1) })
		close(done)
	}()

	for range 5 {
		in <- fsnotify.Event{}
		time.Sleep(2 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	close(in)
	<-done

	if got := fired.Load(); got != 1 {
		t.Errorf("fired = %d, want 1", got)
	}
}

func TestDebounce_SeparateBurstsFireTwice(t *testing.T) {
	in := make(chan fsnotify.Event, 16)
	var fired atomic.Int32
	done := make(chan struct{})
	go func() {
		debounce(in, 20*time.Millisecond, func() { fired.Add(1) })
		close(done)
	}()

	in <- fsnotify.Event{}
	time.Sleep(80 * time.Millisecond)
	in <- fsnotify.Event{}
	time.Sleep(80 * time.Millisecond)
	close(in)
	<-done

	if got := fired.Load(); got != 2 {
		t.Errorf("fired = %d, want 2", got)
	}
}

func TestDebounce_CloseReturns(t *testing.T) {
	in := make(chan fsnotify.Event)
	done := make(chan struct{})
	go func() {
		debounce(in, time.Second, func() {})
		close(done)
	}()
	close(in)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("debounce did not return after channel close")
	}
}

// serveFixture lays a tiny site (1 post, real templates, 1 static file)
// into root and returns a Config pointing at it.
func serveFixture(t *testing.T, root string) Config {
	t.Helper()

	for _, d := range []string{"content/posts", "templates", "static"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	must := func(path, contents string) {
		if err := os.WriteFile(filepath.Join(root, path), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	must("content/posts/hello.md", "---\ntitle: Hello\ndate: 2026-05-22\n---\nhi\n")
	must("static/style.css", "body{}\n")

	for _, name := range []string{"base.html", "post.html", "index.html"} {
		src, err := os.ReadFile(filepath.Join("..", "..", "templates", name))
		if err != nil {
			t.Fatal(err)
		}
		must("templates/"+name, string(src))
	}

	return Config{
		ContentDir:   filepath.Join(root, "content"),
		TemplatesDir: filepath.Join(root, "templates"),
		StaticDir:    filepath.Join(root, "static"),
		OutDir:       filepath.Join(root, "public"),
	}
}

func TestServe_ServesIndex(t *testing.T) {
	cfg := serveFixture(t, t.TempDir())

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- serveOnListener(ctx, cfg, listener) }()

	url := "http://" + listener.Addr().String() + "/"
	resp, err := httpGetWithRetry(t, url, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "Hello") {
		t.Errorf("response body does not contain post title; got: %s", body)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("serveOnListener returned: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveOnListener did not return within 2s after ctx cancel")
	}
}

func httpGetWithRetry(t *testing.T, url string, timeout time.Duration) (*http.Response, error) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		time.Sleep(20 * time.Millisecond)
	}
	return nil, lastErr
}

func TestServe_RebuildsOnContentChange(t *testing.T) {
	root := t.TempDir()
	cfg := serveFixture(t, root)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- serveOnListener(ctx, cfg, listener) }()

	url := "http://" + listener.Addr().String() + "/posts/hello/"
	resp, err := httpGetWithRetry(t, url, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "Hello") {
		t.Fatalf("initial body missing original title: %s", body)
	}

	postPath := filepath.Join(cfg.ContentDir, "posts", "hello.md")
	if err := os.WriteFile(postPath,
		[]byte("---\ntitle: Updated\ndate: 2026-05-22\n---\nbye\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if strings.Contains(string(body), "Updated") {
			cancel()
			<-errCh
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("server did not pick up the new title within 3s")
}
