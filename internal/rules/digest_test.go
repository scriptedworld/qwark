package rules_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/scriptedworld/qwark/internal/rules"
)

// planted writes a minimal loadable rule set into a directory and returns it.
func plantedSet(t *testing.T, body string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "00.toml"), []byte(body), 0o600); err != nil {
		t.Fatalf("writing the rule file: %v", err)
	}
	return dir
}

// minimal is a rule set that loads and decides nothing.
const minimal = "[shell]\nallow=[\"/bin/bash\"]\n"

// COVERS: FR-4.9b | positive
func TestALoadedSetCarriesADigest(t *testing.T) {
	t.Parallel()

	set, err := rules.Load([]string{plantedSet(t, minimal)})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}

	if set.Digest == "" {
		t.Fatal("Digest is empty, so a log entry cannot say which policy judged it")
	}
}

// COVERS: FR-4.9b | property
func TestTheDigestIsOverContentAndNotPaths(t *testing.T) {
	t.Parallel()

	// This is what makes the digest a drift check as well as an identifier. The
	// same rules installed live and sitting in the repository must hash the
	// same, because the question anybody asks of two copies is whether they are
	// the same policy, and a digest including the path could never answer it.
	first, err := rules.Load([]string{plantedSet(t, minimal)})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}
	second, err := rules.Load([]string{plantedSet(t, minimal)})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}

	if first.Digest != second.Digest {
		t.Errorf("same content in different directories gave %q and %q, want one digest",
			first.Digest, second.Digest)
	}
}

// COVERS: FR-4.9b | negative
func TestChangingARuleChangesTheDigest(t *testing.T) {
	t.Parallel()

	// A digest that survived an edit would let entries from two policies be
	// compared as though they came from one, which is the whole thing this
	// exists to stop.
	before, err := rules.Load([]string{plantedSet(t, minimal)})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}

	after, err := rules.Load([]string{plantedSet(t, minimal+
		"\n[[rule]]\nid=\"x\"\naction=\"deny\"\nreason=\"y\"\n"+
		"  [[rule.clause]]\n  index=\"0\"\n  value=\"z\"\n")})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}

	if before.Digest == after.Digest {
		t.Errorf("adding a rule left the digest at %q", before.Digest)
	}
}

// COVERS: FR-4.9b | edge
func TestTheDigestIsStableAcrossLoads(t *testing.T) {
	t.Parallel()

	// Reading the same files twice must give the same answer, or every entry
	// written after a restart looks like it came from a different policy.
	dir := plantedSet(t, minimal)

	first, err := rules.Load([]string{dir})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}
	second, err := rules.Load([]string{dir})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}

	if first.Digest != second.Digest {
		t.Errorf("two loads of one directory gave %q and %q", first.Digest, second.Digest)
	}
}
