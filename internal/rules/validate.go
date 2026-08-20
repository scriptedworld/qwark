package rules

import (
	"fmt"

	"github.com/scriptedworld/qwark/internal/command"
	"github.com/scriptedworld/qwark/internal/shell"
)

// validate checks what only the aggregate can answer: that no rule id is used
// twice across files, that every group a clause names exists, and that no
// clause is empty.
//
// A clause that selects nothing would match nothing, so a rule carrying one
// never applies. That reads exactly like a rule that is working, which is the
// most dangerous way for a gate to be broken -- so it is refused at load rather
// than tolerated at evaluation.
func (s *Set) validate() error {
	seen := make(map[string]bool, len(s.Rules))

	for name, group := range s.Groups {
		if len(group.Members) == 0 {
			return fmt.Errorf("%w: %s", ErrEmptyGroup, name)
		}
		// A group compares its members one way for all of them, and only two
		// comparisons make sense for a set: the whole value, or any part of
		// it. A pattern belongs in the clause, where it is read alongside what
		// it applies to.
		if group.Match != "" && group.Match != FormValue && group.Match != FormPartial {
			return fmt.Errorf("%w: group %s: %q", ErrGroupMatch, name, group.Match)
		}
	}

	for _, rule := range s.Rules {
		if seen[rule.ID] {
			return fmt.Errorf("%w: %s", ErrDuplicateRule, rule.ID)
		}
		seen[rule.ID] = true

		if err := s.validateRule(rule); err != nil {
			return err
		}
	}

	return nil
}

func (s *Set) validateRule(rule Rule) error {
	if !rule.Action.known() {
		return fmt.Errorf("%w: rule %s: %q", ErrUnknownAction, rule.ID, rule.Action)
	}
	if (rule.Action == ActionTag || rule.Action == ActionUntag) && rule.Tag == "" {
		return fmt.Errorf("%w: rule %s", ErrTagMissing, rule.ID)
	}
	if len(rule.Clause) == 0 {
		return fmt.Errorf("%w: rule %s", ErrNoClauses, rule.ID)
	}

	for i, clause := range rule.Clause {
		if err := s.validateClause(rule.ID, i, clause); err != nil {
			return err
		}
	}
	return nil
}

func (s *Set) validateClause(id string, position int, clause Clause) error {
	if !clause.statesAnything() {
		return fmt.Errorf("%w: rule %s, clause %d", ErrClauseEmpty, id, position)
	}

	if clause.Group != "" {
		if _, declared := s.Groups[clause.Group]; !declared {
			return fmt.Errorf("%w: rule %s names %q",
				ErrUnknownGroup, id, clause.Group)
		}
	}

	// A clause may carry an inline test or a group, never both, and a pattern
	// that will not compile is refused here rather than at the first command
	// it silently fails to match.
	if clause.statesTest() && clause.Group == "" {
		if _, err := clause.spec().Build(); err != nil {
			return fmt.Errorf("rule %s, clause %d: %w", id, position, err)
		}
	}

	if _, err := ParseReading(clause.Reading); err != nil {
		return fmt.Errorf("rule %s, clause %d: %w", id, position, err)
	}

	if clause.Index != "" {
		if _, err := command.ParseIndex(clause.Index); err != nil {
			return fmt.Errorf("rule %s, clause %d: %w", id, position, err)
		}
	}

	return s.validateVocabulary(id, position, clause)
}

// validateVocabulary refuses a name the parser does not have.
//
// This is what keeps naming the parser's own vocabulary safe. A clause naming a
// node type that does not exist would otherwise match nothing for ever, which
// reads exactly like a rule that is working -- and if the library ever renames
// one, this fails loudly at load rather than quietly at every command.
func (s *Set) validateVocabulary(id string, position int, clause Clause) error {
	for _, name := range clause.Nodes {
		if !shell.KnownNode(name) {
			return fmt.Errorf("%w: rule %s, clause %d: %q",
				ErrUnknownNode, id, position, name)
		}
	}
	for _, name := range clause.Flags {
		if !shell.KnownFlag(name) {
			return fmt.Errorf("%w: rule %s, clause %d: %q",
				ErrUnknownFlag, id, position, name)
		}
	}
	return nil
}

// statesAnything reports whether a clause says anything at all. One that says
// nothing is refused: it would match nothing, and a rule that never applies
// looks identical to one that is working.
//
// A test with no selector is not empty. **An absent `index` means any
// position**, so `value = "rm"` is a complete clause meaning "some word of this
// command is rm". The index narrows a clause; it is not what makes it one.
func (c Clause) statesAnything() bool {
	return c.selectsSomething() || c.statesTest()
}

// selectsSomething reports whether a clause names part of the command to look
// at, as opposed to stating what to look for.
func (c Clause) selectsSomething() bool {
	return len(c.Nodes) > 0 ||
		len(c.Flags) > 0 ||
		len(c.Ops) > 0 ||
		c.Fact != "" ||
		c.Index != "" ||
		c.Option != "" ||
		c.Kind != "" ||
		c.Tag != "" ||
		c.Agent != nil
}
