package main

import "testing"

func TestColsFromEnv(t *testing.T) {
	for value, want := range map[string]struct {
		cols int
		ok   bool
	}{
		"":       {0, false},
		"  ":     {0, false},
		"abc":    {0, false},
		"-5":     {0, false},
		"0":      {0, false}, // zero is not a width; keep probing
		"80":     {80, true},
		" 120 ":  {120, true},
		"999999": {maxCols, true}, // capped, or the line grows to a megabyte
	} {
		t.Run(value, func(t *testing.T) {
			cols, ok := colsFromEnv(value)
			if cols != want.cols || ok != want.ok {
				t.Errorf("colsFromEnv(%q) = (%d, %v), want (%d, %v)",
					value, cols, ok, want.cols, want.ok)
			}
		})
	}
}
