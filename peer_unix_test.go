//go:build unix

package main

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// A FIFO named like a registry file would block ReadFile forever and hang the
// status line with it.
func TestPeerNameSkipsFIFO(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	write(t, filepath.Join(dir, "sessions", "2.json"),
		`{"sessionId":"session-123","name":"expected-peer"}`)
	if err := syscall.Mkfifo(filepath.Join(dir, "sessions", "1.json"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := make(chan string, 1)
	go func() { got <- peerName("session-123") }()
	select {
	case name := <-got:
		if name != "expected-peer" {
			t.Errorf("peerName = %q, want expected-peer", name)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("peerName blocked on a FIFO")
	}
}
