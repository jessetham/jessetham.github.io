package main

import (
	"bufio"
	"fmt"
	"os"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

// genhl prints the chroma class-based stylesheet for the github-dark theme.
// The site is always dark, so a single unscoped theme is all that's needed.
func main() {
	f := chromahtml.New(chromahtml.WithClasses(true))
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	fmt.Fprintln(w, "/* Syntax highlighting (chroma, github-dark). Regenerate with: go run ./cmd/genhl */")
	if err := f.WriteCSS(w, styles.Get("github-dark")); err != nil {
		panic(err)
	}
}
