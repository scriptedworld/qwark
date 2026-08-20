package shell_test

import (
	"testing"

	"github.com/scriptedworld/qwark/internal/shell"
)

// factsOf parses and gathers, failing the test rather than returning an error,
// because every command in these tables is meant to be valid Bash.
func factsOf(t *testing.T, src string) *shell.Facts {
	t.Helper()

	parsed, err := shell.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q) = %v", src, err)
	}
	return parsed.Facts()
}

// COVERS: FR-2.1 | positive
func TestOneGatherAnswersForEveryFact(t *testing.T) {
	t.Parallel()

	const src = `a | b && c > d $(e) &`

	facts := factsOf(t, src)

	want := []shell.Fact{
		shell.FactPipe,
		shell.FactLogical,
		shell.FactRedirect,
		shell.FactSubstitutionCommand,
		shell.FactBackground,
	}
	for _, fact := range want {
		if !facts.Has(fact) {
			t.Errorf("Has(%q) = false, want true from a single gather", fact)
		}
	}
	if len(facts.Names()) < len(want) {
		t.Errorf("Names() = %v, want at least %d distinct", facts.Names(), len(want))
	}
}

// COVERS: FR-2.2 | property
func TestAFactRecordsEveryLevelItSatisfies(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		src    string
		parent shell.Fact
		child  shell.Fact
	}{
		{
			name: "command substitution", src: `echo $(date)`,
			parent: shell.FactSubstitution, child: shell.FactSubstitutionCommand,
		},
		{
			name: "truncating redirect", src: `echo a > b`,
			parent: shell.FactRedirect, child: shell.FactRedirectTruncate,
		},
		{
			name: "extended glob", src: `ls !(a|b)`,
			parent: shell.FactGlob, child: shell.FactExtGlob,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			facts := factsOf(t, c.src)
			if !facts.Has(c.parent) {
				t.Errorf("Has(%q) = false; naming the family must match", c.parent)
			}
			if !facts.Has(c.child) {
				t.Errorf("Has(%q) = false; naming the member must match", c.child)
			}
		})
	}
}

// COVERS: FR-2.3 | positive
func TestRedirectionsAreDistinguishedByWhatTheyDo(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		src  string
		want shell.Fact
	}{
		{name: "truncate", src: `echo a > f`, want: shell.FactRedirectTruncate},
		{name: "clobber", src: `echo a >| f`, want: shell.FactRedirectTruncate},
		{name: "append", src: `echo a >> f`, want: shell.FactRedirectAppend},
		{name: "input", src: `cat < f`, want: shell.FactRedirectInput},
		{name: "duplicate", src: `echo a 2>&1`, want: shell.FactRedirectDup},
		{name: "heredoc", src: "cat <<E\nx\nE", want: shell.FactHeredoc},
		{name: "herestring", src: `cat <<< x`, want: shell.FactHeredoc},
		{name: "all streams", src: `echo a &> f`, want: shell.FactRedirectTruncate},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			facts := factsOf(t, c.src)
			if !facts.Has(c.want) {
				t.Errorf("Has(%q) = false for %q; got %v", c.want, c.src, facts.Names())
			}
			if !facts.Has(shell.FactRedirect) {
				t.Errorf("Has(%q) = false; every redirection is one", shell.FactRedirect)
			}
		})
	}
}

// COVERS: FR-2.4 | negative
func TestAPipeIsNotALogicalConcatenation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		src     string
		want    shell.Fact
		notWant shell.Fact
	}{
		{name: "pipe", src: `a | b`, want: shell.FactPipe, notWant: shell.FactLogical},
		{name: "pipe both", src: `a |& b`, want: shell.FactPipe, notWant: shell.FactLogical},
		{name: "and", src: `a && b`, want: shell.FactLogical, notWant: shell.FactPipe},
		{name: "or", src: `a || b`, want: shell.FactLogical, notWant: shell.FactPipe},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			facts := factsOf(t, c.src)
			if !facts.Has(c.want) {
				t.Errorf("Has(%q) = false for %q", c.want, c.src)
			}
			if facts.Has(c.notWant) {
				t.Errorf("Has(%q) = true for %q; the operators differ", c.notWant, c.src)
			}
		})
	}
}

// COVERS: FR-2.5 | positive
func TestTheFourSubstitutionsAreNamedSeparately(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		src  string
		want shell.Fact
	}{
		{name: "command", src: `echo $(date)`, want: shell.FactSubstitutionCommand},
		{name: "backquoted", src: "echo `date`", want: shell.FactSubstitutionCommand},
		{name: "process", src: `diff <(a) <(b)`, want: shell.FactSubstitutionProcess},
		{name: "parameter braced", src: `echo ${HOME}`, want: shell.FactSubstitutionParameter},
		{name: "parameter bare", src: `echo $HOME`, want: shell.FactSubstitutionParameter},
		{name: "arithmetic", src: `echo $((1 + 1))`, want: shell.FactSubstitutionArithmetic},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			facts := factsOf(t, c.src)
			if !facts.Has(c.want) {
				t.Errorf("Has(%q) = false for %q; got %v", c.want, c.src, facts.Names())
			}
		})
	}
}

