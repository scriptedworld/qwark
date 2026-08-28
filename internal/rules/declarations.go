package rules

// A DeclarationPolicy says whether a command must be described before it runs.
//
// # Why this exists, and why the default is the strict one
//
// FR-4.16 refuses a command qwark holds no declaration for. That is the heart of
// what qwark is: when it cannot account for something, it refuses, so a gate's
// confusion is never the way through it.
//
// It is also all-or-nothing, and it arrives before shape decides anything. A
// rule set carrying only structural rules therefore refuses every command rather
// than judging the ones it understands, because the declaration check fires
// first. Measured 2026-08-28: a set of `00-structure.toml` plus a permissive
// allow rule answered `(engine) declared commands only` to `ls`, `rm -rf` and
// `git add -N` alike.
//
// # What the setting is for
//
// Introducing qwark in stages. The structural rules are the ones that hold
// whatever else is true, so they go on first, with everything else observed
// rather than refused. Requiring declarations in that phase means refusing
// roughly two thirds of what sessions actually run before any replacement for
// those commands exists, which stops the work rather than shaping it.
//
// # What turning it off gives up, stated plainly
//
// **Any command word runs, if its shape is clean.** That includes interpreters:
// `python3 -c '<program>'` is a single command with no redirection,
// substitution, pipe or glob, so nothing structural objects to it, and what it
// executes is not visible to any rule. `sudo`, `curl`, `chmod` and every task
// runner are likewise permitted by shape alone.
//
// So this phase is observation and not containment. It is worth being explicit
// that the gate is weaker with this off than the corpus figures suggest, because
// those were measured against a rule set that refused undeclared words.
//
// # The way back
//
// Declare commands, then remove the setting. It is deliberately a single
// boolean with no middle ground: a partial version, where some commands need
// declaring and others do not, would be a list of exceptions that nobody could
// read as a policy.
type DeclarationPolicy struct {
	// Required says a command must be declared. Absent is true, so a rule set
	// that says nothing about this gets FR-4.16 as written.
	Required *bool `toml:"required"`

	// Accounted says every option a declared command carries must appear in its
	// declaration. Absent is true, which is FR-6.7 as written.
	//
	// **This is a separate switch from Required and it has to be**, because the
	// two refusals happen at different levels and turning off the first does not
	// reach the second. An undeclared command's options are never decomposed, so
	// with `required = false` and nothing declared, no option is checked and the
	// setting looks unnecessary. Declare one command for any reason, including
	// wanting to write a rule about it later, and every option it carries starts
	// being refused again.
	//
	// So the structural phase says both, and says them out loud. The alternative
	// is a phase whose behaviour depends on whether a file happens to be loaded,
	// which is the kind of thing that changes under somebody months from now for
	// a reason unrelated to what it breaks.
	Accounted *bool `toml:"accounted"`
}

// required reports whether this set refuses undeclared commands.
//
// The pointer is what distinguishes "not stated" from "stated false", and only
// the second turns the check off. A missing table cannot quietly disable a
// refusal, which is the same reasoning that makes the shell policy a
// declaration rather than a clause.
func (s *Set) required() bool {
	if s.Declarations == nil || s.Declarations.Required == nil {
		return true
	}
	return *s.Declarations.Required
}

// accounted reports whether this set refuses an option no declaration names.
//
// Same shape as required, and separate from it on purpose: the two refusals sit
// at different levels and one does not imply the other.
func (s *Set) accounted() bool {
	if s.Declarations == nil || s.Declarations.Accounted == nil {
		return true
	}
	return *s.Declarations.Accounted
}
