package rules_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scriptedworld/qwark/internal/rules"
)

// inFiles writes a rule set to a fresh directory and returns its path. Files
// are named by the caller because which file a definition came from is part of
// what is being tested.
func inFiles(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	return dir
}

// mustFail loads a rule set that is expected to be refused, and returns the
// refusal's message.
//
// The message rather than the error, because that is what the caller checks and
// because a helper handing an error back up would be laundering one across a
// package boundary without wrapping it.
func mustFail(t *testing.T, files map[string]string, want error) string {
	t.Helper()

	_, err := rules.Load([]string{inFiles(t, files)})
	if err == nil {
		t.Fatal("Load accepted a rule set that should have been refused")
	}
	if !errors.Is(err, want) {
		t.Errorf("Load = %v, want %v", err, want)
		return ""
	}
	return err.Error()
}

// COVERS: FR-4.15 | positive
func TestADirectoryContributesEveryRuleFileInIt(t *testing.T) {
	t.Parallel()

	dir := inFiles(t, map[string]string{
		"01-first.toml":  "[[rule]]\nid = \"a\"\naction = \"deny\"\n[[rule.clause]]\nfact = \"pipe\"\n",
		"02-second.toml": "[[rule]]\nid = \"b\"\naction = \"ask\"\n[[rule.clause]]\nfact = \"glob\"\n",
		"notes.md":       "not a rule file",
	})

	set, err := rules.Load([]string{dir})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}
	if len(set.Rules) != 2 {
		t.Errorf("loaded %d rules, want 2 (and nothing from notes.md)", len(set.Rules))
	}
}

// COVERS: FR-4.15 | negative
func TestNothingToLoadIsARefusal(t *testing.T) {
	t.Parallel()

	// An empty rule set would permit everything a deny rule would have caught.
	// Finding no rules is not the same as having no rules to apply.
	if _, err := rules.Load(nil); !errors.Is(err, rules.ErrNoRuleFiles) {
		t.Errorf("Load(nil) = %v, want %v", err, rules.ErrNoRuleFiles)
	}
	if _, err := rules.Load([]string{t.TempDir()}); !errors.Is(err, rules.ErrNoRulesFound) {
		t.Errorf("Load(empty dir) = %v, want %v", err, rules.ErrNoRulesFound)
	}
	if _, err := rules.Load([]string{"/nonexistent/x.toml"}); !errors.Is(err, rules.ErrUnreadable) {
		t.Errorf("Load(missing) = %v, want %v", err, rules.ErrUnreadable)
	}
}

// COVERS: FR-4.6 | negative
func TestASyntaxErrorNamesTheFileAndThePosition(t *testing.T) {
	t.Parallel()

	// Until this is fixed every command is denied, so the message is the only
	// thing the reader has to work from.
	message := mustFail(t, map[string]string{
		"broken.toml": "this is not = valid toml [[[\n",
	}, rules.ErrSyntax)

	for _, want := range []string{"broken.toml", "line 1"} {
		if !strings.Contains(message, want) {
			t.Errorf("message %q does not mention %q", message, want)
		}
	}
}

// COVERS: FR-4.13a | negative
func TestOneFileMayNotRedefineAnothersDefinition(t *testing.T) {
	t.Parallel()

	// Naming only the loser would leave the reader hunting for the other one.
	message := mustFail(t, map[string]string{
		"01.toml": "[group.tools]\nmembers = [\"a\"]\n",
		"02.toml": "[group.tools]\nmembers = [\"b\"]\n",
	}, rules.ErrRedefined)

	for _, want := range []string{"01.toml", "02.toml", "tools"} {
		if !strings.Contains(message, want) {
			t.Errorf("message %q does not name %q", message, want)
		}
	}
}

// COVERS: FR-4.13a | positive
func TestAFileMayCreateItsOwnDefinitions(t *testing.T) {
	t.Parallel()

	set, err := rules.Load([]string{inFiles(t, map[string]string{
		"01.toml": "[group.tools]\nmembers = [\"a\"]\n",
		"02.toml": "[group.others]\nmembers = [\"b\"]\n[command.rm]\noperands = \"path\"\n",
	})})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}

	if len(set.Groups) != 2 {
		t.Errorf("loaded %d groups, want 2", len(set.Groups))
	}
	if _, declared := set.Commands["rm"]; !declared {
		t.Error("a file's own command declaration was not kept")
	}
}

