package rules

import "github.com/scriptedworld/qwark/internal/command"

// A File is one rule file as written. It is the TOML shape and nothing more:
// what a file may say, not what any of it means. Meaning is established when
// files are aggregated, because several of the checks worth making -- that a
// definition is not redefined elsewhere, that a group a clause names exists --
// cannot be made by looking at one file.
type File struct {
	// Shell is the set of shells this rule set permits. It is a declaration
	// rather than a rule because it does not depend on the command.
	Shell *ShellPolicy `toml:"shell"`

	// Group holds named sets a clause can test membership of.
	Group map[string]Group `toml:"group"`

	// Command declares what options a command has and what its words denote.
	// A command with no declaration is denied, so this is where eligibility
	// comes from -- and eligibility is not permission: an explicit deny rule
	// outranks any declaration.
	Command map[string]command.Declaration `toml:"command"`

	// Rule is what actually decides.
	Rule []Rule `toml:"rule"`
}

// A Group is a named set of values a clause can test against.
//
// Match says how members are compared, because the answer differs by what the
// set holds: command names are compared whole, and protected paths are
// prefixes. A group of paths compared for equality would match
// `/home/x/.claude` and miss `/home/x/.claude/settings.json`, which is every
// case that matters -- and it would do so silently.
type Group struct {
	Match   Form     `toml:"match"`
	Members []string `toml:"members"`
}

// A Rule is a decision and the clauses that must all hold for it to apply.
//
// There is no disjunction inside a rule. Alternatives are separate rules, so
// that each one can be checked by reading it alone.
type Rule struct {
	ID     string `toml:"id"`
	Action Action `toml:"action"`
	Reason string `toml:"reason"`

	// Tag is the name a `tag` or `untag` rule sets or clears.
	Tag string `toml:"tag"`

	// TTL is how many allowed commands a tag survives. Tags do not stack:
	// setting one that is already set replaces it, TTL and all.
	TTL int `toml:"ttl"`

	Clause []Clause `toml:"clause"`
}

// An Action is what a rule does when all of its clauses hold.
type Action string

// The actions. When several rules apply, the strictest wins -- deny over ask
// over allow -- so rule order never changes a verdict and no file can weaken
// another by being read later.
//
// There is deliberately no overridable deny. The two tiers of refusal are
// already here: ActionAsk is a refusal a person can lift, one command at a
// time and on the record, and ActionDeny is one nobody can. A rule that could
// override another rule would put that power in configuration, which sits far
// closer to the subject than a person does.
const (
	ActionAllow Action = "allow"
	ActionAsk   Action = "ask"
	ActionDeny  Action = "deny"
	ActionTag   Action = "tag"
	ActionUntag Action = "untag"
)

// Decides reports whether an action produces a verdict. Tagging does not: it
// attaches a name for later rules to match, and decides nothing itself.
func (a Action) Decides() bool {
	return a == ActionAllow || a == ActionAsk || a == ActionDeny
}

// How the deciding actions order against each other. Named rather than written
// as numbers at the point of return, so that "deny outranks ask" is stated once
// and cannot be disagreed with by a second comparison written elsewhere.
const (
	strictnessNone  = 0
	strictnessAllow = 1
	strictnessAsk   = 2
	strictnessDeny  = 3
)

// Strictness orders the deciding actions so the strictest of several can be
// taken. Non-deciding actions sort below all of them and are never a verdict.
func (a Action) Strictness() int {
	switch a {
	case ActionDeny:
		return strictnessDeny
	case ActionAsk:
		return strictnessAsk
	case ActionAllow:
		return strictnessAllow
	case ActionTag, ActionUntag:
		return strictnessNone
	default:
		return strictnessNone
	}
}

// known reports whether this is an action at all. A rule file naming something
// else is refused rather than treated as one of these.
func (a Action) known() bool {
	switch a {
	case ActionAllow, ActionAsk, ActionDeny, ActionTag, ActionUntag:
		return true
	default:
		return false
	}
}

// A Clause selects part of a command and tests it. Every clause of a rule must
// hold for the rule to apply.
//
// A clause names at most one selector and at most one test. The selectors that
// need no test -- nodes, flags, ops, fact, tag -- are satisfied by presence.
type Clause struct {
	// Selectors over the tree. These name the parser's own vocabulary rather
	// than a summary of it, which is what keeps them from falling behind: a
	// maintained mapping can be silently incomplete, and was.
	Nodes []string `toml:"nodes"`
	Flags []string `toml:"flags"`
	Ops   []string `toml:"ops"`
	Fact  string   `toml:"fact"`

	// Selectors over the command's words.
	//
	// **An absent Index means any argument.** It narrows a clause rather than
	// making it one, so `value = "rm"` on its own asks whether some argument
	// is `rm`. Writing `..` for the same thing is refused: one meaning with
	// two spellings is a thing to remove.
	//
	// Ordinal 0 is the command and is reachable only by naming it: arguments
	// do not start at 0, so an open-ended range never reaches it either. A
	// clause about the command itself says `index = "0"`.
	Index  string `toml:"index"`
	Option string `toml:"option"`
	Kind   string `toml:"kind"`

	// Tag matches while a tag is live, whether it was set by an earlier
	// command or calculated afresh from the world.
	Tag string `toml:"tag"`

	// Reading says which form of a word is tested: the interpreted value the
	// shell will pass, or the source as written. Interpreted is the default,
	// because testing what was written is what lets `/home/x/.cl\aude/y` past
	// a rule about `.claude`.
	Reading string `toml:"reading"`

	// Tests. Exactly one may be stated, and Group supplies many values for
	// whichever comparison its own declaration names.
	Group   string  `toml:"group"`
	Value   *string `toml:"value"`
	Partial *string `toml:"partial"`
	Pattern *string `toml:"pattern"`

	// Absent inverts the clause: it holds when what it names is NOT there.
	//
	// This is how a conditional refusal is written -- "git commit is forbidden
	// unless it is signed" is one deny rule with a clause saying the signing
	// option is absent. The exception therefore lives inside the rule it
	// modifies, where a reader of that rule sees it, rather than in a second
	// rule that outranks the first. That distinction is the whole reason there
	// is no overridable deny: an exception stated here is visible, and one
	// stated by precedence between files is not.
	//
	// Where a selector names several positions, a plain clause holds if SOME
	// of them satisfy it, so an inverted clause holds when NONE do. That falls
	// out of one definition rather than being a second rule to remember.
	//
	// Inversion is satisfied by absence, including absence that qwark caused
	// by not understanding something -- so in a deny rule it fails safe, and
	// in an allow rule it is worth reading twice.
	Absent bool `toml:"absent"`
}

// spec returns the clause's inline test, for Spec.Build to validate.
func (c Clause) spec() Spec {
	return Spec{Value: c.Value, Partial: c.Partial, Pattern: c.Pattern}
}

// statesTest reports whether the clause carries a test of its own, as opposed
// to being satisfied by the presence of what it selects.
func (c Clause) statesTest() bool {
	return c.Value != nil || c.Partial != nil || c.Pattern != nil || c.Group != ""
}
