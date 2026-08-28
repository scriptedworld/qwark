package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// payload builds a PreToolUse request for one Bash command.
func payload(t *testing.T, command string) string {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"hook_event_name": "PreToolUse",
		"tool_name":       "Bash",
		"tool_input":      map[string]any{"command": command},
	})
	if err != nil {
		t.Fatalf("building the payload: %v", err)
	}
	return string(body)
}

// decisionIn reads the verdict and its reason out of a reply.
func decisionIn(t *testing.T, out string) (string, string) {
	t.Helper()

	var reply struct {
		Specific struct {
			Event    string `json:"hookEventName"`
			Decision string `json:"permissionDecision"`
			Reason   string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(out), &reply); err != nil {
		t.Fatalf("the reply is not valid JSON: %v\nreply was: %q", err, out)
	}

	// Claude Code validates a reply against the event it asked about, so
	// answering a different one is not a partial answer but none at all.
	if reply.Specific.Event != "PreToolUse" {
		t.Errorf("event = %q, want PreToolUse", reply.Specific.Event)
	}
	return reply.Specific.Decision, reply.Specific.Reason
}

// ruleFile writes a rule file into a directory this user can write, which is
// every directory a test can make.
//
// Until 2026-08-28 that was the one condition the hook refused outright, and
// these tests existed to prove it. FR-4.17 is retired, so a writable rule set
// now loads like any other and what remains here is the loading half. What the
// gate does once it has a rule set is `internal/gate`'s to prove, and it is a
// separate package for that reason.
func ruleFile(t *testing.T, body string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "00.toml"), []byte(body), 0o600); err != nil {
		t.Fatalf("writing the rule file: %v", err)
	}
	return dir
}

// COVERS: FR-4.5, FR-4.6 | negative
func TestABrokenRuleSetPermitsNothingAndSaysWhere(t *testing.T) {
	t.Parallel()

	// A gate that becomes permissive when its own configuration is broken
	// reports success while guarding nothing. The cost is that a typo denies
	// every command until it is fixed, so the refusal has to name where, and
	// the way out must not itself need Bash.
	//
	// This once asserted that BOTH faults were reported, the rule set being
	// unparseable and rewritable at once. FR-4.17 is retired, so loading is the
	// only fault preflight can now find and there is no second one to list.
	broken := ruleFile(t, "[[rule]]\nid = \"unclosed\n")

	out, _, status := invoke(t, payload(t, "git status"), "hook", broken)

	if status != statusOK {
		t.Fatalf("status = %d, want a decision rather than a dead process", status)
	}

	decision, reason := decisionIn(t, out)
	if decision != "deny" {
		t.Errorf("decision = %q, want deny", decision)
	}
	if !strings.Contains(reason, "00.toml") {
		t.Errorf("reason = %q, want it to name the file", reason)
	}
	if !strings.Contains(reason, "Edit tool") {
		t.Errorf("reason = %q, want a way out that does not need Bash, which "+
			"is the one thing this refusal has taken away", reason)
	}
}

// COVERS: FR-10.3a | negative
func TestTheHookExitsTwoWhenItCannotDecide(t *testing.T) {
	t.Parallel()

	// **Exit 2 is the only status that blocks.** Exit 0 with
	// no JSON is no decision and the call proceeds; every other non-zero status
	// is a non_blocking_error and the call also proceeds. So a truncated
	// payload has to exit 2, or a broken connection becomes an approval.
	_, errOut, status := invoke(t, `{"tool_name"`, "hook", ruleFile(t, ""))

	if status != 2 {
		t.Errorf("status = %d, want 2: any other status lets the command run", status)
	}
	if !strings.Contains(errOut, "fault in the gate") {
		t.Errorf("stderr = %q, want it to distinguish a broken gate from a "+
			"judgement about the command", errOut)
	}
}

// COVERS: FR-10.3a | edge
func TestTheHookWithNoRulesPathBlocks(t *testing.T) {
	t.Parallel()

	// A usage error that blocks reads oddly and is the only correct answer.
	// qwark invoked without a policy has not decided anything, and the statuses
	// that conventionally mean "you used me wrongly" all let the command run.
	_, errOut, status := invoke(t, payload(t, "rm x"), "hook")

	if status != 2 {
		t.Errorf("status = %d, want 2", status)
	}
	if !strings.Contains(errOut, "rules path") {
		t.Errorf("stderr = %q, want it to say what was missing", errOut)
	}
}
