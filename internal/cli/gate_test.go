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
// That is exactly the condition the hook refuses, so these tests are about the
// refusals. What the gate does once it believes its rule set is
// `internal/gate`'s to prove, and it is a separate package for that reason.
func ruleFile(t *testing.T, body string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "00.toml"), []byte(body), 0o600); err != nil {
		t.Fatalf("writing the rule file: %v", err)
	}
	return dir
}

// COVERS: FR-4.17 | negative
func TestARewritableRuleSetPermitsNothing(t *testing.T) {
	t.Parallel()

	// The refusal that matters most, and the one that was easiest not to
	// notice: CheckOwnership was written, tested and exported, and called by
	// nothing, so the gate would happily load rules the subject can edit.
	//
	// An agent constrained by rules it can rewrite is constrained by nothing,
	// and it needs no shell to rewrite them: Write and Edit reach those files
	// without passing through this gate at all. So a writable rule set is not a
	// degraded gate to run with a warning: it is the absence of one.
	//
	// This is also why the repository's own `rules/` cannot be the installed
	// path. A draft is writable by whoever is drafting it.
	writable := ruleFile(t, "[[rule]]\nid = \"x\"\naction = \"deny\"\n"+
		"reason = \"y\"\n  [[rule.clause]]\n  index = \"0\"\n  value = \"z\"\n")

	out, _, status := invoke(t, payload(t, "git status"), "hook", writable)

	if status != statusOK {
		t.Fatalf("status = %d, want a decision rather than a dead process", status)
	}

	decision, reason := decisionIn(t, out)
	if decision != "deny" {
		t.Errorf("decision = %q, want deny for a rule set this user can rewrite",
			decision)
	}
	if !strings.Contains(reason, "rewritten by the user qwark runs as") {
		t.Errorf("reason = %q, want it to say the rule set is not a constraint",
			reason)
	}
	if !strings.Contains(reason, "root-owned") {
		t.Errorf("reason = %q, want it to say where the rules should live: the "+
			"fix is deployment, not configuration", reason)
	}
}

// COVERS: FR-4.5, FR-4.6 | negative
func TestABrokenRuleSetPermitsNothingAndSaysWhere(t *testing.T) {
	t.Parallel()

	// A gate that becomes permissive when its own configuration is broken
	// reports success while guarding nothing. The cost is that a typo denies
	// every command until it is fixed, so the refusal has to name where, and
	// the way out must not itself need Bash.
	//
	// Both faults are reported. This rule set is unparseable AND rewritable,
	// and saying only the graver one sends its reader back for the other:
	// the same reason a refusal lists every rule that objected rather than the
	// first one.
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
	if !strings.Contains(reason, "rewritten by the user") {
		t.Errorf("reason = %q, want both faults reported, not only the graver one",
			reason)
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
