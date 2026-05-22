# Go Static Site Generator — Design

A bare-minimum static site generator written in Go for a single personal blog.
Converts markdown posts with YAML frontmatter into a static HTML site.

## Goals and non-goals

**Goals**

- Convert markdown files in `content/posts/` to HTML pages in `public/`.
- Generate an index page at `/` listing all posts (newest first).
- Copy `static/` assets into the build output verbatim.
- Single `blog build` command. No daemon, no watcher, no live reload.
- Be small enough to read end-to-end and modify without ceremony.

**Non-goals (explicitly deferred until needed)**

- Tags, categories, taxonomies.
- RSS / Atom feeds.
- Pagination.
- Drafts.
- Custom pages beyond posts (no `about.md` etc.).
- Multiple themes or layouts.
- Image processing, sitemap.xml, search index.
- Live reload / file watching.
- Config file (TOML/YAML). Flags + defaults are the only configuration.

## Repository layout

```
blog/
├── cmd/blog/
│   └── main.go              # CLI entry: parse flags, call site.Build
├── internal/site/
│   ├── build.go             # Build orchestrates the whole pipeline
│   ├── post.go              # Post struct + LoadPosts(contentDir)
│   ├── frontmatter.go       # parse YAML frontmatter + body split
│   └── render.go            # markdown → HTML, template execution
├── content/
│   └── posts/
│       └── hello-world.md   # one .md per post
├── templates/
│   ├── base.html            # outer page chrome
│   ├── post.html            # single post body
│   └── index.html           # list of posts
├── static/                  # copied verbatim into public/
│   └── style.css
├── public/                  # build output; gitignored
├── go.mod
├── go.sum
└── README.md
```

URL rules:

- A post at `content/posts/hello-world.md` becomes `public/posts/hello-world/index.html`, served at `/posts/hello-world/`.
- The index lands at `public/index.html` (served at `/`).
- Anything under `static/` is copied to `public/` preserving paths: `static/style.css` → `public/style.css`, referenced as `/style.css`.

## Data model

```go
type Post struct {
    Slug  string        // "hello-world" (from filename without .md, lowercased)
    Title string        // from frontmatter
    Date  time.Time     // from frontmatter (YYYY-MM-DD parsed at midnight UTC)
    Body  template.HTML // rendered markdown (safe HTML)
}
```

**Frontmatter (YAML)** — two required fields, nothing else accepted at this scope:

```yaml
---
title: Hello, world
date: 2026-05-22
---
Post body in markdown goes here.
```

Rules:

- A `.md` file without frontmatter is a hard error (so a post is never accidentally published with no title or date).
- `date` is parsed as plain `YYYY-MM-DD`; no time-of-day, no timezones.
- `Slug` is derived from filename (without `.md`, lowercased). No frontmatter `slug` override.
- Posts are sorted by `Date` descending; ties broken by `Slug` ascending for stable ordering.

## Templates

Three templates using `html/template`'s `{{define}}` / `{{template}}` composition.

`templates/base.html` — page shell every page uses:

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{template "title" .}}</title>
  <link rel="stylesheet" href="/style.css">
</head>
<body>
  <header><a href="/">Home</a></header>
  <main>{{template "body" .}}</main>
