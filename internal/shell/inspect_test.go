package shell_test

import (
	"strings"
	"testing"

	"github.com/scriptedworld/qwark/internal/shell"
)

// outlineOf renders the outline of a command, failing the test if either step
// does, so the tables below read as assertions about the text.
func outlineOf(t *testing.T, src string) string {
	t.Helper()

	parsed, err := shell.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q) = %v", src, err)
	}

	var out strings.Builder
	if err := parsed.Inspect(&out); err != nil {
		t.Fatalf("Inspect(%q) = %v", src, err)
	}
	return out.String()
}

// COVERS: FR-3.1 | positive
func TestTheOutlineNamesNodeTypesAsARuleMustSpellThem(t *testing.T) {
	t.Parallel()

	got := outlineOf(t, `cat a | grep b > c`)

	// These are the names a rule file has to use. If the outline prints
	// anything else, a rule written from reading it will not match.
	want := []string{"File", "Stmt", "BinaryCmd", "CallExpr", "Word", "Lit", "Redirect"}
	for _, name := range want {
		if !strings.Contains(got, name) {
			t.Errorf("outline does not name %q:\n%s", name, got)
		}
	}
}

// COVERS: FR-3.1 | property
func TestTheOutlineIsOneNodePerLine(t *testing.T) {
	t.Parallel()

	got := outlineOf(t, "cat <<E\nline one\nline two\nE")

	for line := range strings.SplitSeq(strings.TrimRight(got, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			t.Errorf("outline contains a blank line:\n%s", got)
		}
	}
}

// COVERS: FR-3.1 | positive
func TestTheOutlineDistinguishesOperators(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		src  string
		want string
	}{
		{name: "pipe", src: `a | b`, want: "op=|"},
		{name: "and", src: `a && b`, want: "op=&&"},
		{name: "append", src: `a >> f`, want: "op=>>"},
		{name: "parameter", src: `echo $HOME`, want: "param=HOME"},
		{name: "assignment", src: `X=1 echo`, want: "name=X"},
		{name: "function", src: `f() { echo; }`, want: "name=f"},
		{name: "background", src: `sleep 1 &`, want: "background"},
		{name: "negation", src: `! false`, want: "negated"},
		{name: "backquotes", src: "echo `date`", want: "backquotes"},
		{name: "test operator", src: `[[ -f x ]]`, want: "op=-f"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			if got := outlineOf(t, c.src); !strings.Contains(got, c.want) {
				t.Errorf("outline of %q lacks %q:\n%s", c.src, c.want, got)
			}
		})
	}
}

// COVERS: FR-3.1 | edge
func TestALongNodeIsTruncatedRatherThanWrapped(t *testing.T) {
	t.Parallel()

	const width = 120

	got := outlineOf(t, "echo "+strings.Repeat("word ", width))

	for line := range strings.SplitSeq(strings.TrimRight(got, "\n"), "\n") {
		if len([]rune(line)) > width {
			t.Errorf("line is %d runes, want the outline to stay readable:\n%s",
				len([]rune(line)), line)
		}
	}
	if !strings.Contains(got, "…") {
		t.Error("nothing was marked as truncated, so the reader cannot tell")
	}
}

// COVERS: FR-3.1 | edge
func TestDeepNestingStillIndents(t *testing.T) {
	t.Parallel()

	got := outlineOf(t, `a $(b $(c $(d)))`)

	if !strings.Contains(got, "          ") {
		t.Errorf("deeply nested nodes are not indented:\n%s", got)
	}
}
