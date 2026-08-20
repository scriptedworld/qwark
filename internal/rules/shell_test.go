package rules_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scriptedworld/qwark/internal/rules"
)

// permitted is the declaration these tests verify against. FACT 2026-08-20:
// /bin is a symlink to usr/bin on the machine this was written for, so these
// are two spellings of one root-owned binary rather than two binaries.
func permitted() rules.ShellPolicy {
	return rules.ShellPolicy{Allow: []string{"/bin/bash", "/usr/bin/bash"}}
}

// COVERS: FR-1.7 | positive
func TestAPermittedShellIsAccepted(t *testing.T) {
	t.Parallel()

	for _, reported := range []string{"/bin/bash", "/usr/bin/bash", "  /bin/bash  "} {
		t.Run(reported, func(t *testing.T) {
			t.Parallel()

			if err := permitted().Verify(reported); err != nil {
				t.Errorf("Verify(%q) = %v, want acceptance", reported, err)
			}
		})
	}
}

// COVERS: FR-1.5 | negative
func TestAnotherShellIsRefused(t *testing.T) {
	t.Parallel()

	// zsh is the case this was written for: FACT 2026-08-20, the tool named
	// Bash was running zsh 5.9 on the machine where qwark was written.
	for _, reported := range []string{"/bin/zsh", "/bin/dash", "/bin/sh", "/usr/bin/fish"} {
		t.Run(reported, func(t *testing.T) {
			t.Parallel()

			err := permitted().Verify(reported)
			if !errors.Is(err, rules.ErrShellMismatch) {
				t.Errorf("Verify(%q) = %v, want %v", reported, err, rules.ErrShellMismatch)
			}
		})
	}
}

// COVERS: FR-1.9 | negative
func TestSomethingMerelyNamedBashIsRefused(t *testing.T) {
	t.Parallel()

	// This is the whole reason paths are compared and not names. Every one of
	// these is called bash, and a gate whose subject can create files must not
	// accept a shell on the strength of what it is called.
	for _, reported := range []string{
		"/tmp/bash",
		"/home/ancient/.local/bin/bash",
		"/usr/local/bin/bash",
		"bash",
		"./bash",
		"/usr/bin/bash2",
		"/usr/bin/../tmp/bash",
	} {
		t.Run(reported, func(t *testing.T) {
			t.Parallel()

			err := permitted().Verify(reported)
			if !errors.Is(err, rules.ErrShellMismatch) {
				t.Errorf("Verify(%q) = %v, want %v", reported, err, rules.ErrShellMismatch)
			}
		})
	}
}

// COVERS: FR-1.10 | positive
func TestTwoSpellingsOfOneShellReachOneAnswer(t *testing.T) {
	t.Parallel()

	// FACT 2026-08-20: /bin is a symlink to usr/bin on this machine, so the
	// two permitted paths are one file. A rule about a shell must not be a
	// rule about one way of writing its name.
	dir := t.TempDir()
	target := filepath.Join(dir, "realbash")
	link := filepath.Join(dir, "linkbash")
	if err := os.WriteFile(target, []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatalf("writing %s: %v", target, err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("linking: %v", err)
	}

	// Permitted by one name, reported by the other.
	policy := rules.ShellPolicy{Allow: []string{target}}
	if err := policy.Verify(link); err != nil {
		t.Errorf("Verify(link to a permitted shell) = %v, want acceptance", err)
	}

	// And the other way round, which is the case that matters: the permitted
	// path is the link, and what it points at is what actually runs.
	byLink := rules.ShellPolicy{Allow: []string{link}}
	if err := byLink.Verify(target); err != nil {
		t.Errorf("Verify(target of a permitted link) = %v, want acceptance", err)
	}
}

// COVERS: FR-1.10 | negative
func TestAPermittedNameLinkedElsewhereIsRefused(t *testing.T) {
	t.Parallel()

	// The reason resolution is not merely tidiness. A permitted path replaced
	// by a link to something else still matches as text, and no longer names
	// the program that will run.
	dir := t.TempDir()
	impostor := filepath.Join(dir, "impostor")
	permitted := filepath.Join(dir, "bash")
	if err := os.WriteFile(impostor, []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatalf("writing %s: %v", impostor, err)
	}
	if err := os.Symlink(impostor, permitted); err != nil {
		t.Fatalf("linking: %v", err)
	}

	// The allow list names a real shell somewhere else entirely.
	policy := rules.ShellPolicy{Allow: []string{"/usr/bin/bash"}}

	if err := policy.Verify(permitted); !errors.Is(err, rules.ErrShellMismatch) {
		t.Errorf("Verify(path linked to an impostor) = %v, want %v",
			err, rules.ErrShellMismatch)
	}
}

// COVERS: FR-1.7 | negative
func TestOmittingTheDeclarationIsARefusal(t *testing.T) {
	t.Parallel()

	// A rule file that simply left this out would otherwise disable the check
	// silently, which is the failure this design keeps closing. Absence is not
	// permission.
	var undeclared rules.ShellPolicy

	if err := undeclared.Verify("/bin/bash"); !errors.Is(err, rules.ErrShellUndeclared) {
		t.Errorf("Verify with nothing declared = %v, want %v", err, rules.ErrShellUndeclared)
	}
	if err := (rules.ShellPolicy{Allow: []string{}}).Verify("/bin/bash"); err == nil {
		t.Error("an empty allow list was treated as permission")
	}
}

// COVERS: FR-1.9 | negative
func TestARelativeEntryIsAConfigurationError(t *testing.T) {
	t.Parallel()

	// Declaring `bash` would quietly restore name matching, which is the
	// weakness absolute paths exist to remove. It is refused in the file
	// rather than tolerated at comparison time.
	for _, entry := range []string{"bash", "./bash", "usr/bin/bash", ""} {
		t.Run(entry, func(t *testing.T) {
			t.Parallel()

			policy := rules.ShellPolicy{Allow: []string{"/bin/bash", entry}}
			if err := policy.Verify("/bin/bash"); !errors.Is(err, rules.ErrShellRelative) {
				t.Errorf("Verify with %q declared = %v, want %v",
					entry, err, rules.ErrShellRelative)
			}
		})
	}
}

// COVERS: FR-1.8 | negative
func TestAnUnreportedShellIsARefusal(t *testing.T) {
	t.Parallel()

	// An environment that says nothing is not an environment that says bash.
	for _, reported := range []string{"", "   "} {
		if err := permitted().Verify(reported); !errors.Is(err, rules.ErrShellUnreported) {
			t.Errorf("Verify(%q) = %v, want %v", reported, err, rules.ErrShellUnreported)
		}
	}
}

// COVERS: FR-1.8 | property
func TestARefusalSaysWhatRanAndWhatWasWanted(t *testing.T) {
	t.Parallel()

	err := permitted().Verify("/bin/zsh")
	if err == nil {
		t.Fatal("Verify accepted zsh where bash was required")
	}

	// A denial that says only "wrong shell" leaves the reader to find out
	// which one they have, in the one situation where every command is failing.
	message := err.Error()
	for _, want := range []string{"/bin/zsh", "/bin/bash", "/usr/bin/bash"} {
		if !strings.Contains(message, want) {
			t.Errorf("message %q does not name %q", message, want)
		}
	}
}
