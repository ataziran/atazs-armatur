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
		"2.json": `{"sessionId":"d9be6c2e-4d31","name":"atazs-armatur-06"}`,
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
