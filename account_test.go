package main

import (
	"path/filepath"
	"strings"
	"testing"
)

const loggedIn = `{"oauthAccount":{"emailAddress":"user@example.com"}}`

func TestAccountEmail(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"logged in", `{"numStartups":3,"oauthAccount":{"emailAddress":"user@example.com","displayName":"U"}}`, "user@example.com"},
		{"api key", `{"numStartups":3}`, ""},
		{"wrong type", `{"oauthAccount":{"emailAddress":42}}`, ""},
		{"account not an object", `{"oauthAccount":"x"}`, ""},
		{"empty", ``, ""},
		{"null", `null`, ""},
		{"half written", `{"oauthAccount":{"emailAddress":"user@exa`, ""},
		// Valid JSON, but past the cap: not read at all.
		{"too large", loggedIn + strings.Repeat(" ", maxConfig), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("CLAUDE_CONFIG_DIR", dir)
			write(t, filepath.Join(dir, ".claude.json"), tc.body)
			if got := accountEmail(); got != tc.want {
				t.Errorf("accountEmail() = %q, want %q", got, tc.want)
			}
		})
	}
	for _, tc := range []struct {
		name        string
		configDir   bool              // set CLAUDE_CONFIG_DIR
		customOAuth bool              // set CLAUDE_CODE_CUSTOM_OAUTH_URL
		files       map[string]string // relative to the home directory
		want        string
	}{
		{"home", false, false, map[string]string{".claude.json": loggedIn}, "user@example.com"},
		// Set but without the file: the home one belongs to another config.
		{"config dir without file", true, false, map[string]string{".claude.json": loggedIn}, ""},
		{"legacy name wins", false, false, map[string]string{
			".claude/.config.json": `{"oauthAccount":{"emailAddress":"legacy@example.com"}}`,
			".claude.json":         loggedIn,
		}, "legacy@example.com"},
		{"legacy name in config dir", true, false, map[string]string{
			"cfg/.config.json": `{"oauthAccount":{"emailAddress":"legacy@example.com"}}`,
			"cfg/.claude.json": loggedIn,
		}, "legacy@example.com"},
		{"directory", false, false, map[string]string{".claude.json/x": loggedIn}, ""},
		// Claude Code picks an existing .config.json, readable or not.
		{"legacy name not a file", false, false, map[string]string{
			".claude/.config.json/x": loggedIn,
			".claude.json":           loggedIn,
		}, ""},
		{"custom oauth", false, true, map[string]string{
			".claude-custom-oauth.json": `{"oauthAccount":{"emailAddress":"fed@example.com"}}`,
			".claude.json":              loggedIn,
		}, "fed@example.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("CLAUDE_CONFIG_DIR", "")
			t.Setenv("CLAUDE_CODE_CUSTOM_OAUTH_URL", "")
			if tc.customOAuth {
				t.Setenv("CLAUDE_CODE_CUSTOM_OAUTH_URL", "https://claude.fedstart.com")
			}
			if tc.configDir {
				t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, "cfg"))
			}
			for name, body := range tc.files {
				write(t, filepath.Join(home, name), body)
			}
			if got := accountEmail(); got != tc.want {
				t.Errorf("accountEmail() = %q, want %q", got, tc.want)
			}
		})
	}
}
