package audit_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/scriptedworld/qwark/internal/audit"
)

// decode reads one entry back out of a recorded line, which is the only way to
// assert on what was written rather than on what was passed in.
func decode(t *testing.T, line string) audit.Entry {
	t.Helper()

	var entry audit.Entry
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		t.Fatalf("the log line is not readable JSON: %v\n  %s", err, line)
	}
	return entry
}

// COVERS: FR-4.8 | positive
func TestADecisionIsRecordedWithWhatBoreOnIt(t *testing.T) {
	t.Parallel()

	// The point of the record is that a rule set can be changed on evidence.
	// An entry that says a command was denied without saying which rules did
	// it, or under which policy, cannot be read alongside one from another day.
	var out strings.Builder
	log := audit.Writing(&out)

	err := log.Record(audit.Entry{
		At:       time.Unix(0, 0).UTC(),
		RuleSet:  "abc123",
		Decision: "deny",
		Tool:     "Bash",
		Command:  "rm -rf /",
		Rules:    []string{"rm-force", "rm-recursive"},
		Agent:    "gate-runner",
		Cwd:      "/home/x/project",
	})
	if err != nil {
		t.Fatalf("Record = %v, want it to write", err)
	}

	entry := decode(t, out.String())

	if entry.RuleSet != "abc123" {
		t.Errorf("rule set = %q, want the digest that produced the verdict", entry.RuleSet)
	}
	if entry.Decision != "deny" {
		t.Errorf("decision = %q, want deny", entry.Decision)
	}
	if entry.Command != "rm -rf /" {
		t.Errorf("command = %q, want the command as judged", entry.Command)
	}
	if len(entry.Rules) != 2 {
		t.Errorf("rules = %v, want every rule that objected", entry.Rules)
	}
}

// COVERS: FR-4.8 | property
func TestEachEntryIsOneLine(t *testing.T) {
	t.Parallel()

	// One object per line, not one array. A kill part way through leaves the
	// final record unreadable rather than the whole file unparseable, and a
	// reader can consume it a line at a time without holding all of it.
	var out strings.Builder
	log := audit.Writing(&out)

	for _, command := range []string{"git status", "ls -la", "rm x"} {
		if err := log.Record(audit.Entry{Command: command, Decision: "allow"}); err != nil {
			t.Fatalf("Record(%q) = %v", command, err)
		}
	}

	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("wrote %d lines, want one per entry", len(lines))
	}
	for _, line := range lines {
		if strings.Contains(line, "\n") {
			t.Errorf("a line carries a newline, so a reader cannot split on it")
		}
		decode(t, line)
	}
}

// COVERS: FR-4.9 | property
func TestAWithheldValueIsRecordedAsPresent(t *testing.T) {
	t.Parallel()

	// A withheld value must not look like an absent variable. Those are
	// different facts, and only one of them is about secrecy: omitting the row
	// would have the log assert that the variable was not set.
	var out strings.Builder
	log := audit.Writing(&out)

	if err := log.Record(audit.Entry{
		Decision: "allow",
		Env: []audit.Var{
			{Name: "PATH", Value: "/usr/bin"},
			{Name: "API_TOKEN", Withheld: true},
		},
	}); err != nil {
		t.Fatalf("Record = %v", err)
	}

	entry := decode(t, out.String())

	if len(entry.Env) != 2 {
		t.Fatalf("env has %d entries, want the withheld one still present", len(entry.Env))
	}
	if entry.Env[1].Name != "API_TOKEN" || !entry.Env[1].Withheld {
		t.Errorf("env[1] = %+v, want it named and marked withheld", entry.Env[1])
	}
	if entry.Env[1].Value != "" {
		t.Errorf("env[1] carries a value, which is the one thing withholding means it must not")
	}
	if strings.Contains(out.String(), "secret") {
		t.Errorf("the line leaked something it should not have: %s", out.String())
	}
}

// COVERS: FR-4.8 | property
func TestOpeningALogAppendsRatherThanTruncating(t *testing.T) {
	t.Parallel()

	// A log with earlier entries missing reads as a clean history, which is
	// worse than no log: it answers "what happened" confidently and wrongly.
	path := filepath.Join(t.TempDir(), "state", "decisions.jsonl")

	first, err := audit.To(path)
	if err != nil {
		t.Fatalf("To = %v, want it to create the directory", err)
	}
	if err := first.Record(audit.Entry{Command: "one", Decision: "allow"}); err != nil {
		t.Fatalf("Record = %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close = %v", err)
	}

	second, err := audit.To(path)
	if err != nil {
		t.Fatalf("reopening = %v", err)
	}
	if err := second.Record(audit.Entry{Command: "two", Decision: "deny"}); err != nil {
		t.Fatalf("Record = %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("Close = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the log: %v", err)
	}
	if !strings.Contains(string(content), `"one"`) {
		t.Errorf("reopening lost the earlier entry:\n%s", content)
	}
	if !strings.Contains(string(content), `"two"`) {
		t.Errorf("the second entry is missing:\n%s", content)
	}
}

// COVERS: FR-4.8 | negative
func TestARecorderWithNowhereToWriteIsNotAnError(t *testing.T) {
	t.Parallel()

	// A nil recorder is what a caller holds when the log could not be opened,
	// and it must stay usable: recording is not allowed to become the reason a
	// command is refused, so the call has to succeed at doing nothing.
	var nowhere *audit.Recorder

	if err := nowhere.Record(audit.Entry{Command: "git status"}); err != nil {
		t.Errorf("Record on a nil recorder = %v, want it to do nothing quietly", err)
	}
	if err := nowhere.Close(); err != nil {
		t.Errorf("Close on a nil recorder = %v", err)
	}
}

// COVERS: FR-4.8 | edge
func TestTheLogDirectoryIsCreatedAndNotWorldReadable(t *testing.T) {
	t.Parallel()

	// The log carries every command a session ran, which is not a thing to
	// leave readable by other users on principle, whatever this machine's
	// account list happens to look like today.
	path := filepath.Join(t.TempDir(), "nested", "deeper", "decisions.jsonl")

	log, err := audit.To(path)
	if err != nil {
		t.Fatalf("To = %v, want nested directories created", err)
	}
	defer func() { _ = log.Close() }()

	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat on the log directory: %v", err)
	}
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		t.Errorf("log directory mode = %#o, want nothing for group or other", mode)
	}
}
