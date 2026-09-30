package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAccountEmail(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"logged in", `{"numStartups":3,"oauthAccount":{"emailAddress":"user@example.com","displayName":"U"}}`, "user@example.com"},
		{"api key", `{"numStartups":3}`, ""},
		{"wrong type", `{"oauthAccount":{"emailAddress":42}}`, ""},
		{"account not an object", `{"oauthAccount":"x"}`, ""},
		{"not json", `not json`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("CLAUDE_CONFIG_DIR", dir)
			if err := os.WriteFile(filepath.Join(dir, ".claude.json"), []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			if got := accountEmail(); got != tc.want {
				t.Errorf("accountEmail() = %q, want %q", got, tc.want)
			}
		})
	}
	t.Run("no file", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
		if got := accountEmail(); got != "" {
			t.Errorf("accountEmail() = %q, want empty", got)
		}
	})
	t.Run("home without config dir", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", "")
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		body := `{"oauthAccount":{"emailAddress":"home@example.com"}}`
		if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := accountEmail(); got != "home@example.com" {
			t.Errorf("accountEmail() = %q, want home@example.com", got)
		}
	})
}
