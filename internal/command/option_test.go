package command_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/scriptedworld/qwark/internal/command"
)

// table is the declaration these tests decompose against. rm and tar are here
// because they disagree about `-f`, which is the whole reason a table exists.
func table() command.Table {
	return command.Table{Commands: map[string]command.Declaration{
		"rm": {
			Short: map[string]command.Option{
				"f": {Means: "force"},
				"r": {Means: "recursive"},
				"R": {Means: "recursive"},
				"v": {Means: "verbose"},
			},
			Long: map[string]command.Option{
				"force":     {Means: "force"},
				"recursive": {Means: "recursive"},
				"verbose":   {Means: "verbose"},
			},
			Operands: command.KindPath,
		},
		"tar": {
			Short: map[string]command.Option{
				"f": {Means: "file", TakesValue: true, Kind: command.KindPath},
				"x": {Means: "extract"},
			},
			Long: map[string]command.Option{
				"file":           {Means: "file", TakesValue: true, Kind: command.KindPath},
				"files-from":     {Means: "file-list", TakesValue: true, Kind: command.KindPath},
				"preserve":       {Means: "preserve"},
				"preserve-order": {Means: "preserve-order"},
			},
			Operands: command.KindPath,
		},
		// git is here for one distinction: -m carries prose, not a filename.
		"git": {
			Short: map[string]command.Option{
				"m": {Means: "message", TakesValue: true, Kind: command.KindText},
			},
			Long: map[string]command.Option{
				"message": {Means: "message", TakesValue: true, Kind: command.KindText},
			},
			Operands: command.KindText,
		},
	}}
}

// decomposeOf parses one command line and decomposes its outermost command.
func decomposeOf(t *testing.T, src string) command.Options {
	t.Helper()

	options, err := command.Decompose(outerSimple(t, src), table())
	if err != nil {
		t.Fatalf("Decompose(%q) = %v", src, err)
	}
	return options
}

// COVERS: FR-6.2, FR-6.4 | positive
func TestForcingIsRecognisedHoweverItIsSpelled(t *testing.T) {
	t.Parallel()

	// Every one of these forces. A matcher comparing words against `-f` or
	// `--force` sees it in only two of them.
	for _, src := range []string{
		`rm -f x`,
		`rm -rf x`,
		`rm -fr x`,
		`rm -r -f x`,
		`rm -vrf x`,
		`rm --force x`,
		`rm --forc x`,
		`rm --fo x`,
		`rm --f x`,
	} {
		t.Run(src, func(t *testing.T) {
			t.Parallel()

			if !decomposeOf(t, src).Has("force") {
				t.Errorf("%q was not recognised as forcing", src)
			}
		})
	}
}

// COVERS: FR-6.5 | negative
func TestNothingAfterTheTerminatorIsAnOption(t *testing.T) {
	t.Parallel()

	options := decomposeOf(t, `rm -- -f -rf`)

	if options.Has("force") {
		t.Error("`rm -- -f` was read as forcing; after `--` the -f is a filename")
	}
	if want := []int{2, 3}; !slices.Equal(options.Ordinals(), want) {
		t.Errorf("Operands = %v, want %v", options.Operands, want)
	}
}

// COVERS: FR-6.9 | edge
func TestALoneDashIsAnOperand(t *testing.T) {
	t.Parallel()

	options := decomposeOf(t, `rm -`)

	if len(options.Faults) != 0 {
		t.Errorf("Faults = %v, want none; `-` is a name, not an option", options.Faults)
	}
	if want := []int{1}; !slices.Equal(options.Ordinals(), want) {
		t.Errorf("Operands = %v, want %v", options.Operands, want)
	}
}

// COVERS: FR-6.1, FR-6.6 | positive
func TestTheSameLetterMeansDifferentThingsToDifferentCommands(t *testing.T) {
	t.Parallel()

	// tar's -f is a file, and it consumes the word after it. rm's is force.
	options := decomposeOf(t, `tar -x -f archive.tar`)

	if options.Has("force") {
		t.Error("tar -f was read as forcing; it names a file")
	}
	if !options.Has("file") {
		t.Error("tar -f was not recognised as naming a file")
	}
	for _, given := range options.Given {
		if given.Means == "file" && given.Value != "archive.tar" {
			t.Errorf("file option value = %q, want %q", given.Value, "archive.tar")
		}
	}
	if len(options.Operands) != 0 {
		t.Errorf("Operands = %v, want none; the archive is the option's value",
			options.Operands)
	}
}

