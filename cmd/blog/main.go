package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"blog/internal/site"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "build":
		if err := runBuild(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "serve":
		if err := runServe(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func runBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	cfg := site.Config{}
	fs.StringVar(&cfg.ContentDir, "content", "content", "content directory")
	fs.StringVar(&cfg.TemplatesDir, "templates", "templates", "templates directory")
	fs.StringVar(&cfg.StaticDir, "static", "static", "static assets directory")
	fs.StringVar(&cfg.OutDir, "out", "public", "build output directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return site.Build(cfg)
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	cfg := site.Config{}
	var addr string
	fs.StringVar(&cfg.ContentDir, "content", "content", "content directory")
	fs.StringVar(&cfg.TemplatesDir, "templates", "templates", "templates directory")
	fs.StringVar(&cfg.StaticDir, "static", "static", "static assets directory")
	fs.StringVar(&cfg.OutDir, "out", "public", "build output directory")
	fs.StringVar(&addr, "addr", ":8080", "HTTP listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return site.Serve(ctx, cfg, addr)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  blog build [--content DIR] [--templates DIR] [--static DIR] [--out DIR]")
	fmt.Fprintln(os.Stderr, "  blog serve [--addr :8080] [--content DIR] [--templates DIR] [--static DIR] [--out DIR]")
}
