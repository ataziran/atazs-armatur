package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGitBranchSymlinkedDirectory(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	write(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/target\n")
	subdir := filepath.Join(repo, "sub")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, inRepo := range []bool{false, true} {
		name := "outside-repository"
		if inRepo {
			name = "inside-other-repository"
		}
		t.Run(name, func(t *testing.T) {
			parent := filepath.Join(root, name)
			if err := os.Mkdir(parent, 0o755); err != nil {
				t.Fatal(err)
			}
			if inRepo {
				write(t, filepath.Join(parent, ".git", "HEAD"), "ref: refs/heads/wrong\n")
			}
			link := filepath.Join(parent, "linked-subdir")
			if err := os.Symlink(subdir, link); err != nil {
				if runtime.GOOS == "windows" {
					t.Skipf("symlink creation unavailable: %v", err)
				}
				t.Fatal(err)
			}
			if got := gitBranch(link); got != "target" {
				t.Errorf("gitBranch(%q) = %q, want target", link, got)
			}
		})
	}
}

// A working directory that no longer exists (rm -rf while the session sits
// in it) cannot be resolved; the walk falls back to the path as given.
func TestGitBranchDeletedDirectory(t *testing.T) {
	repo := t.TempDir()
	write(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/main\n")
	gone := filepath.Join(repo, "build", "gone")
	if got := gitBranch(gone); got != "main" {
		t.Errorf("gitBranch(%q) = %q, want main", gone, got)
	}
}

func TestGitBranch(t *testing.T) {
	root := t.TempDir()

	// A normal repository on a branch, with a subdirectory.
	repo := filepath.Join(root, "repo")
	write(t, filepath.Join(repo, ".git", "HEAD"), "ref: refs/heads/main\n")
	if err := os.MkdirAll(filepath.Join(repo, "sub", "deeper"), 0o755); err != nil {
		t.Fatal(err)
	}

	// A linked worktree: .git is a file pointing at the real git dir.
	wtGitDir := filepath.Join(repo, ".git", "worktrees", "wt")
	write(t, filepath.Join(wtGitDir, "HEAD"), "ref: refs/heads/feature\n")
	wt := filepath.Join(root, "wt")
	write(t, filepath.Join(wt, ".git"), "gitdir: "+wtGitDir+"\n")

	// A detached HEAD holds an object id instead of a ref.
	detached := filepath.Join(root, "detached")
	write(t, filepath.Join(detached, ".git", "HEAD"),
		"9ac8b38e410c20078b1d160ffc408333d7d0c5d5\n")

	// HEAD with a ref that is not a branch, e.g. mid-rebase.
	otherRef := filepath.Join(root, "otherref")
	write(t, filepath.Join(otherRef, ".git", "HEAD"), "ref: refs/tags/v1\n")

	// HEAD with content that is neither a ref nor an object id.
	garbage := filepath.Join(root, "garbage")
	write(t, filepath.Join(garbage, ".git", "HEAD"), "not an object id\n")

	// An object id too short to shorten.
	stubby := filepath.Join(root, "stubby")
	write(t, filepath.Join(stubby, ".git", "HEAD"), "9ac8b3\n")

	// Not a repository at all.
	plain := filepath.Join(root, "plain")
	if err := os.MkdirAll(plain, 0o755); err != nil {
		t.Fatal(err)
	}

	for name, tc := range map[string]struct{ dir, want string }{
		"repository root": {repo, "main"},
		"subdirectory":    {filepath.Join(repo, "sub", "deeper"), "main"},
		"linked worktree": {wt, "feature"},
		"detached head":   {detached, "@9ac8b38"},
		"non-branch ref":  {otherRef, ""},
		"garbage head":    {garbage, ""},
		"short object":    {stubby, ""},
		"no repository":   {plain, ""},
		"missing path":    {filepath.Join(root, "nope"), ""},
	} {
		t.Run(name, func(t *testing.T) {
			if got := gitBranch(tc.dir); got != tc.want {
				t.Errorf("gitBranch(%q) = %q, want %q", tc.dir, got, tc.want)
			}
		})
	}
}
