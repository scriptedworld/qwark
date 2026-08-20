package rules_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scriptedworld/qwark/internal/rules"
)

// planted writes a rule file at the given mode into a writable directory, and
// returns the directory.
//
// Nothing here makes the directory unwritable. It does not need to: the file is
// checked before the directory it sits in, so a writable file is reported as
// one. Locking the directory down would also make the temporary tree
// undeletable, which is a cost paid for nothing.
func planted(t *testing.T, fileMode os.FileMode) string {
	t.Helper()

	dir := filepath.Join(t.TempDir(), "rules")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("creating %s: %v", dir, err)
	}

	file := filepath.Join(dir, "00.toml")
	if err := os.WriteFile(file, []byte("[shell]\nallow=[\"/bin/bash\"]\n"), 0o600); err != nil {
		t.Fatalf("writing %s: %v", file, err)
	}
	if err := os.Chmod(file, fileMode); err != nil {
		t.Fatalf("chmod %s: %v", file, err)
	}

	return dir
}

// notRoot skips a test that no arrangement of permission bits can satisfy.
func notRoot(t *testing.T) {
	t.Helper()

	if os.Geteuid() == 0 {
		t.Skip("running as root, which can write anything whatever the bits say")
	}
}

// COVERS: FR-4.17, FR-4.17a | negative
func TestARuleFileThisUserCanWriteIsRefused(t *testing.T) {
	t.Parallel()
	notRoot(t)

	// Owned by this user and writable by its owner, which is what an ordinary
	// `chmod 644` leaves. An agent able to edit the rules is constrained by
	// nothing, and it reaches them with the Write tool without passing through
	// this gate at all.
	err := rules.CheckOwnership([]string{planted(t, 0o644)})

	if !errors.Is(err, rules.ErrWritableFile) {
		t.Fatalf("CheckOwnership = %v, want %v", err, rules.ErrWritableFile)
	}
	if !strings.Contains(err.Error(), "writable by its owner") {
		t.Errorf("message = %q, want it to say by which route", err.Error())
	}
}

// COVERS: FR-4.17a | negative
func TestAWorldWritableRuleFileNamesThatRoute(t *testing.T) {
	t.Parallel()
	notRoot(t)

	// "A rule file is writable" without saying which bit is a message nobody
	// can act on, and the broadest route is the one to name first.
	err := rules.CheckOwnership([]string{planted(t, 0o666)})

	if !errors.Is(err, rules.ErrWritableFile) {
		t.Fatalf("CheckOwnership = %v, want %v", err, rules.ErrWritableFile)
	}
	if !strings.Contains(err.Error(), "world-writable") {
		t.Errorf("message = %q, want it to name the broadest route", err.Error())
	}
}

// COVERS: FR-4.17 | negative
func TestARuleDirectoryThisUserCanWriteIsRefused(t *testing.T) {
	t.Parallel()
	notRoot(t)

	// The file itself is read-only, and that is not enough: a writable
	// directory permits unlink-and-replace, which removes the file and puts
	// another in its place. Every permission on the original is then beside
	// the point.
	err := rules.CheckOwnership([]string{planted(t, 0o444)})

	if !errors.Is(err, rules.ErrWritableDir) {
		t.Errorf("CheckOwnership = %v, want %v", err, rules.ErrWritableDir)
	}
}

// COVERS: FR-4.17 | positive
func TestARuleSetThisUserCannotWriteIsAccepted(t *testing.T) {
	t.Parallel()
	notRoot(t)

	// A real root-owned file in a real root-owned directory, rather than a
	// temporary one made unwritable: this is the arrangement a deployment
	// actually has, and it costs no chmod that would then have to be undone
	// before the tree could be deleted.
	const system = "/etc/passwd"

	info, err := os.Stat(system)
	if err != nil {
		t.Skipf("%s is not present to check against: %v", system, err)
	}
	if info.Mode().Perm()&0o022 != 0 {
		t.Skipf("%s is group- or world-writable on this machine", system)
	}

	if err := rules.CheckOwnership([]string{system}); err != nil {
		t.Errorf("CheckOwnership(%s) = %v, want acceptance", system, err)
	}
}

// COVERS: FR-4.17 | negative
func TestAPathThatCannotBeStattedIsRefused(t *testing.T) {
	t.Parallel()

	// Not knowing who owns a rule file is not the same as knowing nobody can
	// write it.
	if err := rules.CheckOwnership([]string{"/nonexistent/rules"}); err == nil {
		t.Error("CheckOwnership accepted a rule set it could not find")
	}
}

// COVERS: FR-4.17b | property
func TestRunningAsRootIsReportedRatherThanPassedOver(t *testing.T) {
	t.Parallel()

	if os.Geteuid() != 0 {
		t.Skip("not running as root; the branch this pins cannot be reached")
	}

	// No arrangement of permission bits makes a file unwritable by root, so
	// "not writable by this user" is not a property root can have. A check
	// that quietly succeeded here would be worse than none.
	err := rules.CheckOwnership([]string{planted(t, 0o444)})
	if err == nil {
		t.Error("CheckOwnership accepted a rule set while running as root")
	}
	if err != nil && !strings.Contains(err.Error(), "root") {
		t.Errorf("message = %q, want it to say why the check cannot hold", err.Error())
	}
}
