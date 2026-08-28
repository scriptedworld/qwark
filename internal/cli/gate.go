package cli

import (
	"fmt"
	"io"

	"github.com/scriptedworld/qwark/internal/gate"
	"github.com/scriptedworld/qwark/internal/hook"
	"github.com/scriptedworld/qwark/internal/rules"
)

// runHook runs qwark as the hook itself: read one proposed call from stdin,
// judge it, and answer on stdout.
//
// This is the subcommand `install/settings-fragment.json` names, and until it
// existed everything else here was a way of asking qwark questions rather than
// a gate. `internal/hook.Run` was built and tested with nothing calling it.
//
// **A usage error here exits 2, which blocks.** That reads oddly for a usage
// error and is the only correct answer: qwark invoked without rule paths has
// not decided anything, and every other non-zero status is a
// `non_blocking_error` that lets the command run.
func runHook(paths []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(paths) == 0 {
		_, _ = fmt.Fprint(stderr,
			"qwark: hook wants at least one rules path\n"+
				"Without one there is no policy, and qwark has not decided anything.\n")
		return statusUsage
	}

	set, refusals := preflight(paths)
	if len(refusals) > 0 {
		return hook.Run(stdin, stdout, stderr, refusing(refusals))
	}

	return hook.Run(stdin, stdout, stderr, gate.Decider(set))
}

// preflight settles whether this rule set can be believed, before any command
// is judged against it.
//
// **Both questions are asked and both answers are reported.** A rule set can be
// rewritable and unparseable at once, and saying only the first sends its
// reader back for the second: the same reason a refusal lists every rule that
// objected rather than the first.
//
// Ownership comes first in the message because it is the graver finding: a
// parse error means the policy is broken, and a writable rule set means there
// is no policy, only a file that happened to say something this time.
func preflight(paths []string) (*rules.Set, []string) {
	var refusals []string

	if err := rules.CheckOwnership(paths); err != nil {
		refusals = append(refusals, untrusted(err))
	}

	set, err := rules.Load(paths)
	if err != nil {
		refusals = append(refusals, unloadable(err))
	}

	return set, refusals
}

// refusing answers every request with the same refusal, whatever it asks.
//
// A gate that cannot believe its own rule set has one thing to say and should
// say it to everything, rather than judging commands against a policy it has
// already reported as untrustworthy.
func refusing(refusals []string) hook.Decider {
	reason := joinReasons(refusals)
	return func(hook.Request) (hook.Decision, string) {
		return hook.DecisionDeny, reason
	}
}

func joinReasons(refusals []string) string {
	joined := ""
	for i, refusal := range refusals {
		if i > 0 {
			joined += "\n"
		}
		joined += refusal
	}
	return joined
}

// untrusted is the reason a rewritable rule set gives for permitting nothing.
//
// This is the refusal that matters most, and the one it would be most tempting
// to soften. qwark's whole premise is an agent constrained by rules it did not
// write; an agent that can edit them is constrained by nothing, and it needs no
// shell to do it. So a writable rule set is not a degraded gate to be run with
// a warning: it is the absence of one, and it has to say so by refusing.
//
// The fix is deployment rather than configuration, which is why the message
// says where to put the rules rather than what to change in them.
func untrusted(err error) string {
	return fmt.Sprintf(
		"qwark's rule set can be rewritten by the user qwark runs as, so it is "+
			"not a constraint on anything:\n  %v\n"+
			"Move the rules somewhere this user cannot write, a root-owned "+
			"directory such as /etc/qwark/rules, and name that path in the "+
			"hook registration.", err)
}

// unloadable is the reason a broken rule set gives for permitting nothing.
//
// A gate that becomes permissive when its own configuration is broken reports
// success while guarding nothing, so the answer is a refusal rather than a
// shrug. The cost is that a typo denies every command until it is fixed, which
// is why the message carries the parser's position and names a way out that
// does not itself need Bash: editing the rule file with the Edit tool.
func unloadable(err error) string {
	return fmt.Sprintf(
		"qwark's rule set will not load, so nothing is permitted:\n  %v\n"+
			"Fix the rule file with the Edit tool; that does not require Bash.", err)
}