// COVERS: FR-6.6 | positive
func TestAValueMayArriveThreeWays(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		src  string
		want string
	}{
		{name: "following word", src: `tar -f archive.tar`, want: "archive.tar"},
		{name: "rest of the bundle", src: `tar -farchive.tar`, want: "archive.tar"},
		{name: "bundled after a flag", src: `tar -xfarchive.tar`, want: "archive.tar"},
		{name: "after equals", src: `tar --file=archive.tar`, want: "archive.tar"},
		{name: "long following word", src: `tar --file archive.tar`, want: "archive.tar"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			options := decomposeOf(t, c.src)
			if len(options.Faults) != 0 {
				t.Fatalf("Faults = %v, want none", options.Faults)
			}

			var got string
			for _, given := range options.Given {
				if given.Means == "file" {
					got = given.Value
				}
			}
			if got != c.want {
				t.Errorf("value = %q, want %q", got, c.want)
			}
		})
	}
}

// COVERS: FR-6.3 | edge
func TestAnExactNameBeatsAnAbbreviationOfAnother(t *testing.T) {
	t.Parallel()

	// `--preserve` is a declared option and also a prefix of
	// `--preserve-order`. getopt_long resolves it to itself, not to ambiguity.
	options := decomposeOf(t, `tar --preserve`)

	if len(options.Faults) != 0 {
		t.Fatalf("Faults = %v, want none", options.Faults)
	}
	if !options.Has("preserve") {
		t.Error("--preserve did not resolve to itself")
	}
}

// COVERS: FR-6.7 | negative
func TestAnAmbiguousAbbreviationIsRefused(t *testing.T) {
	t.Parallel()

	// `--f` could be --file, --files-from. The shell would refuse it, and a
	// gate that picked one would be deciding about a command that will not run.
	options := decomposeOf(t, `tar --f x`)

	if len(options.Faults) == 0 {
		t.Fatal("an ambiguous abbreviation was accepted")
	}
	if !errors.Is(options.Faults[0], command.ErrAmbiguousOption) {
		t.Errorf("fault = %v, want %v", options.Faults[0], command.ErrAmbiguousOption)
	}
}

// COVERS: FR-6.7 | negative
func TestAnUndeclaredOptionIsRefused(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		src  string
		want error
	}{
		{name: "unknown long", src: `rm --wibble x`, want: command.ErrUndeclaredOption},
		{name: "unknown short", src: `rm -q x`, want: command.ErrUndeclaredOption},
		{name: "unknown in a bundle", src: `rm -rqf x`, want: command.ErrUndeclaredOption},
		{name: "missing value", src: `tar -f`, want: command.ErrMissingValue},
		{name: "missing long value", src: `tar --file`, want: command.ErrMissingValue},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			options := decomposeOf(t, c.src)
			if len(options.Faults) == 0 {
				t.Fatalf("%q produced no fault", c.src)
			}
			if !errors.Is(options.Faults[0], c.want) {
				t.Errorf("fault = %v, want %v", options.Faults[0], c.want)
			}
		})
	}
}

// COVERS: FR-6.8 | property
func TestEveryFaultIsReportedNotOnlyTheFirst(t *testing.T) {
	t.Parallel()

	// Two undeclared letters in one bundle and one undeclared long option.
	options := decomposeOf(t, `rm -qz --wibble x`)

	const wantFaults = 3
	if len(options.Faults) != wantFaults {
		t.Errorf("Faults = %v, want %d of them", options.Faults, wantFaults)
	}

	// The declared letters around them still resolve.
	if options.Has("force") {
		t.Error("nothing here forces")
	}
}

