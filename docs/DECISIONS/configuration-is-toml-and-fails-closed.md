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

    qwark hook <dir holding `[[rule]` >
      deny  qwark's rule set will not load, so nothing is permitted:
              rule file is not valid TOML: …/00.toml:
              toml: error: expected end of table array name delimiter ']'

## No rule fired and no rule ran are the same silence, and here neither is silent

The failure this guards against is worth stating in the general form, because it
is what a rules-driven gate fails at rather than something particular to TOML.

A gate that found nothing and a gate that never looked produce the same output.
Measured elsewhere in this estate: one malformed rule turned thirteen deny cases
into passes in an `ast-grep` ruleset, and the harness scored it clean because it
read stdout and ignored an exit status of 8. The ruleset had stopped gating and
nothing in the output said so.

**Deny by default is what makes that shape unreachable here**, and it covers the
half an unparseable-file check does not: a rule set that parses and matches
nothing.

    qwark judge <dir holding a valid file with no rules> -- ls -la
      deny  (engine) declared commands only
            (engine) deny by default
              Nothing permitted this. Being allowed means an allow rule
              matched, and none did.

Being allowed *means* an allow rule matched, so an empty policy permits nothing
and a ruleset that silently stopped matching denies everything rather than
approving everything. The two silences are both refusals, which is the only
arrangement where forgetting to check costs a false denial rather than a false
allow.

The general rule for a gate without that property: assert the rules loaded before
trusting a verdict, and the strongest form is to run the set against a
known-violating input at startup and require a finding, which also catches a file
that parses and matches nothing. It is the same family as reading the artifact
rather than the exit status, pointing the other way. That one says a zero exit
does not mean the work was good; this one says an empty result does not mean the
work was done.
