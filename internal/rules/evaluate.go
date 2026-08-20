package rules

import (
	"slices"

	"github.com/scriptedworld/qwark/internal/command"
	"github.com/scriptedworld/qwark/internal/shell"
)

// A Context is what an evaluation knows besides the command itself.
type Context struct {
	// Tags live at this moment, whether set by an earlier command or
	// calculated afresh from the world.
	Tags map[string]bool
}

// A Finding is one rule that applied, with what it said and what set it off.
type Finding struct {
	Rule   string
	Action Action
	Reason string
	Cause  string
}

// A TagChange is a tag a rule wants set or cleared.
type TagChange struct {
	Name string
	TTL  int
	Set  bool
}

// An Outcome is the verdict, everything that produced it, and what should
// happen to the tags afterwards.
type Outcome struct {
	Action   Action
	Findings []Finding
	Tags     []TagChange
}

// Reasons the engine refuses without consulting a rule. These are not rules
// because a rule file could omit them, and each is a precondition of qwark
// being able to judge anything at all.
const (
	reasonNotOneCommand = "One command at a time. This call holds more than one, " +
		"however they were joined -- by a sequence, a pipe, a logical " +
		"concatenation, or a substitution carrying a command of its own."
	reasonUndeclared = "This command has no declaration, so qwark cannot account " +
		"for what its options mean or which of its words are paths. Nothing runs " +
		"unless it has been described."
	reasonNothingAllowed = "Nothing permitted this. Being allowed means an allow " +
		"rule matched, and none did."
)

// Ids for the refusals the engine makes itself, so a message can name what
// refused just as it names a rule.
const (
	ruleOneCommand  = "(engine) one command at a time"
	ruleDeclared    = "(engine) declared commands only"
	ruleNoAllowance = "(engine) deny by default"
)

// Evaluate judges one parsed command.
//
// **Deny is the default and it is the engine's, not a rule's.** A command is
// refused unless an allow rule matched it: being in the allowed list *means* an
// allow rule matched. A rule set containing no allow rules therefore permits
// nothing, which is the correct reading of an empty policy.
//
// A deny settles the verdict -- nothing stricter exists, so no later rule can
// change it -- but evaluation continues so the refusal can list everything that
// was wrong rather than sending its reader round three times.
//
// **A denied command has no effect of any kind**, so tag changes are returned
// only when the verdict is not a denial.
func (s *Set) Evaluate(parsed *shell.Parsed, ctx Context) Outcome {
	facts := parsed.Facts()

	simples := command.Simples(parsed)
	if len(simples) != 1 {
		return refusal(ruleOneCommand, reasonNotOneCommand, parsed.Src)
	}
	simple := simples[0]

	// A command with no declaration is refused -- but the rules are consulted
	// first, because many of them can answer without one. A clause naming node
	// types, operators, flags or a fact needs no table, and letting the
	// declaration check short-circuit them means the refusal says only "this
	// is undescribed" about a command that also piped, redirected and
	// substituted. The verdict is the same either way; what changes is whether
	// the reader is told everything that was wrong or one thing that was.
	options, undeclared := command.Decompose(simple, s.table())

	out := s.judge(&subject{
		facts:   facts,
		simple:  simple,
		options: options,
		tags:    ctx.Tags,
		groups:  s.Groups,
	})

	if undeclared != nil {
		out.Action = ActionDeny
		out.Findings = append([]Finding{{
			Rule:   ruleDeclared,
			Action: ActionDeny,
			Reason: reasonUndeclared,
			Cause:  simple.Name(),
		}}, out.Findings...)
	}

	return settle(out)
}

// judge runs every rule and gathers what applied. The verdict is settled by
// the caller, which knows whether the command was declared.
func (s *Set) judge(sub *subject) Outcome {
	out := Outcome{Action: ActionDeny}
	strictest := 0

	for _, rule := range order(s.Rules) {
		cause, applies := sub.satisfies(rule)
		if !applies {
			continue
		}

		out.Findings = append(out.Findings, Finding{
			Rule:   rule.ID,
			Action: rule.Action,
			Reason: rule.Reason,
			Cause:  cause,
		})

		if rule.Action.Decides() && rule.Action.Strictness() > strictest {
			strictest, out.Action = rule.Action.Strictness(), rule.Action
		}
		if rule.Action == ActionTag || rule.Action == ActionUntag {
			out.Tags = append(out.Tags, TagChange{
				Name: rule.Tag,
				TTL:  rule.TTL,
				Set:  rule.Action == ActionTag,
			})
		}
	}

	if strictest == 0 {
		out.Findings = append(out.Findings, Finding{
			Rule:   ruleNoAllowance,
			Action: ActionDeny,
			Reason: reasonNothingAllowed,
		})
		out.Action = ActionDeny
	}
	return out
}

// settle reduces an outcome to the findings that produced it, and strips the
// tag changes from a refusal.
//
// A denied command has no effect of any kind: it sets and clears no tags and
// advances no countdown, because it did not happen.
func settle(out Outcome) Outcome {
	if out.Action == ActionDeny {
		out.Tags = nil
	}
	out.Findings = producing(out.Findings, out.Action)
	return out
}

// order decides which rules are evaluated first.
//
// **Today it is identity, and that is correct rather than merely convenient.**
// The strictest action wins, so no ordering can change a verdict; ordering is
// about how much work is done before the answer is known, not about what the
// answer is.
//
// It exists as a seam. Evaluating cheap structural clauses before expensive
// ones -- a node lookup before a path resolution before a call into stored
// state -- is a later version's optimisation, and leaving somewhere to put it
// costs one function now against restructuring the evaluator later.
func order(rules []Rule) []Rule { return rules }

// producing keeps the findings that reached the verdict.
//
// A rule that matched but was outranked did not cause the outcome, and listing
// it among the reasons reads as though it had: being told a command was refused
// AND permitted, in one message, tells the reader nothing about which to act on.
func producing(findings []Finding, verdict Action) []Finding {
	kept := make([]Finding, 0, len(findings))
	for _, finding := range findings {
		if finding.Action == verdict {
			kept = append(kept, finding)
		}
	}
	return kept
}

func refusal(rule, reason, cause string) Outcome {
	return Outcome{
		Action: ActionDeny,
		Findings: []Finding{
			{Rule: rule, Action: ActionDeny, Reason: reason, Cause: cause},
		},
	}
}

// table assembles the option declarations for command.Decompose.
func (s *Set) table() command.Table {
	return command.Table{Commands: s.Commands}
}

// Denied reports whether the outcome refuses the command.
func (o Outcome) Denied() bool { return o.Action == ActionDeny }

// Reasons returns every reason the verdict was reached, in the order the rules
// were read.
func (o Outcome) Reasons() []string {
	reasons := make([]string, 0, len(o.Findings))
	for _, finding := range o.Findings {
		reasons = append(reasons, finding.Reason)
	}
	return slices.Clip(reasons)
}
