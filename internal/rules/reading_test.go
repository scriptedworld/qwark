package rules_test

import (
	"errors"
	"testing"

	"github.com/scriptedworld/qwark/internal/command"
	"github.com/scriptedworld/qwark/internal/rules"
	"github.com/scriptedworld/qwark/internal/shell"
)

// wordAt parses a command line and returns one of its words.
func wordAt(t *testing.T, src string, ordinal int) command.Word {
	t.Helper()

	parsed, err := shell.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q) = %v", src, err)
	}
	found := command.Simples(parsed)
	if len(found) == 0 {
		t.Fatalf("Simples(%q) found no commands", src)
	}
	word, ok := found[0].At(ordinal)
	if !ok {
		t.Fatalf("no word at ordinal %d in %q", ordinal, src)
	}
	return word
}

// COVERS: FR-7.8 | positive
func TestTheTwoReadingsOfAWordDiffer(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		src      string
		ordinal  int
		wantText string
		wantVal  string
	}{
		{name: "escaped command", src: `\ls -la`, ordinal: 0, wantText: `\ls`, wantVal: `ls`},
		{name: "escaped space", src: `rm a\ b`, ordinal: 1, wantText: `a\ b`, wantVal: `a b`},
		{name: "quoted", src: `rm "a b"`, ordinal: 1, wantText: `"a b"`, wantVal: `a b`},
		{
			name:     "hidden dot dir",
			src:      `rm .cl\aude`,
			ordinal:  1,
			wantText: `.cl\aude`,
			wantVal:  `.claude`,
		},
		{name: "plain", src: `rm x`, ordinal: 1, wantText: `x`, wantVal: `x`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			word := wordAt(t, c.src, c.ordinal)

			text, ok := rules.ReadingWritten.Of(word)
			if !ok || text != c.wantText {
				t.Errorf("ReadingWritten = %q (%v), want %q", text, ok, c.wantText)
			}
			value, ok := rules.ReadingInterpreted.Of(word)
			if !ok || value != c.wantVal {
				t.Errorf("ReadingInterpreted = %q (%v), want %q", value, ok, c.wantVal)
			}
		})
	}
}

// COVERS: FR-7.8 | regression
func TestAClauseOnTheValueSeesThroughAnEscapedPath(t *testing.T) {
	t.Parallel()

	// This is the bypass the default exists to close. The pattern protects
	// .claude; the command hides it with an escape the shell removes.
	guard, err := rules.Pattern(`\.claude`)
	if err != nil {
		t.Fatalf("Pattern = %v", err)
	}

	word := wordAt(t, `rm .cl\aude`, 1)

	value, _ := rules.ReadingInterpreted.Of(word)
	if !guard.Matches(value) {
		t.Errorf("the value reading %q escaped a rule about .claude", value)
	}

	// Read as written, it does not match -- which is exactly why the value is
	// the default and the text has to be asked for.
	text, _ := rules.ReadingWritten.Of(word)
	if guard.Matches(text) {
		t.Errorf("the text reading %q matched, so this test proves nothing", text)
	}
}

// COVERS: FR-7.9 | negative
func TestAValueThatDoesNotExistDoesNotMatch(t *testing.T) {
	t.Parallel()

	// Nothing is expanded, so `$HOME` has no value. A clause on the value must
	// find nothing to test rather than testing the empty string, which would
	// match a pattern like `.*`.
	word := wordAt(t, `rm $HOME`, 1)

	if _, ok := rules.ReadingInterpreted.Of(word); ok {
		t.Error("an unexpanded word reported a value")
	}
	if text, ok := rules.ReadingWritten.Of(word); !ok || text != `$HOME` {
		t.Errorf("ReadingWritten = %q (%v), want %q", text, ok, `$HOME`)
	}
}

// COVERS: FR-7.8 | edge
func TestTheDefaultReadingIsTheInterpretedValue(t *testing.T) {
	t.Parallel()

	reading, err := rules.ParseReading("")
	if err != nil {
		t.Fatalf("ParseReading(\"\") = %v", err)
	}
	if reading != rules.ReadingInterpreted {
		t.Errorf("default reading = %q, want %q", reading, rules.ReadingInterpreted)
	}

	for _, name := range []string{"interpreted", "written"} {
		if _, err := rules.ParseReading(name); err != nil {
			t.Errorf("ParseReading(%q) = %v", name, err)
		}
	}

	if _, err := rules.ParseReading("wibble"); !errors.Is(err, rules.ErrReading) {
		t.Errorf("ParseReading(\"wibble\") = %v, want %v", err, rules.ErrReading)
	}
}
