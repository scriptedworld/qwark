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

	// Agent is the `agent_type` the request carried, empty for a main-session
	// call. It comes from the payload, not the environment, because the
	// subject can reach an environment variable and cannot set this.
	Agent string

	// Cwd is the directory the call was made from, as the payload reported it
	// (FR-10.1). Same trust model as Agent and for the same reason: it is
	// payload and not environment, so the subject cannot set it.
	//
	// It partitions policy on the axis that actually varies. Scoping by agent
	// alone gives ten sessions of one type in ten repositories a single policy,
	// while the limits wanted here are contextual by repository: a tree
	// mid-rebuild and a tree under a freeze are the same `agent_type` and want
	// different answers.
	Cwd string
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
		"however they were joined: by a sequence, a pipe, a logical " +
		"concatenation, or a substitution carrying a command of its own."
	reasonUndeclared = "This command has no declaration, so qwark cannot account " +
		"for what its options mean or which of its words are paths. Nothing runs " +
		"unless it has been described."
	reasonUnaccounted = "This word is not accounted for by the command's " +
		"declaration, so what the command was told to do is not fully known. A " +
		"declaration exists to stop a verdict resting on a reading qwark already " +
		"knows to be incomplete:"
	reasonNothingAllowed = "Nothing permitted this. Being allowed means an allow " +
		"rule matched, and none did."
	reasonObserving = "No rule decided this, and the rule set allows what no " +
		"rule decides while the gate is observing."
)

// Ids for the refusals the engine makes itself, so a message can name what
// refused just as it names a rule.
const (
	ruleOneCommand  = "(engine) one command at a time"
	ruleDeclared    = "(engine) declared commands only"
	ruleAccounted   = "(engine) accounted options only"
	ruleNoAllowance = "(engine) deny by default"
	ruleObserving   = "(engine) allow by default"
)

// Evaluate judges one parsed command.
//
// Deny is the default and it is the engine's, not a rule's. A command is
// refused unless an allow rule matched it: being in the allowed list *means* an
// allow rule matched. A rule set containing no allow rules therefore permits
// nothing, which is the correct reading of an empty policy.
//
// Every rule is evaluated whatever has already matched, so a refusal can list
// everything that was wrong instead of sending its reader round three times.
//
// The engine's own refusals are blocks: no allow lifts a command qwark could
// not account for. A refused command has no effect of any kind, so tag
// changes are returned only when the verdict is not a refusal.
func (s *Set) Evaluate(parsed *shell.Parsed, ctx Context) Outcome {
	facts := parsed.Facts()

	simples := command.Simples(parsed)
	if len(simples) != 1 {
		return refusal(ruleOneCommand, reasonNotOneCommand, parsed.Src)
	}
	simple := simples[0]

	// A command with no declaration is refused, but the rules are consulted
	// first, because many of them can answer without one. A clause naming node
	// types, operators, flags or a fact needs no table, and letting the
	// declaration check short-circuit them means the refusal says only "this
	// is undescribed" about a command that also piped, redirected and
	// substituted. The verdict is the same either way, and consulting the rules
	// first lets the refusal list everything that was wrong.
	options, undeclared := command.Decompose(simple, s.table())

	out := s.judge(&subject{
		facts:   facts,
		simple:  simple,
		options: options,
		tags:    ctx.Tags,
		agent:   ctx.Agent,
		cwd:     ctx.Cwd,
		groups:  s.Groups,
	})

	return settle(s.declarationsHold(out, simple, options, undeclared))
}

