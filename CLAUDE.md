# CLAUDE.md

Guidance for Claude Code working in this repository.

## What this is

A minimal Go static site generator for a personal blog. Markdown posts with
YAML frontmatter become a static HTML site under `public/`.

The full design lives in `docs/superpowers/specs/`. Read the relevant spec
before non-trivial changes.

## CLI

The CLI lives at `cmd/blog/main.go` and compiles to `./blog`.

```bash
go build ./cmd/blog
```

### `blog build` — render the site once

```
blog build [--content DIR] [--templates DIR] [--static DIR] [--out DIR]
```

Reads markdown from `--content` (default `content`), parses templates from
`--templates` (default `templates`), copies assets from `--static` (default
`static`), and writes the site to `--out` (default `public`). Wipes
`--out` at the start of every build.

### `blog serve` — local dev server with hot rebuild

```
blog serve [--addr :8080] [--content DIR] [--templates DIR] [--static DIR] [--out DIR]
```

Runs an initial build, starts an HTTP server on `--addr`, and watches the
three source directories. Any change triggers a rebuild after a 100ms
debounce. The browser is not auto-refreshed — hit Cmd-R/Ctrl-R after a
rebuild to see changes.

Build failures during serve log to stderr but do not crash the server; the
previous good output keeps serving. Ctrl-C performs a graceful shutdown
within 2 seconds.

## Content authoring

Posts live at `content/posts/<slug>.md`. The slug comes from the filename
(lowercased, no `.md`). Frontmatter is YAML and exactly two fields are
recognized:

```markdown
---
title: Hello, world
date: 2026-05-22
---

Post body in markdown.
```

Both fields are required. `date` parses as plain `YYYY-MM-DD` (no
timezone). Missing frontmatter, missing fields, or unparseable dates fail
the build with the offending file path.

Posts sort by date descending; slug-ascending breaks ties. Duplicate slugs
fail the build.

## Layout

```
cmd/blog/        CLI entrypoint
internal/site/   Build pipeline, frontmatter parser, renderer, serve
content/posts/   Source markdown
templates/       html/template files: base.html + post.html + index.html
static/          Files copied verbatim into public/ (optional)
public/          Build output, gitignored
docs/superpowers/specs/   Design specs for each feature
docs/superpowers/plans/   Implementation plans
```

## Tests

```bash
go test ./...
```

Tests live alongside the code in `internal/site/`. The integration tests
spin up real HTTP servers on `127.0.0.1:0` and use `t.TempDir()` fixtures.
