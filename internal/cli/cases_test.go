package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// COVERS: FR-4.37 | positive
func TestCasesPrintsAPassingEnvelope(t *testing.T) {
	t.Parallel()

	out, errOut, status := invoke(t, "", "cases", "../../rules/testdata")

	if status != statusOK {
		t.Fatalf("status = %d, want %d (stderr: %s)", status, statusOK, errOut)
	}
	if !strings.Contains(out, `"success": true`) || !strings.Contains(out, `"rules_triggered"`) {
		t.Errorf("envelope lacks the verdict or the rule count:\n%s", out)
	}
}

// COVERS: FR-4.37 | negative
func TestCasesPrintsAFailingEnvelopeAndExitsNonZero(t *testing.T) {
	t.Parallel()

	// A rule no case triggers: the envelope still prints, so the reason can
	// be read, and the status says it failed.
	dir := t.TempDir()
	for rel, body := range map[string]string{
		"rules.toml": "[declarations]\ndefault = \"allow\"\nrequired = false\naccounted = false\n" +
			"[[rule]]\nid = \"never\"\naction = \"block\"\nreason = \"r\"\n" +
			"  [[rule.clause]]\n  value = \"never-typed\"\n",
		"s/set.txt":           "../rules.toml\n",
		"s/allow/default.cmd": "echo hello\n",
	} {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	out, _, status := invoke(t, "", "cases", dir)

	if status != statusError {
		t.Errorf("status = %d, want %d", status, statusError)
	}
	failed := strings.Contains(out, `"success": false`)
	if !failed || !strings.Contains(out, "never: no case triggered") {
		t.Errorf("envelope does not name the untriggered rule:\n%s", out)
	}
}

// COVERS: FR-4.37 | negative
func TestCasesRefusesADirectoryThatIsNotASuite(t *testing.T) {
	t.Parallel()

	out, errOut, status := invoke(t, "", "cases", t.TempDir())
	if status != statusError || out != "" || !strings.Contains(errOut, "set.txt") {
		t.Errorf("status %d, stdout %q, stderr %q: want an error and no envelope", status, out, errOut)
	}

	if _, _, status := invoke(t, "", "cases"); status != statusUsage {
		t.Errorf("status with no directory = %d, want %d", status, statusUsage)
	}
}
