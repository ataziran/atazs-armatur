//go:build unix

package main

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// A FIFO in place of .claude.json would block ReadFile forever and hang the
// status line with it.
func TestAccountEmailSkipsFIFO(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	if err := syscall.Mkfifo(filepath.Join(dir, ".claude.json"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := make(chan string, 1)
	go func() { got <- accountEmail() }()
	select {
	case email := <-got:
		if email != "" {
			t.Errorf("accountEmail = %q, want empty", email)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("accountEmail blocked on a FIFO")
	}
}
