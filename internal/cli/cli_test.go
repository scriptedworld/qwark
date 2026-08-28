package cli_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/scriptedworld/qwark/internal/cli"
)

const (
	statusOK    = 0
	statusError = 1
	statusUsage = 2
)

// invoke runs one command line against string streams and returns everything
// the caller could observe: what went to each stream, and the status.
func invoke(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()

	var out, errOut strings.Builder
	status := cli.Main(args, strings.NewReader(stdin), &out, &errOut)
	return out.String(), errOut.String(), status
}

// COVERS: FR-3.1 | positive
func TestASTOutlinesTheTree(t *testing.T) {
	t.Parallel()

	out, errOut, status := invoke(t, "", "ast", "cat a | grep b")

	if status != statusOK {
		t.Fatalf("status = %d, want %d (stderr: %s)", status, statusOK, errOut)
	}
	if !strings.Contains(out, "BinaryCmd") {
		t.Errorf("outline lacks BinaryCmd:\n%s", out)
	}
}

// COVERS: FR-3.1 | positive
func TestASTDebugPrintsTheNodeStructs(t *testing.T) {
	t.Parallel()

	out, _, status := invoke(t, "", "ast", "--debug", "echo a")

	if status != statusOK {
		t.Fatalf("status = %d, want %d", status, statusOK)
	}
	if !strings.Contains(out, "CallExpr") {
		t.Errorf("debug output lacks the node structs:\n%s", out)
	}
}

// COVERS: FR-3.2 | positive
func TestFactsListsWhatACommandEstablishes(t *testing.T) {
	t.Parallel()

	out, _, status := invoke(t, "", "facts", "cat a | grep b > c")

	if status != statusOK {
		t.Fatalf("status = %d, want %d", status, statusOK)
	}
	for _, want := range []string{"pipe", "redirect.truncate"} {
		if !strings.Contains(out, want) {
			t.Errorf("facts output lacks %q:\n%s", want, out)
		}
	}
}

// COVERS: FR-3.2 | negative
func TestFactsSaysNothingForABareInvocation(t *testing.T) {
	t.Parallel()

	out, _, status := invoke(t, "", "facts", "git status")

	if status != statusOK {
		t.Fatalf("status = %d, want %d", status, statusOK)
	}
	if out != "" {
		t.Errorf("output = %q, want empty for a command with no facts", out)
	}
}

// COVERS: FR-3.3 | positive
func TestACommandMayComeFromStdin(t *testing.T) {
	t.Parallel()

	out, _, status := invoke(t, "cat a | grep b", "facts")

	if status != statusOK {
		t.Fatalf("status = %d, want %d", status, statusOK)
	}
	if !strings.Contains(out, "pipe") {
		t.Errorf("facts from stdin lacks pipe:\n%s", out)
	}
}

// COVERS: FR-3.3 | edge
func TestAnUnquotedCommandIsJoinedRatherThanTruncated(t *testing.T) {
	t.Parallel()

	// The shell split this into three arguments before qwark saw it. Taking
	// only the first would silently judge `cat` instead of the whole line.
	out, _, status := invoke(t, "", "facts", "cat", "a", "|", "grep", "b")

	if status != statusOK {
		t.Fatalf("status = %d, want %d", status, statusOK)
	}
	if !strings.Contains(out, "pipe") {
		t.Errorf("the arguments were not rejoined into one command:\n%s", out)
	}
}

// COVERS: FR-1.2 | negative
func TestAnUnparseableCommandFails(t *testing.T) {
	t.Parallel()

	out, errOut, status := invoke(t, "", "facts", "echo a )")

	if status != statusError {
		t.Errorf("status = %d, want %d", status, statusError)
	}
	if out != "" {
		t.Errorf("stdout = %q, want nothing written on failure", out)
	}
	if !strings.Contains(errOut, "qwark:") {
		t.Errorf("stderr = %q, want a reported reason", errOut)
	}
}

// COVERS: FR-4.4 | positive
func TestRulesReportsWhatARuleSetHolds(t *testing.T) {
	t.Parallel()

	out, errOut, status := invoke(t, "", "rules", "../../rules")

	if status != statusOK {
		t.Fatalf("status = %d, want %d (stderr: %s)", status, statusOK, errOut)
	}
	// A rule set can be found wrong here, before it is the reason every
	// command is failing.
	for _, want := range []string{"shells:", "groups:", "declarations:", "rules:", "deny"} {
		if !strings.Contains(out, want) {
			t.Errorf("report lacks %q:\n%s", want, out)
		}
	}
}

// COVERS: FR-4.5 | negative
func TestRulesReportsWhyASetWillNotLoad(t *testing.T) {
	t.Parallel()

	out, errOut, status := invoke(t, "", "rules", "/nonexistent/rules")

	if status != statusError {
		t.Errorf("status = %d, want %d", status, statusError)
	}
	if out != "" {
		t.Errorf("stdout = %q, want nothing written on failure", out)
	}
	if !strings.Contains(errOut, "/nonexistent/rules") {
		t.Errorf("stderr = %q, want it to name what could not be read", errOut)
	}
}

