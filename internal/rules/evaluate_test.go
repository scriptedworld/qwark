package rules_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/scriptedworld/qwark/internal/rules"
	"github.com/scriptedworld/qwark/internal/shell"
)

// declarations describes rm well enough for option clauses to be answerable.
// A declaration grants understanding, not permission: nothing here permits
// anything, and the tests below have to say so explicitly.
const declarations = `
[command.rm]
operands = "path"
short.f = { means = "force" }
short.r = { means = "recursive" }
long.force = { means = "force" }

[command.git]
operands = "text"
long.oneline = { means = "oneline" }
`

// judgeWith loads a rule set from the given files and judges one command.
func judgeWith(t *testing.T, files map[string]string, src string) rules.Outcome {
	t.Helper()

	set, err := rules.Load([]string{inFiles(t, files)})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}
	parsed, err := shell.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q) = %v", src, err)
	}
	return set.Evaluate(parsed, rules.Context{})
}

// judgeAs judges one command as a given agent type. The empty type is the main
// session rather than an absent value, so it is a case worth passing on purpose.
func judgeAs(t *testing.T, files map[string]string, agent, src string) rules.Outcome {
	t.Helper()

	set, err := rules.Load([]string{inFiles(t, files)})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}
	return set.Evaluate(parseFor(t, src), rules.Context{Agent: agent})
}

// duties is one rule set carrying two roles, which is the whole point of the
// agent clause: the policy for every caller is in one file, named once where
// the hook is registered, rather than spread across files a launcher has to
// select between.
const duties = `
[[rule]]
id = "runner-may-read"
action = "allow"
reason = "The gate runner reads the repository."
  [[rule.clause]]
  agent = "gate-runner"
  [[rule.clause]]
  index = "0"
  value = "git"

[[rule]]
id = "writer-may-not-read"
action = "deny"
reason = "The writer has no business running git."
  [[rule.clause]]
  agent = "file-writer"
  [[rule.clause]]
  index = "0"
  value = "git"

[[rule]]
id = "main-session-may-read"
action = "allow"
reason = "The main session reads the repository."
  [[rule.clause]]
  agent = ""
  [[rule.clause]]
  index = "0"
  value = "git"
`

// COVERS: FR-7.12 | positive
func TestAClauseNamesTheAgentTheRequestCameFrom(t *testing.T) {
	t.Parallel()

	// One rule set, two roles, opposite verdicts on the same command. This is
	// what separation of duties needs from the engine: the agent that may write
	// a task definition is not the agent that may run it, and saying so should
	// not mean a launcher swapping files between sessions.
	files := ruleSet(duties)

	if outcome := judgeAs(t, files, "gate-runner", `git status`); outcome.Denied() {
		t.Errorf("Action = %q, want allow for the runner", outcome.Action)
	}
	if outcome := judgeAs(t, files, "file-writer", `git status`); !outcome.Denied() {
		t.Errorf("Action = %q, want deny for the writer", outcome.Action)
	}
}

// COVERS: FR-7.12 | negative
func TestAnAgentClauseDoesNotApplyToAnotherAgent(t *testing.T) {
	t.Parallel()

	// A role cannot pick up another role's allowance by being some third thing.
	// The agent type is compared whole: it is a name a dispatcher assigned, not
	// a path, so there is nothing for a prefix to be right about.
	outcome := judgeAs(t, ruleSet(duties), "gate-runner-2", `git status`)

	if !outcome.Denied() {
		t.Errorf("Action = %q, want deny: no rule names this agent, and being "+
			"permitted means an allow rule matched", outcome.Action)
	}
}

// COVERS: FR-7.12 | edge
func TestAnAgentAllowanceReachesOnlyTheCommandItsRuleNames(t *testing.T) {
	t.Parallel()

	// The agent clause narrows a rule; it is not a role saying "and this agent
	// may do things". `runner-may-read` names git as well as the agent, and all
	// clauses must hold, so the runner's allowance stops at git and does not
	// become a general permission attached to the role.
	//
	// This is the direction worth testing, because getting it wrong turns a
	// per-command allowance into a per-agent one, which is how a role quietly
	// accumulates everything anybody ever granted it.
	outcome := judgeAs(t, ruleSet(duties), "gate-runner", `rm x`)

	if !outcome.Denied() {
		t.Errorf("Action = %q, want deny: no rule permits rm for this agent, "+
			"and being allowed means an allow rule matched", outcome.Action)
	}
}

