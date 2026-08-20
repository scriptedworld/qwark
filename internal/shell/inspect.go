package shell

import (
	"fmt"
	"io"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// Inspect writes a readable outline of the tree: one line per node, indented by
// depth, giving the node type, whatever distinguishes that node from others of
// its type, and the source text it covers.
//
// This exists to be read by a person authoring a rule. A rule names node types,
// so the vocabulary a rule may use has to be visible before the rule is
// written -- guessing at it is how the string-matching guard this replaces got
// its blind spot.
func (p *Parsed) Inspect(w io.Writer) error {
	depth := 0
	var werr error

	syntax.Walk(p.File, func(node syntax.Node) bool {
		if node == nil {
			depth--
			return true
		}
		if werr == nil {
			werr = p.writeNode(w, node, depth)
		}
		depth++
		return true
	})

	return werr
}

func (p *Parsed) writeNode(w io.Writer, node syntax.Node, depth int) error {
	const typeColumn = 44

	label := strings.Repeat("  ", depth) + typeName(node)
	pad := typeColumn - len(label)
	if pad < 1 {
		pad = 1
	}

	line := label + strings.Repeat(" ", pad) + column(oneLine(p.Text(node)))
	if d := detail(node); d != "" {
		line += "   " + d
	}

	if _, err := fmt.Fprintln(w, line); err != nil {
		return fmt.Errorf("writing node outline: %w", err)
	}
	return nil
}

// typeName is the node's Go type without the package qualifier -- CallExpr,
// BinaryCmd, Redirect. These names are the rule vocabulary, so they are printed
// exactly as a rule file will have to spell them.
func typeName(node syntax.Node) string {
	return strings.TrimPrefix(fmt.Sprintf("%T", node), "*syntax.")
}

// detail reports what distinguishes this node from others of the same type: the
// operator, the name, the flags. Node types whose source text already says
// everything are left alone.
func detail(node syntax.Node) string {
	if op := operatorOf(node); op != "" {
		return "op=" + op
	}
	if name := nameOf(node); name != "" {
		return name
	}
	switch n := node.(type) {
	case *syntax.CmdSubst:
		if n.Backquotes {
			return "backquotes"
		}
		return ""
	case *syntax.Stmt:
		return flags(n)
	default:
		return ""
	}
}

// operatorOf reports the operator of the node types whose whole meaning is
// their operator. A BinaryCmd is a pipe or a logical concatenation depending on
// nothing else, so printing the type without the operator says almost nothing.
func operatorOf(node syntax.Node) string {
	switch n := node.(type) {
	case *syntax.BinaryCmd:
		return n.Op.String()
	case *syntax.Redirect:
		return n.Op.String()
	case *syntax.UnaryArithm:
		return n.Op.String()
	case *syntax.BinaryArithm:
		return n.Op.String()
	case *syntax.UnaryTest:
		return n.Op.String()
	case *syntax.BinaryTest:
		return n.Op.String()
	default:
		return ""
	}
}

// nameOf reports the identifier a node introduces or refers to.
func nameOf(node syntax.Node) string {
	switch n := node.(type) {
	case *syntax.ParamExp:
		return "param=" + n.Param.Value
	case *syntax.Assign:
		return "name=" + n.Name.Value
	case *syntax.FuncDecl:
		return "name=" + n.Name.Value
	default:
		return ""
	}
}

func flags(stmt *syntax.Stmt) string {
	var set []string
	if stmt.Negated {
		set = append(set, "negated")
	}
	if stmt.Background {
		set = append(set, "background")
	}
	if stmt.Coprocess {
		set = append(set, "coprocess")
	}
	if stmt.Disown {
		set = append(set, "disown")
	}
	if len(set) == 0 {
		return ""
	}
	return strings.Join(set, ",")
}

// oneLine collapses a node's source onto a single line, and caps it so a long
// here-document cannot push a report off screen.
//
// It returns the text and nothing else. An earlier version returned it already
// decorated for the outline, and that decoration then travelled into every
// message quoting what set a rule off -- "caused by: │ $HOME". Presentation
// belongs where something is presented.
func oneLine(src string) string {
	const maximum = 64

	flat := strings.Join(strings.Fields(src), " ")
	if len(flat) > maximum {
		flat = flat[:maximum-1] + "…"
	}
	return flat
}

// column renders a node's source for the outline, where a rule separates it
// from the node type beside it.
func column(text string) string {
	if text == "" {
		return ""
	}
	return "│ " + text
}
