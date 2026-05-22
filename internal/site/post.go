package site

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
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

func loadPost(path string) (Post, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Post{}, err
	}
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		return Post{}, err
	}
	title, date, err := parseFrontmatter(fm)
	if err != nil {
		return Post{}, err
	}
	html, err := markdownToHTML(body)
	if err != nil {
		return Post{}, err
	}
	return Post{
		Slug:  slugFromPath(path),
		Title: title,
		Date:  date,
		Body:  template.HTML(html),
	}, nil
}

func loadPosts(dir string) ([]Post, error) {
	var posts []Post
	seen := map[string]string{} // slug -> path of first occurrence
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		p, err := loadPost(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if prev, ok := seen[p.Slug]; ok {
			return fmt.Errorf("duplicate slug %q: %s and %s", p.Slug, prev, path)
		}
		seen[p.Slug] = path
		posts = append(posts, p)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sortPosts(posts)
	return posts, nil
}
