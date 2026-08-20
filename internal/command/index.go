package command

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Errors a malformed index can produce. A rule file carrying one is a
// configuration error, and configuration errors are fatal by design: an index
// nobody can read is a rule nobody can check.
var (
	ErrEmptyIndex = errors.New("index selects nothing")
	ErrOrdinal    = errors.New("ordinal is not a whole number")
	ErrRange      = errors.New("range wants exactly two endpoints")

	// ErrUnbounded refuses `..`, which names every argument -- and so does
	// omitting the index entirely. Two spellings of one meaning is a thing to
	// remove, not to support: whichever a reader has not seen before costs
	// them a trip to the documentation to learn it meant the other.
	ErrUnbounded = errors.New("a range with neither end is the same as no index; remove the index")
)

// rangeSeparator is spelled `..` rather than `-` because an endpoint may be
// negative, and `-3--1` cannot be read at a glance. The separator that works in
// both directions is the one that is used in both directions.
const rangeSeparator = ".."

// An Index selects word ordinals within a simple command.
//
// Ordinal 0 is the command name and arguments run from 1. A negative ordinal
// counts from the end, -1 being the last word, so `1..-1` is every argument
// regardless of how many there are.
//
// One end of a range may be left off: `1..` runs to the last word and `..3`
// runs from the first argument. Leaving off both is a configuration error --
// `..` names every argument, and so does stating no index at all.
//
// **An open end never reaches the command.** An omitted start is 1, because
// arguments do not start at 0 -- the command does. Ordinal 0 is reachable only
// by naming it. A test written without an index asks about what the command was
// given, and matching the command's own name too would make `value = "rm"` true
// of `echo rm`.
//
// Negative ordinals are the stable way to name an operand, because bundling
// moves the positive ones: `rm -r -f x` puts x at 3 and `rm -rf x` puts it at
// 2, while -1 names it in both.
type Index struct {
	spec  string
	terms []term
}

type term struct {
	from    int
	to      int
	isRange bool
}

// ParseIndex reads an index specification: a comma-separated list of ordinals
// and `..` ranges, such as `1`, `2..4`, `-3..-1` or `1,3..5,-1`.
func ParseIndex(spec string) (Index, error) {
	fields := strings.Split(spec, ",")
	terms := make([]term, 0, len(fields))

	for _, field := range fields {
		parsed, err := parseTerm(strings.TrimSpace(field))
		if err != nil {
			return Index{}, fmt.Errorf("index %q: %w", spec, err)
		}
		terms = append(terms, parsed)
	}

	if len(terms) == 0 {
		return Index{}, fmt.Errorf("index %q: %w", spec, ErrEmptyIndex)
	}

	return Index{spec: spec, terms: terms}, nil
}

func parseTerm(field string) (term, error) {
	if field == "" {
		return term{}, ErrEmptyIndex
	}

	if !strings.Contains(field, rangeSeparator) {
		ordinal, err := parseOrdinal(field)
		if err != nil {
			return term{}, err
		}
		return term{from: ordinal, to: ordinal}, nil
	}

	from, to, found := strings.Cut(field, rangeSeparator)
	if !found || strings.Contains(to, rangeSeparator) {
		return term{}, fmt.Errorf("%q: %w", field, ErrRange)
	}

	// ONE end may be left off. `1..` runs to the last word and `..3` runs from
	// the first argument. Leaving off both is refused: it says every argument,
	// which is what stating no index already says.
	if strings.TrimSpace(from) == "" && strings.TrimSpace(to) == "" {
		return term{}, fmt.Errorf("%q: %w", field, ErrUnbounded)
	}

	start, err := endpoint(from, firstOrdinal)
	if err != nil {
		return term{}, err
	}
	end, err := endpoint(to, lastOrdinal)
	if err != nil {
		return term{}, err
	}

	return term{from: start, to: end, isRange: true}, nil
}

// The ordinals an open end of a range stands for.
//
// **An omitted start is 1, not 0.** Arguments do not start at 0; the command
// does. So `..2` is the first two arguments, and an open end never reaches the
// command at all.
//
// Ordinal 0 is therefore reachable only by naming it. That is the whole point:
// a range is about what the command was given, and the command is not one of
// the things it was given.
//
// An omitted end is -1, the last word, whatever the count. Omitting both is
// refused -- see ErrUnbounded.
const (
	firstOrdinal = 1
	lastOrdinal  = -1
)

// endpoint reads one end of a range, or supplies the default when it is absent.
func endpoint(text string, whenAbsent int) (int, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return whenAbsent, nil
	}
	return parseOrdinal(trimmed)
}

func parseOrdinal(text string) (int, error) {
	ordinal, err := strconv.Atoi(text)
	if err != nil {
		return 0, fmt.Errorf("%q: %w", text, ErrOrdinal)
	}
	return ordinal, nil
}

// String returns the specification as written, so a message about a rule can
// quote the index the rule's author typed.
func (i Index) String() string { return i.spec }

// Select resolves the index against a command whose highest ordinal is last,
// returning the ordinals it names in ascending order, without duplicates.
//
// An ordinal outside the command selects nothing rather than failing: a rule
// about the third argument simply does not apply to a command with one. A range
// whose endpoints resolve backwards likewise names nothing.
func (i Index) Select(last int) []int {
	chosen := make(map[int]bool)

	for _, t := range i.terms {
		from, fromOK := resolve(t.from, last)
		if !t.isRange {
			if fromOK {
				chosen[from] = true
			}
			continue
		}

		to, toOK := resolve(t.to, last)
		if !fromOK || !toOK {
			continue
		}
		for ordinal := from; ordinal <= to; ordinal++ {
			chosen[ordinal] = true
		}
	}

	return sorted(chosen)
}

// resolve turns an ordinal as written into a position, reporting whether the
// command is long enough to have one. -1 is the last word, so a negative
// ordinal counts back from one past the end.
func resolve(ordinal, last int) (int, bool) {
	if ordinal < 0 {
		ordinal = last + 1 + ordinal
	}
	if ordinal < 0 || ordinal > last {
		return 0, false
	}
	return ordinal, true
}

func sorted(set map[int]bool) []int {
	out := make([]int, 0, len(set))
	for ordinal := range set {
		out = append(out, ordinal)
	}
	sort.Ints(out)
	return out
}
