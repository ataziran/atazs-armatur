package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPeerName(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	sessions := filepath.Join(dir, "sessions")
	if err := os.Mkdir(sessions, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"1.json": `{"sessionId":"aaaa-1","name":"other-01"}`,
		"2.json": `{"sessionId":"d9be6c2e-4d31","name":"atazs-armatur-06","updatedAt":2}`,
		"9.json": `{"sessionId":"d9be6c2e-4d31","name":"stale-old-pid","updatedAt":1}`,
		"3.json": `not json`,
		"4.key":  `{"sessionId":"bbbbbbbb-2","name":"ignored"}`,
	} {
		if err := os.WriteFile(filepath.Join(sessions, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for id, want := range map[string]string{
		"d9be6c2e-4d31": "atazs-armatur-06",
		"bbbbbbbb-2":    "bbbbbbbb",
		"short":         "short",
		"":              "",
	} {
		if got := peerName(id); got != want {
			t.Errorf("peerName(%q) = %q, want %q", id, got, want)
		}
	}
}

func TestPeerNameLiteralConfigPath(t *testing.T) {
	for _, name := range []string{"config[1]", "config["} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, name)
			t.Setenv("CLAUDE_CONFIG_DIR", dir)
			write(t, filepath.Join(dir, "sessions", "1.json"),
				`{"sessionId":"session-123","name":"expected-peer"}`)
			// A glob would treat [1] as a pattern and read this other config.
			write(t, filepath.Join(root, "config1", "sessions", "1.json"),
				`{"sessionId":"session-123","name":"wrong-peer"}`)
			if got := peerName("session-123"); got != "expected-peer" {
				t.Errorf("peerName with config %q = %q, want expected-peer", dir, got)
			}
		})
	}
}
