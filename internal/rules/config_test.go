package rules_test

import (
	"testing"

	"github.com/scriptedworld/qwark/internal/rules"
)

// COVERS: FR-4.14 | property
func TestTheStrictestActionOutranksTheRest(t *testing.T) {
	t.Parallel()

	// This ordering is the reason rule order never changes a verdict, and the
	// reason no file can weaken another by being read later. It is stated once
	// here so a second comparison written elsewhere cannot disagree with it.
	ordered := []rules.Action{rules.ActionAllow, rules.ActionAsk, rules.ActionDeny}

	for i := 1; i < len(ordered); i++ {
		looser, stricter := ordered[i-1], ordered[i]
		if stricter.Strictness() <= looser.Strictness() {
			t.Errorf("%q does not outrank %q", stricter, looser)
		}
	}
}

// COVERS: FR-8.1, FR-4.25a | negative
func TestTaggingDecidesNothing(t *testing.T) {
	t.Parallel()

	// A tag attaches a name for later rules to match. If it ranked alongside
	// the verdicts, a tag rule would be able to permit a command by existing.
	for _, action := range []rules.Action{rules.ActionTag, rules.ActionUntag} {
		if action.Decides() {
			t.Errorf("%q reports itself as a verdict", action)
		}
		if got := action.Strictness(); got != 0 {
			t.Errorf("%q has strictness %d, want 0", action, got)
		}
	}

	for _, action := range []rules.Action{
		rules.ActionAllow, rules.ActionAsk, rules.ActionDeny,
	} {
		if !action.Decides() {
			t.Errorf("%q does not report itself as a verdict", action)
		}
	}
}

// COVERS: FR-4.4 | edge
func TestSomethingThatIsNotAnActionRanksBelowEverything(t *testing.T) {
	t.Parallel()

	// Load refuses an unknown action, so this can only be reached by building
	// a Rule in code. It must still fail closed: never a verdict, and never
	// outranking one.
	var unknown rules.Action = "wibble"

	if unknown.Decides() {
		t.Error("an unknown action reported itself as a verdict")
	}
	if got := unknown.Strictness(); got != 0 {
		t.Errorf("strictness = %d, want 0", got)
	}
}