// COVERS: FR-7.13 | positive
func TestTheMainSessionIsNamedByHavingNoAgentType(t *testing.T) {
	t.Parallel()

	// **agent_id and agent_type appear only for a subagent**, so a
	// main-session call carries neither. That is what makes absence
	// dependable rather than a gap: the main session is the one caller reliably
	// without an agent type, so `agent = ""` names it exactly and one rule set
	// covers every caller.
	//
	// Without this the main session would be the one role no clause could
	// address, and a launcher would have to vary what it passes, which is the
	// symlink and environment-variable management this exists to avoid.
	files := ruleSet(duties)

	if outcome := judgeAs(t, files, "", `git status`); outcome.Denied() {
		t.Errorf("Action = %q, want allow: the main session is a role, not a gap",
			outcome.Action)
	}

	// And it is a role rather than a wildcard: a subagent the rule set says
	// nothing about is not covered by the main session's allowance.
	if outcome := judgeAs(t, files, "unnamed-agent", `git status`); !outcome.Denied() {
		t.Errorf("Action = %q, want deny: `agent = \"\"` names the main session, "+
			"not every caller", outcome.Action)
	}
}

// COVERS: FR-7.13 | edge
func TestTheMainSessionClauseIsToldFromNoClauseAtAll(t *testing.T) {
	t.Parallel()

	// The distinction the pointer exists for. A rule that states no agent
	// applies to every caller; a rule stating `agent = ""` applies to the main
	// session only. Written as a plain string those two would be one, and the
	// second would silently become the first, turning a rule meant for one
	// role into a rule for all of them, in an allow rule, which is the
	// direction that hands out permission nobody granted.
	const anyCaller = `
[[rule]]
id = "any-caller"
action = "allow"
reason = "No agent is named, so every caller is covered."
  [[rule.clause]]
  index = "0"
  value = "git"
`

	if outcome := judgeAs(t, ruleSet(anyCaller), "some-agent", `git status`); outcome.Denied() {
		t.Errorf("Action = %q, want allow: a rule naming no agent covers every caller",
			outcome.Action)
	}
	if outcome := judgeAs(t, ruleSet(duties), "some-agent", `git status`); !outcome.Denied() {
		t.Errorf("Action = %q, want deny: `agent = \"\"` is the main session, and "+
			"stating no agent at all is what covers everybody", outcome.Action)
	}
}

// judgeFrom judges `git status` as a call made from a directory.
//
// The command is fixed because these tests vary one thing: where the call came
// from. `scoped` names git as its other clause, so the command is the constant
// the directory is compared against.
func judgeFrom(t *testing.T, files map[string]string, cwd string) rules.Outcome {
	t.Helper()

	set, err := rules.Load([]string{inFiles(t, files)})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}
	return set.Evaluate(parseFor(t, `git status`), rules.Context{Cwd: cwd})
}

// scoped permits a command in one tree and nowhere else, which is the case the
// cwd clause was built for: `go test` compiles and runs code from the working
// tree, so it stays refused everywhere, except in the repository whose own
// tests are the thing being run.
const scoped = `
[[rule]]
id = "tests-run-in-their-own-tree"
action = "allow"
reason = "A project may run its own tests."
  [[rule.clause]]
  cwd = "/srv/project"
  [[rule.clause]]
  index = "0"
  value = "git"
`

// COVERS: FR-7.14 | positive
func TestACwdClausePermitsInsideItsTreeOnly(t *testing.T) {
	t.Parallel()

	// The partition agent-scoping cannot express. Both calls below are the same
	// agent type running the same command, and the only thing telling them
	// apart is which repository they came from.
	if outcome := judgeFrom(t, ruleSet(scoped), "/srv/project"); outcome.Denied() {
		t.Errorf("Action = %q, want allow: the call came from the named tree",
			outcome.Action)
	}
	if outcome := judgeFrom(t, ruleSet(scoped), "/srv/other"); !outcome.Denied() {
		t.Errorf("Action = %q, want deny: a rule scoped to one tree must not "+
			"carry into another", outcome.Action)
	}
}

