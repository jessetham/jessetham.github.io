package site

import (
	"encoding/json"
	"encoding/xml"
	"html"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	defaultBaseURL    = "https://jtham.dev"
	siteTitle         = "Jesse Tham"
	siteAuthor        = "Jesse Tham"
	siteDescription   = "Writing by Jesse Tham."
	descriptionMaxLen = 160
)

var (
	paragraphRe = regexp.MustCompile(`(?is)<p[^>]*>(.*?)</p>`)
	tagRe       = regexp.MustCompile(`<[^>]*>`)
	whitespace  = regexp.MustCompile(`\s+`)
)

// normalizeBaseURL trims a trailing slash and falls back to the default site
// URL when empty, so callers can always join paths with a single "/".
func normalizeBaseURL(u string) string {
	if u == "" {
		u = defaultBaseURL
	}
	return strings.TrimRight(u, "/")
}

// deriveDescription extracts a plain-text summary from rendered HTML, using the
// first paragraph collapsed onto one line and truncated to descriptionMaxLen.
func deriveDescription(body template.HTML) string {
	s := string(body)
	if m := paragraphRe.FindStringSubmatch(s); m != nil {
		s = m[1]
	}
	s = tagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.TrimSpace(whitespace.ReplaceAllString(s, " "))
	return truncate(s, descriptionMaxLen)
}

// truncate shortens s to at most max runes, breaking on a word boundary and
// appending an ellipsis when it cuts.
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	cut := string(r[:max])
	if i := strings.LastIndex(cut, " "); i > 0 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ") + "…"
}

// jsonLDScript marshals m as JSON-LD and wraps it in a script element. The
// json package escapes <, >, and & to \u00xx, so the result is safe to emit as
// verbatim HTML without a </script> breakout.
func jsonLDScript(m map[string]any) template.HTML {
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return template.HTML(`<script type="application/ld+json">` + string(b) + `</script>`)
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

func writeSitemap(cfg Config, posts []Post) error {
	set := sitemapURLSet{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	set.URLs = append(set.URLs, sitemapURL{Loc: cfg.BaseURL + "/", LastMod: latestDate(posts)})
	for _, p := range posts {
		set.URLs = append(set.URLs, sitemapURL{
			Loc:     postURL(cfg, p),
			LastMod: p.Date.Format("2006-01-02"),
		})
	}
	return writeXML(filepath.Join(cfg.OutDir, "sitemap.xml"), set)
}

func writeRobots(cfg Config) error {
	body := "User-agent: *\nAllow: /\n\nSitemap: " + cfg.BaseURL + "/sitemap.xml\n"
	return os.WriteFile(filepath.Join(cfg.OutDir, "robots.txt"), []byte(body), 0644)
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []rssItem `xml:"item"`
}

type rssRoot struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

func writeFeed(cfg Config, posts []Post) error {
	feed := rssRoot{
		Version: "2.0",
		Channel: rssChannel{
			Title:       siteTitle,
			Link:        cfg.BaseURL + "/",
			Description: siteDescription,
		},
	}
	for _, p := range posts {
		url := postURL(cfg, p)
		feed.Channel.Items = append(feed.Channel.Items, rssItem{
			Title:       p.Title,
			Link:        url,
			GUID:        url,
			PubDate:     p.Date.Format("Mon, 02 Jan 2006 15:04:05 -0700"),
			Description: p.Description,
		})
	}
	return writeXML(filepath.Join(cfg.OutDir, "feed.xml"), feed)
}

func postURL(cfg Config, p Post) string {
	return cfg.BaseURL + "/posts/" + p.Slug + "/"
}

// latestDate returns the most recent post date (posts are sorted descending),
// formatted for sitemap lastmod. Empty when there are no posts.
func latestDate(posts []Post) string {
	if len(posts) == 0 {
		return ""
	}
	return posts[0].Date.Format("2006-01-02")
}

func writeXML(path string, v any) error {
	out, err := xml.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append([]byte(xml.Header), append(out, '\n')...), 0644)
}