// COVERS: FR-4.22 | negative
func TestJudgeRefusesWhatNothingAllows(t *testing.T) {
	t.Parallel()

	// The repository's own rules carry no allow rules and no declarations, so
	// every command is refused. That is the correct reading of a policy that
	// is all denials, and seeing it is the point of the command.
	out, errOut, status := invoke(t, "", "judge", "../../rules", "--", "rm", "-rf", "/")

	if status != statusOK {
		t.Fatalf("status = %d, want %d (stderr: %s)", status, statusOK, errOut)
	}
	if !strings.HasPrefix(out, "deny") {
		t.Errorf("verdict = %q, want a refusal", out)
	}
}

// COVERS: FR-1.2, FR-4.12 | negative
func TestJudgeRefusesWhatItCannotParse(t *testing.T) {
	t.Parallel()

	// A command qwark cannot parse is one it cannot judge, and the parser's
	// own message is what comes back.
	out, _, status := invoke(t, "", "judge", "../../rules", "--", "echo", "a", ")")

	if status != statusOK {
		t.Errorf("status = %d, want %d", status, statusOK)
	}
	if !strings.HasPrefix(out, "deny") {
		t.Errorf("verdict = %q, want a refusal", out)
	}
	if !strings.Contains(out, "unparseable") {
		t.Errorf("output %q does not say it could not be parsed", out)
	}
}

// COVERS: FR-4.15 | negative
func TestJudgeNeedsBothARuleSetAndACommand(t *testing.T) {
	t.Parallel()

	_, errOut, status := invoke(t, "", "judge")
	if status != statusUsage {
		t.Errorf("status = %d, want %d", status, statusUsage)
	}
	if !strings.Contains(errOut, "rules path") {
		t.Errorf("stderr = %q, want it to say what was missing", errOut)
	}

	_, _, status = invoke(t, "", "judge", "/nonexistent", "--", "rm", "x")
	if status != statusError {
		t.Errorf("status = %d for an unreadable rule set, want %d", status, statusError)
	}
}

// COVERS: FR-3.4 | negative
func TestAnUnknownSubcommandIsAnError(t *testing.T) {
	t.Parallel()

	_, errOut, status := invoke(t, "", "wibble")

	if status != statusUsage {
		t.Errorf("status = %d, want %d", status, statusUsage)
	}
	if !strings.Contains(errOut, "wibble") {
		t.Errorf("stderr = %q, want it to name what was not understood", errOut)
	}
}

// COVERS: FR-3.4 | negative
func TestNoArgumentsIsAUsageError(t *testing.T) {
	t.Parallel()

	_, errOut, status := invoke(t, "")

	if status != statusUsage {
		t.Errorf("status = %d, want %d", status, statusUsage)
	}
	if !strings.Contains(errOut, "usage:") {
		t.Errorf("stderr = %q, want the usage text", errOut)
	}
}

// COVERS: FR-3.4 | positive
func TestHelpIsAskedForRatherThanStumbledInto(t *testing.T) {
	t.Parallel()

	for _, arg := range []string{"help", "-h", "--help"} {
		t.Run(arg, func(t *testing.T) {
			t.Parallel()

			out, _, status := invoke(t, "", arg)

			if status != statusOK {
				t.Errorf("status = %d for %q, want %d", status, arg, statusOK)
			}
			if !strings.Contains(out, "usage:") {
				t.Errorf("help went somewhere other than stdout: %q", out)
			}
		})
	}
}

// failingReader stands in for a stdin that cannot be read. It is a real
// implementation of io.Reader rather than a mock: there is nothing to assert
// about how it was called, only what the code does with what it returns.
type failingReader struct{}

var errUnreadable = errors.New("stdin is unreadable")

func (failingReader) Read([]byte) (int, error) { return 0, errUnreadable }

// COVERS: FR-3.3 | negative
func TestAnUnreadableStdinIsReported(t *testing.T) {
	t.Parallel()

	var out, errOut strings.Builder
	status := cli.Main([]string{"facts"}, failingReader{}, &out, &errOut)

	if status != statusError {
		t.Errorf("status = %d, want %d", status, statusError)
	}
	if got := errOut.String(); !strings.Contains(got, "reading command from stdin") {
		t.Errorf("stderr = %q, want it to say what could not be read", got)
	}
}

// COVERS: FR-7.12 | positive
func TestJudgeCanBeToldWhichAgentIsAsking(t *testing.T) {
	t.Parallel()

	// A rule set that has never judged anything is a policy nobody has run, and
	// that goes double for one whose verdicts differ by role: an agent clause
	// nobody can exercise from the command line is a rule nobody can check
	// before it is the reason something failed.
	//
	// The flag is taken from the front only. Everything after the rules path
	// may be the command being judged, and a gate that ate an argument out of
	// the middle of a command would be judging something other than what was
	// typed, so this also proves the split still works around it.
	out, errOut, status := invoke(t, "",
		"judge", "--agent=gate-runner", "../../rules", "--", "git", "status")

	if status != statusOK {
		t.Fatalf("status = %d, want %d; stderr = %q", status, statusOK, errOut)
	}
	if !strings.HasPrefix(out, "allow") {
		t.Errorf("verdict = %q, want allow: the flag should have been consumed "+
			"rather than read as a rules path or as part of the command", out)
	}
}
