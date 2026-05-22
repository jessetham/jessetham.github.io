package site

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// Serve starts an HTTP server bound to addr that serves cfg.OutDir, runs an
// initial Build, and rebuilds on file changes under cfg.ContentDir,
// cfg.TemplatesDir, and cfg.StaticDir. Blocks until ctx is cancelled or a
// fatal error occurs.
func Serve(ctx context.Context, cfg Config, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return serveOnListener(ctx, cfg, listener)
}

func serveOnListener(ctx context.Context, cfg Config, listener net.Listener) error {
	if err := Build(cfg); err != nil {
		return fmt.Errorf("initial build: %w", err)
	}

	var mu sync.RWMutex
	fileSrv := http.FileServer(http.Dir(cfg.OutDir))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		fileSrv.ServeHTTP(w, r)
	})

	srv := &http.Server{Handler: handler}

	fmt.Printf("serving %s at http://%s  (watching %s, %s, %s)\n",
		cfg.OutDir, listener.Addr(), cfg.ContentDir, cfg.TemplatesDir, cfg.StaticDir)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stdout, "shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-serveErr:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
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
