package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

// genhl prints the chroma class-based stylesheet for the light theme at the
// top level and the dark theme inside a prefers-color-scheme media query.
func main() {
	f := chromahtml.New(chromahtml.WithClasses(true))
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	fmt.Fprintln(w, "/* Syntax highlighting (chroma, github / github-dark). Regenerate with: go run ./cmd/genhl */")
	if err := f.WriteCSS(w, styles.Get("github")); err != nil {
		panic(err)
	}

	var dark strings.Builder
	if err := f.WriteCSS(&dark, styles.Get("github-dark")); err != nil {
		panic(err)
	}
	fmt.Fprintln(w, "@media (prefers-color-scheme: dark) {")
	for _, line := range strings.Split(strings.TrimRight(dark.String(), "\n"), "\n") {
		fmt.Fprintf(w, "  %s\n", line)
	}
	fmt.Fprintln(w, "}")
}
