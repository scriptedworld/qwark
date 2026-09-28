package rules_test

import (
	"path/filepath"
	"testing"

	"github.com/scriptedworld/qwark/internal/rules"
)

// liveTrio is the observation-phase set as it is installed: permission by
// default, the structural blocks, and the inline-code blocks.
func liveTrio(t *testing.T) *rules.Set {
	t.Helper()

	var paths []string
	for _, name := range []string{"00-allow.toml", "01-structure.toml", "02-inline-code.toml"} {
		paths = append(paths, filepath.Join("..", "..", "rules", name))
	}
	set, err := rules.Load(paths)
	if err != nil {
		t.Fatalf("the live rule files do not load: %v", err)
	}
	return set
}

// COVERS: FR-4.36 | positive
func TestTheLiveSetBlocksAProgramWrittenIntoTheCommandLine(t *testing.T) {
	t.Parallel()

	// One or more spellings per interpreter family, bundled and attached
	// forms included.
	set := liveTrio(t)
	for _, src := range []string{
		`python3 -c 'print(1)'`, `/usr/bin/python3 -c pass`, `python3 -Ic pass`,
		`python3 -cpass`, `bash -c ls`, `zsh -lc ls`, `fish --command ls`,
		`perl -ne 'print'`, `ruby -e 'p 1'`, `node -pe 1`, `node --eval=1`,
		`deno eval 1`, `php -r 'echo 1;'`,
	} {
		if got := set.Evaluate(parseFor(t, src), rules.Context{}).Action; got != rules.ActionBlock {
			t.Errorf("%q = %q, want block", src, got)
		}
	}
}

// COVERS: FR-4.36 | negative
func TestTheLiveSetStillRunsAScriptFile(t *testing.T) {
	t.Parallel()

	// The refusal is of code that was never a file. A script is a file, and
	// so is a module run with -m. A -c belonging to another command, or an
	// interpreter's name given as an argument, is not an inline program.
	set := liveTrio(t)
	for _, src := range []string{
		`python3 tool.py`, `python3 -m pytest`, `bash script.sh`, `node server.js`,
		`grep -c python3 notes`, `git log -c`,
	} {
		if got := set.Evaluate(parseFor(t, src), rules.Context{}).Action; got != rules.ActionAllow {
			t.Errorf("%q = %q, want allow", src, got)
		}
	}
}
