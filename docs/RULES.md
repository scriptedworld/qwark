# The rules

What a rule is made of, what a declaration adds, and what the shipped set holds.
For the shapes a rule can be written in and which one fits a given intent, read
`docs/PATTERNS/the-mechanicals-the-shapes-a-rule-can-be-written-in.md`.

## A rule

An id, an action, a reason and clauses. **All clauses must hold** for the rule to
apply. There is no `or` inside a rule, so alternatives are separate rules and
each can be checked by reading it alone.

    [[rule]]
    id     = "no-skipping-hooks"
    action = "deny"
    reason = """
    `--no-verify` skips the hooks, which are the checks that run without being
    asked for and are therefore the ones a mistake cannot route around.
    If a hook is wrong often enough to be worth skipping, fix the hook."""

      [[rule.clause]]
      option = "no-verify"

That clause names no command. It holds wherever the declaration table says some
option means `no-verify`, so one rule covers every command that has such an
option.

**The reason is not a comment.** It is what a refused agent is shown, and it is
the only thing standing between a denial and a session that does not understand
why. Write it for the reader who has just hit it.

## The four actions

    allow   auto-approve; the command runs without a prompt
    deny    block it
    ask     force the normal permission prompt
    tag     decide nothing; attach a name to the evaluation

`ask` hands the decision to the person at the terminal. It fits a command
legitimate often enough that refusing it outright would be wrong, but costly
enough that nobody should run it without looking. Under an unattended agent an
ask does not proceed on its own.

An `allow` verdict replaces the normal permission flow rather than passing to
it, so the prompt Claude Code would otherwise raise does not appear.

`tag` is what lets the other three compose: a cheap structural rule annotates a
command, and an expensive rule is written over the name instead of walking the
tree again. Tags carry lifetimes and are deferred;
`docs/DECISIONS/tags-have-lifetimes.md` says why, and `rules/40-state.toml` is
the worked example.

## Clauses

A clause selects part of the call and then tests it.

    nodes, flags, ops, fact    over the parse tree
    index, option, kind        over the command's words
    tag                        over state
    agent, cwd                 over the request

    value, partial, pattern, group    the tests
    reading = "interpreted" | "written"

`nodes`, `flags` and `ops` name the parser's own vocabulary, exactly what
`qwark ast` prints, and qwark refuses at load a clause naming one the parser
does not have. A maintained mapping of node type to fact can be silently
incomplete, and was.

`absent = true` inverts a clause, which is how a conditional refusal is written:
forbidden unless the signing option is given is one deny rule with a clause
saying the option is absent. Where a selector covers several positions a plain
clause holds if some satisfy it, so an inverted one holds when none do.

**Inversion is satisfied by absence, including absence qwark caused by not
understanding something.** That fails safe in a deny rule and is worth reading
twice in an allow rule.

The full vocabulary, with what each selector means and the values it takes,
is in the header of `rules/00-structure.toml`, beside the rules that use it.

## Declarations

A rule can only ask whether an option meaning force was given if something says
what `-f` means for that command. That is a declaration, and it is what lets one
clause cover `-f`, `-rf`, `--force` and `--f` while not matching `tar -f`.

    [command.rm]
    operands = "path"

      short.f = { means = "force" }
      short.r = { means = "recursive" }
      short.R = { means = "recursive" }

      long.force     = { means = "force" }
      long.recursive = { means = "recursive" }

`operands` says what the command's non-option words denote, which is what lets a
path rule find the paths in `rm a b c` without knowing what `rm` is. A long
option may be abbreviated to any unambiguous prefix, so `--f` resolves to
`--force` here, and an abbreviation matching two declared names is refused
rather than picked between.

A declaration grants understanding, not permission. Nothing runs because it was
declared; deny is still the default and an allow rule still has to match.

**It is still the eligibility surface, and that makes the declaration file the
most dangerous one in a rule set.** An undeclared command is refused outright,
so adding a declaration moves a command from refused-because-unaccountable to
eligible-and-now-decided-by-rules. `[command.sh]` is a complete bypass written
as configuration.

