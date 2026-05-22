# Serve Subcommand Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `blog serve` subcommand that runs a localhost HTTP server serving `public/`, watches the source directories, and rebuilds the site automatically on file changes.

**Architecture:** Reuse `site.Build(cfg)` as-is. A new `site.Serve(ctx, cfg, addr)` starts an HTTP file server against `cfg.OutDir` guarded by a `sync.RWMutex`; an fsnotify watcher fires events through a filter and a debounce stage into a rebuilder that takes the write lock and calls `Build`.

**Tech Stack:** Go 1.26, stdlib (`net/http`, `sync`, `context`, `time`, `os/signal`), `github.com/fsnotify/fsnotify`. No new dependencies beyond fsnotify.

**Spec:** `docs/superpowers/specs/2026-05-22-serve-subcommand-design.md`

**File map:**

| File | Action | Responsibility |
|---|---|---|
| `go.mod`, `go.sum` | Modify | Add fsnotify direct dep |
| `internal/site/serve.go` | Create | `Serve`, `serveOnListener`, `shouldWatch`, `debounce`, watcher setup |
| `internal/site/serve_test.go` | Create | Unit tests for `shouldWatch` + `debounce`; integration tests for `serveOnListener` |
| `cmd/blog/main.go` | Modify | Add `case "serve"`, `runServe`, update `usage()` |

---

## Task 1: Add fsnotify dependency

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add the dependency**

Run from repo root:

```bash
go get github.com/fsnotify/fsnotify
```

- [ ] **Step 2: Verify it's a direct dep**

```bash
go mod tidy && grep fsnotify go.mod
```

Expected: `fsnotify` appears in `go.mod` as a direct require (not `// indirect`).

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "build: add fsnotify dependency for serve watcher"
```

---

## Task 2: shouldWatch path filter (TDD)

**Files:**
- Create: `internal/site/serve.go`
- Create: `internal/site/serve_test.go`

Filters out hidden files and anything inside `OutDir` before they trigger a rebuild.

- [ ] **Step 1: Write the failing tests**

Create `internal/site/serve_test.go`:

```go
package site

import "testing"

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
```

- [ ] **Step 2: Run tests, verify they fail**

```bash
go test ./internal/site/ -run TestShouldWatch
```

Expected: build fails (`undefined: shouldWatch`).

- [ ] **Step 3: Implement `shouldWatch`**

Create `internal/site/serve.go` with:

```go
package site

import (
	"path/filepath"
	"strings"
)

