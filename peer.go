package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// peerName returns the name other Claude Code sessions use to message this
// one (SendMessage, ListAgents): Claude Code keeps one registry file per
// running session in <config>/sessions/<pid>.json, and the one whose
// sessionId matches carries it. Without a match the first eight characters of
// the session id stand in; "" when the payload names no session.
func peerName(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	dir := os.Getenv("CLAUDE_CONFIG_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return shortID(sessionID)
		}
		dir = filepath.Join(home, ".claude")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "sessions", "*.json"))
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var reg struct {
			SessionID string `json:"sessionId"`
			Name      string `json:"name"`
		}
		if json.Unmarshal(raw, &reg) == nil && reg.SessionID == sessionID && reg.Name != "" {
			return reg.Name
		}
	}
	return shortID(sessionID)
}

func shortID(id string) string {
	if r := []rune(id); len(r) > 8 {
		return string(r[:8])
	}
	return id
}
