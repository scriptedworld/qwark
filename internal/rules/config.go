package rules

import "github.com/scriptedworld/qwark/internal/command"

// A File is one rule file as written. It is the TOML shape and nothing more:
// what a file may say, not what any of it means. Meaning is established when
// files are aggregated, because several of the checks worth making (that a
// definition is not redefined elsewhere, that a group a clause names exists)
// cannot be made by looking at one file.
type File struct {
	// Shell is the set of shells this rule set permits. It is a declaration
	// and not a rule because it does not depend on the command.
	Shell *ShellPolicy `toml:"shell"`

	// Declarations says whether a command must be described before it may run.
	// Absent means yes, which is FR-4.16: a command qwark holds no declaration
	// for is refused.
	Declarations *DeclarationPolicy `toml:"declarations"`

	// Group holds named sets a clause can test membership of.
	Group map[string]Group `toml:"group"`

	// Command declares what options a command has and what its words denote.
	// A command with no declaration is denied, so this is where eligibility
	// comes from, and eligibility is not permission: an explicit deny rule
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
// case that matters, and it would do so silently.
type Group struct {
	Match   Form     `toml:"match"`
	Members []string `toml:"members"`

	// resolved holds what a whole-path member resolves to, where that differs
	// from how it was written. Members stays as written, so a listing and a
	// refusal quote the rule file and not this machine's links.
	resolved []string
}

// A Rule is a decision and the clauses that decide whether it applies.
//
// Match says how the clauses combine: all of them, the default, or any one.
// Without any, a rule that should fire on one of several unlike things has to
// be written once per thing, and the copies drift apart.
type Rule struct {
	ID     string      `toml:"id"`
	Action Action      `toml:"action"`
	Reason string      `toml:"reason"`
	Match  Combination `toml:"match"`

	// Tag is the name a `tag` or `untag` rule sets or clears.
	Tag string `toml:"tag"`

	// TTL is how many allowed commands a tag survives. Tags do not stack:
	// setting one that is already set replaces it, TTL and all.
	TTL int `toml:"ttl"`

	Clause []Clause `toml:"clause"`
}

// A Combination is how a rule's clauses combine.
type Combination string

// The combinations. Unstated is all, so a rule with no match key requires
// every clause.
const (
	CombineAll Combination = "all"
	CombineAny Combination = "any"
)

// known reports whether this is a combination at all. A rule file naming
// something else is refused, never read as one of these.
func (c Combination) known() bool {
	return c == "" || c == CombineAll || c == CombineAny
}

// Widens reports whether an action makes more commands run when it applies.
// Under any, each clause is enough on its own, so one clause written too
// broadly is enough to fire the rule. On an action that refuses, that refuses
// too much; on one that widens, it permits too much. The second is refused at
// load.
func (a Action) Widens() bool {
	return a == ActionAllow || a == ActionUntag
}

// An Action is what a rule does when its clauses hold.
type Action string

// The actions. When several rules apply, the highest precedence wins: block
// over allow over ask over deny. Rule order never changes a verdict.
//
// A block is a refusal nothing lifts. A deny is a refusal with a reason that
// any matching allow or ask lifts, however many denies matched. An ask shows
// me a warning and lets me decide, which is how a narrow case of something
// denied is handed back to me: deleting a branch is denied, deleting your own
// is asked.
const (
	ActionBlock Action = "block"
	ActionAllow Action = "allow"
	ActionDeny  Action = "deny"
	ActionAsk   Action = "ask"
	ActionTag   Action = "tag"
	ActionUntag Action = "untag"
)

// Decides reports whether an action produces a verdict. Tagging does not: it
// attaches a name for later rules to match, and decides nothing itself.
func (a Action) Decides() bool {
	return a.Precedence() > precedenceNone
}

// Refuses reports whether a verdict stops the command.
func (a Action) Refuses() bool {
	return a == ActionBlock || a == ActionDeny
}

// How the deciding actions order against each other. Named, not written as
// numbers at the point of return, so that "allow outranks deny" is stated once
// and cannot be disagreed with by a second comparison written elsewhere.
const (
	precedenceNone  = 0
	precedenceDeny  = 1
	precedenceAsk   = 2
	precedenceAllow = 3
	precedenceBlock = 4
)

// Precedence orders the deciding actions so the one that wins among several
// can be taken. Non-deciding actions sort below all of them and are never a
// verdict.
func (a Action) Precedence() int {
	switch a {
	case ActionBlock:
		return precedenceBlock
	case ActionAllow:
		return precedenceAllow
	case ActionDeny:
		return precedenceDeny
	case ActionAsk:
		return precedenceAsk
	case ActionTag, ActionUntag:
		return precedenceNone
	default:
		return precedenceNone
	}
}

// known reports whether this is an action at all. A rule file naming something
// else is refused, never treated as one of these.
func (a Action) known() bool {
	switch a {
	case ActionBlock, ActionAllow, ActionAsk, ActionDeny, ActionTag, ActionUntag:
		return true
	default:
		return false
	}
}

// A Clause selects part of a command and tests it. The rule's match says
// whether every clause must hold or any one.
//
// A clause names at most one selector and at most one test. The selectors that
// need no test (nodes, flags, ops, fact, tag) are satisfied by presence.
type Clause struct {
	// Selectors over the tree. These name the parser's own vocabulary, not a
	// summary of it, which is what keeps them from falling behind: a
	// maintained mapping can be silently incomplete.
	Nodes []string `toml:"nodes"`
	Flags []string `toml:"flags"`
	Ops   []string `toml:"ops"`
	Fact  string   `toml:"fact"`

	// Selectors over the command's words.
	//
	// An absent Index means any argument. It narrows a clause; it does not
	// make one, so `value = "rm"` on its own asks whether some argument
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

	// Agent names which agent the request came from, so one rule set can carry
	// a different policy per role: the agent that may write a task definition
	// is not the agent that may run it.
	//
	// The value is the `agent_type` from the payload. It is deliberately not an
	// environment variable: the subject can reach one of those and cannot set
	// its own agent type.
	//
	// `agent = ""` names a main-session call, which is why this is a pointer.
	// A main-session call reliably carries no agent type, so absence is the
	// one identity that is always available, and telling
	// "not stated" from "stated as empty" is the whole difference between a
	// clause that ignores the agent and a clause that requires there to be
	// none.
	Agent *string `toml:"agent"`

	// Cwd names a directory, and the clause holds while the call was made from
	// it or from anywhere inside it.
	//
	// The test is containment, not comparison. A session works in
	// subdirectories of the tree it was started in, so an exact test would hold in the root and fail
	// one level down, which is not a policy anybody means. It is compared by
	// path components with symlinks resolved, by the same machinery and for the
	// same reason as the blast radius: as text, `/home/x/proj` contains
	// `/home/x/project`, and as directories neither contains the other.
	//
	// The value must be absolute. A relative one would be resolved against the
	// working directory of whichever process asked, which has nothing to do
	// with where the agent was started.
	//
	// A request carrying no cwd does not satisfy this clause, like every other
	// clause qwark cannot answer, so an allow rule resting on it does not match.
	Cwd string `toml:"cwd"`

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

	// Absent inverts the clause: it holds when what it names is not there.
	//
	// This is how a conditional refusal is written: "git commit is forbidden
	// unless it is signed" is one deny rule with a clause saying the signing
	// option is absent. The exception therefore lives inside the rule it
	// modifies, where a reader of that rule sees it. The other way is a deny
	// with a narrower allow beside it, which a block cannot have.
	//
	// Where a selector names several positions, a plain clause holds if some
	// of them satisfy it, so an inverted clause holds when none do. That falls
	// out of one definition instead of being a second rule to remember.
	//
	// Inversion is satisfied by absence, including absence that qwark caused
	// by not understanding something, so in a deny rule it fails safe, and
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
