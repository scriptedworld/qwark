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
// It reports a list rather than one error because a refusal names everything
// wrong at once, on the same reasoning that has a verdict list every rule that
// objected rather than the first. Only loading can fail today.
//
// **Whether the running user could rewrite the rule set is deliberately not
// asked.** It was, until 2026-08-28, and the check cost more than it bought:
// enforcing it means a root-owned live set, so every change to a rule needs a
// root command and the gate cannot be developed by the session it gates. What
// holds the property now sits outside qwark, in the rule that an agent does not
// edit these files without a person, and in the `permissions.deny` twin that
// keeps the Write and Edit tools off them. See FR-4.17, retired, in
// REQUIREMENTS.md, which records the condition for bringing it back.
func preflight(paths []string) (*rules.Set, []string) {
	var refusals []string

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
