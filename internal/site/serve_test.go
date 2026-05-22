package site

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

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

func TestDebounce_SingleEventFiresOnce(t *testing.T) {
	in := make(chan fsnotify.Event, 1)
	var fired atomic.Int32
	done := make(chan struct{})
	go func() {
		debounce(in, 20*time.Millisecond, func() { fired.Add(1) })
		close(done)
	}()

	in <- fsnotify.Event{}
	time.Sleep(80 * time.Millisecond)
	close(in)
	<-done

	if got := fired.Load(); got != 1 {
		t.Errorf("fired = %d, want 1", got)
	}
}

func TestDebounce_BurstFiresOnce(t *testing.T) {
	in := make(chan fsnotify.Event, 16)
	var fired atomic.Int32
	done := make(chan struct{})
	go func() {
		debounce(in, 30*time.Millisecond, func() { fired.Add(1) })
		close(done)
	}()

	for range 5 {
		in <- fsnotify.Event{}
		time.Sleep(2 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	close(in)
	<-done

	if got := fired.Load(); got != 1 {
		t.Errorf("fired = %d, want 1", got)
	}
}

func TestDebounce_SeparateBurstsFireTwice(t *testing.T) {
	in := make(chan fsnotify.Event, 16)
	var fired atomic.Int32
	done := make(chan struct{})
	go func() {
		debounce(in, 20*time.Millisecond, func() { fired.Add(1) })
		close(done)
	}()

	in <- fsnotify.Event{}
	time.Sleep(80 * time.Millisecond)
	in <- fsnotify.Event{}
	time.Sleep(80 * time.Millisecond)
	close(in)
	<-done

	if got := fired.Load(); got != 2 {
		t.Errorf("fired = %d, want 2", got)
	}
}

func TestDebounce_CloseReturns(t *testing.T) {
	in := make(chan fsnotify.Event)
	done := make(chan struct{})
	go func() {
		debounce(in, time.Second, func() {})
		close(done)
	}()
	close(in)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("debounce did not return after channel close")
	}
}
