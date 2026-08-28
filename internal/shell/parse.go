// Package shell turns a Bash command string into a syntax tree and answers
// questions about it. It is the only place in qwark that knows the command is
// text; everything above it works on the tree.
package shell

import (
	"errors"
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// Parsed is a command string and the tree it produced. Src is retained because
// node positions are byte offsets into it, so the exact source text of any node
// is recoverable without re-printing the tree.
type Parsed struct {
	Src  string
	File *syntax.File
}

// ParseError reports a command that is not valid Bash. A command qwark cannot
// parse is a command qwark cannot judge, which callers must treat as a verdict
// in its own right rather than as an absence of findings.
type ParseError struct {
	Src  string
	Err  error
	Line uint
	Col  uint
	// Incomplete is true when the command is well-formed so far but ends
	// mid-construct: an unclosed quote, a dangling `|`, a `for` with no
	// `done`. Claude Code hands us whole commands, so this indicates
	// truncation rather than a syntax mistake.
	Incomplete bool
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d col %d: %v", e.Line, e.Col, e.Err)
}

func (e *ParseError) Unwrap() error { return e.Err }

// Parse reads one command string as Bash.
//
// **The shell has to be established, not inferred from the tool's name.**
// Claude Code's Bash tool runs zsh 5.9 on the machine this was written on,
// where `$0` is `/bin/zsh` and `BASH_VERSION` is unset, and the
// decision recorded in DESIGN-NOTES is to force that shell to bash rather than
// to teach qwark zsh.
//
// That decision is what makes this variant correct, so it is a precondition and
// not a default. Reading zsh as bash fails silently rather than loudly: of ten
// zsh constructs measured, two were rejected and four parsed cleanly while
// meaning something else. A gate reading the wrong language does not error, it
// answers wrongly.
func Parse(src string) (*Parsed, error) {
	parser := syntax.NewParser(
		syntax.Variant(syntax.LangBash),
		syntax.KeepComments(true),
	)

	file, err := parser.Parse(strings.NewReader(src), "")
	if err != nil {
		perr := &ParseError{Src: src, Err: err, Incomplete: syntax.IsIncomplete(err)}
		if pe, ok := errors.AsType[syntax.ParseError](err); ok {
			perr.Line, perr.Col = pe.Pos.Line(), pe.Pos.Col()
		}
		return nil, perr
	}

	return &Parsed{Src: src, File: file}, nil
}

// Text returns the exact source text of a node, sliced from the original
// command by the node's own byte offsets. This is the source as written:
// quoting, spacing and all, not a re-print of the tree.
func (p *Parsed) Text(node syntax.Node) string {
	start, end := node.Pos().Offset(), node.End().Offset()
	if start > end || end > uint(len(p.Src)) {
		return ""
	}
	return p.Src[start:end]
}