// COVERS: FR-7.14a | property
func TestACwdClauseReachesIntoSubdirectoriesAndNotOntoNeighbours(t *testing.T) {
	t.Parallel()

	// Two halves of one comparison, and getting either wrong is silent.
	//
	// A session works in subdirectories of the tree it was started in, so a
	// clause that held only at the root would be a policy nobody means.
	if outcome := judgeFrom(t, ruleSet(scoped), "/srv/project/internal/deep"); outcome.Denied() {
		t.Errorf("Action = %q, want allow: a session works below the root it "+
			"was started in", outcome.Action)
	}

	// And `/srv/project-old` is a neighbour of `/srv/project`, not a child of
	// it. As text one is a prefix of the other; as directories neither contains
	// the other, and a prefix comparison would hand one repository's permission
	// to a different repository on the strength of a shared spelling.
	if outcome := judgeFrom(t, ruleSet(scoped), "/srv/project-old"); !outcome.Denied() {
		t.Errorf("Action = %q, want deny: `/srv/project-old` is not inside "+
			"`/srv/project`, whatever their spellings share", outcome.Action)
	}
}

// COVERS: FR-7.14b | negative
func TestACwdClauseDeclinesWhenTheCallNamesNoDirectory(t *testing.T) {
	t.Parallel()

	// Every real call carries a cwd, so this is the shape of a request qwark
	// could not read rather than a caller it will meet. It declines, on the
	// same reading every other unanswerable clause gets: an allow rule must not
	// match on the strength of qwark not knowing where the call came from.
	if outcome := judgeFrom(t, ruleSet(scoped), ""); !outcome.Denied() {
		t.Errorf("Action = %q, want deny: a request with no cwd cannot satisfy "+
			"a clause about where it came from", outcome.Action)
	}
}

// COVERS: FR-7.14b | negative
func TestARelativeCwdIsRefusedAtLoad(t *testing.T) {
	t.Parallel()

	// Refused at load rather than declining at every command. A relative
	// directory would be resolved against whichever process asked, which has
	// nothing to do with where the agent was started, and a scoping clause that
	// never holds reads exactly like one that is working.
	const relative = `
[[rule]]
id = "scoped-to-nowhere"
action = "allow"
reason = "The directory is relative and cannot be placed."
  [[rule.clause]]
  cwd = "project"
`

	_, err := rules.Load([]string{inFiles(t, ruleSet(relative))})
	if err == nil {
		t.Fatal("Load = nil: a relative cwd must be refused at load, or the " +
			"rule silently applies nowhere")
	}
	if !errors.Is(err, rules.ErrRelativeCwd) {
		t.Errorf("Load = %v, want ErrRelativeCwd so the message says which "+
			"clause and why", err)
	}
}

// parseFor parses a command, failing the test if it will not.
func parseFor(t *testing.T, src string) *shell.Parsed {
	t.Helper()

	parsed, err := shell.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q) = %v", src, err)
	}
	return parsed
}

// ruleSet is the fixture most of these tests judge against.
func ruleSet(extra string) map[string]string {
	return map[string]string{"00.toml": declarations + extra}
}

// COVERS: FR-4.22 | negative
func TestNothingIsPermittedByDefault(t *testing.T) {
	t.Parallel()

	// Being in the allowed list MEANS an allow rule matched. A rule set with
	// no allow rules permits nothing, which is the correct reading of an empty
	// policy rather than a gap in one.
	outcome := judgeWith(t, ruleSet(""), `rm x`)

	if !outcome.Denied() {
		t.Errorf("Action = %q, want a refusal with no allow rule present", outcome.Action)
	}
	if len(outcome.Findings) != 1 {
		t.Fatalf("Findings = %v, want one", outcome.Findings)
	}
	if !strings.Contains(outcome.Findings[0].Reason, "allow rule matched") {
		t.Errorf("reason %q does not say what was missing", outcome.Findings[0].Reason)
	}
}