// COVERS: FR-2.6 | edge
func TestAWildcardCountsOnlyWhereTheShellWouldExpandOne(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		src  string
		want bool
	}{
		{name: "bare star", src: `ls *`, want: true},
		{name: "bare question", src: `ls a?c`, want: true},
		{name: "bracket class", src: `ls [ab]c`, want: true},
		{name: "double quoted", src: `grep "*" f`, want: false},
		{name: "single quoted", src: `grep '*' f`, want: false},
		{name: "no meta", src: `ls a`, want: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			if got := factsOf(t, c.src).Has(shell.FactGlob); got != c.want {
				t.Errorf("Has(glob) = %v for %q, want %v", got, c.src, c.want)
			}
		})
	}
}

// COVERS: FR-2.7 | positive
func TestEveryFindingCarriesItsPositionAndText(t *testing.T) {
	t.Parallel()

	const src = `echo a > out.log`

	facts := factsOf(t, src)

	found, ok := facts.First(shell.FactRedirectTruncate)
	if !ok {
		t.Fatalf("First(redirect.truncate) not found in %q", src)
	}
	if found.Line == 0 || found.Col == 0 {
		t.Errorf("position = %d:%d, want both non-zero", found.Line, found.Col)
	}
	if found.Text == "" {
		t.Error("Text is empty; a denial that cannot quote what caused it cannot be checked")
	}
	for _, finding := range facts.All() {
		if finding.Fact == "" {
			t.Error("a finding carries no fact name")
		}
	}
}

// COVERS: FR-2.7 | negative
func TestAnAbsentFactHasNoFinding(t *testing.T) {
	t.Parallel()

	facts := factsOf(t, `git status`)

	if got := facts.Count(shell.FactPipe); got != 0 {
		t.Errorf("Count(pipe) = %d for a simple command, want 0", got)
	}
	if _, ok := facts.First(shell.FactPipe); ok {
		t.Error("First(pipe) reported a finding for a command with no pipe")
	}
	if names := facts.Names(); len(names) != 0 {
		t.Errorf("Names() = %v, want none for a bare invocation", names)
	}
}

// COVERS: FR-2.8 | positive
func TestEveryCommandFormCarriesAFact(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		src  string
		want shell.Fact
	}{
		{name: "time prefix", src: `time rm x`, want: shell.FactTime},
		{name: "coprocess", src: `coproc foo`, want: shell.FactCoproc},
		{name: "export", src: `export PATH=/tmp/evil`, want: shell.FactDeclaration},
		{name: "declare", src: `declare -x A=b`, want: shell.FactDeclaration},
		{name: "readonly", src: `readonly PATH=/tmp`, want: shell.FactDeclaration},
		{name: "arithmetic command", src: `((x=1))`, want: shell.FactArithmetic},
		{name: "let", src: `let x=1`, want: shell.FactArithmetic},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			facts := factsOf(t, c.src)
			if !facts.Has(c.want) {
				t.Errorf("Has(%q) = false for %q; got %v", c.want, c.src, facts.Names())
			}
		})
	}
}

// COVERS: FR-2.9 | regression
func TestAStatementWithNoCommandNameIsStillAddressable(t *testing.T) {
	t.Parallel()

	// These carry no usable command name: `time rm x` puts `rm` at ordinal
	// zero rather than `time`, and the other two put nothing there. A rule can
	// only reach them by fact, and each of them reported no facts at all until
	// FR-2.8 was met -- which left them addressable by no rule that could be
	// written, in a gate whose default is to deny what it cannot account for.
	for _, src := range []string{`time rm x`, `((x=1))`, `let x=1`, `coproc foo`} {
		t.Run(src, func(t *testing.T) {
			t.Parallel()

			if names := factsOf(t, src).Names(); len(names) == 0 {
				t.Errorf("%q established no facts, so no rule can name it", src)
			}
		})
	}
}

// COVERS: FR-2.1 | property
func TestCountReportsEveryOccurrence(t *testing.T) {
	t.Parallel()

	const src = `a > x 2> y >> z`
	const wantRedirects = 3

	if got := factsOf(t, src).Count(shell.FactRedirect); got != wantRedirects {
		t.Errorf("Count(redirect) = %d for %q, want %d", got, src, wantRedirects)
	}
}
