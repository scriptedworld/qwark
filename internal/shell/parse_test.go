package shell_test

import (
	"errors"
	"testing"

	"github.com/scriptedworld/qwark/internal/shell"
	"mvdan.cc/sh/v3/syntax"
)

// COVERS: FR-1.1 | positive
func TestParseAcceptsBashBeyondPOSIX(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		src  string
	}{
		{name: "double bracket test", src: `[[ -f x ]] && echo y`},
		{name: "process substitution", src: `diff <(sort a) <(sort b)`},
		{name: "array assignment", src: `arr=(one two three)`},
		{name: "brace group", src: `{ echo a; echo b; }`},
		{name: "here string", src: `grep x <<< "$body"`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			if _, err := shell.Parse(c.src); err != nil {
				t.Fatalf("Parse(%q) = %v, want no error", c.src, err)
			}
		})
	}
}

// COVERS: FR-1.2 | negative
func TestParseReportsWhereItStopped(t *testing.T) {
	t.Parallel()

	const src = "echo a )"

	_, err := shell.Parse(src)
	if err == nil {
		t.Fatalf("Parse(%q) = nil, want an error", src)
	}

	var perr *shell.ParseError
	if !errors.As(err, &perr) {
		t.Fatalf("Parse(%q) returned %T, want *shell.ParseError", src, err)
	}
	if perr.Line == 0 || perr.Col == 0 {
		t.Errorf("position = %d:%d, want both non-zero", perr.Line, perr.Col)
	}
	if perr.Src != src {
		t.Errorf("Src = %q, want %q", perr.Src, src)
	}
	if perr.Error() == "" {
		t.Error("Error() is empty; a verdict nobody can read is not a verdict")
	}
}

// COVERS: FR-1.2 | property
func TestParseErrorUnwrapsToTheParser(t *testing.T) {
	t.Parallel()

	_, err := shell.Parse("echo a )")

	perr, ok := errors.AsType[*shell.ParseError](err)
	if !ok {
		t.Fatalf("got %T, want *shell.ParseError", err)
	}

	if _, ok := errors.AsType[syntax.ParseError](perr); !ok {
		t.Error("errors.As could not reach the parser's own error through Unwrap")
	}
}

// COVERS: FR-1.3 | edge
func TestParseDistinguishesTruncationFromMalformation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		src            string
		wantIncomplete bool
	}{
		{name: "unclosed quote", src: `echo "abc`, wantIncomplete: true},
		{name: "dangling pipe", src: `cat a |`, wantIncomplete: true},
		{name: "unterminated loop", src: `for f in a; do echo`, wantIncomplete: true},
		{name: "stray paren", src: `echo a )`, wantIncomplete: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			_, err := shell.Parse(c.src)

			var perr *shell.ParseError
			if !errors.As(err, &perr) {
				t.Fatalf("Parse(%q) returned %v, want a *shell.ParseError", c.src, err)
			}
			if perr.Incomplete != c.wantIncomplete {
				t.Errorf("Incomplete = %v, want %v", perr.Incomplete, c.wantIncomplete)
			}
		})
	}
}

// COVERS: FR-1.4 | positive
func TestTextReturnsSourceAsWritten(t *testing.T) {
	t.Parallel()

	const src = `grep   "a  b"   'c'`

	parsed, err := shell.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q) = %v", src, err)
	}

	var got []string
	syntax.Walk(parsed.File, func(node syntax.Node) bool {
		if word, ok := node.(*syntax.Word); ok {
			got = append(got, parsed.Text(word))
		}
		return true
	})

	want := []string{`grep`, `"a  b"`, `'c'`}
	if len(got) != len(want) {
		t.Fatalf("got %d words %q, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("word %d = %q, want %q (quoting must survive)", i, got[i], want[i])
		}
	}
}

// COVERS: FR-1.4 | edge
func TestTextRefusesAnOutOfRangeNode(t *testing.T) {
	t.Parallel()

	parsed, err := shell.Parse("echo a")
	if err != nil {
		t.Fatalf("Parse = %v", err)
	}

	// A node belonging to a different source has offsets that do not address
	// this one. Slicing on them would panic or return another command's text;
	// both are worse than an empty answer.
	other, err := shell.Parse("a much longer command line entirely")
	if err != nil {
		t.Fatalf("Parse = %v", err)
	}

	if got := parsed.Text(other.File); got != "" {
		t.Errorf("Text(foreign node) = %q, want empty", got)
	}
}
