package command_test

import (
	"testing"

	"github.com/scriptedworld/qwark/internal/command"
	"github.com/scriptedworld/qwark/internal/shell"
)

// simplesOf parses a command line and lifts every simple command out of it.
func simplesOf(t *testing.T, src string) []command.Simple {
	t.Helper()

	parsed, err := shell.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q) = %v", src, err)
	}
	return command.Simples(parsed)
}

// onlySimple insists the source held exactly one simple command and returns it.
func onlySimple(t *testing.T, src string) command.Simple {
	t.Helper()

	found := simplesOf(t, src)
	if len(found) != 1 {
		t.Fatalf("Simples(%q) found %d commands, want 1", src, len(found))
	}
	return found[0]
}

// outerSimple returns the outermost simple command, which the walk reaches
// first. A command substitution holds a command of its own, so `rm $(pwd)`
// legitimately contains two and only the first is the one being run.
func outerSimple(t *testing.T, src string) command.Simple {
	t.Helper()

	found := simplesOf(t, src)
	if len(found) == 0 {
		t.Fatalf("Simples(%q) found no commands", src)
	}
	return found[0]
}

// COVERS: FR-5.1 | positive
func TestWordsCarryTheirOrdinalAndTheirText(t *testing.T) {
	t.Parallel()

	simple := onlySimple(t, `rm -rf "some dir"`)

	want := []struct {
		ordinal int
		text    string
		value   string
	}{
		{ordinal: 0, text: `rm`, value: `rm`},
		{ordinal: 1, text: `-rf`, value: `-rf`},
		{ordinal: 2, text: `"some dir"`, value: `some dir`},
	}

	if len(simple.Words) != len(want) {
		t.Fatalf("got %d words, want %d", len(simple.Words), len(want))
	}
	for i, w := range want {
		got := simple.Words[i]
		if got.Ordinal != w.ordinal || got.Text != w.text || got.Value != w.value {
			t.Errorf("word %d = {%d %q %q}, want {%d %q %q}",
				i, got.Ordinal, got.Text, got.Value, w.ordinal, w.text, w.value)
		}
	}
}

// COVERS: FR-5.1, FR-5.2 | positive
func TestACommandReportsItsNameAndItsLastOrdinal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		src      string
		wantName string
		wantLast int
	}{
		{name: "with arguments", src: `rm -rf x`, wantName: "rm", wantLast: 2},
		{name: "bare", src: `pwd`, wantName: "pwd", wantLast: 0},
		{name: "path", src: `/bin/ls -l`, wantName: "/bin/ls", wantLast: 1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			simple := onlySimple(t, c.src)
			if got := simple.Name(); got != c.wantName {
				t.Errorf("Name() = %q, want %q", got, c.wantName)
			}
			if got := simple.Last(); got != c.wantLast {
				t.Errorf("Last() = %d, want %d", got, c.wantLast)
			}
		})
	}
}

// COVERS: FR-5.7 | negative
func TestAWordHoldingASubstitutionIsUndetermined(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		src  string
	}{
		{name: "bare parameter", src: `rm $HOME`},
		{name: "braced parameter", src: `rm ${HOME}`},
		{name: "parameter in a path", src: `rm $HOME/x`},
		{name: "command substitution", src: `rm $(pwd)`},
		{name: "backquoted", src: "rm `pwd`"},
		{name: "arithmetic", src: `rm $((1+1))`},
		{name: "inside double quotes", src: `rm "$HOME/x"`},
		{name: "ansi c quoting", src: `rm $'\x41'`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			simple := outerSimple(t, c.src)
			word, ok := simple.At(1)
			if !ok {
				t.Fatalf("no argument at ordinal 1 in %q", c.src)
			}
			if word.Determined {
				t.Errorf("%q reported as determined with value %q; nothing is expanded, "+
					"so it must be reported as unknown rather than guessed", c.src, word.Value)
			}
			if word.Value != "" {
				t.Errorf("Value = %q, want empty for an undetermined word", word.Value)
			}
		})
	}
}

// COVERS: FR-5.7 | positive
func TestQuotingIsResolvedWithoutExpandingAnything(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		src  string
		want string
	}{
		{name: "unquoted", src: `rm plain`, want: `plain`},
		{name: "single quoted", src: `rm 'a b'`, want: `a b`},
		{name: "double quoted", src: `rm "a b"`, want: `a b`},
		{name: "single quoted dollar", src: `rm '$HOME'`, want: `$HOME`},
		{name: "adjacent quoting", src: `rm a"b"'c'`, want: `abc`},
		{name: "glob is fixed text", src: `rm *.go`, want: `*.go`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			word, ok := onlySimple(t, c.src).At(1)
			if !ok {
				t.Fatalf("no argument at ordinal 1 in %q", c.src)
			}
			if !word.Determined {
				t.Fatalf("%q reported as undetermined; its text fixes it", c.src)
			}
			if word.Value != c.want {
				t.Errorf("Value = %q, want %q", word.Value, c.want)
			}
		})
	}
}

