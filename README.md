# qwark

**A gate that decides which shell commands a coding agent is allowed to run.**

Coding agents run shell commands on your machine. Most of the time that is the
point of them, and the permission prompt asking whether `rm` may run tells you
almost nothing about what it is going to remove. qwark sits in front of that: it
parses the proposed command the way a shell would, judges it against rules you
write, and answers allow, ask or deny with a reason the agent can read.

It runs as a `PreToolUse` hook for Claude Code, which means it is handed the
tool call on stdin before the command executes and its answer decides whether it
does.

    $ qwark judge rules -- git push --force origin main
    deny
      no-git-hook-running    This git subcommand runs hooks from .git/hooks, or a
                             filter or merge driver named in .git/config, all of
                             them files that can be written with no shell
                             involved. Reading the repository does not.
                             caused by: git
      no-git-reaching-the-network
                             This git subcommand reaches a machine that is not
                             this one, publishing what is here, or fetching what
                             will then be trusted here. Neither is a thing to do
                             without somebody having decided to.
                             caused by: git

A refusal lists every rule that objected, each with the reason that rule carries
and the text that set it off.

## What it is for

qwark is a containment tool and its subject is an agent: a process that may be
careless, may be wrong about what a command does, and in the limit may be
working against the person who ran it.

So when qwark cannot account for something, it refuses. An unparseable command
is denied. A rule file that will not load means no Bash at all. An undeclared
command is denied, and so is an option the declaration does not name. A word
whose value is not fixed by its own text is refused rather than guessed at.

A gate that degrades to permissive whenever it is confused is a gate whose
confusion is the way through it.

## Installing

