package hook_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/scriptedworld/qwark/internal/hook"
)

// payload is one request as Claude Code assembles it, with the field names read
// out of the installed binary rather than recalled.
const payload = `{
  "session_id": "1e422d1b",
  "transcript_path": "/home/user/.claude/projects/x/1e422d1b.jsonl",
  "cwd": "/home/user/.projects/qwark",
  "prompt_id": "p-1",
  "permission_mode": "default",
  "agent_id": "a-7",
  "agent_type": "Explore",
  "hook_event_name": "PreToolUse",
  "tool_name": "Bash",
  "tool_use_id": "t-9",
  "tool_input": {"command": "rm -rf x", "description": "remove", "timeout": 120}
}`

// COVERS: FR-10.1 | positive
func TestARequestIsReadAsItIsSent(t *testing.T) {
	t.Parallel()

	request, err := hook.Read(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("Read = %v", err)
	}

	cases := []struct {
		field string
		got   string
		want  string
	}{
		{field: "session_id", got: request.SessionID, want: "1e422d1b"},
		{field: "cwd", got: request.Cwd, want: "/home/user/.projects/qwark"},
		{field: "permission_mode", got: request.PermissionMode, want: "default"},
		{field: "hook_event_name", got: request.HookEventName, want: "PreToolUse"},
		{field: "tool_name", got: request.ToolName, want: "Bash"},
		{field: "tool_use_id", got: request.ToolUseID, want: "t-9"},
	}

	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.field, c.got, c.want)
		}
	}
}

// COVERS: FR-10.6 | positive
func TestTheAskingAgentIsIdentified(t *testing.T) {
	t.Parallel()

	// This is what makes a rule set that differs between a coding agent and a
	// reviewing agent implementable at all. Without it the mode would have to
	// arrive through the environment, which the agent can reach.
	request, err := hook.Read(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("Read = %v", err)
	}

	if request.AgentID != "a-7" {
		t.Errorf("agent_id = %q, want %q", request.AgentID, "a-7")
	}
	if request.AgentType != "Explore" {
		t.Errorf("agent_type = %q, want %q", request.AgentType, "Explore")
	}
}

// COVERS: FR-10.1 | positive
func TestTheBashCallIsReadFromTheToolInput(t *testing.T) {
	t.Parallel()

	request, err := hook.Read(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("Read = %v", err)
	}

	call, err := request.Bash()
	if err != nil {
		t.Fatalf("Bash = %v", err)
	}
	if call.Command != "rm -rf x" {
		t.Errorf("command = %q, want %q", call.Command, "rm -rf x")
	}
	if call.Background {
		t.Error("run_in_background reported for a call that did not set it")
	}
}

// COVERS: FR-10.2 | negative
func TestAPayloadThatCannotBeReadIsARefusal(t *testing.T) {
	t.Parallel()

	// Decoding a truncated pipe into a zero value would turn a broken
	// connection into an approval, since a zero Request names no tool and
	// carries no command.
	for _, body := range []string{"", "not json", `{"tool_input":`, "[]"} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()

			if _, err := hook.Read(strings.NewReader(body)); !errors.Is(err, hook.ErrNotJSON) {
				t.Errorf("Read(%q) = %v, want %v", body, err, hook.ErrNotJSON)
			}
		})
	}
}

// COVERS: FR-10.2 | negative
func TestAToolInputThatIsNotTheToolsIsARefusal(t *testing.T) {
	t.Parallel()

	request, err := hook.Read(strings.NewReader(
		`{"tool_name":"Bash","tool_input":"not an object"}`))
	if err != nil {
		t.Fatalf("Read = %v", err)
	}

	if _, err := request.Bash(); !errors.Is(err, hook.ErrNotJSON) {
		t.Errorf("Bash = %v, want %v", err, hook.ErrNotJSON)
	}
}

