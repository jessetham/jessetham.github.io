package site

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// splitFrontmatter splits raw file bytes into (frontmatter, body).
// The file must start with "---\n" and contain a closing "\n---" line.
// A trailing newline after the closing fence is consumed.
func splitFrontmatter(data []byte) (fm, body []byte, err error) {
	const opener = "---\n"
	if !bytes.HasPrefix(data, []byte(opener)) {
		return nil, nil, errors.New("missing opening frontmatter delimiter (--- on the first line)")
	}
	rest := data[len(opener):]
	idx := bytes.Index(rest, []byte("\n---"))
	if idx == -1 {
		return nil, nil, errors.New("missing closing frontmatter delimiter (---)")
	}
	fm = rest[:idx+1] // include the trailing newline so YAML sees a clean doc
	body = rest[idx+len("\n---"):]
	body = bytes.TrimPrefix(body, []byte("\n"))
	return fm, body, nil
}

type rawFrontmatter struct {
	Title string    `yaml:"title"`
	Date  time.Time `yaml:"date"`
}

func parseFrontmatter(fm []byte) (title string, date time.Time, err error) {
	var raw rawFrontmatter
	if err := yaml.Unmarshal(fm, &raw); err != nil {
		// yaml.v3 reports unparseable timestamps here.
		// When a timestamp is unparseable, include "date" in the error.
		return "", time.Time{}, fmt.Errorf("invalid frontmatter date: %w", err)
	}
	if raw.Title == "" {
		return "", time.Time{}, fmt.Errorf("frontmatter: title is required")
	}
	if raw.Date.IsZero() {
		return "", time.Time{}, fmt.Errorf("frontmatter: date is required (YYYY-MM-DD)")
	}
	return raw.Title, raw.Date, nil
}