Build the binary, put a rule set somewhere the hook can name, and register it.

    go build -o bin/qwark ./cmd/qwark
    install -d ~/.config/qwark/rules
    install -m 0644 rules/*.toml ~/.config/qwark/rules/

The rules directory is an argument to `qwark hook`, so it can live anywhere the
registration can name and a deployment can keep several. Copying the shipped set
rather than pointing at the source tree is deliberate: the live policy and the
policy under development are then separate things, and changing one does not
change the other by accident. Nothing compares the two for you.

`install/settings-fragment.json` is the registration to merge into
`settings.json`, and it is commented at length because two things in it carry
weight.

The first is `|| exit 2`. Claude Code treats exit 0 with no JSON as no decision,
and every non-zero status other than 2 as a non-blocking error, so the command
proceeds in both cases. Only exit 2 blocks. qwark recovers from its own panics
and runs in a single goroutine so that it can, but nothing in-process catches a
fatal runtime error, an out-of-memory kill or a signal. The shell wrapper does.

    "command": "qwark hook /etc/qwark/rules || exit 2"

The second is `permissions.deny`. qwark gates Bash and nothing else, so the
Write and Edit tools reach the rule files, the shell snapshot, `.git/hooks` and
`settings.json` without passing through it. Every path a rule protects needs a
twin entry in the registration, or the rule is enforced against a shell and
against nothing at all. The fragment carries one representative of each class
the rule files protect, and the paths in it are this machine's; resolve symlinks
on the machine you install on rather than copying them.

**A change to an existing hook takes effect on the very next command, with no
restart.** If a rule set wedges the tree, move the settings file aside. That
needs no shell and no root.

## Commands

    qwark ast [--debug] [command]   outline the syntax tree of a command
    qwark facts [command]           list the properties a command establishes
    qwark rules PATH...             load rule files and report what they hold
    qwark judge [--agent=T] [--cwd=DIR] RULES COMMAND...
    qwark hook RULES...             run as the PreToolUse hook
    qwark help

`ast` and `facts` read from stdin when given no command argument. Everything
except `hook` exists so a rule can be tried before it is the reason a command
failed, and `judge` is the one to reach for: a rule set that has never judged
anything is a policy nobody has run.

Several rule paths may be given. A directory contributes every `.toml` file in
it and nothing else. With `judge`, everything before `--` is a rule path.

    qwark judge rules/00-structure.toml rules/10-commands.toml -- git push

## How a decision is reached

Deny is the engine's default. A command is refused unless an allow rule matched
it, so a rule set containing no allow rules permits nothing, which is the
correct reading of an empty policy.

**When several rules match, the strictest action wins.** Deny beats ask beats
allow. Order never changes a verdict and no rule can be weakened by which file
it arrived in, which is why the files can be split by cost rather than by
precedence. There is no overridable deny: an exception is written inside the
rule it modifies, where a reader of that rule sees it, rather than in a second
rule that outranks the first.

Evaluation continues after a denial, so a refusal lists everything that was
wrong instead of sending its reader round three times. A denied command has no
effect of any kind, so a rule that would have set state does not.

`ask` hands the decision to the person at the terminal, and it is the right
action where the command is legitimate often enough that refusing it outright
would be wrong, but costly enough that nobody should run it without looking.
Under an unattended agent an ask does not proceed on its own.

An action qwark does not recognise is treated as a denial. Permitting on the
strength of a verdict nothing understood is the one direction that cannot be
taken back.

## Writing a rule

A rule is an id, an action, a reason and clauses. **All clauses must hold** for
the rule to apply. There is no `or` inside a rule, so alternatives are separate
rules and each can be checked by reading it alone.

    [[rule]]
    id     = "no-skipping-hooks"
    action = "deny"
    reason = """
    `--no-verify` skips the hooks, which are the checks that run without being
    asked for and are therefore the ones a mistake cannot route around.
    If a hook is wrong often enough to be worth skipping, fix the hook."""

      [[rule.clause]]
      option = "no-verify"

The clause names no command. It holds wherever the declaration table says some
option means `no-verify`, so one rule covers every command that has such an
option, and a command whose table does not name one is refused a step earlier
for carrying a word nothing accounts for.

The reason is not a comment. It is what a refused agent is shown, and it is the
only thing standing between a denial and a session that does not understand why.
Write it for the reader who just hit it.

A clause selects part of the call and then tests it. The selectors are `nodes`,
`flags`, `ops` and `fact` over the parse tree; `index`, `option` and `kind` over
the command's words; `tag` over state; and `agent` and `cwd` over the request.
The tests are `value`, `partial`, `pattern` and `group`.

`nodes`, `flags` and `ops` name the parser's own vocabulary, exactly what
`qwark ast` prints, and qwark refuses at load a clause naming one the parser
does not have. A maintained mapping of node type to fact can be silently
incomplete, and was.

`absent = true` inverts a clause, which is how a conditional refusal is written:
"this is forbidden unless the signing option is given" is one deny rule with a
clause saying the option is absent. Where a selector covers several positions a
plain clause holds if some satisfy it, so an inverted one holds when none do.
Inversion is satisfied by absence, including absence qwark caused by not
understanding something, so it fails safe in a deny rule and is worth reading
twice in an allow rule.

The full clause vocabulary is documented in the header of
`rules/00-structure.toml`, beside the rules that use it.
`docs/PATTERNS/the-mechanicals-the-shapes-a-rule-can-be-written-in.md` covers
the shapes a rule takes and which one fits a given intent.

## Declarations

A rule can only ask "was an option meaning force given" if something says what
`-f` means for that command. That is a declaration, and it is what lets one
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

Two switches govern how strict this is, and one does not imply the other.

    [declarations]
    required  = true   an undeclared command is refused
    accounted = true   an option the declaration does not name is refused

Declaring a command means declaring every option it is used with, because
`accounted` refuses the rest. Short options bundle, so `git log --oneline -10`
decomposes to `-1` and `-0` and needs both. The unit of work is a command plus
its option set, and twenty commands is not twenty lines.

Setting `required = false` is how a rule set judges by shape alone while the
policy is still being written. It is a single boolean with no middle ground on
purpose: a partial version, where some commands need declaring and others do
not, is a list of exceptions nobody can read as a policy.

## The shipped rules

`rules/` holds the policy in seven files, layered by what a check costs rather
than by precedence, since precedence is fixed by strictness.

`00-structure.toml` refuses by shape and needs no declarations to do it: command
substitution, globs, redirections, pipes, logical concatenation, here-documents,
sequences, backgrounding, subshells, loops and function definitions, `time`,
coprocesses, arithmetic commands, and setting a variable name. Everything here
is about what a command line can be made to mean rather than what any particular
command does, which is why it is first and why it holds without a table.

`05-declarations.toml` is the table: eighteen commands and their options.
`06-allow.toml` carries the permission a deny-by-default engine needs and the
declaration switches. `10-commands.toml` classifies commands by what they do to
the world, including git split into a read-only allowance and six denied
classes, each carrying its own reason, so a subcommand in more than one is
refused with every reason given. `20-paths.toml` protects control surfaces: qwark's
own rules and state, Claude Code's configuration, shell startup files, git
hooks, task definitions, and directories on `PATH`. `30-options.toml` refuses by
option, which is where force and recursion and `--no-verify` live.
`40-state.toml` is a worked example of the tag shape and is not loaded.

A deployment names which files it loads, and the shipped set is not obliged to
be the loaded set.

## Trying a rule before trusting it

`judge` takes the same rule paths the hook does and the same fields a request
carries, so a rule can be exercised as the caller that will meet it.

    qwark judge --agent=test-runner rules -- go test ./...
    qwark judge --cwd=/srv/project rules -- bolt check .

Both options come before the rule paths. Only leading occurrences are taken:
everything after the rules path may be the command being judged, and a gate that
ate an argument out of the middle of a command would be judging something other
than what was typed.

`--agent` defaults to the empty agent, which is the main session rather than a
missing value: a main-session call carries no `agent_type`, so judging with no
option already exercises the caller every session has.

`--cwd` has no default, and that differs for a reason. Every real call carries a
working directory, so a rule with a `cwd` clause is inert until one is named,
and a rule tried without it looks like a rule that does not work. Nothing
defaults it to the process's own directory: where qwark happens to be standing
is not where the call came from.

## The log

Every decision is appended to `$XDG_STATE_HOME/qwark/decisions.jsonl`, falling
back to `~/.local/state/qwark/decisions.jsonl`. One JSON object per line, holding
the verdict, the rules that fired, the command, the agent, the working directory,
the session, and a digest identifying the rule set that judged it. The digest is
what lets entries made under different policies be told apart rather than
compared with each other.

It is opened for append and never truncated, because a log with earlier entries
missing reads as a clean history and answers "what happened" confidently and
wrongly.

**A failure to record does not change a verdict.** The decision is made before it
reaches the log, and refusing when the log is unwritable would turn a full disk
into a way of stopping every command. That is the permissive direction, chosen
deliberately: somebody who can fill the disk can stop the recording without
stopping the commands.

## What it does not do

**qwark gates Bash and nothing else.** The Write and Edit tools reach every path
a rule protects without passing through it, which is what the `permissions.deny`
list in the registration is for. That list enumerates paths that must not be
reached across a space of paths that is effectively infinite, so it is wrong
wherever it is incomplete, and keeping it in step with the rules is manual. It
is the interim rather than the destination.

A coding agent that can write files and run its tests has arbitrary execution
regardless of qwark. `go test` runs code the agent wrote a moment ago. What
qwark constrains is what is typed, not what the typed thing goes on to execute,
and no rule about command lines closes that gap. The rules deny interpreters and
task runners by name for this reason, which narrows the problem without solving
it.

qwark never returns `defer`. A hook may decline to decide and let the dispatcher
carry on past it, and deciding nothing is the outcome this design exists to
prevent. It also never returns `updatedInput`, which would let a hook rewrite
the call it was asked about: a gate that edits what it judges can no longer be
said to have judged it.

The registration is fixed for a session. An external process can choose rule
files when it launches one, but a subagent spawned inside a running session
inherits its parent's command line, so varying policy per subagent is the
engine's job through the `agent` clause rather than the launcher's.

## Building and testing

    go build -o bin/qwark ./cmd/qwark
    go test ./...

Tests live in an external test package and exercise the public API. Each names
the requirement it discharges in a comment directly above it, and the gate fails
a test that cites nothing or cites a requirement `REQUIREMENTS.md` does not
define.

    // COVERS: FR-4.4 | property

`docs/PROJECT.md` describes the layout and how the repository is gated.
`REQUIREMENTS.md` states what must be true, `docs/DECISIONS/` says why, and
`NEXT_STEPS.md` carries what is not done.

## Licence

Apache 2.0. See `LICENSE`, and `NOTICE` for the copyright.
