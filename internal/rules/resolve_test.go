package rules_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/scriptedworld/qwark/internal/rules"
)

// protecting guards one path, written as member, against `rm`, while `rm`
// itself is allowed. A refusal therefore means the path rule fired, and an
// allow means it did not.
func protecting(member string) map[string]string {
	return ruleSet(`
[group.protected]
match = "partial"
members = ["` + member + `"]

[[rule]]
id = "allow-removing"
action = "allow"
reason = "Removing a named file is permitted."
  [[rule.clause]]
  index = "0"
  value = "rm"

[[rule]]
id = "no-touching-the-protected"
action = "block"
reason = "This reaches a protected path."
  [[rule.clause]]
  index = "0"
  value = "rm"
  [[rule.clause]]
  kind = "path"
  group = "protected"
`)
}

// linkedTree makes real/settings.json and a link beside it, link -> real, and
// returns the directory holding both.
func linkedTree(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	target := filepath.Join(dir, "real")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatalf("Mkdir = %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "settings.json"), nil, 0o600); err != nil {
		t.Fatalf("WriteFile = %v", err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "link")); err != nil {
		t.Fatalf("Symlink = %v", err)
	}
	return dir
}

func judgeIn(t *testing.T, files map[string]string, src, cwd string) rules.Outcome {
	t.Helper()

	set, err := rules.Load([]string{inFiles(t, files)})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}
	return set.Evaluate(parseFor(t, src), rules.Context{Cwd: cwd})
}

// COVERS FR-7.15 | positive
func TestAMemberWrittenAsTheLinkGuardsTheFileItPointsAt(t *testing.T) {
	t.Parallel()

	dir := linkedTree(t)
	link := filepath.Join(dir, "link", "settings.json")
	target := filepath.Join(dir, "real", "settings.json")

	outcome := judgeIn(t, protecting(link), "rm "+target, "")
	if !outcome.Denied() {
		t.Errorf("Action = %q, want a refusal: %s is the file the protected "+
			"link names", outcome.Action, target)
	}
}

// COVERS FR-7.15 | positive
func TestAMemberWrittenAsTheTargetGuardsTheLink(t *testing.T) {
	t.Parallel()

	dir := linkedTree(t)
	link := filepath.Join(dir, "link", "settings.json")

	outcome := judgeIn(t, protecting("/real/settings.json"), "rm "+link, "")
	if !outcome.Denied() {
		t.Errorf("Action = %q, want a refusal: %s resolves to the protected "+
			"file", outcome.Action, link)
	}
}

// COVERS FR-7.15 | edge
func TestARelativePathResolvesAgainstTheCallsDirectory(t *testing.T) {
	t.Parallel()

	dir := linkedTree(t)

	outcome := judgeIn(t, protecting("/real/settings.json"), "rm link/settings.json", dir)
	if !outcome.Denied() {
		t.Errorf("Action = %q, want a refusal: link/settings.json from %s is "+
			"the protected file", outcome.Action, dir)
	}
}

// COVERS FR-7.15 | negative
func TestResolvingDoesNotWidenAPathRule(t *testing.T) {
	t.Parallel()

	dir := linkedTree(t)
	other := filepath.Join(dir, "real", "other.json")

	outcome := judgeIn(t, protecting(filepath.Join(dir, "link", "settings.json")), "rm "+other, "")
	if outcome.Denied() {
		t.Errorf("Action = %q, want allow: %s is neither the protected file "+
			"nor a link to it", outcome.Action, other)
	}
}

// COVERS FR-7.15 | edge
func TestAFragmentMemberGuardsOnlyItsOwnSpelling(t *testing.T) {
	t.Parallel()

	// A fragment names no file, so there is nothing to resolve it to. It still
	// guards the link spelling, and the file behind the link stays open: the
	// reason a member that protects a link is written as a whole path.
	dir := linkedTree(t)
	link := filepath.Join(dir, "link", "settings.json")
	target := filepath.Join(dir, "real", "settings.json")
	files := protecting("/link/settings.json")

	if outcome := judgeIn(t, files, "rm "+link, ""); !outcome.Denied() {
		t.Errorf("Action = %q, want a refusal: the fragment matches %s as "+
			"written", outcome.Action, link)
	}
	if outcome := judgeIn(t, files, "rm "+target, ""); outcome.Denied() {
		t.Errorf("Action = %q, want allow: a fragment cannot be resolved, so "+
			"%s is not reached through it", outcome.Action, target)
	}
}
