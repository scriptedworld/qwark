package command

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Problems a command's options can present. Each is a denial: a command whose
// options qwark cannot account for is one it cannot judge, and judging it
// anyway would mean acting on a table known to be wrong.
var (
	ErrUndeclaredCommand = errors.New("command has no declared options")
	ErrUndeclaredOption  = errors.New("option is not declared for this command")
	ErrAmbiguousOption   = errors.New("abbreviation matches more than one option")
	ErrMissingValue      = errors.New("option takes a value and none followed")
	ErrUndeterminedWord  = errors.New("word is not fixed by its text")
)

// A Kind is what a word denotes, which a rule about *where* something points
// needs before it can apply. `rm`'s operands are paths and `tar -f`'s value is
// one; the message in `git commit -m "…"` is not, and a path rule that treated
// it as one would be reading prose as a filename.
//
// KindCommand exists because a word naming another command is the shape that
// slips every rule about that command: in `env rm -rf x` the command word is
// `env`, so a rule about `rm` never fires.
type Kind string

// The declared kinds. A word with no declared kind is KindUnknown, and a rule
// that needs to know cannot apply to it.
const (
	KindUnknown Kind = ""
	KindPath    Kind = "path"
	KindText    Kind = "text"
	KindCommand Kind = "command"
)

// An Option is one option a command declares, as written in TOML:
//
//	[command.rm]
//	short.f = { means = "force" }
//	short.r = { means = "recursive" }
//	long.force = { means = "force" }
//	operands = "path"
//
//	[command.tar]
//	short.f = { means = "file", takes_value = true, kind = "path" }
//	operands = "path"
type Option struct {
	// Means is what the option does, and is what a rule matches on. `-f` is
	// force to rm and file to tar, so the meaning cannot be read off the
	// spelling and has to be declared.
	Means string `toml:"means"`

	// TakesValue reports whether the next word, or the remainder of a bundle,
	// belongs to this option rather than being an operand.
	TakesValue bool `toml:"takes_value"`

	// Kind is what the option's value denotes, where it takes one.
	Kind Kind `toml:"kind"`
}

// A Declaration is one command's options, keyed by name without the dashes.
type Declaration struct {
	Short map[string]Option `toml:"short"`
	Long  map[string]Option `toml:"long"`

	// Operands is what this command's non-option arguments denote. Declaring
	// it is what lets a rule ask where a command reaches.
	Operands Kind `toml:"operands"`
}

// A Table is every declared command.
type Table struct {
	Commands map[string]Declaration `toml:"command"`
}

// A Given is one option a command was actually given.
type Given struct {
	// Ordinal is the word the option was written in. Several bundled options
	// share one.
	Ordinal int

	// Spelling is the option as the author wrote it: `-f`, `--force`, `--f`.
	Spelling string

	// Name is the declared name it resolved to, without dashes.
	Name string

	// Means is the declared meaning, and is what a rule matches on.
	Means string

	// Long reports whether it was written in the long form.
	Long bool

	// Value is the option's value, where it takes one.
	Value string

	// Kind is what that value denotes.
	Kind Kind
}

// An OptionError is one thing about a command's options that could not be
// accounted for, with enough detail to say so in a denial.
type OptionError struct {
	Ordinal int
	Text    string
	Err     error
}

func (f OptionError) Error() string {
	return fmt.Sprintf("argument %d (%q): %v", f.Ordinal, f.Text, f.Err)
}

func (f OptionError) Unwrap() error { return f.Err }

// A Valued is something a command denotes, and where it was written.
//
// Both halves are needed and they are not the same word. In `tar -f archive.tar`
// the path is `archive.tar` but the ordinal is 1, where `-f` was written,
// because that is what a message should point at. A caller given only ordinals
// would read the word at 1, find `-f`, and compare *that* against the protected
// paths, which is a rule that can never fire.
type Valued struct {
	Ordinal int
	Value   string
}

// Options is what a command's words decomposed into.
type Options struct {
	Given    []Given
	Operands []Valued
	Faults   []OptionError

	// OperandKind is what this command's operands denote, from its
	// declaration.
	OperandKind Kind
}

// Ordinals returns just the positions, for a caller that wants to report rather
// than to test.
func (o Options) Ordinals() []int {
	found := make([]int, 0, len(o.Operands))
	for _, operand := range o.Operands {
		found = append(found, operand.Ordinal)
	}
	return found
}

// Values returns every word denoting the given kind, whether it arrived as an
// operand or as an option's value.
//
// This is what a rule about *where* a command reaches asks for. It deliberately
// does not include a word whose kind was never declared: a gate that guessed
// which arguments were paths would be back to reading the command as text.
func (o Options) Values(kind Kind) []Valued {
	if kind == KindUnknown {
		return nil
	}

	var found []Valued
	if o.OperandKind == kind {
		found = append(found, o.Operands...)
	}
	for _, given := range o.Given {
		if given.Kind == kind && given.Value != "" {
			found = append(found, Valued{Ordinal: given.Ordinal, Value: given.Value})
		}
	}

	sort.Slice(found, func(i, j int) bool { return found[i].Ordinal < found[j].Ordinal })
	return found
}

// Has reports whether an option with this declared meaning was given. This is
// what a rule asks: not "was `-f` written" but "was this command told to force",
// which `-rf`, `-f`, `--force` and `--f` all do.
func (o Options) Has(means string) bool {
	for _, given := range o.Given {
		if given.Means == means {
			return true
		}
	}
	return false
}

