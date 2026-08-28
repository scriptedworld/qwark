package rules_test

import (
	"strings"
	"testing"

	"github.com/scriptedworld/qwark/internal/rules"
	"github.com/scriptedworld/qwark/internal/shell"
)

// judged evaluates one command against a rule set written inline.
func judged(t *testing.T, body, command string) rules.Outcome {
	t.Helper()

	set, err := rules.Load([]string{plantedSet(t, body)})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}

	parsed, err := shell.Parse(command)
	if err != nil {
		t.Fatalf("Parse(%q) = %v", command, err)
	}

	return set.Evaluate(parsed, rules.Context{})
}

// permits is a rule set that allows any command carrying a name, so that what
// is being measured is the engine's own refusals rather than the absence of an
// allow rule.
const permits = "[shell]\nallow=[\"/bin/bash\"]\n" +
	"\n[[rule]]\nid=\"allow-anything\"\naction=\"allow\"\nreason=\"testing\"\n" +
	"  [[rule.clause]]\n  index=\"0\"\n  pattern=\".*\"\n"

// declaring is `permits` plus a declaration for `rm` naming one option, so an
// option the table does not carry can be exercised.
const declaring = permits +
	"\n[command.rm]\noperands=\"path\"\n  short.f = { means = \"force\" }\n"

// says returns whether any finding names a rule containing text.
func says(outcome rules.Outcome, text string) bool {
	for _, finding := range outcome.Findings {
		if strings.Contains(finding.Rule, text) {
			return true
		}
	}
	return false
}

// COVERS: FR-4.16 | positive
func TestAnUndeclaredCommandIsRefusedByDefault(t *testing.T) {
	t.Parallel()

	// The default has to be the strict one. A rule set that says nothing about
	// declarations gets FR-4.16 as written, so a file nobody edited cannot
	// quietly be the reason an unknown command ran.
	outcome := judged(t, permits, "somethingnobodydeclared --wild")

	if outcome.Action != rules.ActionDeny {
		t.Errorf("action = %v, want deny for an undeclared command", outcome.Action)
	}
	if !says(outcome, "declared commands only") {
		t.Errorf("findings = %+v, want the declaration refusal", outcome.Findings)
	}
}

// COVERS: FR-4.16 | negative
func TestARuleSetMaySayDeclarationsAreNotRequired(t *testing.T) {
	t.Parallel()

	// This is what makes a structural-only phase possible. FR-4.16 arrives
	// before shape decides anything, so requiring it means refusing every
	// command rather than judging the ones the structural rules understand.
	outcome := judged(t,
		permits+"\n[declarations]\nrequired = false\n",
		"somethingnobodydeclared --wild")

	if outcome.Action != rules.ActionAllow {
		t.Errorf("action = %v, want allow when declarations are not required",
			outcome.Action)
	}
	if says(outcome, "declared commands only") {
		t.Errorf("findings = %+v, want the declaration refusal gone", outcome.Findings)
	}
}

// COVERS: FR-6.7 | positive
func TestAnUndeclaredOptionIsRefusedByDefault(t *testing.T) {
	t.Parallel()

	// `rm` is declared and carries only `-f`, so `-r` is an option qwark cannot
	// account for. Refusing it is what makes the declaration table fail closed:
	// leaving an option out costs a refusal rather than a hole.
	outcome := judged(t, declaring, "rm -r somewhere")

	if outcome.Action != rules.ActionDeny {
		t.Errorf("action = %v, want deny for an option no declaration names",
			outcome.Action)
	}
	if !says(outcome, "accounted options only") {
		t.Errorf("findings = %+v, want the option refusal", outcome.Findings)
	}
}

// COVERS: FR-6.7 | negative
func TestARuleSetMaySayOptionsNeedNotBeAccounted(t *testing.T) {
	t.Parallel()

	// The second switch, and it is separate from the first for a reason worth
	// pinning: the two refusals sit at different levels.
	outcome := judged(t,
		declaring+"\n[declarations]\naccounted = false\n",
		"rm -r somewhere")

	if outcome.Action != rules.ActionAllow {
		t.Errorf("action = %v, want allow when options need not be accounted",
			outcome.Action)
	}
	if says(outcome, "accounted options only") {
		t.Errorf("findings = %+v, want the option refusal gone", outcome.Findings)
	}
}

// COVERS: FR-6.7 | property
func TestTurningOffTheCommandCheckDoesNotTurnOffTheOptionCheck(t *testing.T) {
	t.Parallel()

	// The trap this pins: with `required = false` and nothing declared, no
	// option is ever examined, so `accounted` looks unnecessary. Declare one
	// command and every option it carries starts being refused again. A phase
	// that wants neither has to say so twice, and this is what makes that
	// checkable rather than remembered.
	outcome := judged(t,
		declaring+"\n[declarations]\nrequired = false\n",
		"rm -r somewhere")

	if outcome.Action != rules.ActionDeny {
		t.Errorf("action = %v, want the option check still refusing", outcome.Action)
	}
	if !says(outcome, "accounted options only") {
		t.Errorf("findings = %+v, want `required = false` to leave FR-6.7 alone",
			outcome.Findings)
	}
}

// COVERS: FR-4.13a | negative
func TestTwoFilesCannotBothDecideWhetherDeclarationsAreRequired(t *testing.T) {
	t.Parallel()

	// Turning FR-4.16 off is the widest change a rule file can make, so it must
	// trace to one file that said so. Two files each carrying an opinion would
	// leave the answer depending on which was read last.
	dir := plantedSet(t, permits+"\n[declarations]\nrequired = false\n")
	second := plantedSet(t, "[declarations]\nrequired = true\n")

	_, err := rules.Load([]string{dir, second})

	if err == nil {
		t.Fatal("Load accepted two files both deciding whether declarations are required")
	}
	if !strings.Contains(err.Error(), "declarations") {
		t.Errorf("error = %v, want it to name what collided", err)
	}
}
