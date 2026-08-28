// Package gate turns a rule set and a proposed tool call into a decision.
//
// It is the join between the two halves that otherwise do not know about each
// other: `hook` speaks the PreToolUse contract and knows nothing about rules,
// and `rules` judges commands and knows nothing about how it was asked. Neither
// is the right home for the other, and putting the join in the command line
// would make the thing that decides a detail of how qwark was invoked.
//
// **Whether the rule set may be trusted is deliberately not decided here.**
// That is a question about the deployment rather than about the call in front
// of it. Until 2026-08-28 it was answered by a permission check run before a
// Decider was built; it is now answered outside qwark entirely, by the rule
// that an agent does not edit these files without a person and by the
// `permissions.deny` twin in the hook registration. FR-4.17, retired, records
// what would bring the check back.
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
	judged := Judged(set)

	return func(request hook.Request) (hook.Decision, string) {
		decision, reason, _ := judged(request)
		return decision, reason
	}
}

// A Judge answers like a Decider and also names the rules that produced the
// answer.
type Judge func(hook.Request) (hook.Decision, string, []string)

// Judged is Decider with the rule names kept.
//
// The reason string is written for a model to read and is deliberately prose.
// The names are for the record: an entry in the log saying a command was denied
// without saying which rule did it cannot be counted, grouped, or compared
// against the same command under a different rule set, which is the whole
// purpose of writing it down.
//
// The engine's own refusals are named here too, as `(engine) …`, because a
// refusal for being unparseable or for naming a tool qwark does not model is
// still a decision somebody will want to count.
func Judged(set *rules.Set) Judge {
	return func(request hook.Request) (hook.Decision, string, []string) {
		if request.ToolName != hook.ToolBash {
			return hook.DecisionDeny, wrongTool(request.ToolName),
				[]string{"(engine) tool not modelled"}
		}

		call, err := request.Bash()
		if err != nil {
			// The payload named Bash and did not carry a Bash call. Reading
			// that as an empty command would judge something nobody sent.
			return hook.DecisionDeny, fmt.Sprintf(
					"qwark could not read the Bash call it was asked about:\n  %v", err),
				[]string{"(engine) unreadable payload"}
		}

		parsed, err := shell.Parse(call.Command)
		if err != nil {
			return hook.DecisionDeny, fmt.Sprintf(
					"qwark could not parse this command, so it cannot judge it:\n  %v", err),
				[]string{"(engine) unparseable"}
		}

		outcome := set.Evaluate(parsed, rules.Context{Agent: request.AgentType})
		return decisionOf(outcome.Action), explain(outcome), named(outcome)
	}
}

// named lists the rules behind a verdict, in the order they were collected.
func named(outcome rules.Outcome) []string {
	names := make([]string, 0, len(outcome.Findings))
	for _, finding := range outcome.Findings {
		names = append(names, finding.Rule)
	}
	return names
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
