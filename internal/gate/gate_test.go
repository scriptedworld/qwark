package gate_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scriptedworld/qwark/internal/gate"
	"github.com/scriptedworld/qwark/internal/hook"
	"github.com/scriptedworld/qwark/internal/rules"
)

// setFrom loads a rule set from one file written for the test.
//
// Whether that file could be rewritten by this user is not asked here, and that
// is the point of the split: ownership is a question about the machine qwark
// was deployed onto, settled once before a Decider is built. Asking it here
// would mean these tests could only ever reach the refusal.
func setFrom(t *testing.T, body string) *rules.Set {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "00.toml"), []byte(body), 0o600); err != nil {
		t.Fatalf("writing the rule file: %v", err)
	}

	set, err := rules.Load([]string{dir})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}
	return set
}

// request builds a PreToolUse request for one Bash command by one agent.
func request(t *testing.T, agent, command string) hook.Request {
	t.Helper()

	input, err := json.Marshal(map[string]any{"command": command})
	if err != nil {
		t.Fatalf("building the tool input: %v", err)
	}
	return hook.Request{
		HookEventName: "PreToolUse",
		ToolName:      hook.ToolBash,
		AgentType:     agent,
		ToolInput:     input,
	}
}

// permissive declares echo and permits it, so a decision other than deny is
// reachable at all. Deny is the engine's default, so a rule set that permits
// nothing makes every test look like it passed.
const permissive = `
[command.echo]
operands = "text"

[[rule]]
id = "may-echo"
action = "allow"
reason = "Echo says something and changes nothing."
  [[rule.clause]]
  index = "0"
  value = "echo"

[[rule]]
id = "no-shouting"
action = "deny"
reason = "Shouting is not permitted."
  [[rule.clause]]
  value = "LOUD"

[[rule]]
id = "nothing-loud-first"
action = "deny"
reason = "The first argument may not be that."
  [[rule.clause]]
  index = "1"
  value = "LOUD"
`

// COVERS: FR-4.28 | positive
func TestADecisionNamesItsRuleAndQuotesTheCause(t *testing.T) {
	t.Parallel()

	// A decision nobody can check is one nobody can correct. The reply is the
	// only place the reader sees any of this, so naming the rule and quoting
	// what set it off has to survive into the message rather than stopping at
	// the Outcome.
	decision, reason := gate.Decider(setFrom(t, permissive))(
		request(t, "", "echo hello"))

	if decision != hook.DecisionAllow {
		t.Fatalf("decision = %q, want allow; reason = %q", decision, reason)
	}
	if !strings.Contains(reason, "may-echo") {
		t.Errorf("reason = %q, want it to name the rule that decided", reason)
	}
	if !strings.Contains(reason, "caused by: echo") {
		t.Errorf("reason = %q, want it to quote what satisfied the rule", reason)
	}
}

// COVERS: FR-4.25 | property
func TestEveryReasonReachesTheReply(t *testing.T) {
	t.Parallel()

	// A refusal naming one problem out of two sends its reader round twice. The
	// evaluator gathers every reason precisely so it does not have to, and that
	// is worth nothing unless the reply carries them all.
	//
	// Only the DENY reasons, which is the requirement's own wording and is
	// deliberate: `may-echo` also matched this command, and reporting that a
	// command was refused and permitted in one message tells its reader nothing
	// about which to act on.
	decision, reason := gate.Decider(setFrom(t, permissive))(
		request(t, "", "echo LOUD"))

	if decision != hook.DecisionDeny {
		t.Fatalf("decision = %q, want deny", decision)
	}
	if strings.Contains(reason, "may-echo") {
		t.Errorf("reason = %q, want the outranked allow rule left out", reason)
	}
	for _, want := range []string{"no-shouting", "nothing-loud-first"} {
		if !strings.Contains(reason, want) {
			t.Errorf("reason does not mention %q; reason = %q", want, reason)
		}
	}
}

