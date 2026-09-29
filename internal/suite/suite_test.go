package suite_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scriptedworld/qwark/internal/suite"
)

// twoRules blocks two unlike things, so a suite can trigger one and miss the
// other. Everything else is allowed by default.
const twoRules = `
[declarations]
required  = false
accounted = false
default   = "allow"

[[rule]]
id     = "no-shouting"
action = "block"
reason = "r"
  [[rule.clause]]
  value = "LOUD"

[[rule]]
id     = "no-whispering"
action = "block"
reason = "r"
  [[rule.clause]]
  value = "quiet"
`

// plant writes a suite into a temporary directory: the rule file, a set
// naming it, and the given case files keyed by their path under the set.
func plant(t *testing.T, cases map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	files := map[string]string{
		"rules.toml":   twoRules,
		"only/set.txt": "# the one set\n../rules.toml\n",
	}
	for rel, body := range cases {
		files[filepath.Join("only", rel)] = body
	}
	for rel, body := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func run(t *testing.T, dir string) suite.Envelope {
	t.Helper()

	env, err := suite.Run(dir)
	if err != nil {
		t.Fatalf("Run = %v", err)
	}
	return env
}

func hasReason(env suite.Envelope, kind, text string) bool {
	for _, reason := range env.Reasons {
		if reason.Kind == kind && strings.Contains(reason.Message, text) {
			return true
		}
	}
	return false
}

// COVERS: FR-4.37 | positive
func TestTheShippedCasesPass(t *testing.T) {
	t.Parallel()

	env := run(t, filepath.Join("..", "..", "rules", "testdata"))

	if !env.Success {
		t.Fatalf("the shipped suite failed: %+v", env.Reasons)
	}
	stats := env.Metadata.Statistics
	if stats.Rules == 0 || stats.RulesTriggered != stats.Rules {
		t.Errorf("rules %d, triggered %d: want every included rule triggered",
			stats.Rules, stats.RulesTriggered)
	}
}

// COVERS: FR-4.37 | negative
func TestARuleNoCaseTriggersFailsTheRun(t *testing.T) {
	t.Parallel()

	// Adding a rule without a case is what this exists to catch, so the rule
	// is named in a reason of its own kind.
	env := run(t, plant(t, map[string]string{
		"block/no-shouting.cmd": "echo LOUD\n",
		"allow/default.cmd":     "echo hello\n",
	}))

	if env.Success {
		t.Fatal("a run with an untriggered rule passed")
	}
	if !hasReason(env, suite.KindRuleNotTriggered, "no-whispering") {
		t.Errorf("reasons %+v, want no-whispering named as not triggered", env.Reasons)
	}
	if hasReason(env, suite.KindRuleNotTriggered, "no-shouting") {
		t.Errorf("reasons %+v name a rule a case did trigger", env.Reasons)
	}
	if got := env.Metadata.Statistics; got.Rules != 2 || got.RulesTriggered != 1 {
		t.Errorf("statistics %+v, want 1 of 2 rules triggered", got)
	}
}

// COVERS: FR-4.37a | negative
func TestARuleNoCaseHoldsQuietFailsTheRun(t *testing.T) {
	t.Parallel()

	// Both rules fire, so only the near-miss check can fail this. The quiet
	// header on the rule that fired anyway does not count as holding it.
	env := run(t, plant(t, map[string]string{
		"block/no-shouting.cmd":            "# quiet: no-whispering\necho LOUD\n",
		"block/no-whispering.cmd":          "echo quiet\n",
		"block/no-whispering--both.cmd":    "# quiet: no-shouting\necho quiet LOUD\n",
		"allow/default--nothing-fires.cmd": "echo hello\n",
	}))

	if env.Success {
		t.Fatal("a run with a rule nothing held quiet passed")
	}
	if !hasReason(env, suite.KindRuleNotQuieted, "no-shouting") {
		t.Errorf("reasons %+v, want no-shouting named as not quieted", env.Reasons)
	}
	if hasReason(env, suite.KindRuleNotQuieted, "no-whispering") {
		t.Errorf("reasons %+v name a rule a case did hold quiet", env.Reasons)
	}
	if got := env.Metadata.Statistics; got.RulesTriggered != 2 || got.RulesQuieted != 1 {
		t.Errorf("statistics %+v, want 2 triggered and 1 quieted", got)
	}
}

// COVERS: FR-4.37 | negative
func TestACaseWithTheWrongVerdictFailsTheRun(t *testing.T) {
	t.Parallel()

	env := run(t, plant(t, map[string]string{
		"block/no-shouting.cmd":    "echo LOUD\n",
		"block/no-whispering.cmd":  "echo quiet\n",
		"allow/default--wrong.cmd": "echo LOUD\n",
	}))

	if env.Success || !hasReason(env, suite.KindCaseFailed, "allow/default--wrong.cmd") {
		t.Errorf("reasons %+v, want the misfiled case named", env.Reasons)
	}
	if env.Metadata.Statistics.CasesFailed != 1 {
		t.Errorf("cases_failed = %d, want 1", env.Metadata.Statistics.CasesFailed)
	}
}

// COVERS: FR-4.37 | negative
func TestACaseMustTriggerTheRuleItIsNamedAfter(t *testing.T) {
	t.Parallel()

	// Blocked, as its directory says, but by the other rule: the name is a
	// claim too.
	env := run(t, plant(t, map[string]string{
		"block/no-shouting.cmd":    "echo quiet LOUD\n",
		"block/no-whispering.cmd":  "echo quiet\n",
		"block/no-shouting--x.cmd": "# quiet: no-whispering\n# fires: no-shouting\necho quiet\n",
		"block/engine--pipe.cmd":   "echo LOUD\n",
	}))

	for _, want := range []string{
		"no-shouting--x.cmd: no-shouting did not fire",
		"no-shouting--x.cmd: no-whispering fired",
		"engine--pipe.cmd: the engine did not refuse it",
	} {
		if !hasReason(env, suite.KindCaseFailed, want) {
			t.Errorf("reasons %+v, want %q", env.Reasons, want)
		}
	}
}

// COVERS: FR-4.37 | edge
func TestHeadersSetTheContextACaseIsJudgedIn(t *testing.T) {
	t.Parallel()

	env := run(t, plant(t, map[string]string{
		"block/no-shouting.cmd":    "# tags: anything\n# cwd: /elsewhere\necho LOUD\n",
		"block/no-whispering.cmd":  "echo quiet\n",
		"block/engine--pipe.cmd":   "echo a | echo b\n",
		"block/default--parse.cmd": "echo )\n",
	}))

	if !hasReason(env, suite.KindCaseFailed, "does not parse") {
		t.Errorf("reasons %+v, want the unparseable case reported", env.Reasons)
	}
	held := []string{"no-shouting.cmd", "engine--pipe"}
	if hasReason(env, suite.KindCaseFailed, held[0]) || hasReason(env, suite.KindCaseFailed, held[1]) {
		t.Errorf("reasons %+v fail a case that held", env.Reasons)
	}
}

// COVERS: FR-4.37 | negative
func TestADirectoryThatIsNotASuiteIsAnError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		files map[string]string
		want  error
	}{
		{name: "no cases", files: map[string]string{}, want: suite.ErrNoCases},
		{
			name:  "not a verdict",
			files: map[string]string{"maybe/x.cmd": "ls\n"},
			want:  suite.ErrNoVerdict,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			if _, err := suite.Run(plant(t, c.files)); !errors.Is(err, c.want) {
				t.Errorf("Run = %v, want %v", err, c.want)
			}
		})
	}

	if _, err := suite.Run(t.TempDir()); !errors.Is(err, suite.ErrNoSets) {
		t.Errorf("Run on an empty directory = %v, want %v", err, suite.ErrNoSets)
	}
}

// COVERS: FR-4.37 | negative
func TestACaseUnderASetWithNoListingIsAnError(t *testing.T) {
	t.Parallel()

	// A set is only what its set.txt says it is, so a case filed under a
	// directory with none would be judged against nothing.
	dir := plant(t, map[string]string{"block/no-shouting.cmd": "echo LOUD\n"})
	stray := filepath.Join(dir, "other", "block", "x.cmd")
	if err := os.MkdirAll(filepath.Dir(stray), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stray, []byte("ls\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := suite.Run(dir); !errors.Is(err, suite.ErrUnknown) {
		t.Errorf("Run = %v, want %v", err, suite.ErrUnknown)
	}
}

// COVERS: FR-4.37 | negative
func TestASetWhoseRulesWillNotLoadIsAnError(t *testing.T) {
	t.Parallel()

	dir := plant(t, map[string]string{"block/no-shouting.cmd": "echo LOUD\n"})
	listing := filepath.Join(dir, "only", "set.txt")
	if err := os.WriteFile(listing, []byte("../missing.toml\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := suite.Run(dir); err == nil {
		t.Error("Run accepted a set whose rule file does not exist")
	}
}
