package rules_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/scriptedworld/qwark/internal/rules"
)

// ruleCase is one file under rules/testdata: rules/testdata/README.md says
// what the path and the header lines mean.
type ruleCase struct {
	Path    string
	Set     string
	Verdict rules.Action
	Rule    string
	Command string
	Headers map[string][]string
}

// defaultCwd is where a case's call comes from when it names nowhere.
const defaultCwd = "/srv/project"

// headerPrefix opens a header line in a case file.
const headerPrefix = "# "

// casesFS is the case directory, and nothing outside it is readable through it.
func casesFS() fs.FS { return os.DirFS(filepath.Join("..", "..", "rules", "testdata")) }

// readCase parses one case file, taking what it must decide from its path.
func readCase(t *testing.T, cases fs.FS, rel string) ruleCase {
	t.Helper()

	body, err := fs.ReadFile(cases, rel)
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	parts := strings.Split(rel, "/")
	stem := strings.TrimSuffix(parts[2], ".cmd")
	name, _, _ := strings.Cut(stem, "--")

	c := ruleCase{
		Path: rel, Set: parts[0], Verdict: rules.Action(parts[1]),
		Rule: name, Headers: map[string][]string{},
	}
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	for len(lines) > 0 && strings.HasPrefix(lines[0], headerPrefix) {
		key, value, _ := strings.Cut(strings.TrimPrefix(lines[0], headerPrefix), ":")
		c.Headers[key] = strings.Fields(value)
		lines = lines[1:]
	}
	c.Command = strings.Join(lines, "\n")
	return c
}

// shippedCases reads every case file, and refuses a file placed anywhere the
// layout does not give a meaning to.
func shippedCases(t *testing.T) []ruleCase {
	t.Helper()

	root := casesFS()
	paths, err := fs.Glob(root, "*/*/*.cmd")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no case files under rules/testdata: %v", err)
	}
	cases := make([]ruleCase, 0, len(paths))
	for _, path := range paths {
		c := readCase(t, root, path)
		if c.Set != "live" && c.Set != "full" {
			t.Errorf("%s: set %q is neither live nor full", c.Path, c.Set)
		}
		if !c.Verdict.Decides() {
			t.Errorf("%s: %q is not a verdict", c.Path, c.Verdict)
		}
		cases = append(cases, c)
	}
	return cases
}

// shippedSets loads the two rule sets a case can name.
func shippedSets(t *testing.T) map[string]*rules.Set {
	t.Helper()

	full, err := rules.Load([]string{filepath.Join("..", "..", "rules")})
	if err != nil {
		t.Fatalf("the full rule set does not load: %v", err)
	}
	return map[string]*rules.Set{"live": liveSet(t), "full": full}
}

// judgeCase evaluates one case against the set it names.
func judgeCase(t *testing.T, set *rules.Set, c ruleCase) rules.Outcome {
	t.Helper()

	ctx := rules.Context{Cwd: defaultCwd, Tags: map[string]bool{}}
	if cwd := c.Headers["cwd"]; len(cwd) == 1 {
		ctx.Cwd = cwd[0]
	}
	for _, tag := range c.Headers["tags"] {
		ctx.Tags[tag] = true
	}
	return set.Evaluate(parseFor(t, c.Command), ctx)
}

func fired(outcome rules.Outcome) []string {
	ids := make([]string, 0, len(outcome.Findings))
	for _, finding := range outcome.Findings {
		ids = append(ids, finding.Rule)
	}
	return ids
}

// changedTag reports whether the outcome sets, or clears, the named tag.
func changedTag(outcome rules.Outcome, name string, set bool) bool {
	for _, change := range outcome.Tags {
		if change.Name == name && change.Set == set {
			return true
		}
	}
	return false
}

// findRule returns the rule a case is named after.
func findRule(set *rules.Set, id string) (rules.Rule, bool) {
	for _, rule := range set.Rules {
		if rule.ID == id {
			return rule, true
		}
	}
	return rules.Rule{}, false
}

// Case names that stand for no rule: the engine's own refusal, or the verdict
// for a command no rule decided.
const (
	caseEngine  = "engine"
	caseDefault = "default"
)

// byEngine reports whether the engine itself refused.
func byEngine(outcome rules.Outcome) bool {
	return slices.ContainsFunc(fired(outcome), func(id string) bool {
		return strings.HasPrefix(id, "(engine)")
	})
}

// checkNamedRule reports whether the rule a case is named after did its part.
func checkNamedRule(t *testing.T, set *rules.Set, c ruleCase, outcome rules.Outcome) {
	t.Helper()

	rule, found := findRule(set, c.Rule)
	switch {
	case !found:
		t.Errorf("%s: no rule %q in the %s set", c.Path, c.Rule, c.Set)
	case rule.Action == rules.ActionTag || rule.Action == rules.ActionUntag:
		if !changedTag(outcome, rule.Tag, rule.Action == rules.ActionTag) {
			t.Errorf("%s: %s did not %s tag %s", c.Path, c.Rule, rule.Action, rule.Tag)
		}
	case !slices.Contains(fired(outcome), c.Rule):
		t.Errorf("%s: %s did not fire; reasons %v", c.Path, c.Rule, fired(outcome))
	}
}

// checkCase reports every way an outcome differs from its case.
func checkCase(t *testing.T, set *rules.Set, c ruleCase, outcome rules.Outcome) {
	t.Helper()

	ids := fired(outcome)
	if outcome.Action != c.Verdict {
		t.Errorf("%s: %s, want %s; reasons %v", c.Path, outcome.Action, c.Verdict, ids)
	}
	switch c.Rule {
	case caseDefault:
	case caseEngine:
		if !byEngine(outcome) {
			t.Errorf("%s: the engine did not refuse it; reasons %v", c.Path, ids)
		}
	default:
		checkNamedRule(t, set, c, outcome)
	}
	for _, want := range c.Headers["fires"] {
		if !slices.Contains(ids, want) {
			t.Errorf("%s: %s did not fire; reasons %v", c.Path, want, ids)
		}
	}
	for _, unwanted := range c.Headers["quiet"] {
		if slices.Contains(ids, unwanted) {
			t.Errorf("%s: %s fired and should not have", c.Path, unwanted)
		}
	}
}

// COVERS: FR-4.37 | property
func TestEveryShippedCaseGetsItsVerdict(t *testing.T) {
	t.Parallel()

	sets := shippedSets(t)
	for _, c := range shippedCases(t) {
		if set, known := sets[c.Set]; known {
			checkCase(t, set, c, judgeCase(t, set, c))
		}
	}
}

// COVERS: FR-4.37 | property
func TestEveryShippedRuleHasACase(t *testing.T) {
	t.Parallel()

	// A rule no case exercises is a rule nobody has seen decide anything, and
	// one that can never fire reads exactly like one that is working.
	named := map[string]bool{}
	for _, c := range shippedCases(t) {
		named[c.Rule] = true
	}
	var missing []string
	for _, rule := range shippedSets(t)["full"].Rules {
		if !named[rule.ID] {
			missing = append(missing, rule.ID)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d shipped rules have no case under rules/testdata: %v", len(missing), missing)
	}
}