// COVERS: FR-4.22 | positive
func TestAnAllowRulePermits(t *testing.T) {
	t.Parallel()

	outcome := judgeWith(t, ruleSet(`
[[rule]]
id = "allow-rm"
action = "allow"
reason = "Removing a named file."
  [[rule.clause]]
  index = "0"
  value = "rm"
`), `rm x`)

	if outcome.Action != rules.ActionAllow {
		t.Errorf("Action = %q, want allow", outcome.Action)
	}
}

// COVERS: FR-4.14 | property
func TestTheStrictestRuleWins(t *testing.T) {
	t.Parallel()

	// Both rules match every one of these. Whichever was read first, the
	// verdict is the stricter, which is what makes rule order irrelevant.
	both := `
[[rule]]
id = "allow-rm"
action = "allow"
reason = "Removing a named file."
  [[rule.clause]]
  index = "0"
  value = "rm"

[[rule]]
id = "ask-recursive"
action = "ask"
reason = "This is recursive."
  [[rule.clause]]
  option = "recursive"

[[rule]]
id = "deny-force"
action = "deny"
reason = "Forcing suppresses the check that would have stopped this."
  [[rule.clause]]
  option = "force"
`

	cases := []struct {
		src  string
		want rules.Action
	}{
		{src: `rm x`, want: rules.ActionAllow},
		{src: `rm -r x`, want: rules.ActionAsk},
		{src: `rm -f x`, want: rules.ActionDeny},
		{src: `rm -rf x`, want: rules.ActionDeny},
	}

	for _, c := range cases {
		t.Run(c.src, func(t *testing.T) {
			t.Parallel()

			if got := judgeWith(t, ruleSet(both), c.src).Action; got != c.want {
				t.Errorf("Action for %q = %q, want %q", c.src, got, c.want)
			}
		})
	}
}

// COVERS: FR-4.25 | positive
func TestEveryReasonForARefusalIsListed(t *testing.T) {
	t.Parallel()

	// Two rules refuse this, and a message naming one of them sends its reader
	// round twice. The verdict was settled by the first; the list is what makes
	// the refusal actionable.
	outcome := judgeWith(t, ruleSet(`
[[rule]]
id = "deny-force"
action = "deny"
reason = "Forcing is not permitted."
  [[rule.clause]]
  option = "force"

[[rule]]
id = "deny-recursive-removal"
action = "deny"
reason = "Recursive removal is not permitted."
  [[rule.clause]]
  option = "recursive"
`), `rm -rf x`)

	if !outcome.Denied() {
		t.Fatalf("Action = %q, want deny", outcome.Action)
	}
	if len(outcome.Findings) != 2 {
		t.Errorf("Findings = %v, want both refusals", outcome.Reasons())
	}
}

// COVERS: FR-4.28 | property
func TestAVerdictNamesItsRuleAndQuotesTheCause(t *testing.T) {
	t.Parallel()

	outcome := judgeWith(t, ruleSet(`
[[rule]]
id = "deny-force"
action = "deny"
reason = "Forcing is not permitted."
  [[rule.clause]]
  option = "force"
`), `rm --force x`)

	if len(outcome.Findings) != 1 {
		t.Fatalf("Findings = %v, want one", outcome.Findings)
	}
	found := outcome.Findings[0]

	// A decision nobody can check is one nobody can correct.
	if found.Rule != "deny-force" {
		t.Errorf("Rule = %q, want the rule that refused", found.Rule)
	}
	if found.Reason == "" {
		t.Error("the refusal carries no reason")
	}
	if !strings.Contains(found.Cause, "force") {
		t.Errorf("Cause = %q, want it to quote what set the rule off", found.Cause)
	}
}