// COVERS: FR-10.4 | positive
func TestAReplyNamesTheEventItAnswers(t *testing.T) {
	t.Parallel()

	var out strings.Builder
	if err := hook.Answer(hook.DecisionDeny, "because").Write(&out); err != nil {
		t.Fatalf("Write = %v", err)
	}

	// Decoded rather than string-matched: the field names are the contract,
	// and a test that matched substrings would pass on a reply Claude Code
	// could not read.
	var reply struct {
		Specific struct {
			HookEventName            string `json:"hookEventName"`
			PermissionDecision       string `json:"permissionDecision"`
			PermissionDecisionReason string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(out.String()), &reply); err != nil {
		t.Fatalf("the reply is not readable JSON: %v", err)
	}

	if reply.Specific.HookEventName != "PreToolUse" {
		t.Errorf("hookEventName = %q, want PreToolUse", reply.Specific.HookEventName)
	}
	if reply.Specific.PermissionDecision != "deny" {
		t.Errorf("permissionDecision = %q, want deny", reply.Specific.PermissionDecision)
	}
	if reply.Specific.PermissionDecisionReason != "because" {
		t.Errorf("reason = %q, want it carried through",
			reply.Specific.PermissionDecisionReason)
	}
}

// COVERS: FR-10.5 | property
func TestTheDecisionsAreTheOnesClaudeCodeAccepts(t *testing.T) {
	t.Parallel()

	// Four, not three. `defer` is named here so that its absence from qwark's
	// answers is a decision on the record rather than an oversight.
	for _, decision := range []hook.Decision{
		hook.DecisionAllow, hook.DecisionDeny, hook.DecisionAsk, hook.DecisionDefer,
	} {
		if decision == "" {
			t.Error("a decision has no spelling")
		}
	}

	if hook.DecisionDefer != "defer" {
		t.Errorf("defer is spelled %q", hook.DecisionDefer)
	}
}

// COVERS: FR-10.7 | negative
func TestQwarkNeverRewritesTheCommand(t *testing.T) {
	t.Parallel()

	// Claude Code accepts `updatedInput` and will rewrite the tool call from
	// it. qwark does not send one: rewriting the subject's command would make
	// qwark the author of what runs, and a gate that edits what it judges can
	// no longer be said to have judged it.
	var out strings.Builder
	if err := hook.Answer(hook.DecisionAllow, "fine").Write(&out); err != nil {
		t.Fatalf("Write = %v", err)
	}

	var reply map[string]any
	if err := json.Unmarshal([]byte(out.String()), &reply); err != nil {
		t.Fatalf("the reply is not readable JSON: %v", err)
	}

	if _, rewrote := reply["updatedInput"]; rewrote {
		t.Error("the reply carries updatedInput at the top level")
	}
	specific, ok := reply["hookSpecificOutput"].(map[string]any)
	if !ok {
		t.Fatal("the reply carries no hookSpecificOutput")
	}
	if _, rewrote := specific["updatedInput"]; rewrote {
		t.Error("the reply carries updatedInput inside hookSpecificOutput")
	}
}

// COVERS: FR-10.5 | negative
func TestQwarkNeverAnswersDefer(t *testing.T) {
	t.Parallel()

	// `defer` means the hook declines to decide and the dispatcher continues
	// past it. Deciding nothing is the one outcome this design exists to
	// prevent, so it must never be what Answer produces by default or by
	// accident.
	for _, decision := range []hook.Decision{
		hook.DecisionAllow, hook.DecisionDeny, hook.DecisionAsk,
	} {
		var out strings.Builder
		if err := hook.Answer(decision, "r").Write(&out); err != nil {
			t.Fatalf("Write = %v", err)
		}
		if got := decisionIn(t, out.String()); got == string(hook.DecisionDefer) {
			t.Errorf("Answer(%q) produced a defer", decision)
		}
	}
}

// COVERS: FR-10.6a | edge
func TestAMainSessionCallCarriesNoAgentAndIsStillRead(t *testing.T) {
	t.Parallel()

	// **agent_id and agent_type appear only for a subagent.** A main-session
	// call carries neither, so anything requiring them would work for
	// subagents and quietly not for the session they run inside.
	//
	// REVISED 2026-08-20: this used to add that qwark need not care, because
	// which rule files apply is chosen outside. That was wrong: the
	// registration is fixed for a session, so a subagent inherits its parent's
	// command line and a per-agent partition chosen by the launcher collapses.
	//
	// What the emptiness asserted below is really worth is the opposite of a
	// gap. **The main session is the one caller reliably carrying no agent
	// type**, so absence identifies it exactly, and a rule can name that case
	// the way it names any other role (FR-7.13). This test is what makes that
	// dependable rather than assumed.
	const mainSession = `{
	  "session_id": "s-1",
	  "cwd": "/home/user/.projects/qwark",
	  "permission_mode": "default",
	  "hook_event_name": "PreToolUse",
	  "tool_name": "Bash",
	  "tool_input": {"command": "rm x"}
	}`

	request, err := hook.Read(strings.NewReader(mainSession))
	if err != nil {
		t.Fatalf("Read = %v", err)
	}

	if request.AgentID != "" || request.AgentType != "" {
		t.Errorf("agent fields = %q/%q, want both empty for a main-session call",
			request.AgentID, request.AgentType)
	}
	call, err := request.Bash()
	if err != nil {
		t.Fatalf("Bash = %v", err)
	}
	if call.Command != "rm x" {
		t.Errorf("command = %q, want the call to be readable regardless", call.Command)
	}
}
