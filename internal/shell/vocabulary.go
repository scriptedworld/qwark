package shell

import (
	"slices"

	"mvdan.cc/sh/v3/syntax"
)

// The three vocabularies a clause names besides facts: the node types present,
// the operators used, and the flags a statement carries.
//
// These are the parser's own names rather than a summary of them, which is what
// keeps them from falling behind. A summary can be silently incomplete, and was:
// `TimeClause`, `LetClause`, `ArithmCmd`, `CoprocClause` and `DeclClause`
// established no fact for as long as nobody had written one, leaving `time rm x`
// and `((x=1))` addressable by no rule at all.

// HasNode reports whether a node of this type appears anywhere in the command,
// and the source text of the first one.
func (f *Facts) HasNode(name string) (string, bool) { return lookup(f.nodes, name) }

// HasOp reports whether this operator appears anywhere in the command, and the
// source it appeared in. The spelling is the operator as written: `|`, `&&`,
// `>>`, `<<-`.
func (f *Facts) HasOp(op string) (string, bool) { return lookup(f.ops, op) }

// HasFlag reports whether any statement carries this flag, and the statement.
func (f *Facts) HasFlag(name string) (string, bool) { return lookup(f.flags, name) }

func lookup(from map[string]string, key string) (string, bool) {
	text, present := from[key]
	return text, present
}

// The statement flags, named for the fields they come from.
//
// **Under bash only two of these are reachable.** `Coprocess`
// is mksh's `|&` and `Disown` is zsh's `&|` and `&!`, and the bash parser
// rejects both: a rejected command being a denied one. They are here so that
// the vocabulary does not have to grow if the variant ever changes, which is
// the failure the node types actually suffered.
const (
	FlagNegated    = "Negated"
	FlagBackground = "Background"
	FlagCoprocess  = "Coprocess"
	FlagDisown     = "Disown"
)

// KnownFlag reports whether a rule file names a flag that exists.
func KnownFlag(name string) bool {
	return slices.Contains(everyFlag(), name)
}

func everyFlag() []string {
	return []string{FlagNegated, FlagBackground, FlagCoprocess, FlagDisown}
}

// KnownNode reports whether a rule file names a node type that exists.
//
// This list is maintained by hand, and unlike a fact table its falling behind is
// loud rather than silent: a name missing from it is refused at load, where the
// author sees it immediately. A name missing from a *fact* table used to mean a
// construct no rule could reach, which nobody saw at all.
func KnownNode(name string) bool {
	return slices.Contains(everyNode(), name)
}

func everyNode() []string {
	return []string{
		// Commands.
		"ArithmCmd", "BinaryCmd", "Block", "CallExpr", "CaseClause",
		"CoprocClause", "DeclClause", "ForClause", "FuncDecl", "IfClause",
		"LetClause", "Subshell", "TestClause", "TestDecl", "TimeClause",
		"WhileClause",
		// Structure.
		"File", "Stmt", "Comment", "Assign", "ArrayElem", "ArrayExpr",
		"CaseItem", "Redirect", "Word", "WordIter",
		// Word parts.
		"ArithmExp", "BraceExp", "CmdSubst", "DblQuoted", "ExtGlob",
		"Expansion", "Lit", "ParamExp", "ProcSubst", "Replace", "SglQuoted",
		"Slice",
		// Arithmetic and tests.
		"BinaryArithm", "BinaryTest", "CStyleLoop", "ParenArithm",
		"ParenTest", "UnaryArithm", "UnaryTest",
	}
}

// recordVocabulary notes the node's type, its operator and its flags. It runs
// for every node of the one walk the facts are gathered in, so naming a node
// type in a rule costs nothing that naming a fact does not.
func (p *Parsed) recordVocabulary(f *Facts, node syntax.Node) {
	text := oneLine(p.Text(node))
	remember(f.nodes, typeName(node), text)

	if op := operatorOf(node); op != "" {
		remember(f.ops, op, text)
	}

	stmt, ok := node.(*syntax.Stmt)
	if !ok {
		return
	}
	for name, set := range map[string]bool{
		FlagNegated:    stmt.Negated,
		FlagBackground: stmt.Background,
		FlagCoprocess:  stmt.Coprocess,
		FlagDisown:     stmt.Disown,
	} {
		if set {
			remember(f.flags, name, text)
		}
	}
}

// remember keeps the FIRST occurrence. A message quotes where a rule was first
// satisfied, which is where its author will look.
func remember(into map[string]string, key, text string) {
	if _, seen := into[key]; !seen {
		into[key] = text
	}
}
