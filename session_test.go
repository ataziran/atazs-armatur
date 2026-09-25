package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDecodeShape(t *testing.T) {
	for name, tc := range map[string]struct {
		in     string
		wantOK bool
	}{
		"empty":             {"", true},
		"empty object":      {"{}", true},
		"not json":          {"not json", true},
		"array":             {"[1, 2]", true},
		"trailing garbage":  {`{"cwd": "/a"} junk`, true},
		"good payload":      {`{"context_window": {"used_percentage": 12.5}}`, true},
		"window is string":  {`{"rate_limits": {"five_hour": "x"}}`, false},
		"percent is string": {`{"context_window": {"used_percentage": "50"}}`, false},
		"falsy container":   {`{"rate_limits": 0}`, false},
	} {
		t.Run(name, func(t *testing.T) {
			if _, ok := decode([]byte(tc.in)); ok != tc.wantOK {
				t.Errorf("decode(%q) ok = %v, want %v", tc.in, ok, tc.wantOK)
			}
		})
	}
}

func TestDecodeTrailingData(t *testing.T) {
	const payload = `{"cwd":"/project","context_window":{"used_percentage":42}}`
	for _, suffix := range []string{"}", "]", "} junk", "] {}", "{}", "null", " junk"} {
		t.Run(suffix, func(t *testing.T) {
			s, ok := decode([]byte(payload + suffix))
			if !ok || !reflect.DeepEqual(s, session{}) {
				t.Errorf("decode with suffix %q = (%+v, %v), want empty session and true", suffix, s, ok)
			}
		})
	}
	// JSON whitespace after the payload is valid and must retain its readings.
	s, ok := decode([]byte(payload + " \t\r\n"))
	if !ok || sessionDir(s) != "/project" || s.ContextWindow.UsedPercentage == nil || *s.ContextWindow.UsedPercentage != 42 {
		t.Errorf("decode with trailing whitespace = (%+v, %v), want the original readings", s, ok)
	}
}

func TestDecodeTolerantFields(t *testing.T) {
	s, ok := decode([]byte(`{"workspace": {"current_dir": true}, "cwd": 42,
		"context_window": {"used_percentage": 10}}`))
	if !ok {
		t.Fatal("a wrong type in current_dir/cwd must be tolerated, not fatal")
	}
	if _, isString := s.Workspace.CurrentDir.(string); isString {
		t.Error("current_dir was true, so it must not read as a string")
	}
	if s.ContextWindow.UsedPercentage == nil || *s.ContextWindow.UsedPercentage != 10 {
		t.Error("used_percentage was lost")
	}
}

func TestDecodeKeepsHugeResetsAt(t *testing.T) {
	// 1e999 overflows float64; as a json.Number it must survive decoding, or the
	// whole line would blank instead of just dropping the countdown.
	s, ok := decode([]byte(`{"rate_limits": {"five_hour":
		{"used_percentage": 50, "resets_at": 1e999}}}`))
	if !ok {
		t.Fatal("an out-of-range resets_at must not blank the line")
	}
	if _, isNumber := s.RateLimits.FiveHour.ResetsAt.(json.Number); !isNumber {
		t.Errorf("resets_at is %T, want json.Number", s.RateLimits.FiveHour.ResetsAt)
	}
}
