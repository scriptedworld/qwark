// Package rules holds what a rule is made of. A clause tests one thing about a
// command; this file is how a clause states what it is testing for.
package rules

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Errors a badly stated match produces. All are fatal, like every other
// configuration error: a clause that does not say what it tests for is a rule
// with a hole in it, and a hole reads as a clean run.
var (
	ErrPattern  = errors.New("pattern will not compile")
	ErrNoForm   = errors.New("clause states no match")
	ErrManyForm = errors.New("clause states more than one match")
	ErrEmpty    = errors.New("match is empty and would match everything")
)

// A Form is how a clause states what it is testing for.
type Form string

// The three forms. Each is named for what it does, so which one a clause uses
// is readable without working it out from the syntax.
const (
	// FormValue matches the whole value exactly. `value = "rm"` does not match
	// `rmdir`.
	FormValue Form = "value"

	// FormPartial matches anywhere within the value. `partial = ".claude"`
	// matches any path containing it.
	//
	// This is the form that must be chosen rather than fallen into. The
	// predecessor of this project matched the substring `.archive` and thereby
	// blocked `web.archive.org`, costing a legitimate research route. Nothing
	// here prevents that; naming the form is what makes it a decision the
	// author made and a reader can see.
	FormPartial Form = "partial"

	// FormPattern matches a regular expression against the whole value.
	//
	// Anchored, because `pattern = "rm"` meaning "contains rm" would make every
	// pattern quietly partial, and partial has its own name. A pattern that
	// means "contains" is written with `.*` at both ends, or uses FormPartial.
	//
	// Go's regexp is RE2: no backtracking, linear in the input. A rule file
	// cannot carry a pattern a crafted command makes pathological, which for a
	// program running before every shell command is worth more than the
	// backreferences RE2 gives up.
	FormPattern Form = "pattern"
)

// A Match is what a clause compares a word against.
type Match struct {
	form  Form
	text  string
	regex *regexp.Regexp
}

// Value builds a match for the whole value, exactly.
func Value(text string) Match {
	return Match{form: FormValue, text: text}
}

// Partial builds a match for any part of the value.
func Partial(text string) (Match, error) {
	if text == "" {
		return Match{}, fmt.Errorf("partial: %w", ErrEmpty)
	}
	return Match{form: FormPartial, text: text}, nil
}

// Pattern builds a match from a regular expression, anchored to the whole
// value.
func Pattern(expr string) (Match, error) {
	// Wrapped in a group rather than concatenated: `\Arm|force\z` anchors only
	// its first branch, so `enforcement` would match.
	compiled, err := regexp.Compile(`\A(?:` + expr + `)\z`)
	if err != nil {
		return Match{}, fmt.Errorf("%q: %w: %w", expr, ErrPattern, err)
	}
	return Match{form: FormPattern, text: expr, regex: compiled}, nil
}

// Matches reports whether a value satisfies the match.
func (m Match) Matches(value string) bool {
	switch m.form {
	case FormPartial:
		return strings.Contains(value, m.text)
	case FormPattern:
		return m.regex.MatchString(value)
	case FormValue:
		return value == m.text
	default:
		// A zero Match states nothing, so it tests nothing. Reaching here
		// means a clause was built without going through Spec.Build.
		return false
	}
}

// Form reports how the match was stated, so a message about a rule can say
// which of the three failed.
func (m Match) Form() Form { return m.form }

// String returns the match as its author wrote it, without the anchoring this
// package adds.
func (m Match) String() string { return m.text }

// A Spec is the TOML shape of a clause's match. Exactly one field may be set:
//
//	value   = "rm"          the whole word, exactly
//	partial = ".claude"     anywhere within the word
//	pattern = "rm|rmdir"    a regular expression over the whole word
type Spec struct {
	Value   *string `toml:"value"`
	Partial *string `toml:"partial"`
	Pattern *string `toml:"pattern"`
}

// Build turns a declared match into one that can be evaluated.
//
// Stating none is an error rather than a clause that matches everything, and
// stating two is an error rather than a precedence rule nobody would remember.
func (s Spec) Build() (Match, error) {
	var built Match
	var err error
	stated := 0

	if s.Value != nil {
		stated, built = stated+1, Value(*s.Value)
	}
	if s.Partial != nil {
		stated++
		built, err = Partial(*s.Partial)
	}
	if s.Pattern != nil {
		stated++
		built, err = Pattern(*s.Pattern)
	}

	switch {
	case stated == 0:
		return Match{}, ErrNoForm
	case stated > 1:
		return Match{}, fmt.Errorf("%w: %s", ErrManyForm, strings.Join(s.stated(), ", "))
	case err != nil:
		return Match{}, err
	default:
		return built, nil
	}
}

func (s Spec) stated() []string {
	var names []string
	if s.Value != nil {
		names = append(names, string(FormValue))
	}
	if s.Partial != nil {
		names = append(names, string(FormPartial))
	}
	if s.Pattern != nil {
		names = append(names, string(FormPattern))
	}
	return names
}