// COVERS: FR-5.9 | positive
func TestEscapesResolveTheWayTheShellResolvesThem(t *testing.T) {
	t.Parallel()

	// Every expectation here was read off bash on 2026-08-19 rather than
	// recalled: `bash -c "printf '[%s]' <word>"`. The quoted rule is not the
	// unquoted rule, and only measuring says which is which.
	cases := []struct {
		name string
		word string
		want string
	}{
		{name: "escaped space", word: `a\ b`, want: `a b`},
		{name: "escaped letter", word: `a\qb`, want: `aqb`},
		{name: "escaped dollar", word: `a\$b`, want: `a$b`},
		{name: "quoted escaped dollar", word: `"a\$b"`, want: `a$b`},
		{name: "quoted escaped letter", word: `"a\qb"`, want: `a\qb`},
		{name: "quoted escaped backslash", word: `"a\\b"`, want: `a\b`},
		{name: "single quoted backslash", word: `'a\qb'`, want: `a\qb`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			word, ok := onlySimple(t, "rm "+c.word).At(1)
			if !ok {
				t.Fatalf("no argument at ordinal 1 in %q", c.word)
			}
			if word.Value != c.want {
				t.Errorf("Value of %s = %q, want %q (what bash passes)",
					c.word, word.Value, c.want)
			}
		})
	}
}

// COVERS: FR-5.9 | regression
func TestAnEscapeInsideAPathDoesNotHideIt(t *testing.T) {
	t.Parallel()

	// The shell reaches .claude either way. A rule protecting that directory
	// is defeated by the second spelling unless the escape is resolved first.
	plain, _ := onlySimple(t, `rm /home/ancient/.claude/settings.json`).At(1)
	hidden, _ := onlySimple(t, `rm /home/ancient/.cl\aude/settings.json`).At(1)

	if hidden.Value != plain.Value {
		t.Errorf("escaped path resolved to %q, want %q", hidden.Value, plain.Value)
	}
}

// COVERS: FR-5.10 | positive
func TestAnEscapedCommandNameIsDistinguishable(t *testing.T) {
	t.Parallel()

	// `ls` runs whatever ls is aliased to; `\ls` runs the binary. They name
	// the same file, so the name alone cannot tell them apart.
	aliased := onlySimple(t, `ls -la`).Words[0]
	escaped := onlySimple(t, `\ls -la`).Words[0]

	if aliased.Value != escaped.Value {
		t.Errorf("names differ: %q and %q, want both to resolve to the same file",
			aliased.Value, escaped.Value)
	}
	if aliased.Escaped {
		t.Error("`ls` reported as escaped; nothing suppressed its alias")
	}
	if !escaped.Escaped {
		t.Error("`\\ls` not reported as escaped; that escape is the only thing " +
			"distinguishing it from the alias")
	}
}

// COVERS: FR-5.7 | edge
func TestACommandNamedByASubstitutionHasNoName(t *testing.T) {
	t.Parallel()

	simple := onlySimple(t, `$tool --force`)

	if got := simple.Name(); got != "" {
		t.Errorf("Name() = %q, want empty; nothing here can say what will run", got)
	}
}

// COVERS: FR-5.8 | positive
func TestEverySimpleCommandInAStructureIsFound(t *testing.T) {
	t.Parallel()

	// A pipeline of two, a logical concatenation, and one inside a loop body.
	const src = `cat a | grep b && for f in x; do rm $f; done`

	found := simplesOf(t, src)

	var names []string
	for _, simple := range found {
		names = append(names, simple.Name())
	}

	want := []string{"cat", "grep", "rm"}
	if len(names) != len(want) {
		t.Fatalf("found %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("command %d = %q, want %q", i, names[i], want[i])
		}
	}
}

// COVERS: FR-5.8 | negative
func TestABareAssignmentIsNotACommand(t *testing.T) {
	t.Parallel()

	// `X=1` names nothing to run, so it has no ordinal 0 and no position a
	// rule could address. `X=1 rm y` does name one, and rm is still at 0.
	if found := simplesOf(t, `X=1`); len(found) != 0 {
		t.Errorf("Simples(X=1) found %d commands, want none", len(found))
	}

	simple := onlySimple(t, `X=1 rm y`)
	if got := simple.Name(); got != "rm" {
		t.Errorf("Name() = %q, want %q; the assignment is a prefix, not the command", got, "rm")
	}
}

// COVERS: FR-5.5 | negative
func TestAtRefusesAnOrdinalTheCommandLacks(t *testing.T) {
	t.Parallel()

	simple := onlySimple(t, `pwd`)

	if _, ok := simple.At(1); ok {
		t.Error("At(1) found an argument in a command that has none")
	}
	if _, ok := simple.At(0); !ok {
		t.Error("At(0) did not find the command name")
	}
}
