package main

import (
	"runtime/debug"
	"testing"
)

func TestFormatVersion(t *testing.T) {
	full := &debug.BuildInfo{
		Main: debug.Module{Version: "v0.2.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "9ac8b38e410c20078b1d160ffc408333d7d0c5d5"},
		},
	}
	if got, want := formatVersion(full, true), "atazs-armatur v0.2.0 (9ac8b38)"; got != want {
		t.Errorf("formatVersion(full) = %q, want %q", got, want)
	}

	bare := &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}
	if got, want := formatVersion(bare, true), "atazs-armatur (devel)"; got != want {
		t.Errorf("formatVersion(bare) = %q, want %q", got, want)
	}

	if got, want := formatVersion(nil, false), "atazs-armatur (unknown)"; got != want {
		t.Errorf("formatVersion(nil) = %q, want %q", got, want)
	}
}
