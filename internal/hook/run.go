package hook

import (
	"errors"
	"fmt"
	"io"
)

// Exit statuses, chosen for what Claude Code does with them rather than for
// what they conventionally mean.
//
// Confirmed in the installed binary and the documentation. Both of the obvious
// failure exits are permissive:
//
//	0 with a decision   the decision is honoured
//	0 with no JSON      no decision; the normal permission flow proceeds
//	2                   the call is blocked, stderr goes back to the model
//	any other non-zero  a `non_blocking_error`, SO THE TOOL PROCEEDS
//
// So a gate that crashes, or that exits 1 the way a Unix program usually
// reports failure, lets the command through. Exit 2 is the only status that
// refuses on qwark's behalf when qwark itself has broken, which makes it the
// only honest failure exit here.
const (
	StatusDecided = 0
	StatusBroken  = 2
)

// ErrPanicked reports that judging a command raised a panic. It is a sentinel
// so that a caller can tell a fault in the gate from a fault in the payload,
// which are different things to go and look at.
var ErrPanicked = errors.New("qwark panicked while judging")

// A Decider judges one request. It returns the decision and the reason for it.
type Decider func(Request) (Decision, string)

// Run reads a request, decides, and answers.
//
// **It emits a decision or exits 2, never neither.** Every path out of here
// ends in one or the other, because the two ways of ending in nothing are both
// ways of letting the command run.
//
// A panic in the decider is caught for the same reason. A gate that dies
// mid-judgement has not permitted anything, but Claude Code cannot tell that
// from a gate that had no opinion, and it resolves the ambiguity in the
// command's favour.
func Run(in io.Reader, out, errOut io.Writer, decide Decider) (status int) {
	defer func() {
		if panicked := recover(); panicked != nil {
			status = broke(errOut, fmt.Errorf("%w: %v", ErrPanicked, panicked))
		}
	}()

	request, err := Read(in)
	if err != nil {
		return broke(errOut, err)
	}

	decision, reason := decide(request)

	if err := Answer(decision, reason).Write(out); err != nil {
		// The decision was reached and could not be delivered. Exiting 0 here
		// would deliver silence, which reads as no decision.
		return broke(errOut, err)
	}

	return StatusDecided
}

// broke reports that qwark could not decide, in the only way that refuses.
//
// The message goes to stderr because that is what exit 2 feeds back, and it
// says qwark broke rather than that the command was wrong: those are
// different claims and only one of them is the reader's to act on.
func broke(errOut io.Writer, cause error) int {
	_, _ = fmt.Fprintf(errOut,
		"qwark could not decide, so it refused: %v\n"+
			"This is a fault in the gate, not a judgement about the command.\n",
		cause)
	return StatusBroken
}
