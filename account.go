package main

import (
	"cmp"
	"encoding/json"
	"os"
	"path/filepath"
)

// maxConfig caps the config file read on every repaint: it grows with use,
// and beyond this the parse would cost more than the status line itself.
const maxConfig = 4 << 20

// accountEmail returns oauthAccount.emailAddress from Claude Code's global
// config, looked up as Claude Code does: <config>/.config.json if it exists
// (<config> is CLAUDE_CONFIG_DIR or ~/.claude), else .claude.json in
// CLAUDE_CONFIG_DIR or the home directory, .claude-custom-oauth.json with
// CLAUDE_CODE_CUSTOM_OAUTH_URL set. "" without the file or the field, as
// with an API key.
func accountEmail() string {
	dir := os.Getenv("CLAUDE_CONFIG_DIR")
	home, err := os.UserHomeDir()
	if dir == "" && err != nil {
		return ""
	}
	path := filepath.Join(cmp.Or(dir, filepath.Join(home, ".claude")), ".config.json")
	info, err := os.Stat(path)
	if err != nil {
		name := ".claude.json"
		if os.Getenv("CLAUDE_CODE_CUSTOM_OAUTH_URL") != "" {
			name = ".claude-custom-oauth.json"
		}
		path = filepath.Join(cmp.Or(dir, home), name)
		info, err = os.Stat(path)
	}
	// Regular files only: a FIFO or device would block the read, and a
	// directory cannot be read. Claude Code would fail on them too.
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxConfig {
		return ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	// A wrong type fails the decode and leaves "", which is what it should show.
	var cfg struct {
		OAuthAccount struct {
			EmailAddress string `json:"emailAddress"`
		} `json:"oauthAccount"`
	}
	if json.Unmarshal(raw, &cfg) != nil {
		return ""
	}
	return cfg.OAuthAccount.EmailAddress
}
