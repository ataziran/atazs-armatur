package main

import (
	"bytes"
	"encoding/json"
)

// session is the part of Claude Code's status payload this program reads.
//
// The field types carry the contract for malformed input, and encoding/json
// enforces it: a field of the wrong shape fails the decode, which blanks the
// status line. A missing or null field leaves the zero value: a meter without
// its value shows as waiting, and a missing working directory falls through to
// the next source. Three fields must survive a wrong type instead: current_dir and cwd
// fall through to the next source, and resets_at accepts numbers as well as
// numeric strings. Those are `any` and are checked where they are used.
type session struct {
	Workspace struct {
		CurrentDir any `json:"current_dir"`
	} `json:"workspace"`
	CWD            any    `json:"cwd"`
	TranscriptPath string `json:"transcript_path"`
	ContextWindow  struct {
		UsedPercentage *float64 `json:"used_percentage"`
	} `json:"context_window"`
	RateLimits struct {
		FiveHour *window `json:"five_hour"`
		SevenDay *window `json:"seven_day"`
	} `json:"rate_limits"`
}

type window struct {
	UsedPercentage *float64 `json:"used_percentage"`
	ResetsAt       any      `json:"resets_at"`
}

// sessionDir is the working directory the payload names: current_dir, then
// cwd. Only a non-empty string counts; "" means the payload names none.
func sessionDir(s session) string {
	if dir, _ := s.Workspace.CurrentDir.(string); dir != "" {
		return dir
	}
	dir, _ := s.CWD.(string)
	return dir
}

// decode reads the payload. ok is false only for a JSON object whose shape
// does not match: that is malformed input and the caller renders nothing.
// Anything that is not a single JSON object -- empty stdin, an array, plain
// garbage, or an object with something appended -- yields the zero session and
// true, so the line renders with its fallbacks.
func decode(stdin []byte) (session, bool) {
	trimmed := bytes.TrimLeft(stdin, " \t\r\n")
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return session{}, true
	}
	var s session
	dec := json.NewDecoder(bytes.NewReader(stdin))
	// Numbers in `any` fields stay json.Number: a resets_at of 1e999 overflows
	// float64 and would otherwise fail the whole decode.
	dec.UseNumber()
	if err := dec.Decode(&s); err != nil {
		return session{}, false
	}
	// Anything after the first value makes the payload invalid, not malformed.
	if dec.More() {
		return session{}, true
	}
	return s, true
}
