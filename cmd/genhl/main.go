package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

// genhl prints the chroma class-based stylesheet, with the github and
// github-dark themes each wrapped in its own prefers-color-scheme media query.
//
// Both themes share the same class names, so they must be scoped symmetrically:
// if one were left at the top level, the classes it defines but the other omits
// (each theme drops tokens whose colour matches its base text) would leak across
// the boundary and render with the wrong theme's colour.
func main() {
	f := chromahtml.New(chromahtml.WithClasses(true))
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	fmt.Fprintln(w, "/* Syntax highlighting (chroma, github / github-dark). Regenerate with: go run ./cmd/genhl */")
	writeScoped(w, f, "light", "github")
	writeScoped(w, f, "dark", "github-dark")
}

// writeScoped writes the named chroma style inside a prefers-color-scheme media
// query, indenting each rule for readability.
func writeScoped(w *bufio.Writer, f *chromahtml.Formatter, scheme, style string) {
	var css strings.Builder
	if err := f.WriteCSS(&css, styles.Get(style)); err != nil {
		panic(err)
	}
	fmt.Fprintf(w, "@media (prefers-color-scheme: %s) {\n", scheme)
	for _, line := range strings.Split(strings.TrimRight(css.String(), "\n"), "\n") {
		fmt.Fprintf(w, "  %s\n", line)
	}
	fmt.Fprintln(w, "}")
}
