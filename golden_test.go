package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
				Stdin   string  `json:"stdin"`
				Cols    int     `json:"cols"`
				Now     float64 `json:"now"`
				Branch  *string `json:"branch"`
				Getcwd  string  `json:"getcwd"`
				NoColor bool    `json:"no_color"`
			}
			if err := json.Unmarshal(raw, &spec); err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(strings.TrimSuffix(path, ".json") + ".out")
			if err != nil {
				t.Fatal(err)
			}
			got := render([]byte(spec.Stdin), env{
				cols:   spec.Cols,
				now:    spec.Now,
				color:  !spec.NoColor,
				getcwd: func() string { return spec.Getcwd },
				branchOf: func(string) string {
					if spec.Branch == nil {
						return ""
					}
					return *spec.Branch
				},
			})
			if got != string(want) {
				t.Errorf("output mismatch\n got: %q\nwant: %q", got, want)
			}
		})
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