// Decompose splits a command's arguments into the options it was given and the
// operands it was given them about.
//
// It reports faults rather than stopping at the first, so a denial can name
// everything wrong with a command instead of sending its author round again.
func Decompose(simple Simple, table Table) (Options, error) {
	name := simple.Name()
	declared, ok := table.Commands[name]
	if !ok {
		return Options{}, fmt.Errorf("%q: %w", name, ErrUndeclaredCommand)
	}

	out := Options{OperandKind: declared.Operands}
	terminated := false

	for ordinal := 1; ordinal <= simple.Last(); ordinal++ {
		word, found := simple.At(ordinal)
		if !found {
			continue
		}
		if !word.Determined {
			out.fault(ordinal, word.Text, ErrUndeterminedWord)
			continue
		}
		if terminated || !isOption(word.Value) {
			out.Operands = append(out.Operands, Valued{Ordinal: ordinal, Value: word.Value})
			continue
		}
		if word.Value == terminator {
			terminated = true
			continue
		}
		ordinal = out.takeOption(simple, declared, ordinal, word.Value)
	}

	return out, nil
}

// terminator ends the options, so `rm -- -f` deletes a file called -f rather
// than forcing anything.
const terminator = "--"

// isOption reports whether a word is an option rather than an operand. A lone
// `-` is the conventional name for standard input and is an operand.
func isOption(value string) bool {
	return strings.HasPrefix(value, "-") && value != "-"
}

// takeOption records one option word and returns the ordinal consumed, which is
// one further along when the option took its value from the following word.
func (o *Options) takeOption(simple Simple, declared Declaration, ordinal int, value string) int {
	if strings.HasPrefix(value, terminator) {
		return o.takeLong(simple, declared, ordinal, value)
	}
	return o.takeShort(simple, declared, ordinal, value)
}

func (o *Options) takeLong(simple Simple, declared Declaration, ordinal int, value string) int {
	written, inline, hasInline := strings.Cut(strings.TrimPrefix(value, terminator), "=")

	name, option, err := resolveLong(declared.Long, written)
	if err != nil {
		o.fault(ordinal, value, err)
		return ordinal
	}

	given := Given{
		Ordinal:  ordinal,
		Spelling: terminator + written,
		Name:     name,
		Means:    option.Means,
		Long:     true,
		Value:    inline,
		Kind:     option.Kind,
	}

	if option.TakesValue && !hasInline {
		next, taken := o.takeValue(simple, ordinal, value)
		if !taken {
			return ordinal
		}
		given.Value = next
		ordinal++
	}

	o.Given = append(o.Given, given)
	return ordinal
}

// takeShort walks a bundle. `rm -rf` is two options in one word, which is why a
// matcher comparing arguments against `-f` never sees it.
func (o *Options) takeShort(simple Simple, declared Declaration, ordinal int, value string) int {
	bundle := strings.TrimPrefix(value, "-")

	for i, letter := range bundle {
		name := string(letter)
		option, ok := declared.Short[name]
		if !ok {
			o.fault(ordinal, "-"+name, ErrUndeclaredOption)
			continue
		}

		given := Given{
			Ordinal:  ordinal,
			Spelling: "-" + name,
			Name:     name,
			Means:    option.Means,
			Kind:     option.Kind,
		}
		if !option.TakesValue {
			o.Given = append(o.Given, given)
			continue
		}

		// The rest of the bundle is this option's value, as in `tar -fx.tar`;
		// if it ends the bundle, the value is the next word.
		if rest := bundle[i+len(name):]; rest != "" {
			given.Value = rest
			o.Given = append(o.Given, given)
			return ordinal
		}

		next, taken := o.takeValue(simple, ordinal, "-"+name)
		if !taken {
			return ordinal
		}
		given.Value = next
		o.Given = append(o.Given, given)
		return ordinal + 1
	}

	return ordinal
}

// takeValue reads the word after an option that requires one.
func (o *Options) takeValue(simple Simple, ordinal int, spelling string) (string, bool) {
	next, found := simple.At(ordinal + 1)
	if !found {
		o.fault(ordinal, spelling, ErrMissingValue)
		return "", false
	}
	if !next.Determined {
		o.fault(ordinal+1, next.Text, ErrUndeterminedWord)
		return "", false
	}
	return next.Value, true
}

// resolveLong finds the declared option a long form names.
//
// GNU accepts any unambiguous abbreviation, so `rm --f`,
// `--fo` and `--forc` all force. Matching the text `--force` misses every one
// of them, which is why this resolves rather than compares. An exact name wins
// outright, as it does in getopt_long, so declaring both `--force` and
// `--force-all` leaves `--force` meaning itself.
func resolveLong(declared map[string]Option, written string) (string, Option, error) {
	if option, ok := declared[written]; ok {
		return written, option, nil
	}

	var matched []string
	for name := range declared {
		if strings.HasPrefix(name, written) {
			matched = append(matched, name)
		}
	}
	sort.Strings(matched)

	switch len(matched) {
	case 0:
		return "", Option{}, fmt.Errorf("--%s: %w", written, ErrUndeclaredOption)
	case 1:
		return matched[0], declared[matched[0]], nil
	default:
		return "", Option{}, fmt.Errorf("--%s matches %s: %w",
			written, strings.Join(matched, ", "), ErrAmbiguousOption)
	}
}

func (o *Options) fault(ordinal int, text string, err error) {
	o.Faults = append(o.Faults, OptionError{Ordinal: ordinal, Text: text, Err: err})
}
