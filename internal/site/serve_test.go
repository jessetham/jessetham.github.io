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
