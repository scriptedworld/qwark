// Package repo answers questions about the repository a command is being run
// in, by reading its files rather than by running its tooling.
//
// # Why this reads files instead of running git
//
// **FACT 2026-08-20, demonstrated.** An alias written into a repository's own
// `.git/config` executes on a plain git invocation:
//
//	[alias]
//	  innocent = "!printf EXECUTED_FROM_REPO_CONFIG\n"
//
//	$ git innocent
//	EXECUTED_FROM_REPO_CONFIG
//
// `core.pager`, `core.fsmonitor` and the hooks path do the same thing by other
// routes. That file is writable by the agent with no shell involved, so asking
// git a question from inside the gate would run the subject's code in the
// judge's process -- the same mistake as sourcing the shell snapshot, reached
// from a different direction.
//
// Reading `.git/HEAD` answers the question and executes nothing.
package repo

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// What a repository lookup can fail to establish. Each is a fact about the
// world rather than a malfunction, so a caller decides what it means: a gate
// that denies on "no branch" is making a policy choice, not handling an error.
var (
	ErrNoRepository = errors.New("no repository at or above this directory")
	ErrDetached     = errors.New("no branch is checked out")
	ErrUnreadable   = errors.New("repository state could not be read")
)

// headPrefix introduces a symbolic ref. A HEAD that does not begin with it
// holds a commit id directly, which is a detached head.
const headPrefix = "ref: refs/heads/"

// gitdirPrefix introduces the redirection used when `.git` is a file rather
// than a directory, as it is in a worktree or a submodule.
const gitdirPrefix = "gitdir:"

// maxHeadSize bounds what is read from HEAD. The real file is one short line;
// anything larger is a malformed or hostile repository, and a gate that runs
// before every command should not be readable into memory without limit.
const maxHeadSize = 4096

// Branch reports the branch checked out by the repository at or above dir.
//
// It returns ErrDetached when a commit is checked out directly, and
// ErrNoRepository when there is no repository to ask.
func Branch(dir string) (string, error) {
	gitDir, err := locate(dir)
	if err != nil {
		return "", err
	}

	head, err := readTrimmed(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return "", fmt.Errorf("%w: HEAD: %w", ErrUnreadable, err)
	}

	if !strings.HasPrefix(head, headPrefix) {
		return "", ErrDetached
	}

	name := strings.TrimSpace(strings.TrimPrefix(head, headPrefix))
	if name == "" {
		return "", ErrDetached
	}
	return name, nil
}

// locate walks upwards for a `.git`, resolving the file form used by worktrees
// and submodules. The walk stops at the filesystem root rather than running
// forever on a path that has none.
func locate(dir string) (string, error) {
	current, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUnreadable, err)
	}

	for {
		candidate := filepath.Join(current, ".git")
		info, err := os.Stat(candidate)
		switch {
		case err == nil && info.IsDir():
			return candidate, nil
		case err == nil:
			return redirected(current, candidate)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", ErrNoRepository
		}
		current = parent
	}
}

// redirected follows a `.git` file to the directory it names. The path may be
// relative, in which case it is relative to the directory holding the file.
func redirected(dir, marker string) (string, error) {
	content, err := readTrimmed(marker)
	if err != nil {
		return "", fmt.Errorf("%w: %s: %w", ErrUnreadable, marker, err)
	}

	if !strings.HasPrefix(content, gitdirPrefix) {
		return "", fmt.Errorf("%w: %s names no gitdir", ErrUnreadable, marker)
	}

	target := strings.TrimSpace(strings.TrimPrefix(content, gitdirPrefix))
	if target == "" {
		return "", fmt.Errorf("%w: %s names an empty gitdir", ErrUnreadable, marker)
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(dir, target)
	}
	return target, nil
}

// readTrimmed reads a bounded amount of a small state file.
//
// The path is cleaned before opening. It is assembled from a located `.git`
// rather than taken from a command, but a path built by joining components is
// worth normalising on its own account, and it is what lets this read stand
// without a suppression.
func readTrimmed(path string) (string, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(io.LimitReader(file, maxHeadSize))
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	return strings.TrimSpace(string(content)), nil
}
