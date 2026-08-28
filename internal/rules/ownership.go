package rules

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"syscall"
)

// Reasons a rule set cannot be trusted to be the one that was written.
var (
	ErrWritableFile = errors.New("a rule file is writable by the user qwark runs as")
	ErrWritableDir  = errors.New("a rule directory is writable by the user qwark runs as")
	ErrUnownable    = errors.New("who owns a rule file could not be established")
)

// The permission bits that let somebody write.
const (
	writeByOwner = 0o200
	writeByGroup = 0o020
	writeByOther = 0o002
)

// CheckOwnership refuses a rule set the running user could rewrite.
//
// # Why a rule set the subject can edit is not a rule set
//
// qwark's whole premise is that an agent is constrained by rules it did not
// write. An agent able to edit those rules is constrained by nothing, and it
// does not need a shell to do it: the Write and Edit tools reach the
// filesystem without passing through this gate at all.
//
// # The directory matters as much as the files
//
// A writable directory permits unlink-and-replace, which defeats an unwritable
// file completely: the old file is removed and a new one put in its place, and
// every permission on the original is irrelevant. Both are checked.
//
// # What this is not
//
// It is a check on the ordinary permission bits and says nothing about ACLs, a
// read-only mount, or an immutable attribute. It can report a file as
// unwritable that something else would refuse anyway, which is the harmless
// direction; it cannot report one as writable that is not.
//
// Running as root defeats it entirely, and that is reported rather than passed
// over: root can write anything, so "not writable by this user" is not a
// property root can have.
func CheckOwnership(paths []string) error {
	files, err := expand(paths)
	if err != nil {
		return err
	}

	user := os.Geteuid()
	groups, err := os.Getgroups()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnownable, err)
	}
	groups = append(groups, os.Getegid())

	seen := map[string]bool{}
	for _, file := range files {
		if err := refuseIfWritable(file, user, groups, ErrWritableFile); err != nil {
			return err
		}

		dir := filepath.Dir(file)
		if seen[dir] {
			continue
		}
		seen[dir] = true
		if err := refuseIfWritable(dir, user, groups, ErrWritableDir); err != nil {
			return err
		}
	}

	return nil
}

// refuseIfWritable reports a path the running user could change.
func refuseIfWritable(path string, user int, groups []int, refusal error) error {
	writable, why, err := writableBy(path, user, groups)
	if err != nil {
		return err
	}
	if writable {
		return fmt.Errorf("%w: %s (%s)", refusal, path, why)
	}
	return nil
}

// writableBy reports whether a user can write a path, and by which route, so a
// refusal can say what to change rather than only that something is wrong.
func writableBy(path string, user int, groups []int) (bool, string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, "", fmt.Errorf("%w: %s: %w", ErrUnownable, path, err)
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false, "", fmt.Errorf("%w: %s", ErrUnownable, path)
	}

	// Root is not subject to the permission bits, so no arrangement of them
	// makes this true for root. Saying so is better than passing a check that
	// means nothing.
	if user == 0 {
		return true, "qwark is running as root, which can write anything", nil
	}

	mode := info.Mode().Perm()
	switch {
	case mode&writeByOther != 0:
		return true, "world-writable", nil
	case int(stat.Uid) == user && mode&writeByOwner != 0:
		return true, "owned by this user and writable by its owner", nil
	case slices.Contains(groups, int(stat.Gid)) && mode&writeByGroup != 0:
		return true, "group-writable by a group this user is in", nil
	default:
		return false, "", nil
	}
}
