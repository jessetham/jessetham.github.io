package site

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
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

// debounce reads from in and calls fire after wait has passed since the
// most recent event. Returns when in is closed.
func debounce(in <-chan fsnotify.Event, wait time.Duration, fire func()) {
	var timer *time.Timer
	var timerCh <-chan time.Time
	for {
		select {
		case _, ok := <-in:
			if !ok {
				if timer != nil {
					timer.Stop()
				}
				return
			}
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(wait)
			timerCh = timer.C
		case <-timerCh:
			timerCh = nil
			fire()
		}
	}
}