Two switches govern how strict this is, and one does not imply the other.

    [declarations]
    required  = true   an undeclared command is refused
    accounted = true   an option the declaration does not name is refused

`required` covers the command word and `accounted` covers the options a declared
command carries, so both are said or an absence somewhere else changes what the
policy means. Declaring a command means declaring every option it is used with.
Short options bundle, so `git log --oneline -10` decomposes to `-1` and `-0` and
needs both. The unit of work is a command plus its option set, and twenty
commands is not twenty lines.

Leaving an option out is not a gap. It is refused, so an allowance stays narrow
by construction and forgetting a dangerous flag costs a refusal rather than a
hole.

**A declaration is per command word, and git's options are per subcommand.**
There is one `[command.git]` table consulted for every subcommand alike, so
declaring an option for the sake of one declares it for all of them. `git diff
-w` ignores whitespace and `git help -w` opens a browser. An option belongs in
the table only when it means the same thing on every subcommand that can reach
it and is harmless on all of them.

Setting `required = false` is how a rule set judges by shape alone while the
policy is still being written. It is a single boolean with no middle ground on
purpose: a partial version, where some commands need declaring and others do
not, is a list of exceptions nobody can read as a policy.

## The shipped set

`rules/` holds seven files, layered by what a check costs rather than by
precedence, since precedence is fixed by strictness. Run
`qwark rules rules/` for the counts.

`00-structure.toml` refuses by shape and needs no declarations to do it: command
substitution, globs, redirections, pipes, logical concatenation, here-documents,
sequences, backgrounding, subshells, loops and function definitions, `time`,
coprocesses, arithmetic commands, and setting a variable name. Everything in it
is about what a command line can be made to mean rather than what any particular
command does.

`05-declarations.toml` is the table: eighteen commands and their options, with
the deliberate omissions listed and their reasons, so nobody adds one later
assuming it was forgotten.

`06-allow.toml` carries the permission a deny-by-default engine needs, and the
declaration switches, both currently off.

`10-commands.toml` classifies commands by what they do to the world, including
git split into a read-only allowance and six denied classes. Each class carries
its own reason and classes are expected to overlap, so a subcommand in three of
them is refused with all three reasons stated.

`20-paths.toml` protects control surfaces: qwark's own rules and state, Claude
Code's configuration, shell startup files, git hooks, task definitions, and
directories on `PATH`.

`30-options.toml` refuses by option, which is where force, recursion and
`--no-verify` live.

`40-state.toml` is the worked example of the tag shape.

**A deployment names which files it loads, and the shipped set is not obliged to
be the loaded set.** The whole directory and the two structural files answer
differently, which is the point of the split:

    $ qwark judge rules -- python3 -c "import os"
    deny
      no-interpreters                    This runs code supplied as an argument, …

    $ qwark judge rules/00-structure.toml rules/06-allow.toml -- python3 -c "import os"
    allow
      allow-a-single-plain-command       A command whose effect is fixed by its own text …

Loading the structural pair alone is observation rather than containment. Any
command word runs if its shape is clean, interpreters and task runners included,
and what they go on to execute is invisible to every rule. It is a deliberate
first phase: a rule set that refuses undeclared commands refused roughly two
thirds of what real sessions run, before anything existed to replace them.

## Trying a rule before trusting it

`judge` takes the same rule paths the hook does and the same fields a request
carries, so a rule can be exercised as the caller that will meet it.

    qwark judge --agent=test-runner rules -- go test ./...
    qwark judge --cwd=/srv/project rules -- bolt check .

Both options come before the rule paths, and only leading occurrences are taken.
Everything after the rules path may be the command being judged, and a gate that
ate an argument out of the middle of a command would be judging something other
than what was typed.

`--agent` defaults to the empty agent, which is the main session rather than a
missing value: a main-session call carries no `agent_type`, so judging with no
option already exercises the caller every session has.

`--cwd` has no default. Every real call carries a working directory, so a rule
with a `cwd` clause is inert until one is named, and a rule tried without it
looks like a rule that does not work. Nothing defaults it to the process's own
directory: where qwark happens to be standing is not where the call came from.
