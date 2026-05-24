package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

var selectorRE = regexp.MustCompile(`\.chroma (\.[a-z0-9]+) \{`)

func selectors(css string) map[string]bool {
	out := map[string]bool{}
	for _, m := range selectorRE.FindAllStringSubmatch(css, -1) {
		out[m[1]] = true
	}
	return out
}

// genhl prints the chroma class-based stylesheet for the light theme at the
// top level and the dark theme inside a prefers-color-scheme media query.
//
// Both themes share the same class names, so a class that the dark theme omits
// (because its colour matches the dark base text colour) would otherwise
// inherit the light theme's dark-on-light value in dark mode and render nearly
// invisible. We emit an explicit base-colour override for each such class.
func main() {
	f := chromahtml.New(chromahtml.WithClasses(true))
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	var light, dark strings.Builder
	if err := f.WriteCSS(&light, styles.Get("github")); err != nil {
		panic(err)
	}
	if err := f.WriteCSS(&dark, styles.Get("github-dark")); err != nil {
		panic(err)
	}

	darkBase := styles.Get("github-dark").Get(chroma.Background).Colour.String()
	lightSel, darkSel := selectors(light.String()), selectors(dark.String())
	var missing []string
	for s := range lightSel {
		if !darkSel[s] {
			missing = append(missing, s)
		}
	}
	sort.Strings(missing)

	fmt.Fprintln(w, "/* Syntax highlighting (chroma, github / github-dark). Regenerate with: go run ./cmd/genhl */")
	fmt.Fprint(w, light.String())

	fmt.Fprintln(w, "@media (prefers-color-scheme: dark) {")
	for _, line := range strings.Split(strings.TrimRight(dark.String(), "\n"), "\n") {
		fmt.Fprintf(w, "  %s\n", line)
	}
	for _, s := range missing {
		fmt.Fprintf(w, "  .chroma %s { color: %s }\n", s, darkBase)
	}
	fmt.Fprintln(w, "}")
}