</body>
</html>
```

`templates/index.html` — defines `title` and `body` for the index, receives `[]Post`:

```html
{{define "title"}}Blog{{end}}
{{define "body"}}
  <ul>
  {{range .}}
    <li>
      <a href="/posts/{{.Slug}}/">{{.Title}}</a>
      <time datetime="{{.Date.Format "2006-01-02"}}">{{.Date.Format "Jan 2, 2006"}}</time>
    </li>
  {{end}}
  </ul>
{{end}}
```

`templates/post.html` — defines `title` and `body` for a post, receives a single `Post`:

```html
{{define "title"}}{{.Title}}{{end}}
{{define "body"}}
  <article>
    <h1>{{.Title}}</h1>
    <time datetime="{{.Date.Format "2006-01-02"}}">{{.Date.Format "Jan 2, 2006"}}</time>
    {{.Body}}
  </article>
{{end}}
```

Loading: `template.ParseFiles("templates/base.html", "templates/<page>.html")` per page, then `t.ExecuteTemplate(w, "base.html", data)`. `Post.Body` is `template.HTML` so goldmark output is not double-escaped.

## Build pipeline

`site.Build(cfg)` runs these steps in order. Any error aborts the build and is returned.

1. **Wipe `public/`** — remove and recreate the directory so stale files from renamed/deleted posts do not linger.
2. **Walk `content/posts/`** — collect every `*.md` file. Each becomes a `Post` via `LoadPost(path)`:
   - Read file.
   - Split frontmatter from body (must open with `---` at the very top; error if missing).
   - YAML-unmarshal frontmatter; validate `title` (non-empty) and `date` (parses as `YYYY-MM-DD`).
   - Render body markdown to HTML with goldmark.
   - Slug = filename without `.md`, lowercased.
3. **Sort posts** — `Date` descending, `Slug` ascending tiebreak.
4. **Render each post** — for every post, render `base.html`+`post.html` to `public/posts/<slug>/index.html`. `MkdirAll` for the directory.
5. **Render index** — `base.html`+`index.html` with `[]Post` → `public/index.html`.
6. **Copy `static/` → `public/`** — preserve relative paths. If `static/` does not exist, skip silently.
7. **Print summary** — `built N posts in <duration>` to stdout.

Implementation choices:

- **Templates parsed once.** The two `*template.Template` instances (post, index) are parsed before the post loop and reused.
- **In-memory model.** All posts held in a single `[]Post` slice — trivially fine at any realistic personal-blog scale.

Errors that must be explicit:

- Missing frontmatter → error including file path.
- Unparseable `date` → error including file path and the bad string.
- Empty `title` → error including file path.
- Duplicate slugs (two files resolving to the same slug) → error listing both paths.

## CLI

Binary: `cmd/blog/main.go`, ~30 lines, compiled with `go build ./cmd/blog`.

```
blog build [--content ./content] [--templates ./templates] [--static ./static] [--out ./public]
```

Defaults match the repo layout so the normal invocation is just `blog build`.

A single subcommand is used (rather than bare `blog`) so future additions like `serve` or `new` won't break muscle memory or scripts. Implementation is a `switch` on `os.Args[1]` — no CLI framework dependency.

```go
type Config struct {
    ContentDir   string
    TemplatesDir string
    StaticDir    string
    OutDir       string
}
```

`site.Build(cfg)` is the only exported entry point. `main` parses flags into `Config`, calls `Build`, prints any error to stderr, exits non-zero on failure.

## Dependencies

- `github.com/yuin/goldmark` — markdown parser (CommonMark, pure Go, de-facto standard).
- `gopkg.in/yaml.v3` — YAML frontmatter parsing.
- Everything else from the standard library (`html/template`, `path/filepath`, `os`, `time`, `sort`, `io`, `flag`).

## Testing

**Unit tests (table-driven, no filesystem):**

- `internal/site/frontmatter_test.go`
  - valid frontmatter → correct struct + body
  - missing opening `---` → error
  - missing closing `---` → error
  - empty / missing `title` → error
  - missing or unparseable `date` → error
- `internal/site/post_test.go`
  - slug derived from filename (`Hello-World.md` → `hello-world`)
  - sort order: date desc with slug-asc tiebreak

**Integration test (one, using `t.TempDir()`):**

- `internal/site/build_test.go`
  - Lay out a tiny fixture in a temp dir: 2 posts + 1 static file + the real templates copied in.
  - Run `Build(cfg)`.
  - Assert: `public/index.html` exists and contains both post titles; `public/posts/<slug>/index.html` exists for each post; `public/style.css` matches the source byte-for-byte.
  - Separate fixture for duplicate-slug failure: two files sharing a slug; expect `Build` to return an error mentioning both paths.

**Not tested:**

- goldmark's markdown → HTML output (well-tested upstream).
- Exact template HTML byte-for-byte (brittle).
- `main()` flag wiring (trivial).
