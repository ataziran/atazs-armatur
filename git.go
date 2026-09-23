package main

import (
	"os"
	"path/filepath"
	"strings"
)

// gitBranch returns the checked-out branch for cwd, @<short commit> on a
// detached HEAD, and "" outside a repository or when HEAD cannot be read. It
// reads the files directly: a subprocess costs a fork per repaint, and a git
// that hangs -- a slow network mount, a stuck lock -- would freeze the status
// line with it.
func gitBranch(cwd string) string {
	dir, ok := gitDir(cwd)
	if !ok {
		return ""
	}
	head, err := os.ReadFile(filepath.Join(dir, "HEAD"))
	if err != nil {
		return ""
	}
	// "ref: refs/heads/<branch>" on a branch, a bare object id when detached.
	if branch, onBranch := strings.CutPrefix(strings.TrimSpace(string(head)), "ref: refs/heads/"); onBranch {
		return branch
	}
	return shortCommit(strings.TrimSpace(string(head)))
}

// shortCommit renders a detached HEAD as @<7 hex digits>, and "" for anything
// that is not an object id. The marker keeps it apart from a branch that
// happens to be named like a hash.
func shortCommit(head string) string {
	if len(head) < 7 {
		return ""
	}
	for _, r := range head {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return ""
		}
	}
	return "@" + head[:7]
}

// gitDir walks up from cwd to the first .git and returns the directory that
// holds HEAD. In a linked worktree .git is a file naming the real one.
func gitDir(cwd string) (string, bool) {
	dir := cwd
	for {
		candidate := filepath.Join(dir, ".git")
		if info, err := os.Stat(candidate); err == nil {
			if info.IsDir() {
				return candidate, true
			}
			return gitDirFromFile(dir, candidate)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// gitDirFromFile reads the "gitdir: <path>" line a linked worktree puts in
// place of the .git directory. A relative path is relative to the worktree.
func gitDirFromFile(worktree, path string) (string, bool) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	target, found := strings.CutPrefix(strings.TrimSpace(string(content)), "gitdir: ")
	if !found || target == "" {
		return "", false
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(worktree, target)
	}
	return target, true
}
