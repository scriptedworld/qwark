package rules_test

import (
	"testing"

	"github.com/scriptedworld/qwark/internal/rules"
)

// COVERS: FR-4.38 | positive
func TestTheLiveSetKeepsGitInTheRepositoryItActsOn(t *testing.T) {
	t.Parallel()

	// Every spelling that points git elsewhere, a commit included, and one
	// behind another global option so matching only the word after git would
	// miss it.
	set := liveSet(t)
	for _, src := range []string{
		`git -C /srv/other status`, `git --no-pager -C /srv/other log`,
		`git --git-dir=/srv/other/.git log`, `git --git-dir /srv/other/.git log`,
		`git --work-tree=/srv/other status`, `git -C /srv/other commit f -F m`,
	} {
		if got := set.Evaluate(parseFor(t, src), rules.Context{}).Action; got != rules.ActionBlock {
			t.Errorf("%q = %q, want block", src, got)
		}
	}
}

// COVERS: FR-4.38 | negative
func TestTheLiveSetRunsGitInPlace(t *testing.T) {
	t.Parallel()

	set := liveSet(t)
	for _, src := range []string{`git status`, `git log --find-copies`, `git commit f -F m`} {
		if got := set.Evaluate(parseFor(t, src), rules.Context{}).Action; got != rules.ActionAllow {
			t.Errorf("%q = %q, want allow", src, got)
		}
	}
}
