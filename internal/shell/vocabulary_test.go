package shell_test

import (
	"testing"

	"github.com/scriptedworld/qwark/internal/shell"
)

// COVERS: FR-2.8 | positive
func TestTheNodeTypesPresentAreReported(t *testing.T) {
	t.Parallel()

	// These names are what a rule file writes and what `qwark ast` prints. If
	// the three ever disagree, a rule written from reading the outline stops
	// matching what it was written about.
	facts := factsOf(t, `cat a | grep b > c`)

	for _, name := range []string{"File", "Stmt", "BinaryCmd", "CallExpr", "Word", "Lit", "Redirect"} {
		if _, found := facts.HasNode(name); !found {
			t.Errorf("HasNode(%q) = false", name)
		}
	}
	for _, name := range []string{"Subshell", "FuncDecl", "TimeClause", "Wibble"} {
		if _, found := facts.HasNode(name); found {
			t.Errorf("HasNode(%q) = true for a command that has none", name)
		}
	}
}

// COVERS: FR-2.4 | positive
func TestTheOperatorsUsedAreReported(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		src  string
		want string
	}{
		{name: "pipe", src: `a | b`, want: "|"},
		{name: "and", src: `a && b`, want: "&&"},
		{name: "or", src: `a || b`, want: "||"},
		{name: "truncate", src: `a > f`, want: ">"},
		{name: "append", src: `a >> f`, want: ">>"},
		{name: "heredoc", src: "a <<E\nx\nE", want: "<<"},
		{name: "here string", src: `a <<< x`, want: "<<<"},
		{name: "test operator", src: `[[ -f x ]]`, want: "-f"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			if _, found := factsOf(t, c.src).HasOp(c.want); !found {
				t.Errorf("HasOp(%q) = false for %q", c.want, c.src)
			}
		})
	}
}

// COVERS: FR-2.4 | negative
func TestAnOperatorNotUsedIsNotReported(t *testing.T) {
	t.Parallel()

	// A pipe and a logical concatenation are the same node type. Reporting the
	// type alone would make a rule against one a rule against the other.
	facts := factsOf(t, `a | b`)

	if _, found := facts.HasOp("|"); !found {
		t.Error(`HasOp("|") = false for a pipeline`)
	}
	for _, op := range []string{"&&", "||", ">", "<<"} {
		if _, found := facts.HasOp(op); found {
			t.Errorf("HasOp(%q) = true for a plain pipeline", op)
		}
	}
}

// COVERS: FR-2.8 | positive
func TestTheStatementFlagsSetAreReported(t *testing.T) {
	t.Parallel()

	if _, found := factsOf(t, `sleep 1 &`).HasFlag(shell.FlagBackground); !found {
		t.Error("Background not reported for a backgrounded statement")
	}
	if _, found := factsOf(t, `! false`).HasFlag(shell.FlagNegated); !found {
		t.Error("Negated not reported for a negated statement")
	}

	plain := factsOf(t, `ls`)
	for _, flag := range []string{
		shell.FlagBackground, shell.FlagNegated,
		shell.FlagCoprocess, shell.FlagDisown,
	} {
		if _, found := plain.HasFlag(flag); found {
			t.Errorf("HasFlag(%q) = true for a plain command", flag)
		}
	}
}

// COVERS: FR-4.5 | negative
func TestARuleFileNamingSomethingThatDoesNotExistIsCatchable(t *testing.T) {
	t.Parallel()

	// A name nobody recognises must be refusable at load. A clause naming a
	// node type that does not exist would otherwise match nothing forever,
	// which reads exactly like a rule that is working.
	for _, name := range []string{"CallExpr", "Redirect", "TimeClause", "DeclClause"} {
		if !shell.KnownNode(name) {
			t.Errorf("KnownNode(%q) = false for a real node type", name)
		}
	}
	for _, name := range []string{"Wibble", "callexpr", "", "Command"} {
		if shell.KnownNode(name) {
			t.Errorf("KnownNode(%q) = true for something that is not a node type", name)
		}
	}

	for _, name := range []string{shell.FlagNegated, shell.FlagDisown} {
		if !shell.KnownFlag(name) {
			t.Errorf("KnownFlag(%q) = false for a real flag", name)
		}
	}
	if shell.KnownFlag("Wibble") {
		t.Error(`KnownFlag("Wibble") = true`)
	}
}

// COVERS: FR-2.7 | property
func TestTheVocabularyQuotesTheSourceNotTheName(t *testing.T) {
	t.Parallel()

	// A message says what set a rule off. "caused by: $HOME" can be checked
	// against the command; "caused by: ParamExp" asks the reader to go and
	// find it, in the one situation where their command has just failed.
	facts := factsOf(t, `rm $HOME/x`)

	text, found := facts.HasNode("ParamExp")
	if !found {
		t.Fatal("ParamExp not reported for a command holding one")
	}
	if text != "$HOME" {
		t.Errorf("HasNode(ParamExp) = %q, want the source %q", text, "$HOME")
	}
}

// COVERS: FR-2.7 | edge
func TestTheFirstOccurrenceIsTheOneQuoted(t *testing.T) {
	t.Parallel()

	// Where a rule could have been set off several times, the earliest is
	// where its author will look.
	text, found := factsOf(t, `a > first >> second`).HasNode("Redirect")
	if !found {
		t.Fatal("Redirect not reported")
	}
	if text != "> first" {
		t.Errorf("HasNode(Redirect) = %q, want %q", text, "> first")
	}
}
