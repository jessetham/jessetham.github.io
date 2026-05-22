# Serve subcommand — Design

A `blog serve` subcommand that runs a localhost HTTP server, watches the
source directories, and rebuilds the site on every change. The browser is
manually refreshed by the user; this is not a browser-side live-reload
system.

## Goals and non-goals

**Goals**

- One command (`blog serve`) starts an HTTP server bound to a local address
  and serves the built site from `public/`.
- Edits to files under `content/`, `templates/`, or `static/` trigger an
  automatic rebuild so the next request returns fresh output.
- Build errors during serve do not crash the server; the previous good
  output stays live while errors print to stderr.
- Reuse the existing `site.Build(cfg)` pipeline verbatim — no duplication of
  the build logic.
- Ctrl-C cleanly shuts down the watcher and the HTTP server.

**Non-goals (explicitly deferred until needed)**

- Browser auto-refresh / live reload (no JS injection, no SSE/WebSocket).
- Hot module replacement, partial rebuilds, or CSS-only swaps.
- Auto-opening the browser on startup.
- HTTPS, custom domains, or any production-server concerns.
- Build-output diffing or incremental rebuild detection.
- Pretty error pages in the browser; errors go to terminal stderr.

## CLI

```
blog serve [--addr :8080] [--content DIR] [--templates DIR] [--static DIR] [--out DIR]
```

The four directory flags are identical to `blog build` and have the same
defaults (`content`, `templates`, `static`, `public`). Only new flag is
`--addr`, defaulting to `:8080`. Standard Go `net.Listen` address syntax
(`:8080`, `127.0.0.1:3000`, `[::1]:8080`).

Startup output to stdout:

```
serving public/ at http://localhost:8080  (watching content/, templates/, static/)
```

Ctrl-C output to stdout:

```
shutting down
```

## Architecture

One new package file (`internal/site/serve.go`) and a `runServe` function
in `cmd/blog/main.go` mirroring `runBuild`.

```
cmd/blog/main.go
  runServe(args)  →  parse flags into site.Config + addr  →  site.Serve(ctx, cfg, addr)

internal/site/serve.go
  Serve(ctx context.Context, cfg Config, addr string) error
    ├── initial Build(cfg)
    ├── start fsnotify watcher (goroutine)
    ├── start http.Server (goroutine)
    └── block on ctx.Done(), then shutdown both
```

Inside `serve.go`:

- **`fileServer`** — `http.FileServer(http.Dir(cfg.OutDir))` wrapped to take
  a read lock on a shared `sync.RWMutex` before serving each request.
- **`watcher`** — an `*fsnotify.Watcher` registered against
  `cfg.ContentDir`, `cfg.TemplatesDir`, `cfg.StaticDir`. fsnotify is
  non-recursive by default, so on startup the watcher walks each tree and
  calls `Add` on every directory. New subdirectories created at runtime are
  added on the fly when their parent emits a `Create` event for a dir.
- **`rebuilder`** — a goroutine that drains watcher events, debounces
  bursts with a 100ms timer, then takes the write lock and runs
  `site.Build(cfg)`. Build errors are logged to stderr; the previous
  successful `public/` stays observable to readers because the write lock
  is held for the entire build.

The mutex is the only shared state between the HTTP path and the rebuild
path. No channels of build results, no event types.

## Data flow

Steady-state loop:

```
1. User edits content/posts/hello.md
2. fsnotify emits a Write event on that path
3. rebuilder receives event → resets 100ms debounce timer
4. (more events may arrive — timer keeps resetting)
5. Timer fires → rebuilder takes write lock
6. site.Build(cfg) runs: wipe public/, render, copy static.
   Build's existing "built N posts in <duration>" log line is reused;
   no new print added.
7. Write lock released
8. User hits Cmd-R; request takes read lock, http.FileServer serves fresh file
```

Concurrent-request handling:

- N readers can hold the read lock simultaneously.
- A pending build waits for in-flight requests to finish, then blocks new
  requests for the build duration (~50ms on the current fixture).
- Requests arriving during a build queue on the lock and succeed against
  the fresh `public/` once the build releases.

## Debounce and event filtering

