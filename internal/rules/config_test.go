package rules_test

import (
	"testing"

	"github.com/scriptedworld/qwark/internal/rules"
)

// COVERS FR-4.32 | property
func TestTheHigherPrecedenceActionOutranksTheRest(t *testing.T) {
	t.Parallel()

	// This ordering is the reason rule order never changes a verdict. It is
	// stated once here so a second comparison written elsewhere cannot
	// disagree with it.
	ordered := []rules.Action{
		rules.ActionDeny, rules.ActionAsk, rules.ActionAllow, rules.ActionBlock,
	}

	for i := 1; i < len(ordered); i++ {
		lower, higher := ordered[i-1], ordered[i]
		if higher.Precedence() <= lower.Precedence() {
			t.Errorf("%q does not outrank %q", higher, lower)
		}
	}
}

// COVERS FR-8.1 | negative
func TestTaggingDecidesNothing(t *testing.T) {
	t.Parallel()

	// A tag attaches a name for later rules to match. If it ranked alongside
	// the verdicts, a tag rule would be able to permit a command by existing.
	for _, action := range []rules.Action{rules.ActionTag, rules.ActionUntag} {
		if action.Decides() {
			t.Errorf("%q reports itself as a verdict", action)
		}
		if got := action.Precedence(); got != 0 {
			t.Errorf("%q has precedence %d, want 0", action, got)
		}
	}

	for _, action := range []rules.Action{
		rules.ActionBlock, rules.ActionAllow, rules.ActionAsk, rules.ActionDeny,
	} {
		if !action.Decides() {
			t.Errorf("%q does not report itself as a verdict", action)
		}
	}
}

// COVERS FR-4.4 | edge
func TestSomethingThatIsNotAnActionRanksBelowEverything(t *testing.T) {
	t.Parallel()

	// Load refuses an unknown action, so this can only be reached by building
	// a Rule in code. It must still fail closed: never a verdict, and never
	// outranking one.
	var unknown rules.Action = "wibble"

	if unknown.Decides() {
		t.Error("an unknown action reported itself as a verdict")
	}
	if got := unknown.Precedence(); got != 0 {
		t.Errorf("precedence = %d, want 0", got)
	}
}