// COVERS: FR-4.5 | negative
func TestARuleThatCannotBeTrustedIsRefused(t *testing.T) {
	t.Parallel()

	// Each of these would otherwise produce a rule that never applies, which
	// reads exactly like a rule that is working: the most dangerous way for
	// a gate to be broken. They are refused at load rather than at the command
	// they silently fail to catch.
	cases := []struct {
		name string
		toml string
		want error
	}{
		{
			name: "duplicate rule id",
			toml: "[[rule]]\nid=\"a\"\naction=\"deny\"\n[[rule.clause]]\nfact=\"pipe\"\n" +
				"[[rule]]\nid=\"a\"\naction=\"ask\"\n[[rule.clause]]\nfact=\"glob\"\n",
			want: rules.ErrDuplicateRule,
		},
		{
			name: "unknown action",
			toml: "[[rule]]\nid=\"a\"\naction=\"wibble\"\n[[rule.clause]]\nfact=\"pipe\"\n",
			want: rules.ErrUnknownAction,
		},
		{
			name: "no clauses",
			toml: "[[rule]]\nid=\"a\"\naction=\"deny\"\n",
			want: rules.ErrNoClauses,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			mustFail(t, map[string]string{"x.toml": c.toml}, c.want)
		})
	}
}

// COVERS: FR-4.5 | negative
func TestAClauseThatCannotBeTrustedIsRefused(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		toml string
		want error
	}{
		{
			name: "clause says nothing at all",
			toml: "[[rule]]\nid=\"a\"\naction=\"deny\"\n[[rule.clause]]\n",
			want: rules.ErrClauseEmpty,
		},
		{
			name: "clause names an undeclared group",
			toml: "[[rule]]\nid=\"a\"\naction=\"deny\"\n[[rule.clause]]\nindex=\"0\"\ngroup=\"nope\"\n",
			want: rules.ErrUnknownGroup,
		},
		{
			name: "empty group",
			toml: "[group.tools]\nmembers = []\n",
			want: rules.ErrEmptyGroup,
		},
		{
			name: "tag rule tagging nothing",
			toml: "[[rule]]\nid=\"a\"\naction=\"tag\"\n[[rule.clause]]\nfact=\"pipe\"\n",
			want: rules.ErrTagMissing,
		},
		{
			name: "pattern that will not compile",
			toml: "[[rule]]\nid=\"a\"\naction=\"deny\"\n[[rule.clause]]\nindex=\"0\"\npattern=\"[\"\n",
			want: rules.ErrPattern,
		},
		{
			name: "two tests in one clause",
			toml: "[[rule]]\nid=\"a\"\naction=\"deny\"\n" +
				"[[rule.clause]]\nindex=\"0\"\nvalue=\"rm\"\npartial=\"r\"\n",
			want: rules.ErrManyForm,
		},
		{
			name: "unknown reading",
			toml: "[[rule]]\nid=\"a\"\naction=\"deny\"\n" +
				"[[rule.clause]]\nindex=\"0\"\nvalue=\"rm\"\nreading=\"wibble\"\n",
			want: rules.ErrReading,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			mustFail(t, map[string]string{"x.toml": c.toml}, c.want)
		})
	}
}

// COVERS: FR-7.11 | positive
func TestATestWithNoIndexIsACompleteClause(t *testing.T) {
	t.Parallel()

	// An absent index means any position, so this asks whether some word of
	// the command is `rm`. The index narrows a clause; it is not what makes
	// one, and refusing this would have made "anywhere in the command"
	// inexpressible.
	set, err := rules.Load([]string{inFiles(t, map[string]string{
		"x.toml": "[[rule]]\nid=\"a\"\naction=\"deny\"\n[[rule.clause]]\nvalue=\"rm\"\n",
	})})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}

	if got := set.Rules[0].Clause[0].Index; got != "" {
		t.Errorf("Index = %q, want it left empty to mean any position", got)
	}
}

// COVERS: FR-4.4 | regression
func TestThisRepositorysOwnRuleFilesLoad(t *testing.T) {
	t.Parallel()

	// The drafts in rules/ are the only rule set anybody has read, and a draft
	// that does not load is worse than no draft: it is a policy nobody can
	// run, reviewed as though it were one that could.
	set, err := rules.Load([]string{filepath.Join("..", "..", "rules")})
	if err != nil {
		t.Fatalf("the repository's own rule files do not load: %v", err)
	}

	if len(set.Rules) == 0 {
		t.Error("the repository's rule files loaded no rules")
	}
	if len(set.Shell.Allow) == 0 {
		t.Error("the repository's rule files declare no permitted shell")
	}
}

// COVERS: FR-4.29 | positive
func TestALeadingTildeInAGroupMemberBecomesTheHomeDirectory(t *testing.T) {
	t.Parallel()

	// A shipped rule set must not carry one machine's absolute paths, and a
	// protected-path group that silently lost its home prefix would widen to
	// every user on the host. So the resolution happens at load and a member
	// keeping its tilde would be a member matching nothing.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory on this host: %v", err)
	}

	dir := inFiles(t, map[string]string{
		"00-groups.toml": `
[group.paths]
match = "partial"
members = ["~/bin/", "/usr/bin/", "~name/not-a-home"]
`,
	})

	set, err := rules.Load([]string{dir})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := set.Groups["paths"].Members
	want := []string{filepath.Join(home, "bin"), "/usr/bin/", "~name/not-a-home"}

	if len(got) != len(want) {
		t.Fatalf("members = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("member %d = %q, want %q", i, got[i], want[i])
		}
	}
}
