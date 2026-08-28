package rules_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scriptedworld/qwark/internal/audit"
	"github.com/scriptedworld/qwark/internal/rules"
	"github.com/scriptedworld/qwark/internal/shell"
)

// COVERS: FR-10.9 | property
func TestTheRegistrationRefusesWhenQwarkDies(t *testing.T) {
	t.Parallel()

	// The `|| exit 2` is not belt and braces. **Exit 0 with no JSON is no
	// decision and the command proceeds, and any non-zero exit other than 2
	// is a non_blocking_error and the command also proceeds.** Only exit
	// 2 blocks. A registration without the guard therefore lets the command
	// through whenever qwark segfaults, is killed, or exits 1, and the shipped
	// fragment is where somebody copies that from.
	body, err := os.ReadFile(filepath.Join("..", "..", "install", "settings-fragment.json"))
	if err != nil {
		t.Fatalf("reading the shipped registration: %v", err)
	}

	var fragment struct {
		Hooks struct {
			PreToolUse []struct {
				Matcher string `json:"matcher"`
				Hooks   []struct {
					Type    string `json:"type"`
					Command string `json:"command"`
				} `json:"hooks"`
			} `json:"PreToolUse"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(body, &fragment); err != nil {
		t.Fatalf("the shipped registration is not valid JSON: %v", err)
	}

	if len(fragment.Hooks.PreToolUse) == 0 {
		t.Fatal("the shipped registration declares no PreToolUse hook")
	}
	group := fragment.Hooks.PreToolUse[0]

	if group.Matcher != "Bash" {
		t.Errorf("matcher = %q, want Bash: qwark's first mode gates that tool",
			group.Matcher)
	}
	if len(group.Hooks) == 0 {
		t.Fatal("the matcher group holds no hook")
	}
	if command := group.Hooks[0].Command; !strings.Contains(command, "|| exit 2") {
		t.Errorf("command = %q, want the `|| exit 2` guard: without it a qwark "+
			"that dies lets the command run", command)
	}
}

// COVERS: FR-10.10 | property
func TestTheRegistrationCarriesTheDenyListQwarkCannotEnforce(t *testing.T) {
	t.Parallel()

	// qwark gates Bash and nothing else. Write and Edit reach the rule files,
	// the shell snapshot, .git/hooks and a task definition without passing
	// through it, so a path protected only by a rule in 20-paths.toml is
	// protected against a shell and against nothing else.
	//
	// This was documented in DESIGN-NOTES and shipped as prose: the fragment
	// explained at length that a permissions.deny twin was needed and carried
	// none. A control that exists only in the paragraph describing it is the
	// failure this test exists to catch.
	body, err := os.ReadFile(filepath.Join("..", "..", "install", "settings-fragment.json"))
	if err != nil {
		t.Fatalf("reading the shipped registration: %v", err)
	}

	var fragment struct {
		Permissions struct {
			Deny []string `json:"deny"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(body, &fragment); err != nil {
		t.Fatalf("the shipped registration is not valid JSON: %v", err)
	}

	if len(fragment.Permissions.Deny) == 0 {
		t.Fatal("the shipped registration carries no permissions.deny list, so " +
			"every path rule is enforced against Bash and against nothing else")
	}

	// One representative of each class the rule files protect. Naming them
	// individually means a class dropped from the list fails here rather than
	// being noticed by whoever is attacked through it.
	classes := map[string]string{
		"qwark's own rules":    "/etc/qwark/",
		"the shell snapshot":   "shell-snapshots",
		"a shell startup file": ".zshrc",
		"git's hooks":          ".git/hooks",
		"a task definition":    "justfile",
	}

	joined := strings.Join(fragment.Permissions.Deny, "\n")
	for class, want := range classes {
		if !strings.Contains(joined, want) {
			t.Errorf("permissions.deny covers no path for %s (looked for %q): "+
				"rules/20-paths.toml protects it and Write reaches it anyway",
				class, want)
		}
	}
}

// COVERS: FR-4.21 | positive
func TestTheShippedRulesDenyWrappersByName(t *testing.T) {
	t.Parallel()

	// Wrappers are refused by an explicit rule rather than by being undeclared,
	// so that the refusal states why, records that they were considered rather
	// than forgotten, and survives someone later declaring one for a harmless
	// flag. An absence provides none of the three, and an absence is also
	// what this test would be checking if it merely asserted they do not run.
	set, err := rules.Load([]string{filepath.Join("..", "..", "rules")})
	if err != nil {
		t.Fatalf("the repository's rule files do not load: %v", err)
	}

	group, declared := set.Groups["wrapper"]
	if !declared {
		t.Fatal("no group names the command wrappers")
	}
	for _, want := range []string{"env", "xargs", "sudo", "exec", "command"} {
		if !contains(group.Members, want) {
			t.Errorf("the wrapper group does not name %q", want)
		}
	}

	if !deniedByName(set, "wrapper") {
		t.Error("no deny rule names the wrapper group, so wrappers are refused " +
			"only by being undeclared")
	}
}

// COVERS: FR-4.21 | positive
func TestTheShippedRulesDenyTaskRunnersByName(t *testing.T) {
	t.Parallel()

	// 2026-08-20: `I worry about letting them run bolt ... running
	// ANYTHING that isn't one of our standard jigs ... same with JUST or POE
	// etc ... if they can write a new file, then they can then get the agent to
	// approve anything.`
	//
	// Measured: `just checks` was refused by `no-executors`, which names the
	// threat. `bolt run` was refused by "(engine) deny by default", which
	// names nothing, and bolt is this project's own gate.
	//
	// The list is not what makes this safe: deny-by-default already refuses an
	// unnamed command. What the list buys is a refusal that explains itself, so
	// the same command is not retried in five spellings. That is FR-4.21, and
	// it is why this test asserts the NAME is present rather than asserting the
	// command does not run.
	set, err := rules.Load([]string{filepath.Join("..", "..", "rules")})
	if err != nil {
		t.Fatalf("the repository's rule files do not load: %v", err)
	}

	group, declared := set.Groups["executor"]
	if !declared {
		t.Fatal("no group names the task runners")
	}
	for _, want := range []string{"bolt", "just", "poe", "make", "task", "pre-commit"} {
		if !contains(group.Members, want) {
			t.Errorf("the executor group does not name %q, so its refusal will "+
				"say only that nothing permitted it", want)
		}
	}

	if !deniedByName(set, "executor") {
		t.Error("no deny rule names the executor group, so a task runner is " +
			"refused only by being undeclared")
	}
}

// COVERS: FR-10.11 | regression
func TestTheGuardCoversThePathsQwarkActuallyUses(t *testing.T) {
	t.Parallel()

	// The group named /etc/qwark/ and /var/lib/qwark/ long after the install
	// target moved to ~/.config/qwark/rules and the log to ~/.local/state.
	// Measured 2026-08-28: `cp` over the live 00-structure.toml and `rm` of
	// decisions.jsonl were both ALLOW, while `ls /etc/qwark/rules` was refused.
	// The guard was working perfectly against an address its subject had left.
	//
	// So this asserts the SUBJECT is covered, not that the rule is present.
	// A test naming the rule would have passed throughout, which is how the
	// defect survived a rule set that is otherwise heavily tested.
	set, err := rules.Load([]string{filepath.Join("..", "..", "rules")})
	if err != nil {
		t.Fatalf("the repository's rule files do not load: %v", err)
	}

	if _, declared := set.Groups["qwark-control"]; !declared {
		t.Fatal("no group names qwark's own control surfaces")
	}

	// Judged as commands rather than by matching the group's members here.
	// Partial matching compares fragments and the members carry trailing
	// slashes, so a helper written in this file is a second implementation of
	// the evaluator, free to be wrong in exactly the way it is being asked to
	// detect. The first draft of this test was, and passed the live paths as
	// bare directories that no member matched.
	//
	// Derived rather than written twice: moving the log moves the assertion.
	logPath := audit.DefaultPath()
	rulePath := filepath.Join(home(t), ".config", "qwark", "rules", "00-structure.toml")

	surfaces := map[string]string{
		"overwriting the live rule set": "cp /tmp/evil.toml " + rulePath,
		"deleting the decision log":     "rm " + logPath,
	}

	for name, command := range surfaces {
		parsed, err := shell.Parse(command)
		if err != nil {
			t.Fatalf("parsing %q: %v", command, err)
		}
		if !set.Evaluate(parsed, rules.Context{}).Denied() {
			t.Errorf("%s is permitted: %q. The subject may overwrite the rules "+
				"that gate it or delete the record of what it was refused.",
				name, command)
		}
	}
}

// COVERS: FR-10.11 | regression
func TestTheDenyTwinCoversThePathsQwarkActuallyUses(t *testing.T) {
	t.Parallel()

	// The other half of one control, and it drifted the same way for the same
	// reason: the twin named //etc/qwark/** and //var/lib/qwark/** while the
	// live set sat in ~/.config. A path held by the rule group and absent from
	// the twin is protected against a shell and against nothing else, since
	// Write and Edit never reach qwark at all.
	body, err := os.ReadFile(filepath.Join("..", "..", "install", "settings-fragment.json"))
	if err != nil {
		t.Fatalf("reading the shipped registration: %v", err)
	}

	var fragment struct {
		Permissions struct {
			Deny []string `json:"deny"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(body, &fragment); err != nil {
		t.Fatalf("the shipped registration is not valid JSON: %v", err)
	}

	joined := strings.Join(fragment.Permissions.Deny, "\n")
	for _, want := range []string{"~/.config/qwark/", "~/.local/state/qwark/"} {
		if !strings.Contains(joined, want) {
			t.Errorf("permissions.deny names no entry under %q, so Write and "+
				"Edit reach it whatever rules/20-paths.toml says", want)
		}
	}
}

// home is the directory the install path and the log path are both relative to.
func home(t *testing.T) string {
	t.Helper()
	dir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory to resolve the live paths against: %v", err)
	}
	return dir
}

func contains(members []string, want string) bool {
	for _, member := range members {
		if member == want {
			return true
		}
	}
	return false
}

// deniedByName reports whether some deny rule has a clause naming the group.
func deniedByName(set *rules.Set, group string) bool {
	for _, rule := range set.Rules {
		if rule.Action != rules.ActionDeny {
			continue
		}
		for _, clause := range rule.Clause {
			if clause.Group == group {
				return true
			}
		}
	}
	return false
}

// shippedVerdict judges a command against this repository's own rule files and
// returns the verdict with the rule ids that produced it.
func shippedVerdict(t *testing.T, src string) (rules.Outcome, []string) {
	t.Helper()

	set, err := rules.Load([]string{filepath.Join("..", "..", "rules")})
	if err != nil {
		t.Fatalf("the repository's rule files do not load: %v", err)
	}

	outcome := set.Evaluate(parseFor(t, src), rules.Context{})

	var fired []string
	for _, finding := range outcome.Findings {
		fired = append(fired, finding.Rule)
	}
	return outcome, fired
}

// COVERS: FR-4.3 | positive
func TestTheShippedRulesForbidTierOneAsOneProperty(t *testing.T) {
	t.Parallel()

	// The four are one property: that a command's effect is fixed by its own
	// text, and the shipped rules must actually say so, not merely be
	// described as saying so.
	//
	// A pipe and a logical concatenation are refused before the rules run,
	// because each of them IS more than one command and the engine refuses
	// that outright. Their reason names the construct, so the refusal is still
	// specific; the rules exist for the case where the engine's check is ever
	// relaxed.
	cases := []struct {
		name    string
		src     string
		mention string
	}{
		{name: "redirection", src: `cat a > b`, mention: "Redirections are not permitted"},
		{name: "substitution", src: `cat $HOME`, mention: "Substitutions are not permitted"},
		{name: "pipe", src: `cat a | grep b`, mention: "One command at a time"},
		{name: "logical", src: `cat a && cat b`, mention: "One command at a time"},
		// A glob is the fifth member of the property and was the one with no
		// rule. What `*` matches is decided by the directory at the moment it
		// runs, and it hands a path rule a word that denotes no path, so the
		// protected-path rules are silent on `rm *` entirely.
		{name: "glob", src: `cat *`, mention: "wildcard is not permitted"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			outcome, fired := shippedVerdict(t, c.src)
			if !outcome.Denied() {
				t.Fatalf("%q was not refused; rules that fired: %v", c.src, fired)
			}

			// Everything is refused by this rule set, so "denied" alone proves
			// nothing. The reason has to be about the construct.
			var found bool
			for _, reason := range outcome.Reasons() {
				if strings.Contains(reason, c.mention) {
					found = true
				}
			}
			if !found {
				t.Errorf("%q was refused, but for no reason mentioning %q; reasons: %v",
					c.src, c.mention, outcome.Reasons())
			}
		})
	}
}

// COVERS: FR-4.10 | positive
func TestTheShippedRulesRefuseAHeredocWriteInItsOwnRight(t *testing.T) {
	t.Parallel()

	// The here-document ban is subsumed by the redirection ban, and is stated
	// separately because its reason is separate: such content was never a diff
	// and leaves nothing to review. That separateness is only real if the
	// separate reason actually reaches the reader, otherwise it is a comment
	// in a file rather than something the gate says.
	outcome, fired := shippedVerdict(t, "cat > f.go <<EOF\npackage main\nEOF")

	if !outcome.Denied() {
		t.Fatalf("a here-document write was not refused; rules that fired: %v", fired)
	}

	if !contains(fired, "no-heredoc-write") {
		t.Errorf("no-heredoc-write did not fire; rules that fired: %v", fired)
	}
	// And the redirection rule fires too, which is the point of listing every
	// reason rather than the first.
	if !contains(fired, "no-redirection") {
		t.Errorf("no-redirection did not fire; rules that fired: %v", fired)
	}

	var reviewable bool
	for _, reason := range outcome.Reasons() {
		if strings.Contains(reason, "never a diff") {
			reviewable = true
		}
	}
	if !reviewable {
		t.Errorf("the separate reason never reached the reader; reasons: %v",
			outcome.Reasons())
	}
}
