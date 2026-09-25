package main

import (
	"cmp"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the .out files")

// Each testdata/golden/<case>.json pins stdin, width, clock, branch and
// working directory; <case>.out is the expected output, byte for byte.
func TestGolden(t *testing.T) {
	specs, err := filepath.Glob(filepath.Join("testdata", "golden", "*.json"))
	if err != nil || len(specs) == 0 {
		t.Fatalf("no golden cases found: %v", err)
	}
	for _, path := range specs {
		name := strings.TrimSuffix(filepath.Base(path), ".json")
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var spec struct {
				Stdin   string   `json:"stdin"`
				Cols    int      `json:"cols"`
				Now     float64  `json:"now"`
				Branch  string   `json:"branch"`
				Getcwd  string   `json:"getcwd"`
				NoColor bool     `json:"no_color"`
				Mtime   *float64 `json:"mtime"`
				Clock   string   `json:"clock"`
				Peer    string   `json:"peer"`
			}
			if err := json.Unmarshal(raw, &spec); err != nil {
				t.Fatal(err)
			}
			s, ok := decode([]byte(spec.Stdin))
			e := env{cols: spec.Cols, now: spec.Now, color: !spec.NoColor,
				branch: spec.Branch, peer: spec.Peer, clock: cmp.Or(spec.Clock, "◷")}
			if e.dir = sessionDir(s); e.dir == "" {
				e.dir = spec.Getcwd
			}
			if spec.Mtime != nil {
				e.modTime, e.hasModTime = *spec.Mtime, true
			}
			got := render(s, ok, e)
			out := strings.TrimSuffix(path, ".json") + ".out"
			if *update {
				if err := os.WriteFile(out, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Errorf("output mismatch\n got: %q\nwant: %q", got, want)
			}
		})
	}
}

func TestClockGlyph(t *testing.T) {
	for _, tc := range []struct{ goos, wtSession, want string }{
		{"linux", "", "◷"},
		{"darwin", "x", "◷"},
		{"windows", "", "○"},
		{"windows", "1", "◷"},
	} {
		if got := clockGlyph(tc.goos, tc.wtSession); got != tc.want {
			t.Errorf("clockGlyph(%q, %q) = %q, want %q", tc.goos, tc.wtSession, got, tc.want)
		}
	}
}

func TestBasenameWindowsPath(t *testing.T) {
	for in, want := range map[string]string{
		`C:\Users\me\proj`:  "proj",
		`C:\Users\me\proj\`: "proj",
		"/home/me/proj/":    "proj",
		"/":                 "/",
	} {
		if got := basename(in); got != want {
			t.Errorf("basename(%q) = %q, want %q", in, got, want)
		}
	}
}
