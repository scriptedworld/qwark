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

// ruleFile writes one rule file into a temporary directory and returns the path.
func ruleFile(t *testing.T, body string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "00.toml"), []byte(body), 0o600); err != nil {
		t.Fatalf("writing the rule file: %v", err)
	}
	return dir
}

// COVERS: FR-10.3 | positive
func TestTheHookDecidesAndSaysSoInTheJSON(t *testing.T) {
	t.Parallel()

	// **The exit status is not the answer.** A hook that has decided exits 0
	// and puts the decision in the JSON; a non-zero exit claims the hook failed
	// to run, which is a different thing and one the caller treats differently.
	//
	// Until this subcommand existed, everything else here was a way of asking
	// qwark questions. internal/hook.Run was built and tested with nothing
	// calling it, so no payload had ever produced a decision.
	out, errOut, status := invoke(t, payload(t, "git status"), "hook", "../../rules")

	if status != statusOK {
		t.Fatalf("status = %d, want %d; stderr = %q", status, statusOK, errOut)
	}

	decision, reason := decisionIn(t, out)
	if decision != "allow" {
		t.Errorf("decision = %q, want allow; reason = %q", decision, reason)
	}
	if !strings.Contains(reason, "allow-reading-the-repository") {
		t.Errorf("reason = %q, want it to name the rule that decided", reason)
	}
}

// COVERS: FR-4.25 | property
func TestTheHookReturnsEveryReasonRatherThanTheFirst(t *testing.T) {
	t.Parallel()

	// A refusal naming one problem out of three sends its reader round three
	// times. The evaluator gathers every reason precisely so it does not have
	// to, and that is only worth anything if the reply carries them.
	out, _, status := invoke(t,
		payload(t, "git push --force"), "hook", "../../rules")

	if status != statusOK {
		t.Fatalf("status = %d, want %d", status, statusOK)
	}

	decision, reason := decisionIn(t, out)
	if decision != "deny" {
		t.Fatalf("decision = %q, want deny", decision)
	}

	for _, want := range []string{
		"no-git-hook-running",
		"no-git-reaching-the-network",
		"accounted options only",
	} {
		if !strings.Contains(reason, want) {
			t.Errorf("reason does not mention %q; reason = %q", want, reason)
		}
	}
}

// COVERS: FR-10.3a | negative
func TestTheHookExitsTwoWhenItCannotDecide(t *testing.T) {
	t.Parallel()

	// **FACT 2026-08-20: exit 2 is the only status that blocks.** Exit 0 with
	// no JSON is no decision and the call proceeds; every other non-zero status
	// is a non_blocking_error and the call also proceeds. So a truncated
	// payload has to exit 2, or a broken connection becomes an approval.
	_, errOut, status := invoke(t, `{"tool_name"`, "hook", "../../rules")

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

// COVERS: FR-4.5, FR-4.6 | negative
func TestABrokenRuleSetPermitsNothingAndSaysWhere(t *testing.T) {
	t.Parallel()

	// A gate that becomes permissive when its own configuration is broken
	// reports success while guarding nothing. The cost is that a typo denies
	// every command until it is fixed -- so the refusal has to name where, and
	// the way out must not itself need Bash.
	broken := ruleFile(t, "[[rule]]\nid = \"unclosed\n")

	out, _, status := invoke(t, payload(t, "git status"), "hook", broken)

	if status != statusOK {
		t.Fatalf("status = %d, want a decision rather than a dead process", status)
	}

	decision, reason := decisionIn(t, out)
	if decision != "deny" {
		t.Errorf("decision = %q, want deny: a rule set that will not load "+
			"permits nothing", decision)
	}
	if !strings.Contains(reason, "00.toml") {
		t.Errorf("reason = %q, want it to name the file", reason)
	}
	if !strings.Contains(reason, "Edit tool") {
		t.Errorf("reason = %q, want a way out that does not need Bash -- which "+
			"is the one thing this refusal has taken away", reason)
	}
}

// COVERS: FR-4.12 | negative
func TestTheHookRefusesWhatItCannotParse(t *testing.T) {
	t.Parallel()

	// A command qwark cannot parse is one it cannot judge, and that is a
	// verdict rather than an absence of findings. The parser's own message goes
	// back because it carries the line and column.
	out, _, status := invoke(t, payload(t, "echo a )"), "hook", "../../rules")

	if status != statusOK {
		t.Fatalf("status = %d, want a decision", status)
	}

	decision, reason := decisionIn(t, out)
	if decision != "deny" {
		t.Errorf("decision = %q, want deny", decision)
	}
	if !strings.Contains(reason, "could not parse") {
		t.Errorf("reason = %q, want it to say parsing was what failed", reason)
	}
}

// COVERS: FR-4.20 | negative
func TestTheHookRefusesAToolItDoesNotModel(t *testing.T) {
	t.Parallel()

	// Finding no command to check is not the same as finding nothing to check.
	// A matcher wide enough to send Write here blocks loudly, which is the
	// failure worth having: the alternative is a gate that judges nothing while
	// looking installed.
	request := `{"hook_event_name":"PreToolUse","tool_name":"Write",` +
		`"tool_input":{"file_path":"/tmp/x","content":"y"}}`

	out, _, status := invoke(t, request, "hook", "../../rules")

	if status != statusOK {
		t.Fatalf("status = %d, want a decision", status)
	}

	decision, reason := decisionIn(t, out)
	if decision != "deny" {
		t.Errorf("decision = %q, want deny", decision)
	}
	if !strings.Contains(reason, "matcher") {
		t.Errorf("reason = %q, want it to name the misconfiguration and its fix",
			reason)
	}
}

// COVERS: FR-7.12 | positive
func TestTheHookJudgesAsTheAgentThePayloadNames(t *testing.T) {
	t.Parallel()

	// The payload's agent_type has to reach the evaluator, or the agent clause
	// is a mechanism nothing feeds. This is the wiring that makes separation of
	// duties real rather than expressible.
	rules := ruleFile(t, `
[command.echo]
operands = "text"

[[rule]]
id = "only-the-runner"
action = "allow"
reason = "The gate runner may echo."
  [[rule.clause]]
  agent = "gate-runner"
  [[rule.clause]]
  index = "0"
  value = "echo"
`)

	as := func(agent string) string {
		body, err := json.Marshal(map[string]any{
			"hook_event_name": "PreToolUse",
			"tool_name":       "Bash",
			"agent_type":      agent,
			"tool_input":      map[string]any{"command": "echo hello"},
		})
		if err != nil {
			t.Fatalf("building the payload: %v", err)
		}
		return string(body)
	}

	out, _, _ := invoke(t, as("gate-runner"), "hook", rules)
	if decision, reason := decisionIn(t, out); decision != "allow" {
		t.Errorf("decision = %q for the runner, want allow; reason = %q",
			decision, reason)
	}

	out, _, _ = invoke(t, as("file-writer"), "hook", rules)
	if decision, _ := decisionIn(t, out); decision != "deny" {
		t.Errorf("decision = %q for the writer, want deny", decision)
	}
}
