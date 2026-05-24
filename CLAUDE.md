# blog

This repo contains everything needed to power my personal blog: the
content itself and a minimal Go static site generator that converts
markdown posts with YAML frontmatter into a static HTML site under
`public/`.

This file is the single source of context for both humans and agents
working on the repo. Design specs for each feature live in
`docs/superpowers/specs/`; read the relevant spec before non-trivial
changes.

## CLI

The CLI lives at `cmd/blog/main.go` and compiles to `./blog`.

```bash
go build ./cmd/blog
```

### `blog build` — render the site once

```
blog build [--content DIR] [--templates DIR] [--static DIR] [--out DIR] [--base-url URL]
```

Reads markdown from `--content` (default `content`), parses templates from
`--templates` (default `templates`), copies assets from `--static` (default
`static`), and writes the site to `--out` (default `public`). Wipes
`--out` at the start of every build. `--base-url` (default
`https://jtham.dev`) is the absolute origin used to build canonical links,
the sitemap, and the feed; a trailing slash is trimmed.

### `blog serve` — local dev server with hot rebuild

```
blog serve [--addr :8080] [--content DIR] [--templates DIR] [--static DIR] [--out DIR] [--base-url URL]
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

## Syntax highlighting

Fenced code blocks with a language hint (` ```go `) are highlighted at
build time by the `goldmark-highlighting` (chroma) extension. Output is
class-based — no inline styles, no client-side JS. Indented code blocks and
inline `code` are left untouched and keep the plain `--code-bg` styling.

The colours live in `static/highlight.css`, a generated file holding the
chroma `github` and `github-dark` themes, each wrapped in its own
`prefers-color-scheme` media query. The themes share class names, so they
must be scoped symmetrically — leaving either at the top level lets the
classes one theme omits inherit the other's colours. Regenerate it (e.g. to
change themes) with:

```bash
go run ./cmd/genhl > static/highlight.css
```

Edit the style names in `cmd/genhl/main.go` to pick different themes; run
`go run ./cmd/genhl` with no redirect to preview.

## SEO

Every page renders a full `<head>`: a `<title>` (`<post> — Jesse Tham` on
posts, `Jesse Tham` on the index), `<meta name="description">`, author,
canonical link, Open Graph and Twitter Card tags, and JSON-LD
(`BlogPosting` on posts, `WebSite` on the index). Descriptions are
auto-derived from the first paragraph of the rendered body, collapsed to
one line and truncated to ~160 characters — there is no `description`
frontmatter field.

Each build also writes three crawlability artifacts to the output root:
`sitemap.xml` (index + every post, with `lastmod`), `robots.txt` (points
crawlers at the sitemap), and `feed.xml` (an RSS 2.0 feed of all posts).
All absolute URLs in these come from `--base-url`. The SEO helpers live in
`internal/site/seo.go`; the shared per-page view model is `pageData` in
`internal/site/render.go`.

## Layout

```
cmd/blog/        CLI entrypoint
cmd/genhl/       Regenerates static/highlight.css (dev tool, run manually)
internal/site/   Build pipeline, frontmatter parser, renderer, serve
content/posts/   Source markdown
templates/       html/template files: base.html + post.html + index.html
static/          Files copied verbatim into public/ (style.css, highlight.css, fonts)
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
