// Package hook is qwark's end of the PreToolUse contract: what Claude Code
// sends, and what it will accept back.
//
// # Where this shape comes from
//
// **Read out of the installed binary**, Claude Code 2.1.233, a single-file
// executable with its JavaScript bundled in. The request is assembled there as:
//
//	{ session_id, transcript_path, cwd, prompt_id, permission_mode,
//	  agent_id, agent_type, effort,
//	  hook_event_name: "PreToolUse", tool_name, tool_input, tool_use_id }
//
// and the reply is validated against a schema naming `hookSpecificOutput` with
// `hookEventName`, `permissionDecision`, `permissionDecisionReason`,
// `updatedInput` and `additionalContext`.
//
// Two of those matter:
//
//   - **`agent_id` and `agent_type` are present.** Per-agent rule sets are
//     therefore implementable from the payload, rather than needing the mode to
//     be smuggled in through an environment variable the agent might reach.
//   - **`permissionDecision` has four values, not three.** Alongside allow, deny
//     and ask there is `defer`, which the dispatcher treats as "this hook
//     declines to decide" and continues past. It is the *no opinion* verdict,
//     precisely the one qwark never returns.
package hook

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// ErrNotJSON reports a payload that is not the contract. qwark cannot judge a
// call it cannot read, and reading a truncated payload as an empty one would
// turn a broken pipe into an approval.
var ErrNotJSON = errors.New("the tool call could not be read")

// EventPreToolUse names the event this package speaks for. Claude Code
// validates a reply against the event it asked about, so answering a different
// one is no answer at all.
const EventPreToolUse = "PreToolUse"

// ToolBash names the tool qwark's first mode gates.
const ToolBash = "Bash"

// A Request is one proposed tool call.
//
// Every field is what Claude Code sends; nothing here is derived. ToolInput is
// left raw because its shape is the tool's, and qwark only understands one of
// them.
type Request struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	PromptID       string `json:"prompt_id"`
	PermissionMode string `json:"permission_mode"`

	// AgentID and AgentType identify which agent is asking. These are what
	// make a rule set that differs between a coding agent and a reviewing
	// agent possible at all.
	AgentID   string `json:"agent_id"`
	AgentType string `json:"agent_type"`

	HookEventName string          `json:"hook_event_name"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input"`
	ToolUseID     string          `json:"tool_use_id"`
}

// A BashCall is the Bash tool's own input.
type BashCall struct {
	Command string `json:"command"`

	// Description is the caller's own account of what the command does.
	//
	// **Deliberately unconsulted.** 2026-08-20: asking each tool usage
	// to state its intention is the proxy's job, not this gate's. It is read
	// because reading the payload faithfully is the contract (FR-10.1), and it
	// is not a rule input that somebody forgot to wire up.
	//
	// It could not be one safely without care. This is the only field here the
	// subject authors freely: the command text is fixed by the shell grammar,
	// `agent_type` is assigned by the dispatcher, so a rule permitting on the
	// strength of it would be defeated by writing the expected words.
	Description string `json:"description"`
	Timeout     int    `json:"timeout"`
	Background  bool   `json:"run_in_background"`
}

// Read decodes one request from the stream Claude Code writes to.
func Read(from io.Reader) (Request, error) {
	body, err := io.ReadAll(from)
	if err != nil {
		return Request{}, fmt.Errorf("%w: %w", ErrNotJSON, err)
	}

	var request Request
	if err := json.Unmarshal(body, &request); err != nil {
		return Request{}, fmt.Errorf("%w: %w", ErrNotJSON, err)
	}
	return request, nil
}

// Bash reads the Bash tool's input out of a request.
func (r Request) Bash() (BashCall, error) {
	var call BashCall
	if err := json.Unmarshal(r.ToolInput, &call); err != nil {
		return BashCall{}, fmt.Errorf("%w: tool_input: %w", ErrNotJSON, err)
	}
	return call, nil
}

// A Decision is what qwark says back.
type Decision string

// The decisions Claude Code accepts.
//
// DecisionDefer is listed for completeness and is never returned: it means the
// hook declines to decide, and qwark deciding nothing is the one outcome its
// whole design exists to prevent.
const (
	DecisionAllow Decision = "allow"
	DecisionDeny  Decision = "deny"
	DecisionAsk   Decision = "ask"
	DecisionDefer Decision = "defer"
)

// A Reply is the JSON Claude Code reads back.
type Reply struct {
	Specific Specific `json:"hookSpecificOutput"`

	// SystemMessage is shown to the person rather than to the agent, so a
	// refusal can be visible without being something to argue with.
	SystemMessage string `json:"systemMessage,omitempty"`
}

// Specific is the per-event half of the reply.
type Specific struct {
	HookEventName            string   `json:"hookEventName"`
	PermissionDecision       Decision `json:"permissionDecision"`
	PermissionDecisionReason string   `json:"permissionDecisionReason"`
}

// Answer builds a reply for a PreToolUse call.
func Answer(decision Decision, reason string) Reply {
	return Reply{
		Specific: Specific{
			HookEventName:            EventPreToolUse,
			PermissionDecision:       decision,
			PermissionDecisionReason: reason,
		},
	}
}

// Write sends a reply.
//
// # The exit status is not the answer
//
// A hook that has decided exits 0 and puts its decision in this JSON. A
// non-zero exit says *the hook failed to run*, which is a different claim, and
// one that a caller may reasonably treat differently from a refusal.
//
// The predecessor of this project recorded that distinction in its own header
// and it still holds: "Exit 0 always: the decision travels in the JSON on
// stdout, per the PreToolUse contract. A non-zero exit would report that the
// hook failed to run, which is a different claim."
func (r Reply) Write(to io.Writer) error {
	encoded, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("encoding the decision: %w", err)
	}
	if _, err := to.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("writing the decision: %w", err)
	}
	return nil
}