// COVERS: FR-4.25 | negative
func TestAnOutrankedRuleIsNotListedAmongTheReasons(t *testing.T) {
	t.Parallel()

	// Being told a command was both refused and permitted, in one message,
	// says nothing about which to act on.
	outcome := judgeWith(t, ruleSet(`
[[rule]]
id = "allow-rm"
action = "allow"
reason = "Removing a named file."
  [[rule.clause]]
  index = "0"
  value = "rm"

[[rule]]
id = "deny-force"
action = "deny"
reason = "Forcing is not permitted."
  [[rule.clause]]
  option = "force"
`), `rm -f x`)

	for _, finding := range outcome.Findings {
		if finding.Action != rules.ActionDeny {
			t.Errorf("a %q finding was listed among the reasons for a refusal", finding.Action)
		}
	}
}

// COVERS: FR-4.26 | negative
func TestMoreThanOneCommandIsRefusedOutright(t *testing.T) {
	t.Parallel()

	// The engine's, not a rule's: a rule file cannot omit it, and it holds
	// however the commands were joined.
	permissive := ruleSet(`
[[rule]]
id = "allow-anything"
action = "allow"
reason = "Everything."
  [[rule.clause]]
  index = "0"
  pattern = ".*"
`)

	for _, src := range []string{
		`rm a | rm b`,
		`rm a && rm b`,
		`rm a; rm b`,
		`rm $(echo b)`,
	} {
		t.Run(src, func(t *testing.T) {
			t.Parallel()

			outcome := judgeWith(t, permissive, src)
			if !outcome.Denied() {
				t.Errorf("Action for %q = %q, want deny", src, outcome.Action)
			}
			if !strings.Contains(outcome.Findings[0].Reason, "One command at a time") {
				t.Errorf("reason = %q, want the one-command refusal",
					outcome.Findings[0].Reason)
			}
		})
	}
}

// COVERS: FR-4.16 | negative
func TestAnUndeclaredCommandIsRefused(t *testing.T) {
	t.Parallel()

	// Even with a rule that would otherwise permit it. Nothing runs unless it
	// has been described.
	outcome := judgeWith(t, ruleSet(`
[[rule]]
id = "allow-anything"
action = "allow"
reason = "Everything."
  [[rule.clause]]
  index = "0"
  pattern = ".*"
`), `wibble x`)

	if !outcome.Denied() {
		t.Errorf("Action = %q, want deny for a command with no declaration", outcome.Action)
	}
	if !strings.Contains(outcome.Findings[0].Reason, "no declaration") {
		t.Errorf("reason = %q, want it to say what was missing", outcome.Findings[0].Reason)
	}
}

// COVERS: FR-4.24 | negative
func TestADeniedCommandHasNoEffect(t *testing.T) {
	t.Parallel()

	// A tag records that something happened, and a denied command did not
	// happen. The tag rule below matches; its effect must not be returned.
	tagging := `
[[rule]]
id = "note-it"
action = "tag"
tag = "did-something"
ttl = 6
reason = "Something worth remembering."
  [[rule.clause]]
  index = "0"
  value = "rm"

[[rule]]
id = "deny-force"
action = "deny"
reason = "Forcing is not permitted."
  [[rule.clause]]
  option = "force"

[[rule]]
id = "allow-rm"
action = "allow"
reason = "Removing a named file."
  [[rule.clause]]
  index = "0"
  value = "rm"
`

	denied := judgeWith(t, ruleSet(tagging), `rm -f x`)
	if !denied.Denied() {
		t.Fatalf("Action = %q, want deny", denied.Action)
	}
	if len(denied.Tags) != 0 {
		t.Errorf("Tags = %v, want none: the command did not happen", denied.Tags)
	}

	// The same tag rule on a command that is permitted does take effect.
	allowed := judgeWith(t, ruleSet(tagging), `rm x`)
	if allowed.Denied() {
		t.Fatalf("Action = %q, want allow", allowed.Action)
	}
	if len(allowed.Tags) != 1 {
		t.Fatalf("Tags = %v, want the one the rule set", allowed.Tags)
	}
	if allowed.Tags[0].Name != "did-something" || !allowed.Tags[0].Set || allowed.Tags[0].TTL != 6 {
		t.Errorf("Tags[0] = %+v, want did-something set with a ttl of 6", allowed.Tags[0])
	}
}