// declarationsHold applies the two declaration switches to a verdict the rules
// have already reached.
//
// Both refuse for the same underlying reason, that qwark cannot account for
// what the command was told to do, and both are applied after the rules so
// that a refusal names everything that was wrong instead of one thing that
// was.
func (s *Set) declarationsHold(
	out Outcome, simple command.Simple, options command.Options, undeclared error,
) Outcome {
	// A rule set may say it does not require declarations, which is what makes
	// a structural-only phase possible: FR-4.16 arrives before shape decides
	// anything, so requiring it means refusing every command instead of
	// judging the ones the structural rules understand. Absent, the answer is
	// yes and FR-4.16 holds as written. See DeclarationPolicy for what turning
	// it off gives up.
	if undeclared != nil && s.required() {
		out.Action = ActionBlock
		out.Findings = append([]Finding{{
			Rule:   ruleDeclared,
			Action: ActionBlock,
			Reason: reasonUndeclared,
			Cause:  simple.Name(),
		}}, out.Findings...)
	}

	// A command can be declared and still hold a word the declaration does not
	// account for. Decomposition records each such word and does not stop at
	// the first, and the verdict has to consult that record: an option nobody
	// declared is exactly the case where qwark does not know what the command
	// was told to do, which is the same ignorance that refuses an undeclared
	// command one level up.
	// Gated separately from the check above, because the two sit at different
	// levels: an undeclared command's options are never decomposed, so turning
	// off the declaration requirement does not reach this one. A phase that
	// wants neither has to say so twice. See DeclarationPolicy.
	if s.accounted() {
		if faults := unaccounted(options); len(faults) > 0 {
			out.Action = ActionBlock
			out.Findings = append(faults, out.Findings...)
		}
	}

	return out
}

// unaccounted turns each fault decomposition recorded into a finding of its
// own, so a refusal names every word it could not account for, not only the
// first, and fixing them takes one round trip instead of one per word.
func unaccounted(options command.Options) []Finding {
	findings := make([]Finding, 0, len(options.Faults))
	for _, fault := range options.Faults {
		findings = append(findings, Finding{
			Rule:   ruleAccounted,
			Action: ActionBlock,
			Reason: reasonUnaccounted + " " + fault.Err.Error() + ".",
			Cause:  fault.Text,
		})
	}
	return findings
}

// judge runs every rule and gathers what applied. The verdict is settled by
// the caller, which knows whether the command was declared.
func (s *Set) judge(sub *subject) Outcome {
	out := Outcome{Action: ActionDeny}
	highest := precedenceNone

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

		if rule.Action.Precedence() > highest {
			highest, out.Action = rule.Action.Precedence(), rule.Action
		}
		if rule.Action == ActionTag || rule.Action == ActionUntag {
			out.Tags = append(out.Tags, TagChange{
				Name: rule.Tag,
				TTL:  rule.TTL,
				Set:  rule.Action == ActionTag,
			})
		}
	}

	if highest == precedenceNone {
		finding := Finding{Rule: ruleNoAllowance, Action: ActionDeny, Reason: reasonNothingAllowed}
		if s.fallback() == ActionAllow {
			finding = Finding{Rule: ruleObserving, Action: ActionAllow, Reason: reasonObserving}
		}
		out.Findings = append(out.Findings, finding)
		out.Action = finding.Action
	}
	return out
}

// settle reduces an outcome to the findings that produced it, and strips the
// tag changes from a refusal.
//
// A refused command never runs, so it sets and clears no tags and advances no
// countdown.
func settle(out Outcome) Outcome {
	if out.Action.Refuses() {
		out.Tags = nil
	}
	out.Findings = producing(out.Findings, out.Action)
	return out
}

// order decides which rules are evaluated first.
//
// It is identity, which is correct and not merely convenient. The action with
// the highest precedence wins, so no ordering can change a verdict; ordering is about how much
// work is done before the answer is known, not about what the answer is.
//
// It exists as a seam. Evaluating cheap structural clauses before expensive
// ones (a node lookup before a path resolution before a call into stored
// state) is a later version's optimisation, and leaving somewhere to put it
// costs one function now against restructuring the evaluator later.
func order(rules []Rule) []Rule { return rules }

// producing keeps the findings that reached the verdict.
//
// A rule that matched but was outranked did not cause the outcome, and listing
// it among the reasons reads as though it had: being told a command was refused
// and permitted, in one message, tells the reader nothing about which to act on.
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
		Action: ActionBlock,
		Findings: []Finding{
			{Rule: rule, Action: ActionBlock, Reason: reason, Cause: cause},
		},
	}
}

// table assembles the option declarations for command.Decompose.
func (s *Set) table() command.Table {
	return command.Table{Commands: s.Commands}
}

// Denied reports whether the outcome refuses the command, by block or deny.
func (o Outcome) Denied() bool { return o.Action.Refuses() }

// Reasons returns every reason the verdict was reached, in the order the rules
// were read.
func (o Outcome) Reasons() []string {
	reasons := make([]string, 0, len(o.Findings))
	for _, finding := range o.Findings {
		reasons = append(reasons, finding.Reason)
	}
	return slices.Clip(reasons)
}
