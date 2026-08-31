package rules_test

import (
	"testing"

	"github.com/scriptedworld/qwark/internal/rules"
)

// clauseSet wraps one clause in a deny rule, so a test can ask whether that
// clause held by asking whether the command was refused.
func clauseSet(clause string) map[string]string {
	return map[string]string{"00.toml": declarations + `
[group.protected]
match = "partial"
members = ["/etc/", "/home/user/.claude/"]

[group.subcommands]
members = ["log", "status"]

[[rule]]
id = "allow-everything-declared"
action = "allow"
reason = "So that a refusal below means the clause held."
  [[rule.clause]]
  index = "0"
  pattern = "rm|git"

[[rule]]
id = "under-test"
action = "deny"
reason = "The clause under test held."
` + clause}
}

// held reports whether the clause under test matched, by whether the command
// was refused.
func held(t *testing.T, clause, src string) bool {
	t.Helper()

	return judgeWith(t, clauseSet(clause), src).Denied()
}

// COVERS: FR-2.8 | positive
func TestAClauseNamingNodeTypes(t *testing.T) {
	t.Parallel()

	const clause = "  [[rule.clause]]\n  nodes = [\"ParamExp\", \"CmdSubst\"]\n"

	if !held(t, clause, `rm $HOME`) {
		t.Error("a clause naming ParamExp did not match a command holding one")
	}
	if held(t, clause, `rm plain`) {
		t.Error("a clause naming ParamExp matched a command holding none")
	}
}

// COVERS: FR-2.4 | positive
func TestAClauseNamingOperators(t *testing.T) {
	t.Parallel()

	// `rm a > b` is one command with a redirection; the operator is what
	// separates it from an append.
	const clause = "  [[rule.clause]]\n  ops = [\">\", \">|\"]\n"

	if !held(t, clause, `rm a > b`) {
		t.Error("a clause naming `>` did not match a truncating redirection")
	}
	if held(t, clause, `rm a >> b`) {
		t.Error("a clause naming `>` matched an append")
	}
}

// COVERS: FR-2.8 | positive
func TestAClauseNamingStatementFlags(t *testing.T) {
	t.Parallel()

	const clause = "  [[rule.clause]]\n  flags = [\"Background\"]\n"

	if !held(t, clause, `rm x &`) {
		t.Error("a clause naming Background did not match a backgrounded command")
	}
	if held(t, clause, `rm x`) {
		t.Error("a clause naming Background matched an ordinary command")
	}
}

// COVERS: FR-2.6 | positive
func TestAClauseNamingAFact(t *testing.T) {
	t.Parallel()

	// glob is one of the two properties no node, flag or operator can express:
	// quoting decides it.
	const clause = "  [[rule.clause]]\n  fact = \"glob\"\n"

	if !held(t, clause, `rm *.go`) {
		t.Error("a clause naming glob did not match an unquoted wildcard")
	}
	if held(t, clause, `rm "*.go"`) {
		t.Error("a clause naming glob matched a quoted wildcard, which is an argument")
	}
}

// COVERS: FR-6.10 | positive
func TestAClauseSelectingPaths(t *testing.T) {
	t.Parallel()

	// rm's operands are declared as paths; git's are text. A rule about where
	// a command reaches must read one and not the other.
	const clause = "  [[rule.clause]]\n  kind = \"path\"\n  group = \"protected\"\n"

	if !held(t, clause, `rm /etc/passwd`) {
		t.Error("a path clause did not match a protected path")
	}
	if held(t, clause, `rm /home/user/work/x`) {
		t.Error("a path clause matched a path outside the group")
	}
	if held(t, clause, `git /etc/passwd`) {
		t.Error("a path clause matched a command whose operands are declared as text")
	}
}

// COVERS: FR-7.2 | positive
func TestAGroupComparesTheWayItDeclares(t *testing.T) {
	t.Parallel()

	// A group of paths must compare partially: comparing whole would match the
	// directory and miss everything in it, which is every case that matters.
	const partial = "  [[rule.clause]]\n  kind = \"path\"\n  group = \"protected\"\n"
	if !held(t, partial, `rm /etc/passwd`) {
		t.Error("a partial group did not match a path inside one of its members")
	}

	// A group of subcommands compares whole, so `logs` is not `log`.
	const whole = "  [[rule.clause]]\n  index = \"1\"\n  group = \"subcommands\"\n"
	if !held(t, whole, `git log`) {
		t.Error("a value group did not match an exact member")
	}
	if held(t, whole, `git logs`) {
		t.Error("a value group matched a value that merely starts like a member")
	}
}

// COVERS: FR-7.11, FR-5.12 | positive
func TestAClauseWithNoIndexAsksAboutTheArguments(t *testing.T) {
	t.Parallel()

	const clause = "  [[rule.clause]]\n  value = \"rm\"\n"

	// Some argument is `rm`.
	if !held(t, clause, `git rm`) {
		t.Error("a clause with no index did not look at the arguments")
	}
	// The command being `rm` is not an argument being `rm`.
	if held(t, clause, `rm x`) {
		t.Error("a clause with no index matched the command name at ordinal 0")
	}
}

// COVERS: FR-7.8 | positive
func TestAClauseChoosesWhichReadingItTests(t *testing.T) {
	t.Parallel()

	// The shell reaches .claude either way. Reading what was written sees
	// `.cl\aude`; reading the interpreted value sees through the escape.
	const interpreted = "  [[rule.clause]]\n  partial = \".claude\"\n"
	const written = "  [[rule.clause]]\n  partial = \".claude\"\n  reading = \"written\"\n"

	if !held(t, interpreted, `rm /home/user/.cl\aude/x`) {
		t.Error("the interpreted reading did not see through an escape")
	}
	if held(t, written, `rm /home/user/.cl\aude/x`) {
		t.Error("the written reading saw through an escape, so this proves nothing")
	}
}

// COVERS: FR-4.27 | positive
func TestAnInvertedClauseHoldsWhenNothingMatches(t *testing.T) {
	t.Parallel()

	// "forbidden unless" written as one rule: the exception lives inside the
	// rule it modifies, where a reader of that rule sees it.
	const clause = "  [[rule.clause]]\n  option = \"force\"\n  absent = true\n"

	if !held(t, clause, `rm x`) {
		t.Error("an inverted clause did not hold when the option was absent")
	}
	if held(t, clause, `rm -f x`) {
		t.Error("an inverted clause held when the option was present")
	}
}

// COVERS: FR-8.3 | positive
func TestAClauseNamingATag(t *testing.T) {
	t.Parallel()

	set, err := rules.Load([]string{inFiles(t, clauseSet(
		"  [[rule.clause]]\n  tag = \"post-rebase\"\n"))})
	if err != nil {
		t.Fatalf("Load = %v", err)
	}

	parsed := parseFor(t, `rm x`)

	live := set.Evaluate(parsed, rules.Context{Tags: map[string]bool{"post-rebase": true}})
	if !live.Denied() {
		t.Error("a tag clause did not hold while the tag was live")
	}

	absent := set.Evaluate(parsed, rules.Context{})
	if absent.Denied() {
		t.Error("a tag clause held with no tags live at all")
	}
}
