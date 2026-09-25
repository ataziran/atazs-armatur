package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// peerName returns the name other Claude Code sessions use to message this
// one (SendMessage, ListAgents). Claude Code keeps one registry file per
// running session in <config>/sessions/<pid>.json, and the one whose
// sessionId matches carries it. A resumed session keeps its id under a new
// pid, and the old file may linger, so of several matches the most recently
// updated wins. "" without a match: without a registry entry the session
// cannot be messaged, so no stand-in would be an address.
func peerName(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	dir := os.Getenv("CLAUDE_CONFIG_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".claude")
	}
	dir = filepath.Join(dir, "sessions")
	// Read the directory literally: config paths may contain glob characters.
	files, _ := os.ReadDir(dir)
	name, newest := "", -1.0
	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			continue
		}
		var reg struct {
			SessionID string  `json:"sessionId"`
			Name      string  `json:"name"`
			UpdatedAt float64 `json:"updatedAt"`
		}
		if json.Unmarshal(raw, &reg) != nil || reg.SessionID != sessionID || reg.Name == "" {
			continue
		}
		if reg.UpdatedAt > newest {
			name, newest = reg.Name, reg.UpdatedAt
		}
	}
	return name
}
