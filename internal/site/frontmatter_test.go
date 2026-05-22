package site

import (
	"strings"
	"testing"
)

func TestSplitFrontmatter(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		wantFM   string
		wantBody string
		wantErr  string // substring; "" means no error
	}{
		{
			name:     "happy path",
			in:       "---\ntitle: Hello\ndate: 2026-05-22\n---\nBody text.\n",
			wantFM:   "title: Hello\ndate: 2026-05-22\n",
			wantBody: "Body text.\n",
		},
		{
			name:     "empty body after closing fence",
			in:       "---\ntitle: x\ndate: 2026-05-22\n---\n",
			wantFM:   "title: x\ndate: 2026-05-22\n",
			wantBody: "",
		},
		{
			name:    "missing opening fence",
			in:      "title: Hello\ndate: 2026-05-22\nBody text.\n",
			wantErr: "opening",
		},
		{
			name:    "missing closing fence",
			in:      "---\ntitle: Hello\ndate: 2026-05-22\nBody text.\n",
			wantErr: "closing",
		},
		{
			name:    "empty file",
			in:      "",
			wantErr: "opening",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fm, body, err := splitFrontmatter([]byte(tc.in))
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("want error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(fm) != tc.wantFM {
				t.Errorf("fm: got %q want %q", fm, tc.wantFM)
			}
			if string(body) != tc.wantBody {
				t.Errorf("body: got %q want %q", body, tc.wantBody)
			}
		})
	}
}

func TestParseFrontmatter(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		wantTitle string
		wantDate  string // YYYY-MM-DD; "" if error expected
		wantErr   string // substring; "" means no error
	}{
		{
			name:      "happy path",
			in:        "title: Hello, world\ndate: 2026-05-22\n",
			wantTitle: "Hello, world",
			wantDate:  "2026-05-22",
		},
		{
			name:    "missing title",
			in:      "date: 2026-05-22\n",
			wantErr: "title",
		},
		{
			name:    "empty title",
			in:      "title: \"\"\ndate: 2026-05-22\n",
			wantErr: "title",
		},
		{
			name:    "missing date",
			in:      "title: Hello\n",
			wantErr: "date",
		},
		{
			name:    "unparseable date",
			in:      "title: Hello\ndate: yesterday\n",
			wantErr: "date",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			title, date, err := parseFrontmatter([]byte(tc.in))
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("want error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if title != tc.wantTitle {
				t.Errorf("title: got %q want %q", title, tc.wantTitle)
			}
			if got := date.Format("2006-01-02"); got != tc.wantDate {
				t.Errorf("date: got %q want %q", got, tc.wantDate)
			}
		})
	}
}
