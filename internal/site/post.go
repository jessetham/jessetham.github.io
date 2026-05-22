package site

import (
	"html/template"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Post struct {
	Slug  string
	Title string
	Date  time.Time
	Body  template.HTML
}

func slugFromPath(p string) string {
	base := filepath.Base(p)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return strings.ToLower(name)
}

func sortPosts(posts []Post) {
	sort.Slice(posts, func(i, j int) bool {
		if !posts[i].Date.Equal(posts[j].Date) {
			return posts[i].Date.After(posts[j].Date)
		}
		return posts[i].Slug < posts[j].Slug
	})
}
