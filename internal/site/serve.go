package site

import (
	"path/filepath"
	"strings"
)

// shouldWatch returns true if path should trigger a rebuild.
// Hidden files (any component starting with '.') and paths inside outDir
// are excluded.
func shouldWatch(path, outDir string) bool {
	clean := filepath.Clean(path)
	out := filepath.Clean(outDir)
	sep := string(filepath.Separator)
	if clean == out || strings.HasPrefix(clean, out+sep) {
		return false
	}
	for part := range strings.SplitSeq(clean, sep) {
		if part != "" && part != "." && strings.HasPrefix(part, ".") {
			return false
		}
	}
	return true
}
