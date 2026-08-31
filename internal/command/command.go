// Package command lifts a parsed shell tree into the shape a rule talks about:
// a simple command whose words are addressed by ordinal.
//
// A rule names a position and asks something of the word there. That requires
// two things this package provides and the tree alone does not: a stable way to
// address a word, and an honest answer about what the word's value will be.
package command

import (
	"strings"

	"github.com/scriptedworld/qwark/internal/shell"
	"mvdan.cc/sh/v3/syntax"
)

// A Word is one word of a simple command, at a known ordinal.
type Word struct {
	// Ordinal is the word's position. 0 is the command name itself, so
	// arguments begin at 1.
	Ordinal int

	// Text is the source as written, quoting and all.
	Text string

	// Value is the string the shell will pass, and is meaningful only when
	// Determined is true.
	Value string

	// Determined reports whether Value is fixed by the text alone. A word
	// containing any substitution is not determined, and Value is empty.
	//
	// This says nothing about globbing: `*.go` is determined (its text is
	// fixed) yet the shell will replace it with a list of filenames. That is
	// a separate property, reported as a fact by the shell package.
	Determined bool

	// Escaped reports whether the word carried a backslash escape outside of
	// quotes. It matters in its own right rather than as a detail of parsing:
	// a leading escape is what suppresses alias expansion, so `ls` and `\ls`
	// name the same file and run different programs.
	Escaped bool
}

// A Simple is one simple command: a command name and its arguments, in order.
//
// Only simple commands have this shape. A pipeline, a subshell or a loop is a
// structure containing them, and is described by facts rather than ordinals.
type Simple struct {
	Words []Word
	Line  uint
	Col   uint
}

// Name returns the command name, or an empty string when the command word is
// not determined by its text: `$cmd arg`, where nothing here can say what
// will run.
func (s Simple) Name() string {
	if len(s.Words) == 0 || !s.Words[0].Determined {
		return ""
	}
	return s.Words[0].Value
}

// Last returns the highest ordinal in the command, which is the one a negative
// index counts back from. A command with no arguments has a Last of 0.
func (s Simple) Last() int {
	if len(s.Words) == 0 {
		return 0
	}
	return s.Words[len(s.Words)-1].Ordinal
}

// At returns the word at an ordinal.
func (s Simple) At(ordinal int) (Word, bool) {
	for _, word := range s.Words {
		if word.Ordinal == ordinal {
			return word, true
		}
	}
	return Word{}, false
}

// Simples returns every simple command in the tree, in source order.
//
// A bare assignment such as `X=1` is not one: it names no command, so it has no
// ordinal 0 and nothing a positional rule could address.
func Simples(parsed *shell.Parsed) []Simple {
	var found []Simple

	syntax.Walk(parsed.File, func(node syntax.Node) bool {
		call, ok := node.(*syntax.CallExpr)
		if ok && len(call.Args) > 0 {
			found = append(found, simpleOf(parsed, call))
		}
		return true
	})

	return found
}

func simpleOf(parsed *shell.Parsed, call *syntax.CallExpr) Simple {
	words := make([]Word, 0, len(call.Args))
	for ordinal, arg := range call.Args {
		value, escaped, determined := literal(arg)
		words = append(words, Word{
			Ordinal:    ordinal,
			Text:       parsed.Text(arg),
			Value:      value,
			Determined: determined,
			Escaped:    escaped,
		})
	}

	pos := call.Pos()
	return Simple{Words: words, Line: pos.Line(), Col: pos.Col()}
}

// literal reports the value a word will carry, and whether the text alone fixes
// it.
//
// This is deliberately hand-written rather than delegating to expand.Literal.
// Measured: that function resolves `$HOME` to the empty
// string and returns no error, so a caller cannot tell a fixed word from one it
// silently guessed at. It refuses command substitution properly, but the silent
// case is the dangerous one: deciding about `rm -rf /x` when the shell will act
// on `rm -rf /home/user/x` is exactly the mistake a parser was adopted to
// stop making.
func literal(word *syntax.Word) (value string, escaped, determined bool) {
	var built strings.Builder

	for _, part := range word.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			text, hadEscape := unescape(p.Value)
			built.WriteString(text)
			escaped = escaped || hadEscape
		case *syntax.SglQuoted:
			// $'...' processes escapes, so its value is not its text.
			if p.Dollar {
				return "", false, false
			}
			built.WriteString(p.Value)
		case *syntax.DblQuoted:
			if !writeQuoted(&built, p) {
				return "", false, false
			}
		default:
			return "", false, false
		}
	}

	return built.String(), escaped, true
}

// writeQuoted appends a double-quoted run, refusing it if anything inside is
// expanded rather than literal.
func writeQuoted(built *strings.Builder, quoted *syntax.DblQuoted) bool {
	for _, part := range quoted.Parts {
		lit, ok := part.(*syntax.Lit)
		if !ok {
			return false
		}
		built.WriteString(unescapeQuoted(lit.Value))
	}
	return true
}

// quotedEscapes are the only characters a backslash escapes inside double
// quotes. Before anything else the backslash is an ordinary character and both
// it and what follows survive.
const quotedEscapes = "$`\"\\\n"

// unescape resolves the backslash escapes of an unquoted word, and reports
// whether there were any.
//
// The parser keeps escapes in a literal's value: **`a\ b` arrives as `a\ b`
// while bash passes `a b`**, so a gate comparing that value
// against a path is comparing a string the shell will never produce. Written as
// `rm /home/user/.cl\aude/x`, the shell reaches `.claude` and an unresolved
// comparison does not.
func unescape(raw string) (string, bool) {
	if !strings.ContainsRune(raw, '\\') {
		return raw, false
	}

	var built strings.Builder
	escaped := false

	for i := 0; i < len(raw); i++ {
		if raw[i] != '\\' || i+1 == len(raw) {
			built.WriteByte(raw[i])
			continue
		}
		escaped = true
		i++
		// A backslash before a newline is a line continuation: both go.
		if raw[i] != '\n' {
			built.WriteByte(raw[i])
		}
	}

	return built.String(), escaped
}

// unescapeQuoted resolves the escapes a double-quoted run permits.
//
// **From bash:** `"a\$b"` yields `a$b` but `"a\qb"` yields
// `a\qb`: the backslash survives before anything not in quotedEscapes. The
// unquoted rule is not the quoted rule, and applying it here would drop
// backslashes the shell keeps.
func unescapeQuoted(raw string) string {
	if !strings.ContainsRune(raw, '\\') {
		return raw
	}

	var built strings.Builder

	for i := 0; i < len(raw); i++ {
		if raw[i] != '\\' || i+1 == len(raw) || !strings.ContainsRune(quotedEscapes, rune(raw[i+1])) {
			built.WriteByte(raw[i])
			continue
		}
		i++
		if raw[i] != '\n' {
			built.WriteByte(raw[i])
		}
	}

	return built.String()
}
