package site

import "testing"

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
