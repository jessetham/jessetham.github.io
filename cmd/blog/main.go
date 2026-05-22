package main

import (
	"flag"
	"fmt"
	"os"

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

func usage() {
	fmt.Fprintln(os.Stderr, "usage: blog build [--content DIR] [--templates DIR] [--static DIR] [--out DIR]")
}
