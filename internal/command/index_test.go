package command_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/scriptedworld/qwark/internal/command"
)

// selectOf parses an index and resolves it, failing the test if the index does
// not parse, so the tables below read as assertions about what is selected.
func selectOf(t *testing.T, spec string, last int) []int {
	t.Helper()

	index, err := command.ParseIndex(spec)
	if err != nil {
		t.Fatalf("ParseIndex(%q) = %v", spec, err)
	}
	return index.Select(last)
}

// COVERS: FR-5.1 | positive
func TestOrdinalZeroIsTheCommandAndArgumentsRunFromOne(t *testing.T) {
	t.Parallel()

	// rm -rf x  ->  0:rm  1:-rf  2:x
	const last = 2

	cases := []struct {
		spec string
		want []int
	}{
		{spec: "0", want: []int{0}},
		{spec: "1", want: []int{1}},
		{spec: "2", want: []int{2}},
	}

	for _, c := range cases {
		t.Run(c.spec, func(t *testing.T) {
			t.Parallel()

			if got := selectOf(t, c.spec, last); !slices.Equal(got, c.want) {
				t.Errorf("Select(%q) = %v, want %v", c.spec, got, c.want)
			}
		})
	}
}

// COVERS: FR-5.2 | positive
func TestNegativeOrdinalsCountFromTheEnd(t *testing.T) {
	t.Parallel()

	// rm -r -f x  ->  0:rm  1:-r  2:-f  3:x
	const last = 3

	cases := []struct {
		spec string
		want []int
	}{
		{spec: "-1", want: []int{3}},
		{spec: "-2", want: []int{2}},
		{spec: "-3", want: []int{1}},
		{spec: "-4", want: []int{0}},
	}

	for _, c := range cases {
		t.Run(c.spec, func(t *testing.T) {
			t.Parallel()

			if got := selectOf(t, c.spec, last); !slices.Equal(got, c.want) {
				t.Errorf("Select(%q) = %v, want %v", c.spec, got, c.want)
			}
		})
	}
}

// COVERS: FR-5.2 | property
func TestTheLastOrdinalIsStableAgainstBundling(t *testing.T) {
	t.Parallel()

	// The operand sits at a different positive ordinal depending only on how
	// the options were bundled. -1 names it in both, which is why a rule about
	// an operand should be written that way.
	const bundled = 2   // rm -rf x
	const unbundled = 3 // rm -r -f x

	if got := selectOf(t, "-1", bundled); !slices.Equal(got, []int{bundled}) {
		t.Errorf("Select(-1) = %v for the bundled form, want [%d]", got, bundled)
	}
	if got := selectOf(t, "-1", unbundled); !slices.Equal(got, []int{unbundled}) {
		t.Errorf("Select(-1) = %v for the unbundled form, want [%d]", got, unbundled)
	}
}

// COVERS: FR-5.3, FR-5.4 | positive
func TestAnIndexMayNameSeveralOrdinals(t *testing.T) {
	t.Parallel()

	const last = 5

	cases := []struct {
		name string
		spec string
		want []int
	}{
		{name: "list", spec: "1,3", want: []int{1, 3}},
		{name: "forward range", spec: "2..4", want: []int{2, 3, 4}},
		{name: "trailing range", spec: "-3..-1", want: []int{3, 4, 5}},
		{name: "every argument", spec: "1..-1", want: []int{1, 2, 3, 4, 5}},
		{name: "mixed", spec: "1,3..5,-1", want: []int{1, 3, 4, 5}},
		{name: "spaced", spec: " 1 , 3 .. 4 ", want: []int{1, 3, 4}},
		{name: "overlapping", spec: "1..3,2..4", want: []int{1, 2, 3, 4}},
		{name: "single point range", spec: "2..2", want: []int{2}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			if got := selectOf(t, c.spec, last); !slices.Equal(got, c.want) {
				t.Errorf("Select(%q) = %v, want %v", c.spec, got, c.want)
			}
		})
	}
}

