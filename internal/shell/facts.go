package shell

import (
	"slices"

	"mvdan.cc/sh/v3/pattern"
	"mvdan.cc/sh/v3/syntax"
)

// A Fact is a named property of a command's syntax tree.
//
// Facts are hierarchical, and a node records every level it satisfies: a `$(…)`
// records both `substitution` and `substitution.command`. A rule may therefore
// forbid a whole family by naming the parent, or one member by naming the child,
// without the rule file needing to know which node types make up the family.
type Fact string

// The plumbing and composition facts. These are what tier-one rules name.
//
// Their shared purpose is narrow and worth stating: each one is a way for a
// command's effect to stop being determined by its own text. A command carrying
// none of them does exactly what it appears to do, which is the precondition
// every later tier depends on: reasoning about which paths a command reaches
// is unsound the moment a substitution can produce a path at runtime.
const (
	FactRedirect         Fact = "redirect"
	FactRedirectTruncate Fact = "redirect.truncate"
	FactRedirectAppend   Fact = "redirect.append"
	FactRedirectInput    Fact = "redirect.input"
	FactRedirectDup      Fact = "redirect.dup"
	FactHeredoc          Fact = "redirect.heredoc"

	FactPipe    Fact = "pipe"
	FactLogical Fact = "logical"

	FactSubstitution           Fact = "substitution"
	FactSubstitutionCommand    Fact = "substitution.command"
	FactSubstitutionProcess    Fact = "substitution.process"
	FactSubstitutionParameter  Fact = "substitution.parameter"
	FactSubstitutionArithmetic Fact = "substitution.arithmetic"

	FactSequence   Fact = "sequence"
	FactBackground Fact = "background"
	FactSubshell   Fact = "subshell"
	FactBlock      Fact = "block"
	FactNegation   Fact = "negation"

	FactAssignment  Fact = "assignment"
	FactFunction    Fact = "function"
	FactLoop        Fact = "loop"
	FactConditional Fact = "conditional"
	FactGlob        Fact = "glob"
	FactExtGlob     Fact = "glob.extended"
)

// Command forms that are not a plain invocation.
//
// These matter more than their obscurity suggests. A rule reaches a command
// either by naming the word at ordinal zero or by naming a fact, and each of
// these has no word at ordinal zero: `time rm x` puts `rm` there, and
// `((x=1))` and `let x=1` put nothing there at all. Before these facts existed
// such a statement carried no fact and offered no name, which left it
// addressable by no rule that could be written.
const (
	FactTime        Fact = "time"
	FactCoproc      Fact = "coproc"
	FactDeclaration Fact = "declaration"
	FactArithmetic  Fact = "arithmetic"
)

// A Finding is one node that established a fact, with enough of the source to
// name it in a message. A denial that quotes the offending text and its position
// can be checked by the person reading it; one that only names the rule cannot.
type Finding struct {
	Fact Fact
	Line uint
	Col  uint
	Text string
}

// Facts is every property of one command, gathered in a single walk. Cost is set
// by the size of the tree, not by the number of rules consulting it, so a
// tier-one rule added later costs nothing to evaluate.
type Facts struct {
	findings []Finding
	index    map[Fact][]int

	// The parser's own vocabularies, counted in the same walk: which node
	// types appeared, which operators, and which statement flags were set.
	// Keyed by name, holding the source text of the first occurrence, so a
	// message can quote what set a rule off rather than naming the category
	// it fell into. "caused by: $HOME" is checkable; "caused by: ParamExp"
	// asks the reader to go and find it.
	nodes map[string]string
	ops   map[string]string
	flags map[string]string
}

// Has reports whether any node established the fact.
func (f *Facts) Has(fact Fact) bool { return len(f.index[fact]) > 0 }

// Count reports how many nodes established the fact.
func (f *Facts) Count(fact Fact) int { return len(f.index[fact]) }

// First returns the earliest node that established the fact.
func (f *Facts) First(fact Fact) (Finding, bool) {
	idx := f.index[fact]
	if len(idx) == 0 {
		return Finding{}, false
	}
	return f.findings[idx[0]], true
}

// All returns every finding, in source order.
func (f *Facts) All() []Finding { return f.findings }

// Names returns the distinct facts established, sorted, for display.
func (f *Facts) Names() []Fact {
	names := make([]Fact, 0, len(f.index))
	for fact := range f.index {
		names = append(names, fact)
	}
	slices.Sort(names)
	return names
}

func (f *Facts) record(p *Parsed, node syntax.Node, facts ...Fact) {
	pos := node.Pos()
	for _, fact := range facts {
		f.index[fact] = append(f.index[fact], len(f.findings))
		f.findings = append(f.findings, Finding{
			Fact: fact,
			Line: pos.Line(),
			Col:  pos.Col(),
			Text: oneLine(p.Text(node)),
		})
	}
}

// Facts walks the tree once and reports every property it establishes.
func (p *Parsed) Facts() *Facts {
	f := &Facts{
		index: map[Fact][]int{},
		nodes: map[string]string{},
		ops:   map[string]string{},
		flags: map[string]string{},
	}

	if len(p.File.Stmts) > 1 {
		f.record(p, p.File, FactSequence)
	}

	syntax.Walk(p.File, func(node syntax.Node) bool {
		if node == nil {
			return true
		}
		p.recordVocabulary(f, node)
		p.recordNode(f, node)
		return true
	})

	return f
}