// shouldWatch returns true if path should trigger a rebuild.
// Hidden files (any component starting with '.') and paths inside outDir
// are excluded.
func shouldWatch(path, outDir string) bool {
	clean := filepath.Clean(path)
	out := filepath.Clean(outDir)
	sep := string(filepath.Separator)
	if clean == out || strings.HasPrefix(clean, out+sep) {
		return false
	}
	for _, part := range strings.Split(clean, sep) {
		if part != "" && part != "." && strings.HasPrefix(part, ".") {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Run tests, verify they pass**

```bash
go test ./internal/site/ -run TestShouldWatch -v
```

Expected: all subtests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/site/serve.go internal/site/serve_test.go
git commit -m "feat(serve): add shouldWatch path filter"
```

---

## Task 3: debounce function (TDD)

**Files:**
- Modify: `internal/site/serve.go`
- Modify: `internal/site/serve_test.go`

Collapses bursts of fsnotify events into a single rebuild trigger.

- [ ] **Step 1: Write the failing tests**

Append to `internal/site/serve_test.go`:

```go
import (
	// add to existing import block
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
)

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

	// Burst of 5 events well within the debounce window.
	for i := 0; i < 5; i++ {
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
	time.Sleep(80 * time.Millisecond) // > wait, first fire happens
	in <- fsnotify.Event{}
	time.Sleep(80 * time.Millisecond) // > wait, second fire happens
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
```

- [ ] **Step 2: Run tests, verify they fail**

```bash
go test ./internal/site/ -run TestDebounce
```

Expected: build fails (`undefined: debounce`).

- [ ] **Step 3: Implement `debounce`**

Append to `internal/site/serve.go`:

```go
import (
	// add to existing import block
	"time"

	"github.com/fsnotify/fsnotify"
)

// debounce reads from in and calls fire after wait has passed since the
// most recent event. Returns when in is closed.
func debounce(in <-chan fsnotify.Event, wait time.Duration, fire func()) {
	var timer *time.Timer
	var timerCh <-chan time.Time
	for {
		select {
		case _, ok := <-in:
			if !ok {
				if timer != nil {
					timer.Stop()
				}
				return
			}
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(wait)
			timerCh = timer.C
		case <-timerCh:
			timerCh = nil
			fire()
		}
	}
}
```

- [ ] **Step 4: Run tests, verify they pass**

```bash
go test ./internal/site/ -run TestDebounce -v
```

Expected: all four subtests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/site/serve.go internal/site/serve_test.go
git commit -m "feat(serve): add event debouncer"
```

---

## Task 4: Serve lifecycle — initial build + HTTP server (TDD)

**Files:**
- Modify: `internal/site/serve.go`
- Modify: `internal/site/serve_test.go`

Set up the public surface (`Serve`), a testable inner (`serveOnListener`), the locked file handler, the initial build, the HTTP server, and the ctx-driven shutdown. No watcher yet — that comes in Task 5.

- [ ] **Step 1: Write the failing integration test**

Append to `internal/site/serve_test.go`:

```go
import (
	// add to existing import block
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

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

	// Copy real templates so we don't duplicate template text in tests.
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

// httpGetWithRetry polls until the server is responsive or timeout elapses.
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
```

- [ ] **Step 2: Run test, verify it fails**

```bash
go test ./internal/site/ -run TestServe_ServesIndex
```

Expected: build fails (`undefined: serveOnListener`).

- [ ] **Step 3: Implement `Serve` and `serveOnListener`**

Append to `internal/site/serve.go`:

```go
import (
	// add to existing import block
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
)

// Serve starts an HTTP server bound to addr that serves cfg.OutDir, runs an
// initial Build, and rebuilds on file changes under cfg.ContentDir,
// cfg.TemplatesDir, and cfg.StaticDir. Blocks until ctx is cancelled or a
// fatal error occurs.
func Serve(ctx context.Context, cfg Config, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return serveOnListener(ctx, cfg, listener)
}

func serveOnListener(ctx context.Context, cfg Config, listener net.Listener) error {
	if err := Build(cfg); err != nil {
		return fmt.Errorf("initial build: %w", err)
	}

	var mu sync.RWMutex
	fileSrv := http.FileServer(http.Dir(cfg.OutDir))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		fileSrv.ServeHTTP(w, r)
	})

	srv := &http.Server{Handler: handler}

	fmt.Printf("serving %s at http://%s  (watching %s, %s, %s)\n",
		cfg.OutDir, listener.Addr(), cfg.ContentDir, cfg.TemplatesDir, cfg.StaticDir)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stdout, "shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-serveErr:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}
```

- [ ] **Step 4: Run test, verify it passes**

```bash
go test ./internal/site/ -run TestServe_ServesIndex -v
```

Expected: PASS within ~1s.

- [ ] **Step 5: Run the full package test suite to confirm no regressions**

```bash
go test ./internal/site/...
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/site/serve.go internal/site/serve_test.go
git commit -m "feat(serve): add Serve and serveOnListener with locked file handler"
```

---

## Task 5: Add watcher + rebuilder to Serve (TDD)

**Files:**
- Modify: `internal/site/serve.go`
- Modify: `internal/site/serve_test.go`

Wire fsnotify into `serveOnListener`: walk each watched root, register every subdirectory, drain events through `shouldWatch`, push survivors into a channel, run `debounce` on that channel, and rebuild under the write lock when it fires.

- [ ] **Step 1: Write the failing integration test**

Append to `internal/site/serve_test.go`:

```go
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

	// Rewrite the post with a new title.
	postPath := filepath.Join(cfg.ContentDir, "posts", "hello.md")
	if err := os.WriteFile(postPath,
		[]byte("---\ntitle: Updated\ndate: 2026-05-22\n---\nbye\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Poll for the new title to appear.
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
```

- [ ] **Step 2: Run test, verify it fails**

```bash
go test ./internal/site/ -run TestServe_RebuildsOnContentChange
```

Expected: FAIL — the server keeps serving the original title because nothing watches the file system yet.

- [ ] **Step 3: Add watcher setup and rebuilder to `serveOnListener`**

Modify `internal/site/serve.go`. Add imports (`errors`, `io/fs`) and replace `serveOnListener` with the version below:

```go
import (
	// add these to the existing import block
	"errors"
	"io/fs"
)

const debounceWait = 100 * time.Millisecond

func serveOnListener(ctx context.Context, cfg Config, listener net.Listener) error {
	if err := Build(cfg); err != nil {
		return fmt.Errorf("initial build: %w", err)
	}

	var mu sync.RWMutex
	fs := http.FileServer(http.Dir(cfg.OutDir))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		fs.ServeHTTP(w, r)
	})

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	for _, dir := range []string{cfg.ContentDir, cfg.TemplatesDir, cfg.StaticDir} {
		if err := addDirsRecursive(watcher, dir); err != nil {
			return err
		}
	}

	filtered := make(chan fsnotify.Event, 64)
	go func() {
		defer close(filtered)
		for {
			select {
			case ev, ok := <-watcher.Events:
				if !ok {
					return
				}
				if ev.Op&fsnotify.Create != 0 {
					if info, statErr := os.Stat(ev.Name); statErr == nil && info.IsDir() {
						_ = watcher.Add(ev.Name)
					}
				}
				if shouldWatch(ev.Name, cfg.OutDir) {
					select {
					case filtered <- ev:
					default:
					}
				}
			case werr := <-watcher.Errors:
				fmt.Fprintln(os.Stderr, "watcher error:", werr)
			case <-ctx.Done():
				return
			}
		}
	}()

	go debounce(filtered, debounceWait, func() {
		mu.Lock()
		defer mu.Unlock()
		if err := Build(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "rebuild failed:", err)
		}
	})

	srv := &http.Server{Handler: handler}

	fmt.Printf("serving %s at http://%s  (watching %s, %s, %s)\n",
		cfg.OutDir, listener.Addr(), cfg.ContentDir, cfg.TemplatesDir, cfg.StaticDir)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stdout, "shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-serveErr:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

// addDirsRecursive registers root and every subdirectory beneath it with the
// watcher. Missing roots are skipped silently (static/ may legitimately not
// exist).
func addDirsRecursive(w *fsnotify.Watcher, root string) error {
	if _, err := os.Stat(root); errors.Is(err, fs.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return w.Add(path)
		}
		return nil
	})
}
```

Note: the `fileSrv` variable from Task 4 is intentionally already named to avoid colliding with the `io/fs` package import added in this task. Keep it as `fileSrv`.

- [ ] **Step 4: Run test, verify it passes**

```bash
go test ./internal/site/ -run TestServe_RebuildsOnContentChange -v
```

Expected: PASS within ~1-2s. fsnotify on macOS uses FSEvents which can take ~100ms to deliver events, plus the 100ms debounce, so this should comfortably fit in the 3s poll window.

- [ ] **Step 5: Run the full package test suite**

```bash
go test ./internal/site/...
```

Expected: all tests pass (including TestServe_ServesIndex from Task 4 and the build tests from before this work started).

- [ ] **Step 6: Commit**

```bash
git add internal/site/serve.go internal/site/serve_test.go
git commit -m "feat(serve): watch source dirs and rebuild on change"
```

---

## Task 6: Wire `runServe` into the CLI

**Files:**
- Modify: `cmd/blog/main.go`

- [ ] **Step 1: Add `runServe` and dispatch**

Replace the contents of `cmd/blog/main.go` with:

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"blog/internal/site"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "build":
		if err := runBuild(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "serve":
		if err := runServe(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func runBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	cfg := site.Config{}
	fs.StringVar(&cfg.ContentDir, "content", "content", "content directory")
	fs.StringVar(&cfg.TemplatesDir, "templates", "templates", "templates directory")
	fs.StringVar(&cfg.StaticDir, "static", "static", "static assets directory")
	fs.StringVar(&cfg.OutDir, "out", "public", "build output directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return site.Build(cfg)
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	cfg := site.Config{}
	var addr string
	fs.StringVar(&cfg.ContentDir, "content", "content", "content directory")
	fs.StringVar(&cfg.TemplatesDir, "templates", "templates", "templates directory")
	fs.StringVar(&cfg.StaticDir, "static", "static", "static assets directory")
	fs.StringVar(&cfg.OutDir, "out", "public", "build output directory")
	fs.StringVar(&addr, "addr", ":8080", "HTTP listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return site.Serve(ctx, cfg, addr)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  blog build [--content DIR] [--templates DIR] [--static DIR] [--out DIR]")
	fmt.Fprintln(os.Stderr, "  blog serve [--addr :8080] [--content DIR] [--templates DIR] [--static DIR] [--out DIR]")
}
```

- [ ] **Step 2: Build the binary**

```bash
go build ./cmd/blog
```

Expected: produces `./blog` with no errors.

- [ ] **Step 3: Run a smoke test**

In one terminal:

```bash
./blog serve --addr :8081
```

Expected output:
```
built 1 posts in XXms
serving public/ at http://[::]:8081  (watching content, templates, static)
```

In another terminal:

```bash
curl -s http://localhost:8081/ | head -20
```

Expected: HTML containing the existing post title ("Hello, world").

While the server is still running, edit `content/posts/hello-world.md` — change the title — and save. Watch the first terminal for a new `built 1 posts in XXms` line.

```bash
curl -s http://localhost:8081/ | head -20
```

Expected: the new title appears. Press Ctrl-C in the server terminal:

Expected: `shutting down`, then process exits cleanly.

- [ ] **Step 4: Verify the help text**

```bash
./blog help
```

Expected: both `build` and `serve` usage lines printed.

- [ ] **Step 5: Run all tests one more time**

```bash
go test ./...
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add cmd/blog/main.go
git commit -m "feat(cli): wire serve subcommand into blog command"
```

---

## Done criteria

- `blog serve` starts an HTTP server, prints the URL, serves `public/`.
- Saving a file under `content/`, `templates/`, or `static/` triggers a rebuild within ~200ms and the next HTTP request returns fresh content.
- Build errors during serve print to stderr but do not crash the server.
- Ctrl-C cleanly shuts down within 2s.
- `go test ./...` is green.
- Five commits: dependency add, shouldWatch, debounce, Serve scaffold, watcher wiring, CLI plumbing.
