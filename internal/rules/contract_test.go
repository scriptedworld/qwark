package rules_test

import (
	"strings"
	"testing"
	"time"

	"github.com/scriptedworld/qwark/internal/rules"
)

// COVERS: FR-4.1 | positive
func TestEveryActionIsAccepted(t *testing.T) {
	t.Parallel()

	// Five, and the two tagging ones decide nothing. A rule file naming any
	// other is refused, which is tested separately; this pins the set itself.
	for _, action := range []rules.Action{
		rules.ActionAllow, rules.ActionAsk, rules.ActionDeny,
		rules.ActionTag, rules.ActionUntag,
	} {
		t.Run(string(action), func(t *testing.T) {
			t.Parallel()

			body := "[[rule]]\nid=\"a\"\naction=\"" + string(action) + "\"\n" +
				"tag=\"t\"\nttl=1\nreason=\"r\"\n[[rule.clause]]\nfact=\"pipe\"\n"

			set, err := rules.Load([]string{inFiles(t, map[string]string{"x.toml": body})})
			if err != nil {
				t.Fatalf("Load(%q) = %v", action, err)
			}
			if set.Rules[0].Action != action {
				t.Errorf("Action = %q, want %q", set.Rules[0].Action, action)
			}
		})
	}
}

// COVERS: FR-4.11 | negative
func TestEveryClauseMustHoldForARuleToApply(t *testing.T) {
	t.Parallel()

	// There is no disjunction inside a rule. A rule whose first clause matches
	// and whose second does not must not apply, otherwise "all clauses" would
	// quietly mean "any clause", and every conjunctive rule in the drafts would
	// be broader than it reads.
	twoClauses := `
[[rule]]
id = "allow-rm"
action = "allow"
reason = "So the command is not refused for a different reason."
  [[rule.clause]]
  index = "0"
  value = "rm"

[[rule]]
id = "both-or-neither"
action = "deny"
reason = "Both clauses held."
  [[rule.clause]]
  index = "0"
  value = "rm"

  [[rule.clause]]
  option = "force"
`

	if judgeWith(t, ruleSet(twoClauses), `rm -f x`).Action != rules.ActionDeny {
		t.Error("a rule with two matching clauses did not apply")
	}
	if judgeWith(t, ruleSet(twoClauses), `rm x`).Action != rules.ActionAllow {
		t.Error("a rule applied with only one of its two clauses matching")
	}
}

// COVERS: FR-4.13b, FR-4.23 | negative
func TestADeclarationPermitsNothingByItself(t *testing.T) {
	t.Parallel()

	// A declaration says how a command's options decompose so rules can be
	// written about them. It is not an entry on an allowed list, and treating
	// it as one would make describing a command in order to deny it precisely
	// into a way of permitting it.
	outcome := judgeWith(t, ruleSet(""), `rm x`)

	if !outcome.Denied() {
		t.Errorf("Action = %q: declaring rm permitted it", outcome.Action)
	}
	if !strings.Contains(outcome.Findings[0].Reason, "allow rule matched") {
		t.Errorf("reason = %q, want it to say nothing allowed the command",
			outcome.Findings[0].Reason)
	}
}

// COVERS: FR-4.20 | negative
func TestACommandFormWithNoCommandIsRefused(t *testing.T) {
	t.Parallel()

	// `((x=1))` and `let x=1` evaluate rather than run, and hold no command
	// for a declaration to be looked up by. Finding no command to check is not
	// the same as finding nothing to check.
	permissive := ruleSet(`
[[rule]]
id = "allow-anything"
action = "allow"
reason = "Everything."
  [[rule.clause]]
  index = "0"
  pattern = ".*"
`)

	for _, src := range []string{`((x=1))`, `let x=1`, `export PATH=/tmp`} {
		t.Run(src, func(t *testing.T) {
			t.Parallel()

			if !judgeWith(t, permissive, src).Denied() {
				t.Errorf("%q was not refused", src)
			}
		})
	}
}

// COVERS: FR-7.10 | property
func TestNoPatternCanMakeTheGateSlow(t *testing.T) {
	t.Parallel()

	// The classic catastrophic-backtracking shape. Under a backtracking engine
	// this takes exponential time in the length of the input; under RE2 it is
	// linear, which is why a rule file cannot carry a pattern that a crafted
	// command turns into a denial-of-service on the gate itself.
	match, err := rules.Pattern(`(a+)+b`)
	if err != nil {
		t.Fatalf("Pattern = %v", err)
	}

	const budget = 2 * time.Second
	started := time.Now()
	match.Matches(strings.Repeat("a", 64))
	if elapsed := time.Since(started); elapsed > budget {
		t.Errorf("a pathological pattern took %v, want well under %v", elapsed, budget)
	}
}
