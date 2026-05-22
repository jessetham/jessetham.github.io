package site

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	ContentDir   string
	TemplatesDir string
	StaticDir    string
	OutDir       string
}

func copyStatic(src, dst string) error {
	if _, err := os.Stat(src); errors.Is(err, fs.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

func Build(cfg Config) error {
	start := time.Now()

	if err := os.RemoveAll(cfg.OutDir); err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.OutDir, 0755); err != nil {
		return err
	}

	posts, err := loadPosts(filepath.Join(cfg.ContentDir, "posts"))
	if err != nil {
		return err
	}

	postTmpl, indexTmpl, err := parseTemplates(cfg.TemplatesDir)
	if err != nil {
		return err
	}

	for _, p := range posts {
		dir := filepath.Join(cfg.OutDir, "posts", p.Slug)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		f, err := os.Create(filepath.Join(dir, "index.html"))
		if err != nil {
			return err
		}
		if err := renderPost(f, postTmpl, p); err != nil {
			f.Close()
			return fmt.Errorf("render post %s: %w", p.Slug, err)
		}
		if err := f.Close(); err != nil {
			return err
		}
	}

	idx, err := os.Create(filepath.Join(cfg.OutDir, "index.html"))
	if err != nil {
		return err
	}
	if err := renderIndex(idx, indexTmpl, posts); err != nil {
		idx.Close()
		return fmt.Errorf("render index: %w", err)
	}
	if err := idx.Close(); err != nil {
		return err
	}

	if err := copyStatic(cfg.StaticDir, cfg.OutDir); err != nil {
		return err
	}

	fmt.Printf("built %d posts in %s\n", len(posts), time.Since(start).Round(time.Millisecond))
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