- A single `*time.Timer` is reset on every event. Saves that span multiple
  syscalls (vim's `:w` does `write` → `rename`) collapse into one rebuild.
- Events on the `cfg.OutDir` directory (and any path beneath it) are
  filtered out before they reach the timer. This is mostly defensive —
  `OutDir` is never added to the watcher — but documents the invariant.
- Hidden files (any path component starting with `.`) are filtered.
- A pure function `shouldWatch(path string, outDir string) bool` encodes
  these rules so it can be unit-tested.

## Error handling

Build errors during serve are explicitly not fatal:

```
1. site.Build(cfg) returns an error from rebuilder goroutine
2. rebuilder prints "rebuild failed: <err>" to stderr
3. The write lock is released
4. Server keeps serving whatever state Build left in public/
```

The wipe-then-recreate window inside `Build` is handled by holding the
write lock through the entire build. Requests blocked on the read lock
during a failing build see the post-failure state of `public/` (possibly
incomplete) once they unblock. In practice the user fixes the source and
saves again, which triggers a fresh build and repopulates `public/`.

Fatal errors (server can't bind to addr, watcher can't initialize): print
to stderr and exit non-zero. Same shape as `runBuild` today.

Watcher errors (`fsnotify.Watcher.Errors` channel): log to stderr, keep
going. A transient error shouldn't kill the dev server.

Signals: `runServe` in `cmd/blog/main.go` builds the context via
`signal.NotifyContext(context.Background(), os.Interrupt)` and passes it to
`site.Serve`, so Ctrl-C cancels the context Serve is blocked on. The
watcher goroutine returns when its channel closes or ctx is done; the HTTP
goroutine returns after `http.Server.Shutdown(shutdownCtx)` with a 2s
grace timeout (a fresh `context.WithTimeout`, not the cancelled parent).

## Dependencies

- `github.com/fsnotify/fsnotify` — file-system event library. Added to
  `go.mod`. Only new dependency.
- Everything else is stdlib (`net/http`, `context`, `sync`, `time`,
  `os/signal`, `path/filepath`).

## Testing

**Unit tests (`internal/site/serve_test.go`):**

- **Debouncer** — extract the debounce loop as
  `debounce(events <-chan fsnotify.Event, wait time.Duration, fire func())`
  or similar, and test:
  - A single event triggers exactly one `fire` after `wait`.
  - A burst of 5 events within `wait` triggers exactly one `fire`.
  - Two bursts separated by more than `wait` trigger two `fire`s.
  - Use short waits (~10ms) — no real `time.Sleep` chains beyond that.

- **Path filter** — pure function `shouldWatch(path, outDir string) bool`,
  table-driven:
  - Hidden file (`.DS_Store`, `content/.draft.md`) → false.
  - Path inside `outDir` → false.
  - Normal path → true.

**Integration test (`internal/site/serve_test.go`, uses `t.TempDir()`):**

One end-to-end test that exercises the full loop:

1. Lay out a tiny fixture in a temp dir: 1 post, the real templates, 1
   static file.
2. Call `site.Serve(ctx, cfg, "127.0.0.1:0")` in a goroutine; capture the
   actual bound port via the listener.
3. HTTP GET `/posts/<slug>/` — assert the original title appears in the
   response body.
4. Rewrite the markdown file with a new title.
5. Poll-with-timeout (max 2s) HTTP GET until the new title appears.
6. Cancel the context; assert `Serve` returns within 2s.

**Not tested:**

- `runServe` flag wiring in `cmd/blog/main.go` (trivial, mirrors
  `runBuild`).
- fsnotify behavior itself (well-tested upstream).
- Real OS signal handling.

## File-by-file change list

- `cmd/blog/main.go` — add `case "serve"` in the switch, add `runServe`
  function, update `usage()` to mention serve.
- `internal/site/serve.go` — new file: `Serve(ctx, cfg, addr)`,
  `shouldWatch(path, outDir)`, `debounce(events, wait, fire)`,
  lock-wrapping HTTP handler.
- `internal/site/serve_test.go` — new file: unit tests for `shouldWatch`
  and `debounce`, integration test for `Serve`.
- `go.mod` / `go.sum` — `github.com/fsnotify/fsnotify` added as a direct
  dependency.