// COVERS: FR-5.5 | edge
func TestAnOrdinalTheCommandLacksSelectsNothing(t *testing.T) {
	t.Parallel()

	// `ls -l`  ->  0:ls  1:-l
	const last = 1

	cases := []struct {
		name string
		spec string
		want []int
	}{
		{name: "past the end", spec: "3", want: nil},
		{name: "before the start", spec: "-5", want: nil},
		{name: "range past the end", spec: "5..9", want: nil},
		{name: "range clipped at the end", spec: "1..1", want: []int{1}},
		{name: "some present some not", spec: "1,7", want: []int{1}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			got := selectOf(t, c.spec, last)
			if len(got) == 0 && len(c.want) == 0 {
				return
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("Select(%q) = %v, want %v", c.spec, got, c.want)
			}
		})
	}
}

// COVERS: FR-5.5 | edge
func TestARangeRunningBackwardsNamesNothing(t *testing.T) {
	t.Parallel()

	// With five arguments -1 resolves to 5, so `-1..1` runs from 5 down to 1.
	// Descending is not silently reversed: a rule that says it backwards means
	// something different from the rule its author meant to write.
	const last = 5

	if got := selectOf(t, "-1..1", last); len(got) != 0 {
		t.Errorf("Select(-1..1) = %v, want nothing", got)
	}

	// The same span written the right way round does select.
	if got := selectOf(t, "1..-1", last); len(got) != last {
		t.Errorf("Select(1..-1) = %v, want %d ordinals", got, last)
	}
}

// COVERS: FR-5.6, FR-5.13 | negative
func TestAMalformedIndexIsAConfigurationError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		spec string
		want error
	}{
		{name: "empty", spec: "", want: command.ErrEmptyIndex},
		{name: "empty term", spec: "1,,2", want: command.ErrEmptyIndex},
		{name: "trailing comma", spec: "1,", want: command.ErrEmptyIndex},
		{name: "not a number", spec: "first", want: command.ErrOrdinal},
		{name: "float", spec: "1.5", want: command.ErrOrdinal},
		{name: "hyphen range", spec: "2-4", want: command.ErrOrdinal},
		{name: "three endpoints", spec: "1..2..3", want: command.ErrRange},
		{name: "neither end", spec: "..", want: command.ErrUnbounded},
		{name: "neither end, spaced", spec: " .. ", want: command.ErrUnbounded},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			_, err := command.ParseIndex(c.spec)
			if !errors.Is(err, c.want) {
				t.Errorf("ParseIndex(%q) = %v, want %v", c.spec, err, c.want)
			}
		})
	}
}

// COVERS: FR-5.11 | positive
func TestEitherEndOfARangeMayBeLeftOff(t *testing.T) {
	t.Parallel()

	// `rm -r -f x` -> 0:rm 1:-r 2:-f 3:x
	//
	// An open end never reaches the command: an omitted start is 1, because
	// arguments do not start at 0. Reaching ordinal 0 means naming it.
	const last = 3

	cases := []struct {
		name string
		spec string
		want []int
	}{
		{name: "from one to the end", spec: "1..", want: []int{1, 2, 3}},
		{name: "from the first argument to two", spec: "..2", want: []int{1, 2}},
		{name: "last two", spec: "-2..", want: []int{2, 3}},
		{name: "first argument to third from last", spec: "..-3", want: []int{1}},
		{name: "the command, named explicitly, plus a range", spec: "0,2..", want: []int{0, 2, 3}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			if got := selectOf(t, c.spec, last); !slices.Equal(got, c.want) {
				t.Errorf("Select(%q) = %v, want %v", c.spec, got, c.want)
			}
		})
	}
}

// COVERS: FR-5.6 | property
func TestAnIndexQuotesItselfAsWritten(t *testing.T) {
	t.Parallel()

	const spec = "1,3..5,-1"

	index, err := command.ParseIndex(spec)
	if err != nil {
		t.Fatalf("ParseIndex(%q) = %v", spec, err)
	}

	// A message about a rule has to quote the index its author typed, not a
	// normalised form they would not recognise.
	if got := index.String(); got != spec {
		t.Errorf("String() = %q, want %q", got, spec)
	}
}
