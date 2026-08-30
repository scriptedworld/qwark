package hook_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/scriptedworld/qwark/internal/hook"
)

// decisionIn reads the verdict back out of what was written.
func decisionIn(t *testing.T, written string) string {
	t.Helper()

	var reply struct {
		Specific struct {
			PermissionDecision string `json:"permissionDecision"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(written), &reply); err != nil {
		t.Fatalf("the reply is not readable JSON: %v", err)
	}
	return reply.Specific.PermissionDecision
}

// COVERS: FR-10.3 | positive
func TestADecisionExitsZeroAndTravelsInTheJSON(t *testing.T) {
	t.Parallel()

	var out, errOut strings.Builder
	status := hook.Run(strings.NewReader(payload), &out, &errOut,
		func(hook.Request) (hook.Decision, string) {
			return hook.DecisionDeny, "because"
		})

	if status != hook.StatusDecided {
		t.Errorf("status = %d, want %d for a decision", status, hook.StatusDecided)
	}
	if got := decisionIn(t, out.String()); got != "deny" {
		t.Errorf("decision = %q, want deny", got)
	}
	if errOut.String() != "" {
		t.Errorf("stderr = %q, want nothing when the gate worked", errOut.String())
	}
}

// COVERS: FR-10.3a, FR-10.3b | negative
func TestAnUnreadablePayloadExitsTwo(t *testing.T) {
	t.Parallel()

	// Exiting 0 with nothing, or exiting 1, would both let the command run.
	var out, errOut strings.Builder
	status := hook.Run(strings.NewReader("not json"), &out, &errOut,
		func(hook.Request) (hook.Decision, string) {
			t.Error("the decider ran on a payload that could not be read")
			return hook.DecisionAllow, ""
		})

	if status != hook.StatusBroken {
		t.Errorf("status = %d, want %d, the only status that refuses",
			status, hook.StatusBroken)
	}
	if out.String() != "" {
		t.Errorf("stdout = %q, want nothing", out.String())
	}
	if !strings.Contains(errOut.String(), "fault in the gate") {
		t.Errorf("stderr = %q, want it to distinguish a broken gate from a bad command",
			errOut.String())
	}
}

// COVERS: FR-10.3b | negative
func TestAPanicExitsTwoRatherThanLettingTheCommandThrough(t *testing.T) {
	t.Parallel()

	// A gate that dies mid-judgement has permitted nothing, but Claude Code
	// cannot tell that from a gate with no opinion, and resolves the ambiguity
	// in the command's favour.
	var out, errOut strings.Builder
	status := hook.Run(strings.NewReader(payload), &out, &errOut,
		func(hook.Request) (hook.Decision, string) {
			panic("something went wrong deep inside")
		})

	if status != hook.StatusBroken {
		t.Errorf("status = %d, want %d after a panic", status, hook.StatusBroken)
	}
	if !strings.Contains(errOut.String(), "panicked") {
		t.Errorf("stderr = %q, want it to say what happened", errOut.String())
	}
}

// errCannotWrite is what a stdout that has gone away reports.
var errCannotWrite = errors.New("stdout is gone")

// failingWriter stands in for a stdout that cannot be written to. A real
// implementation rather than a mock: there is nothing to assert about how it
// was called, only what the code does with what it returns.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errCannotWrite
}

// COVERS: FR-10.3b | negative
func TestADecisionThatCannotBeDeliveredExitsTwo(t *testing.T) {
	t.Parallel()

	// The decision was reached and could not be delivered. Exiting 0 would
	// deliver silence, which reads as no decision at all.
	var errOut strings.Builder
	status := hook.Run(strings.NewReader(payload), failingWriter{}, &errOut,
		func(hook.Request) (hook.Decision, string) {
			return hook.DecisionDeny, "because"
		})

	if status != hook.StatusBroken {
		t.Errorf("status = %d, want %d", status, hook.StatusBroken)
	}
}

// COVERS: FR-10.3 | property
func TestEveryPathEndsInADecisionOrARefusal(t *testing.T) {
	t.Parallel()

	// The property the whole file exists for: there is no way out of Run that
	// leaves the command neither judged nor blocked.
	cases := []struct {
		name   string
		body   string
		decide hook.Decider
	}{
		{
			name: "decided", body: payload,
			decide: func(hook.Request) (hook.Decision, string) {
				return hook.DecisionAllow, ""
			},
		},
		{name: "unreadable", body: "{", decide: nil},
		{
			name: "panicked", body: payload,
			decide: func(hook.Request) (hook.Decision, string) { panic("x") },
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			decide := c.decide
			if decide == nil {
				decide = func(hook.Request) (hook.Decision, string) {
					return hook.DecisionAllow, ""
				}
			}

			var out, errOut strings.Builder
			status := hook.Run(strings.NewReader(c.body), &out, &errOut, decide)

			decided := status == hook.StatusDecided && out.String() != ""
			refused := status == hook.StatusBroken
			if !decided && !refused {
				t.Errorf("status = %d with stdout %q: neither a decision nor a refusal",
					status, out.String())
			}
		})
	}
}