// COVERS: FR-6.10 | positive
func TestOnlyDeclaredPathsAreReportedAsPaths(t *testing.T) {
	t.Parallel()

	// The VALUES are what a rule about where a command reaches must test. In
	// `tar -f archive.tar` the path is `archive.tar` while the ordinal is 1,
	// where `-f` was written -- so a caller given only ordinals would test
	// `-f` against the protected paths and never fire.
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{name: "operands", src: `rm -rf a b`, want: []string{"a", "b"}},
		{name: "option value", src: `tar -f archive.tar`, want: []string{"archive.tar"}},
		{name: "both", src: `tar -f archive.tar extra`, want: []string{"archive.tar", "extra"}},
		{name: "a message is not a path", src: `git -m "some words"`, want: nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var got []string
			for _, valued := range decomposeOf(t, c.src).Values(command.KindPath) {
				got = append(got, valued.Value)
			}
			if len(got) == 0 && len(c.want) == 0 {
				return
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("Values(path) = %v, want %v", got, c.want)
			}
		})
	}
}

// COVERS: FR-6.10 | property
func TestAPathReportsWhereItWasWritten(t *testing.T) {
	t.Parallel()

	// A message points at the option, because that is what the author typed
	// and what they would have to change.
	found := decomposeOf(t, `tar -f archive.tar extra`).Values(command.KindPath)
	if len(found) != 2 {
		t.Fatalf("found %v, want two paths", found)
	}

	if found[0].Ordinal != 1 || found[0].Value != "archive.tar" {
		t.Errorf("first = %+v, want ordinal 1 holding archive.tar", found[0])
	}
	if found[1].Ordinal != 3 || found[1].Value != "extra" {
		t.Errorf("second = %+v, want ordinal 3 holding extra", found[1])
	}
}

// COVERS: FR-6.11 | negative
func TestAnUndeclaredKindSelectsNothing(t *testing.T) {
	t.Parallel()

	// Guessing which arguments are paths would be reading the command as text
	// again, so a kind nobody declared matches no word at all.
	if got := decomposeOf(t, `rm -rf a`).Values(command.KindUnknown); got != nil {
		t.Errorf("Values(unknown) = %v, want nothing", got)
	}
	if got := decomposeOf(t, `rm -rf a`).Values(command.KindCommand); got != nil {
		t.Errorf("Values(command) = %v, want nothing", got)
	}
}

// COVERS: FR-6.8 | property
func TestAFaultSaysWhichArgumentAndWhy(t *testing.T) {
	t.Parallel()

	options := decomposeOf(t, `rm -q x`)
	if len(options.Faults) == 0 {
		t.Fatal("no fault for an undeclared option")
	}

	// A denial that names only its rule cannot be checked by the reader.
	message := options.Faults[0].Error()
	for _, want := range []string{"-q", "argument 1"} {
		if !strings.Contains(message, want) {
			t.Errorf("fault %q does not mention %q", message, want)
		}
	}
}

// COVERS: FR-4.16 | negative
func TestACommandWithNoDeclarationIsRefused(t *testing.T) {
	t.Parallel()

	_, err := command.Decompose(outerSimple(t, `wibble --force`), table())

	if !errors.Is(err, command.ErrUndeclaredCommand) {
		t.Errorf("Decompose = %v, want %v", err, command.ErrUndeclaredCommand)
	}
}

// COVERS: FR-5.7, FR-6.7 | negative
func TestAWordThatIsNotFixedByItsTextIsAFault(t *testing.T) {
	t.Parallel()

	options := decomposeOf(t, `rm $HOME`)

	if len(options.Faults) == 0 {
		t.Fatal("an unexpanded word produced no fault")
	}
	if !errors.Is(options.Faults[0], command.ErrUndeterminedWord) {
		t.Errorf("fault = %v, want %v", options.Faults[0], command.ErrUndeterminedWord)
	}
}

// COVERS: FR-6.1 | positive
func TestOperandsAreReportedByOrdinal(t *testing.T) {
	t.Parallel()

	// rm -rf a b  ->  0:rm 1:-rf 2:a 3:b
	options := decomposeOf(t, `rm -rf a b`)

	if want := []int{2, 3}; !slices.Equal(options.Ordinals(), want) {
		t.Errorf("Operands = %v, want %v", options.Operands, want)
	}
	if !options.Has("force") || !options.Has("recursive") {
		t.Error("the bundle did not decompose into both of its options")
	}
}
