# Configuration

TOML, in multiple rule files that are aggregated. `github.com/BurntSushi/toml` is
already the house library, used by `dotfiles/go/internal/manifest`, and its
`ParseError.ErrorWithPosition` renders a source excerpt with line and column,
which this design needs:

**If any rule file is unparseable, Bash is unusable.** Fail-closed. A gate that
degrades to permissive when its own configuration is broken is a gate that
reports success while guarding nothing.

The cost is that a typo denies every Bash command until it is fixed, so the
denial has to name the file, the line and the text, and the escape route must not
itself require Bash. Editing the rule file with the Edit tool does not.
