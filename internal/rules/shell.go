package rules

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/scriptedworld/qwark/internal/reach"
)

// Reasons the shell cannot be accepted. Each denies every command, not just
// one, because the parser's grammar is a precondition of every verdict: if the
// language is wrong, no answer qwark gives about any command means anything.
var (
	ErrShellUndeclared = errors.New("no rule file declares which shells are permitted")
	ErrShellRelative   = errors.New("a permitted shell is not an absolute path")
	ErrShellUnreported = errors.New("the environment does not say which shell will run")
	ErrShellMismatch   = errors.New("the shell that will run is not a permitted one")
)

// A ShellPolicy is the set of shells a rule set permits, declared in TOML:
//
//	[shell]
//	allow = ["/bin/bash", "/usr/bin/bash"]
//
// It is a declaration and not a rule because it does not depend on the
// command. Written as a clause it would be evaluated once per command to
// produce the same answer, and, more importantly, a rule file that simply
// omitted it would disable the check silently. So absence is a refusal here,
// as it is everywhere else in the rule set.
//
// # Whole paths, not names
//
// Entries are absolute paths and are compared exactly. Comparing basenames
// would be friendlier to a machine that installs the shell somewhere unusual,
// and it would accept any file named bash anywhere, including one written
// into a directory the agent can reach. For a gate whose subject can create
// files, "it is called bash" is not a property worth checking.
//
// On a merged-/usr system `/bin` is a symlink to `usr/bin`, so `/bin/bash`
// and `/usr/bin/bash` are the same file, both resolving to `/usr/bin/bash`
// and both root-owned with mode 755. Either entry alone would accept both
// spellings, so listing both only makes the list readable without resolving.
type ShellPolicy struct {
	Allow []string `toml:"allow"`
}

// Verify reports whether the shell that will run commands is a permitted one.
//
// # Why this exists
//
// qwark's parser is fixed to one shell's grammar, and reading a command in the
// wrong grammar does not fail loudly. Of ten zsh constructs put through the
// bash parser, it rejects only two, and four parse cleanly while meaning
// something else: `**/`, `*(.)`, `$foo[2]`, and the `noglob` precommand
// modifier. Worse, `rm *(e:'rm -rf /':)` carries no substitution, pipe,
// redirection or logical concatenation, so it satisfies every tier-one rule,
// and zsh executes the quoted code as a glob qualifier.
//
// # What this is not
//
// This is a consistency check on the best available signal, not proof, and a
// passing check is not a guarantee. qwark runs as a child process of the tool
// that will spawn the shell, so it cannot observe that shell directly: `type`
// answers only from inside it, and a child sees the un-aliased view. The
// signal is what the environment reports, and it is trustworthy only to the
// extent that the environment is: whatever names it must be unwritable, exactly
// as the rule files must be.
//
// Both sides are resolved through their symbolic links first, so two spellings
// of one file reach one answer and a replaced link does not slip past on the
// strength of its name.
func (p ShellPolicy) Verify(reported string) error {
	if len(p.Allow) == 0 {
		return ErrShellUndeclared
	}
	for _, permitted := range p.Allow {
		if !filepath.IsAbs(permitted) {
			return fmt.Errorf("%w: %q", ErrShellRelative, permitted)
		}
	}

	running := strings.TrimSpace(reported)
	if running == "" {
		return fmt.Errorf("%w; permitted: %s", ErrShellUnreported, p.list())
	}

	// Both sides are resolved through their symbolic links before comparing.
	// On a merged-/usr system `/bin/bash` and `/usr/bin/bash` are one file
	// under two names, and comparing names would accept only one of them.
	//
	// It also closes what an exact comparison could not: if a permitted path
	// were itself replaced by a link to something else, the text would still
	// match while the program would not, and resolving catches that.
	target := reach.Resolve(running)
	for _, permitted := range p.Allow {
		if reach.Resolve(permitted) == target {
			return nil
		}
	}

	return fmt.Errorf("%w: %q will run, permitted: %s",
		ErrShellMismatch, running, p.list())
}

// list renders the permitted shells for a message. A shell mismatch refuses
// every command, so the refusal has to say which shells would be accepted.
func (p ShellPolicy) list() string {
	return strings.Join(p.Allow, ", ")
}
