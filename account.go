package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// accountEmail returns the email address of the account Claude Code is logged
// in with. The status payload does not carry it; Claude Code keeps it in its
// global config, <config>/.claude.json when CLAUDE_CONFIG_DIR is set and
// ~/.claude.json otherwise, under oauthAccount.emailAddress. "" when there is
// no such file or field, as with an API key: nothing is logged in to show.
func accountEmail() string {
	path := ""
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		path = filepath.Join(dir, ".claude.json")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		path = filepath.Join(home, ".claude.json")
	}
	// Regular files only: a FIFO or device would block the read.
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		return ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	// Only the one field is decoded; `any` keeps a wrong type from failing it.
	var cfg struct {
		OAuthAccount struct {
			EmailAddress any `json:"emailAddress"`
		} `json:"oauthAccount"`
	}
	if json.Unmarshal(raw, &cfg) != nil {
		return ""
	}
	email, _ := cfg.OAuthAccount.EmailAddress.(string)
	return email
}