// recordNode is split three ways by what the facts mean rather than by size:
// how the command is plumbed together, where its words come from, and what it
// declares. A rule reaches for one of those three, rarely for a mixture.
func (p *Parsed) recordNode(f *Facts, node syntax.Node) {
	if p.recordComposition(f, node) {
		return
	}
	if p.recordExpansion(f, node) {
		return
	}
	if p.recordDeclaration(f, node) {
		return
	}
	p.recordCommandForm(f, node)
}

// recordCommandForm covers the statements that are not a plain invocation.
//
// A rule addresses a command by naming the word at ordinal zero or by naming a
// fact. None of these has a usable word at ordinal zero: `time rm x` puts
// `rm` there, `((x=1))` and `let x=1` put nothing there, so without a fact
// apiece they would be reachable by no rule at all.
func (p *Parsed) recordCommandForm(f *Facts, node syntax.Node) {
	switch n := node.(type) {
	case *syntax.TimeClause:
		f.record(p, n, FactTime)
	case *syntax.CoprocClause:
		f.record(p, n, FactCoproc)
	case *syntax.DeclClause:
		// export, declare, readonly, local, typeset. Written as a keyword
		// rather than a call, so `export` is never a command word and a rule
		// naming it would never fire, while `export PATH=…` decides which
		// binary every later command resolves to.
		f.record(p, n, FactDeclaration)
	case *syntax.ArithmCmd:
		f.record(p, n, FactArithmetic)
	case *syntax.LetClause:
		f.record(p, n, FactArithmetic)
	default:
	}
}

// recordComposition covers the ways separate commands are joined and their
// streams rearranged. These are the tier-one facts.
func (p *Parsed) recordComposition(f *Facts, node syntax.Node) bool {
	switch n := node.(type) {
	case *syntax.Redirect:
		f.record(p, n, FactRedirect, redirectKind(n.Op))
	case *syntax.BinaryCmd:
		p.recordBinary(f, n)
	case *syntax.Subshell:
		f.record(p, n, FactSubshell)
	case *syntax.Block:
		f.record(p, n, FactBlock)
	case *syntax.Stmt:
		if n.Background {
			f.record(p, n, FactBackground)
		}
		if n.Negated {
			f.record(p, n, FactNegation)
		}
	default:
		return false
	}
	return true
}

// recordBinary separates the two meanings of a BinaryCmd. A pipe couples two
// commands' streams; a logical concatenation makes one conditional on the
// other. A rule forbidding one does not forbid the other.
func (p *Parsed) recordBinary(f *Facts, node *syntax.BinaryCmd) {
	switch node.Op {
	case syntax.Pipe, syntax.PipeAll:
		f.record(p, node, FactPipe)
	case syntax.AndStmt, syntax.OrStmt:
		f.record(p, node, FactLogical)
	default:
	}
}

// recordExpansion covers every way a word becomes something other than what it
// says: the four substitutions and wildcard expansion.
func (p *Parsed) recordExpansion(f *Facts, node syntax.Node) bool {
	switch n := node.(type) {
	case *syntax.CmdSubst:
		f.record(p, n, FactSubstitution, FactSubstitutionCommand)
	case *syntax.ProcSubst:
		f.record(p, n, FactSubstitution, FactSubstitutionProcess)
	case *syntax.ParamExp:
		f.record(p, n, FactSubstitution, FactSubstitutionParameter)
	case *syntax.ArithmExp:
		f.record(p, n, FactSubstitution, FactSubstitutionArithmetic)
	case *syntax.ExtGlob:
		f.record(p, n, FactGlob, FactExtGlob)
	case *syntax.Word:
		p.recordGlob(f, n)
	default:
		return false
	}
	return true
}

// recordDeclaration covers what a command introduces: names, functions, and the
// control structures that make it a program rather than an invocation.
func (p *Parsed) recordDeclaration(f *Facts, node syntax.Node) bool {
	switch n := node.(type) {
	case *syntax.Assign:
		f.record(p, n, FactAssignment)
	case *syntax.FuncDecl:
		f.record(p, n, FactFunction)
	case *syntax.ForClause, *syntax.WhileClause:
		f.record(p, n, FactLoop)
	case *syntax.IfClause, *syntax.CaseClause, *syntax.TestClause:
		f.record(p, n, FactConditional)
	default:
		return false
	}
	return true
}

// recordGlob reports a wildcard only where the shell would expand one. The check
// runs over a Word's own literal parts rather than over every Lit in the tree,
// because the `*` in `grep "*"` or `grep '*'` is an argument and not a glob:
// the quoting that makes the difference is a property of the Lit's parent.
func (p *Parsed) recordGlob(f *Facts, word *syntax.Word) {
	for _, part := range word.Parts {
		lit, ok := part.(*syntax.Lit)
		if !ok {
			continue
		}
		if pattern.HasMeta(lit.Value, 0) {
			f.record(p, word, FactGlob)
			return
		}
	}
}

func redirectKind(op syntax.RedirOperator) Fact {
	switch op {
	case syntax.RdrOut, syntax.RdrClob, syntax.RdrAll, syntax.RdrAllClob:
		return FactRedirectTruncate
	case syntax.AppOut, syntax.AppClob, syntax.AppAll, syntax.AppAllClob:
		return FactRedirectAppend
	case syntax.RdrIn, syntax.RdrInOut:
		return FactRedirectInput
	case syntax.DplIn, syntax.DplOut:
		return FactRedirectDup
	case syntax.Hdoc, syntax.DashHdoc, syntax.WordHdoc:
		return FactHeredoc
	default:
		return FactRedirect
	}
}
