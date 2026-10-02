package rules_test

import (
	"testing"

	"github.com/scriptedworld/qwark/internal/rules"
)

// setting and clearing are two rules on one command, so each can be judged
// with the tag live and with it absent.
const setting = `
[[rule]]
id = "allow-rm"
action = "allow"
reason = "Removing a named file."
  [[rule.clause]]
  index = "0"
  value = "rm"

[[rule]]
id = "mark-it"
action = "tag"
tag = "marked"
ttl = 6
reason = "Something worth remembering."
  [[rule.clause]]
  index = "0"
  value = "rm"
`

const clearing = `
[[rule]]
id = "allow-rm"
action = "allow"
reason = "Removing a named file."
  [[rule.clause]]
  index = "0"
  value = "rm"

[[rule]]
id = "clear-it"
action = "untag"
tag = "marked"
reason = "The condition is satisfied."
  [[rule.clause]]
  index = "0"
  value = "rm"
`

// tagChange judges `rm x` with the tag live or not and returns the one change
// the rule set produced.
func tagChange(t *testing.T, extra string, live bool) rules.TagChange {
	t.Helper()

	set, err := rules.Load([]string{inFiles(t, ruleSet(extra))})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}
	outcome := set.Evaluate(parseFor(t, `rm x`), rules.Context{Tags: map[string]bool{"marked": live}})
	if len(outcome.Tags) != 1 {
		t.Fatalf("Tags = %v, want exactly one change", outcome.Tags)
	}
	return outcome.Tags[0]
}

// COVERS FR-8.5 | property
func TestATagRuleSetsOrClearsWhateverTheTagWas(t *testing.T) {
	t.Parallel()

	// A toggle would clear a live tag and set an absent one, so its effect
	// would depend on state the reader of the rule cannot see. Each rule here
	// gives the same change from either starting point.
	for _, live := range []bool{false, true} {
		if change := tagChange(t, setting, live); change.Name != "marked" || !change.Set {
			t.Errorf("live=%v: tag rule gave %+v, want marked set", live, change)
		}
		if change := tagChange(t, clearing, live); change.Name != "marked" || change.Set {
			t.Errorf("live=%v: untag rule gave %+v, want marked cleared", live, change)
		}
	}
}
