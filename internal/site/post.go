package site

import (
	"path/filepath"
	"strings"
)

func slugFromPath(p string) string {
	base := filepath.Base(p)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return strings.ToLower(name)
}
