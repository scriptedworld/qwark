package rules

import (
	"slices"

	"github.com/scriptedworld/qwark/internal/command"
	"github.com/scriptedworld/qwark/internal/shell"
)

// A subject is one command, in every form a clause might ask about it.
//
// Everything here was derived once: the facts and the three vocabularies come
// from a single walk, and the options from a single decomposition. A clause is
// therefore a lookup, which is what makes adding rules cheap.
type subject struct {
	facts   *shell.Facts
	simple  command.Simple
	options command.Options
	tags    map[string]bool
	groups  map[string]Group
}

// satisfies reports whether every clause of a rule holds, and returns the text
// that set the rule off so a message can quote it.
//
// All clauses must hold. There is no disjunction inside a rule: alternatives
// are separate rules, so each can be checked by reading it alone.
func (sub *subject) satisfies(rule Rule) (string, bool) {
	cause := ""

	for _, clause := range rule.Clause {
		matched, text := sub.holds(clause)
		if !matched {
			return "", false
		}
		if cause == "" {
			cause = text
		}
	}

	return cause, true
}

// holds reports whether one clause holds.
//
// An inverted clause holds when what it names is not there. Where a selector
// covers several positions a plain clause holds if SOME satisfy it, so an
// inverted one holds when NONE do -- one definition rather than two rules to
// remember.
func (sub *subject) holds(clause Clause) (bool, string) {
	matched, text := sub.evaluate(clause)
	if clause.Absent {
		return !matched, ""
	}
	return matched, text
}

// evaluate answers what the clause names, before any inversion.
//
// A clause that cannot be evaluated does not match. An option clause against a
// command with no declaration has nothing to test, and counting an unanswerable
// clause as satisfied is what would let an allow rule permit on the strength of
// qwark's own ignorance.
func (sub *subject) evaluate(clause Clause) (bool, string) {
	switch {
	case len(clause.Nodes) > 0:
		return sub.anyPresent(clause.Nodes, sub.facts.HasNode)
	case len(clause.Flags) > 0:
		return sub.anyPresent(clause.Flags, sub.facts.HasFlag)
	case len(clause.Ops) > 0:
		return sub.anyPresent(clause.Ops, sub.facts.HasOp)
	case clause.Fact != "":
		return sub.factHolds(clause.Fact)
	case clause.Tag != "":
		return sub.tags[clause.Tag], clause.Tag
	case clause.Option != "":
		return sub.optionHolds(clause)
	case clause.Kind != "":
		return sub.kindHolds(clause)
	default:
		return sub.wordsHold(clause)
	}
}

// anyPresent reports whether any of the names is present, and quotes the source
// it was found in rather than the name itself.
func (sub *subject) anyPresent(
	names []string, present func(string) (string, bool),
) (bool, string) {
	for _, name := range names {
		if text, found := present(name); found {
			return true, text
		}
	}
	return false, ""
}

func (sub *subject) factHolds(name string) (bool, string) {
	found, ok := sub.facts.First(shell.Fact(name))
	if !ok {
		return false, ""
	}
	return true, found.Text
}

// optionHolds tests the options the command was given, by their DECLARED
// meaning rather than their spelling -- so `-f`, `-rf`, `--force` and `--f` all
// satisfy a clause about forcing, and `tar -f` does not.
func (sub *subject) optionHolds(clause Clause) (bool, string) {
	for _, given := range sub.options.Given {
		if given.Means != clause.Option {
			continue
		}
		if !clause.statesTest() {
			return true, given.Spelling
		}
		if sub.test(clause, given.Value) {
			return true, given.Spelling + " " + given.Value
		}
	}
	return false, ""
}

// kindHolds tests every word the declaration says denotes this kind, whether it
// arrived as an operand or as an option's value.
func (sub *subject) kindHolds(clause Clause) (bool, string) {
	for _, valued := range sub.options.Values(command.Kind(clause.Kind)) {
		if !clause.statesTest() || sub.test(clause, valued.Value) {
			return true, valued.Value
		}
	}
	return false, ""
}

// wordsHold tests the words at the ordinals the clause names.
//
// **An absent index means the arguments**, so a test written without one asks
// about what the command was given. Ordinal 0 is the command and is reached
// only by naming it.
func (sub *subject) wordsHold(clause Clause) (bool, string) {
	reading, err := ParseReading(clause.Reading)
	if err != nil {
		return false, ""
	}

	for _, ordinal := range sub.ordinals(clause.Index) {
		word, found := sub.simple.At(ordinal)
		if !found {
			continue
		}
		text, readable := reading.Of(word)
		if !readable {
			continue
		}
		if !clause.statesTest() || sub.test(clause, text) {
			return true, text
		}
	}
	return false, ""
}

// ordinals resolves the clause's index, defaulting to every argument.
func (sub *subject) ordinals(spec string) []int {
	last := sub.simple.Last()

	if spec == "" {
		return argumentOrdinals(last)
	}

	index, err := command.ParseIndex(spec)
	if err != nil {
		// Refused at load, so unreachable from a loaded rule set. Selecting
		// nothing is the safe answer for the paths that do not go through it.
		return nil
	}
	return index.Select(last)
}

func argumentOrdinals(last int) []int {
	ordinals := make([]int, 0, last)
	for ordinal := 1; ordinal <= last; ordinal++ {
		ordinals = append(ordinals, ordinal)
	}
	return ordinals
}

// test applies the clause's test to one value: a group's members, or the
// clause's own inline value, partial or pattern.
func (sub *subject) test(clause Clause, value string) bool {
	if clause.Group != "" {
		return sub.groupHas(clause.Group, value)
	}

	match, err := clause.spec().Build()
	if err != nil {
		return false
	}
	return match.Matches(value)
}

// groupHas reports whether a value satisfies any member of a group, compared
// the way the group declares. A group of paths must say `match = "partial"`,
// because paths are prefixes and comparing them whole would match a directory
// and miss everything in it.
func (sub *subject) groupHas(name, value string) bool {
	group, declared := sub.groups[name]
	if !declared {
		return false
	}

	return slices.ContainsFunc(group.Members, func(member string) bool {
		return group.compare(member, value)
	})
}

// compare applies the group's declared comparison to one member.
func (g Group) compare(member, value string) bool {
	if g.Match == FormPartial {
		partial, err := Partial(member)
		if err != nil {
			return false
		}
		return partial.Matches(value)
	}
	return Value(member).Matches(value)
}
