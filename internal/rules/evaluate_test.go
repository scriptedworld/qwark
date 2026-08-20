package rules_test

import (
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
	// verdict is the stricter -- which is what makes rule order irrelevant.
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
	// Most rules need no declaration to answer -- a clause naming node types,
	// operators, flags or a fact needs no table at all -- and silently not
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
