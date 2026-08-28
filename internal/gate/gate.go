// Package gate turns a rule set and a proposed tool call into a decision.
//
// It is the join between the two halves that otherwise do not know about each
// other: `hook` speaks the PreToolUse contract and knows nothing about rules,
// and `rules` judges commands and knows nothing about how it was asked. Neither
// is the right home for the other, and putting the join in the command line
// would make the thing that decides a detail of how qwark was invoked.
//
// **Whether the rule set may be trusted is deliberately not decided here.**
// That is a question about the filesystem qwark was deployed onto rather than
// about the call in front of it, and it is settled before a Decider is built;
// see the `hook` subcommand.
package gate

import (
	"fmt"
	"strings"

	"github.com/scriptedworld/qwark/internal/hook"
	"github.com/scriptedworld/qwark/internal/rules"
	"github.com/scriptedworld/qwark/internal/shell"
)

// Decider judges each request against one rule set.
//
// Tag changes are deliberately dropped. Tags are deferred to a later version
// and there is no store to put them in, so honouring some of the machinery and
// not the rest would make the half that works look like the whole of it.
func Decider(set *rules.Set) hook.Decider {
	return func(request hook.Request) (hook.Decision, string) {
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

		return Verdict(set, call.Command, request.AgentType)
	}
}

// Verdict judges one command as one agent, and says why.
func Verdict(set *rules.Set, command, agent string) (hook.Decision, string) {
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

// oneLine flattens a reason written as a paragraph onto a single line, since
// what carries it is a JSON string a model reads rather than a terminal.
func oneLine(reason string) string {
	return strings.Join(strings.Fields(reason), " ")
}

// wrongTool is the reason for a call qwark was never registered to judge.
//
// It refuses rather than waving the call through, on the same reasoning as any
// command form qwark does not model: finding no command to check is not the
// same as finding nothing to check. A matcher wide enough to send Write and
// Edit here will therefore block loudly, which is the failure worth having:
// the alternative is a gate that silently judges nothing while looking
// installed.
func wrongTool(name string) string {
	return fmt.Sprintf(
		"qwark gates Bash and was asked to judge %q, which it does not model.\n"+
			"Set the PreToolUse matcher to %q in settings.json; the Edit tool "+
			"reaches it without Bash.", name, hook.ToolBash)
}
