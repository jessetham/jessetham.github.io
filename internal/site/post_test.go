package site

import (
	"strings"
	"testing"
	"time"
)

func TestSlugFromPath(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"content/posts/hello-world.md", "hello-world"},
		{"content/posts/Hello-World.md", "hello-world"},
		{"hello.md", "hello"},
		{"/abs/path/2026-05-22-Post.md", "2026-05-22-post"},
		{"content/posts/nested/dir/HelloAgain.md", "helloagain"},
	}
	for _, tc := range cases {
		got := slugFromPath(tc.in)
		if got != tc.want {
			t.Errorf("slugFromPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSortPosts(t *testing.T) {
	date := func(s string) time.Time {
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatalf("bad date in test: %v", err)
		}
		return d
	}
	posts := []Post{
		{Slug: "older", Date: date("2026-01-01")},
		{Slug: "tied-b", Date: date("2026-05-22")},
		{Slug: "newer", Date: date("2026-06-01")},
		{Slug: "tied-a", Date: date("2026-05-22")},
	}
	sortPosts(posts)
	got := make([]string, len(posts))
	for i, p := range posts {
		got[i] = p.Slug
	}
	want := []string{"newer", "tied-a", "tied-b", "older"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("sort order: got %v want %v", got, want)
	}
}
