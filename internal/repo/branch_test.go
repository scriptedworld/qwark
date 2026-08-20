package repo_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/scriptedworld/qwark/internal/repo"
)

// plant builds a repository whose HEAD holds the given content, and returns its
// working directory. Written by hand rather than by running `git init`, because
// the package under test exists precisely so that git is never run.
func plant(t *testing.T, head string) string {
	t.Helper()

	work := t.TempDir()
	gitDir := filepath.Join(work, ".git")
	if err := os.MkdirAll(gitDir, 0o750); err != nil {
		t.Fatalf("creating %s: %v", gitDir, err)
	}
	write(t, filepath.Join(gitDir, "HEAD"), head)
	return work
}

func write(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// COVERS: FR-8.9 | positive
func TestTheCheckedOutBranchIsRead(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		head string
		want string
	}{
		{name: "main", head: "ref: refs/heads/main\n", want: "main"},
		{name: "master", head: "ref: refs/heads/master\n", want: "master"},
		{name: "slashes", head: "ref: refs/heads/feature/two\n", want: "feature/two"},
		{name: "no trailing newline", head: "ref: refs/heads/main", want: "main"},
		{name: "trailing spaces", head: "ref: refs/heads/main  \n", want: "main"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got, err := repo.Branch(plant(t, c.head))
			if err != nil {
				t.Fatalf("Branch = %v", err)
			}
			if got != c.want {
				t.Errorf("Branch = %q, want %q", got, c.want)
			}
		})
	}
}

// COVERS: FR-8.9 | positive
func TestTheRepositoryIsFoundFromASubdirectory(t *testing.T) {
	t.Parallel()

	work := plant(t, "ref: refs/heads/main\n")
	deep := filepath.Join(work, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o750); err != nil {
		t.Fatalf("creating %s: %v", deep, err)
	}

	got, err := repo.Branch(deep)
	if err != nil {
		t.Fatalf("Branch = %v", err)
	}
	if got != "main" {
		t.Errorf("Branch = %q, want %q", got, "main")
	}
}

// COVERS: FR-8.9 | edge
func TestADetachedHeadHasNoBranch(t *testing.T) {
	t.Parallel()

	// A commit id rather than a symbolic ref. Reporting it as a branch named
	// after the hash would let a rule about `main` silently stop applying.
	for _, head := range []string{
		"4b825dc642cb6eb9a060e54bf8d69288fbee4904\n",
		"ref: refs/heads/\n",
		"",
	} {
		t.Run(head, func(t *testing.T) {
			t.Parallel()

			if _, err := repo.Branch(plant(t, head)); !errors.Is(err, repo.ErrDetached) {
				t.Errorf("Branch = %v, want %v", err, repo.ErrDetached)
			}
		})
	}
}

// COVERS: FR-8.9 | negative
func TestSomewhereWithNoRepositoryReportsSo(t *testing.T) {
	t.Parallel()

	// The walk stops at the filesystem root rather than running forever.
	if _, err := repo.Branch(t.TempDir()); !errors.Is(err, repo.ErrNoRepository) {
		t.Errorf("Branch = %v, want %v", err, repo.ErrNoRepository)
	}
}

// COVERS: FR-8.9 | edge
func TestAWorktreeRedirectionIsFollowed(t *testing.T) {
	t.Parallel()

	// In a worktree or a submodule `.git` is a file naming the real directory.
	// A gate that only understood the directory form would answer
	// ErrNoRepository there, and a rule about branches would quietly not apply.
	cases := []struct {
		name     string
		relative bool
	}{
		{name: "absolute gitdir", relative: false},
		{name: "relative gitdir", relative: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			gitDir := filepath.Join(root, "real-git-dir")
			work := filepath.Join(root, "worktree")
			for _, dir := range []string{gitDir, work} {
				if err := os.MkdirAll(dir, 0o750); err != nil {
					t.Fatalf("creating %s: %v", dir, err)
				}
			}
			write(t, filepath.Join(gitDir, "HEAD"), "ref: refs/heads/side\n")

			target := gitDir
			if c.relative {
				target = filepath.Join("..", "real-git-dir")
			}
			write(t, filepath.Join(work, ".git"), "gitdir: "+target+"\n")

			got, err := repo.Branch(work)
			if err != nil {
				t.Fatalf("Branch = %v", err)
			}
			if got != "side" {
				t.Errorf("Branch = %q, want %q", got, "side")
			}
		})
	}
}

// COVERS: FR-8.9 | negative
func TestAMalformedRedirectionIsReported(t *testing.T) {
	t.Parallel()

	for _, content := range []string{"nonsense\n", "gitdir:\n", ""} {
		t.Run(content, func(t *testing.T) {
			t.Parallel()

			work := t.TempDir()
			write(t, filepath.Join(work, ".git"), content)

			if _, err := repo.Branch(work); !errors.Is(err, repo.ErrUnreadable) {
				t.Errorf("Branch = %v, want %v", err, repo.ErrUnreadable)
			}
		})
	}
}

// COVERS: FR-8.9 | negative
func TestAnAbsentHeadIsReported(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	if err := os.MkdirAll(filepath.Join(work, ".git"), 0o750); err != nil {
		t.Fatalf("creating .git: %v", err)
	}

	if _, err := repo.Branch(work); !errors.Is(err, repo.ErrUnreadable) {
		t.Errorf("Branch = %v, want %v", err, repo.ErrUnreadable)
	}
}
