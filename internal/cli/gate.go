package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/scriptedworld/qwark/internal/hook"
	"github.com/scriptedworld/qwark/internal/rules"
	"github.com/scriptedworld/qwark/internal/shell"
)

// gate runs qwark as the hook itself: read one proposed call from stdin, judge
// it, and answer on stdout.
//
// This is the subcommand `install/settings-fragment.json` names, and until it
// existed everything else here was a way of asking qwark questions rather than
// a gate. `internal/hook.Run` was built and tested with nothing calling it.
//
// **A usage error here exits 2, which blocks.** That reads oddly for a usage
// error and is the only correct answer: qwark invoked without rule paths has
// not decided anything, and every other non-zero status is a
// `non_blocking_error` that lets the command run.
func gate(paths []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(paths) == 0 {
		_, _ = fmt.Fprint(stderr,
			"qwark: hook wants at least one rules path\n"+
				"Without one there is no policy, and qwark has not decided anything.\n")
		return statusUsage
	}

	return hook.Run(stdin, stdout, stderr, decider(paths))
}

// decider loads the rule set and returns what judges each request.
//
// The load happens once, outside the returned function, because a hook process
// handles exactly one call. A failure is carried rather than returned so that
// it becomes a refusal with a reason the reader can act on, instead of a dead
// process whose exit status says only that qwark broke.
func decider(paths []string) hook.Decider {
	set, loadErr := rules.Load(paths)

	return func(request hook.Request) (hook.Decision, string) {
		if loadErr != nil {
			return hook.DecisionDeny, unloadable(loadErr)
		}
		if request.ToolName != hook.ToolBash {
			return hook.DecisionDeny, wrongTool(request.ToolName)
		}

		call, err := request.Bash()
		if err != nil {
			// The payload named Bash and did not carry a Bash call. Reading
			// that as an empty command would judge something nobody sent.
			return hook.DecisionDeny, fmt.Sprintf(
				"qwark could not read the Bash call it was asked about:\n  %v", err)
		}

		return verdict(set, call.Command, request.AgentType)
	}
}

// verdict judges one command and says why.
//
// Tag changes are deliberately dropped. Tags are deferred to a later version
// and there is no store to put them in, so honouring some of the machinery and
// not the rest would make the half that works look like the whole of it.
func verdict(set *rules.Set, command, agent string) (hook.Decision, string) {
	parsed, err := shell.Parse(command)
	if err != nil {
		// A command qwark cannot parse is one it cannot judge, and that is a
		// verdict rather than an absence of findings. The parser's own message
		// is what goes back, because it carries the line and column.
		return hook.DecisionDeny, fmt.Sprintf(
			"qwark could not parse this command, so it cannot judge it:\n  %v", err)
	}

	outcome := set.Evaluate(parsed, rules.Context{Agent: agent})
	return decisionOf(outcome.Action), explain(outcome)
}

// decisionOf maps a verdict onto what Claude Code accepts.
//
// The tagging actions decide nothing and are settled out before this, but they
// are named rather than left to a default: an action that reached here without
// being a decision must refuse, since permitting on the strength of an
// unrecognised verdict is the one direction that cannot be taken back.
func decisionOf(action rules.Action) hook.Decision {
	switch action {
	case rules.ActionAllow:
		return hook.DecisionAllow
	case rules.ActionAsk:
		return hook.DecisionAsk
	case rules.ActionDeny, rules.ActionTag, rules.ActionUntag:
		return hook.DecisionDeny
	default:
		return hook.DecisionDeny
	}
}

// explain renders every reason behind a verdict.
//
// **Every reason, not the first.** A refusal that names one problem out of
// three sends its reader round three times, and the evaluator gathers them all
// precisely so that it does not have to.
func explain(outcome rules.Outcome) string {
	var out strings.Builder
	out.WriteString(heading(outcome.Action))

	for _, finding := range outcome.Findings {
		_, _ = fmt.Fprintf(&out, "\n  %s: %s", finding.Rule, oneLine(finding.Reason))
		if finding.Cause != "" {
			_, _ = fmt.Fprintf(&out, "\n    caused by: %s", finding.Cause)
		}
	}

	return out.String()
}

// heading says what happened before the reasons say why.
func heading(action rules.Action) string {
	switch action {
	case rules.ActionAllow:
		return "qwark permitted this command."
	case rules.ActionAsk:
		return "qwark wants this confirmed before it runs."
	case rules.ActionDeny, rules.ActionTag, rules.ActionUntag:
		return "qwark refused this command."
	default:
		return "qwark refused this command."
	}
}

// unloadable is the reason a broken rule set gives for permitting nothing.
//
// A gate that becomes permissive when its own configuration is broken reports
// success while guarding nothing, so the answer is a refusal rather than a
// shrug. The cost is that a typo denies every command until it is fixed, which
// is why the message carries the parser's position and names a way out that
// does not itself need Bash -- editing the rule file with the Edit tool.
func unloadable(err error) string {
	return fmt.Sprintf(
		"qwark's rule set will not load, so nothing is permitted:\n  %v\n"+
			"Fix the rule file with the Edit tool; that does not require Bash.", err)
}

// wrongTool is the reason for a call qwark was never registered to judge.
//
// It refuses rather than waving the call through, on the same reasoning as any
// command form qwark does not model: finding no command to check is not the
// same as finding nothing to check. A matcher wide enough to send Write and
// Edit here will therefore block loudly, which is the failure worth having --
// the alternative is a gate that silently judges nothing while looking
// installed.
func wrongTool(name string) string {
	return fmt.Sprintf(
		"qwark gates Bash and was asked to judge %q, which it does not model.\n"+
			"Set the PreToolUse matcher to %q in settings.json; the Edit tool "+
			"reaches it without Bash.", name, hook.ToolBash)
}
