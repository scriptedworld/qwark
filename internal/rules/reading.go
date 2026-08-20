package rules

import (
	"errors"
	"fmt"

	"github.com/scriptedworld/qwark/internal/command"
)

// ErrReading reports a reading a rule file names that does not exist.
var ErrReading = errors.New("not a reading")

// A Reading is which form of a word a clause tests.
//
// A word has two readings and they are not the same string. `\ls` is written
// with an escape and runs `ls`; `a\ b` is written with an escape and names the
// file `a b`. Which one a clause means has to be said, because the answer
// differs and the difference is exploitable in one direction.
type Reading string

// The readings a clause may name.
const (
	// ReadingInterpreted is what the shell will actually pass, with quoting and
	// escapes resolved. It is the default, and almost always what a rule
	// wants: it is what the command will do.
	ReadingInterpreted Reading = "interpreted"

	// ReadingWritten is the source as written, quoting and escapes intact. It
	// is for rules about *how* something was spelled rather than what it means
	// -- which is a real thing to want, and a dangerous default.
	ReadingWritten Reading = "written"
)

// ParseReading reads the reading named in a rule file. An empty name is the
// default, so a clause that does not care does not have to say so.
func ParseReading(name string) (Reading, error) {
	switch Reading(name) {
	case "":
		return ReadingInterpreted, nil
	case ReadingInterpreted:
		return ReadingInterpreted, nil
	case ReadingWritten:
		return ReadingWritten, nil
	default:
		return "", fmt.Errorf("%q: %w", name, ErrReading)
	}
}

// Of returns the reading of a word, and whether that reading exists.
//
// The interpreted value of a word containing a substitution does not exist --
// nothing is expanded, so there is no answer -- and a clause testing it does
// not match rather than matching an empty string, which a partial or a `.*`
// pattern otherwise would. A clause reading what was written always has
// something to read.
func (r Reading) Of(word command.Word) (string, bool) {
	if r == ReadingWritten {
		return word.Text, true
	}
	if !word.Determined {
		return "", false
	}
	return word.Value, true
}