// COVERS: FR-4.12 | negative
func TestACommandThatWillNotParseIsRefused(t *testing.T) {
	t.Parallel()

	// A command qwark cannot parse is one it cannot judge, and that is a
	// verdict rather than an absence of findings. The parser's own message goes
	// back because it carries the line and column.
	decision, reason := gate.Decider(setFrom(t, permissive))(
		request(t, "", "echo a )"))

	if decision != hook.DecisionDeny {
		t.Errorf("decision = %q, want deny", decision)
	}
	if !strings.Contains(reason, "could not parse") {
		t.Errorf("reason = %q, want it to say parsing was what failed", reason)
	}
}

// COVERS: FR-4.20 | negative
func TestACallForAToolQwarkDoesNotModelIsRefused(t *testing.T) {
	t.Parallel()

	// Finding no command to check is not the same as finding nothing to check.
	// A matcher wide enough to send Write here blocks loudly, which is the
	// failure worth having: the alternative is a gate that judges nothing while
	// looking installed.
	asked := request(t, "", "")
	asked.ToolName = "Write"

	decision, reason := gate.Decider(setFrom(t, permissive))(asked)

	if decision != hook.DecisionDeny {
		t.Errorf("decision = %q, want deny", decision)
	}
	if !strings.Contains(reason, "matcher") {
		t.Errorf("reason = %q, want it to name the misconfiguration and its fix",
			reason)
	}
}

// COVERS: FR-10.2 | negative
func TestAToolInputThatIsNotABashCallIsRefused(t *testing.T) {
	t.Parallel()

	// The payload said Bash and did not carry a Bash call. Decoding that into a
	// zero value would judge an empty command nobody sent, which is how a
	// broken payload turns into an approval.
	asked := request(t, "", "")
	asked.ToolInput = json.RawMessage(`["not", "an", "object"]`)

	decision, reason := gate.Decider(setFrom(t, permissive))(asked)

	if decision != hook.DecisionDeny {
		t.Errorf("decision = %q, want deny", decision)
	}
	if !strings.Contains(reason, "could not read the Bash call") {
		t.Errorf("reason = %q, want it to say the call itself was unreadable",
			reason)
	}
}

// COVERS: FR-7.12 | positive
func TestTheAgentTypeFromThePayloadReachesTheRules(t *testing.T) {
	t.Parallel()

	// The agent clause is a mechanism nothing feeds unless the payload's
	// agent_type arrives here. This is the wiring that makes separation of
	// duties real rather than merely expressible.
	const perAgent = `
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
`
	decide := gate.Decider(setFrom(t, perAgent))

	decision, reason := decide(request(t, "gate-runner", "echo hi"))
	if decision != hook.DecisionAllow {
		t.Errorf("decision = %q for the runner, want allow; reason = %q",
			decision, reason)
	}
	if decision, _ := decide(request(t, "file-writer", "echo hi")); decision != hook.DecisionDeny {
		t.Errorf("decision = %q for the writer, want deny", decision)
	}
}

// COVERS: FR-4.1 | edge
func TestAnAskReachesTheReplyAsAnAsk(t *testing.T) {
	t.Parallel()

	// Ask is the refusal a person can lift, and it is a different answer from
	// deny rather than a softer wording of it. Collapsing the two here would
	// turn every confirmable command into a refused one.
	const asking = `
[command.echo]
operands = "text"

[[rule]]
id = "may-echo"
action = "allow"
reason = "Echo says something and changes nothing."
  [[rule.clause]]
  index = "0"
  value = "echo"

[[rule]]
id = "check-first"
action = "ask"
reason = "Confirm this is the word you meant."
  [[rule.clause]]
  value = "maybe"
`
	decision, reason := gate.Decider(setFrom(t, asking))(request(t, "", "echo maybe"))

	if decision != hook.DecisionAsk {
		t.Fatalf("decision = %q, want ask", decision)
	}
	if !strings.Contains(reason, "confirmed") {
		t.Errorf("reason = %q, want it to say a person is being asked", reason)
	}
}