// COVERS: FR-4.27 | negative
func TestAClauseThatCannotBeAnsweredDoesNotMatch(t *testing.T) {
	t.Parallel()

	// git is declared, but nothing declares what `force` means to it. An allow
	// rule resting on that clause must not fire: qwark never permits on the
	// strength of its own ignorance.
	outcome := judgeWith(t, ruleSet(`
[[rule]]
id = "allow-forced-git"
action = "allow"
reason = "Only when forcing, which git does not declare."
  [[rule.clause]]
  index = "0"
  value = "git"

  [[rule.clause]]
  option = "force"
`), `git log`)

	if !outcome.Denied() {
		t.Errorf("Action = %q, want deny: the clause had no answer", outcome.Action)
	}
}

// COVERS: FR-4.16a | regression
func TestAnUndeclaredCommandStillGetsItsStructuralReasons(t *testing.T) {
	t.Parallel()

	// The declaration check used to run first and return, so a refusal said
	// only "this is undescribed" about a command that had also redirected.
	// Most rules need no declaration to answer: a clause naming node types,
	// operators, flags or a fact needs no table at all, and silently not
	// asking them made the refusal name one problem out of two.
	//
	// The verdict was never wrong. What was wrong was what the reader was told.
	outcome := judgeWith(t, ruleSet(`
[[rule]]
id = "no-redirection"
action = "deny"
reason = "Redirections are not permitted."
  [[rule.clause]]
  nodes = ["Redirect"]
`), `wibble > f`)

	if !outcome.Denied() {
		t.Fatalf("Action = %q, want deny", outcome.Action)
	}

	var undeclared, structural bool
	for _, finding := range outcome.Findings {
		if strings.Contains(finding.Reason, "no declaration") {
			undeclared = true
		}
		if finding.Rule == "no-redirection" {
			structural = true
		}
	}

	if !undeclared {
		t.Error("the refusal does not say the command was undeclared")
	}
	if !structural {
		t.Error("a rule that needed no declaration to answer was not consulted")
	}
}

// COVERS: FR-6.7 | negative
func TestAnUndeclaredOptionIsRefused(t *testing.T) {
	t.Parallel()

	// Decomposition recorded this from the beginning and the verdict did not
	// consult it, so `rm -Z x` was permitted by the allow rule below while the
	// fault sat unread beside it. An option nobody declared is the same
	// ignorance that refuses an undeclared command one level up: qwark does not
	// know what the command was told to do.
	outcome := judgeWith(t, ruleSet(`
[[rule]]
id = "allow-rm"
action = "allow"
reason = "Removing a named file."
  [[rule.clause]]
  index = "0"
  value = "rm"
`), `rm -Z x`)

	if !outcome.Denied() {
		t.Fatalf("Action = %q, want deny for an option the table does not declare",
			outcome.Action)
	}
	if !strings.Contains(outcome.Findings[0].Cause, "-Z") {
		t.Errorf("cause = %q, want the word that could not be accounted for",
			outcome.Findings[0].Cause)
	}
}

// COVERS: FR-6.8 | property
func TestEveryUnaccountedWordReachesTheVerdict(t *testing.T) {
	t.Parallel()

	// Reporting one fault at a time sends its reader round once per fault. The
	// decomposition gathers all of them precisely so a single refusal can say
	// everything that is wrong, and that is only true if the verdict carries
	// every one through.
	outcome := judgeWith(t, ruleSet(""), `rm -Z --nope x`)

	if !outcome.Denied() {
		t.Fatalf("Action = %q, want deny", outcome.Action)
	}

	var sawShort, sawLong bool
	for _, finding := range outcome.Findings {
		switch finding.Cause {
		case "-Z":
			sawShort = true
		case "--nope":
			sawLong = true
		}
	}

	if !sawShort || !sawLong {
		t.Errorf("findings = %v, want one for -Z and one for --nope: a refusal "+
			"that names the first fault only is a refusal its reader has to earn twice",
			outcome.Findings)
	}
}
