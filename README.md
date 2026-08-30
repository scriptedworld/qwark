# qwark

**A gate that decides which shell commands a coding agent is allowed to run.**

Coding agents run shell commands on your machine, and the prompt asking whether
`rm` may run tells you almost nothing about what it is going to remove. qwark
sits in front of that. It parses the proposed command the way a shell would,
judges it against rules you write, and answers allow, ask or deny with a reason
the agent can read.

    $ qwark judge rules -- rm -rf /
    deny
      rm-force                           `rm` with force suppresses every check it has: …
                                         caused by: rm
      rm-recursive                       Recursive removal is not permitted. …
                                         caused by: rm
      no-forcing-anything                Forcing is not permitted. …
                                         caused by: -f

Every rule that objected is listed, each with the word that set it off. The
reasons are elided above and printed in full: a reason is not a comment, it is
what a refused agent is shown, and it is the only thing between a denial and a
session that does not understand why.

It runs as a `PreToolUse` hook for Claude Code, which hands it the tool call on
stdin before the command executes and lets its answer decide whether it does.

## What it is for

qwark is a containment tool and its subject is an agent: a process that may be
careless, may be wrong about what a command does, and in the limit may be
working against the person who ran it.

So when qwark cannot account for something, it refuses. An unparseable command
is denied. A rule file that will not load means no Bash at all. An undeclared
command is denied, and so is an option the declaration does not name. A word
whose value is not fixed by its own text is refused rather than guessed at. A
gate that degrades to permissive whenever it is confused is a gate whose
confusion is the way through it.

Matching text would not do. `rm -rf /` and `env rm -rf /` and
`PATH=. rm -rf /` are three different programs behind one string, so qwark
parses with the same grammar bash uses and asks its questions of the tree.
`docs/DECISIONS/why-a-parser-rather-than-a-matcher.md` carries the argument.

## How a decision is reached

Deny is the default. A command is refused unless an allow rule matched it, so a
rule set containing no allow rules permits nothing, which is the correct reading
of an empty policy.

A rule is an id, an action, a reason and clauses, and **all of its clauses must
hold** for it to apply. There is no `or` inside a rule, so alternatives are
separate rules and each one can be checked by reading it alone.

When several rules match, the strictest action wins: deny beats ask beats allow.
Order never changes a verdict and no rule is weakened by which file it arrived
in, so the files can be split by what a check costs rather than by precedence.
There is no overridable deny. An exception is written inside the rule it
modifies, where a reader of that rule sees it.

Evaluation continues after a denial, so one refusal lists everything that was
wrong instead of sending its reader round three times. An action qwark does not
recognise is treated as a denial.

## Try it

    go build -o bin/qwark ./cmd/qwark

    bin/qwark judge rules -- git push --force origin main
    bin/qwark ast "rm -rf ./build && echo done"
    bin/qwark facts "ls | wc -l"

`judge` takes the same rule paths and the same request fields the hook does, so
a rule can be exercised as the caller that will meet it before it is the reason
something failed. A rule set that has never judged anything is a policy nobody
has run.

    qwark ast [--debug] [command]   outline the syntax tree of a command
    qwark facts [command]           list the properties a command establishes
    qwark rules PATH...             load rule files and report what they hold
    qwark judge [--agent=T] [--cwd=DIR] RULES COMMAND...
    qwark hook RULES...             run as the PreToolUse hook
    qwark help

`ast` and `facts` read from stdin when given no command argument, and `facts`
prints nothing for a command that establishes none. Several rule paths may be
given, a directory contributes every `.toml` file in it and nothing else, and
with `judge` everything before `--` is a rule path.

## Installing it as a hook

Build the binary, copy the rule set somewhere the registration can name, and
merge `install/settings-fragment.json` into `settings.json`.

    go build -o bin/qwark ./cmd/qwark
    install -d ~/.config/qwark/rules
    install -m 0644 rules/*.toml ~/.config/qwark/rules/

Copying the shipped rules rather than pointing at this tree is deliberate: the
live policy and the policy under development are then separate things. Nothing
compares the two for you.

Two things in that fragment carry weight, and `docs/INSTALLING.md` explains both
before you run anything. `|| exit 2` is the only fail-closed exit, because
Claude Code lets the command proceed on every status except 2. The
`permissions.deny` list is the other half of the control, because Write and Edit
reach the paths a rule protects without passing through qwark at all.

**A change to a registered hook takes effect on the very next command, with no
restart.** If a rule set wedges the tree, move the settings file aside, which
needs no shell and no root.

## What it does not do

**qwark gates Bash and nothing else.** The Write and Edit tools reach every path
a rule protects, which is what the `permissions.deny` list exists for. That list
enumerates paths across a space of paths that is effectively infinite, so it is
wrong wherever it is incomplete, and keeping it in step with the rules is
manual.

A coding agent that can write files and run its tests has arbitrary execution
regardless of qwark. `go test` runs code the agent wrote a moment ago, and
`just`, `make` and `npm run` execute recipes from files in the tree. What qwark
constrains is what is typed, not what the typed thing goes on to execute. The
shipped rules deny interpreters and task runners by name, which narrows the
problem without solving it.
`docs/DECISIONS/what-qwark-does-not-cover.md` is the full statement of the
limits.

qwark never answers `defer`, because deciding nothing is the outcome this design
exists to prevent, and it never rewrites the call it was asked about: a gate that
edits what it judges can no longer be said to have judged it.

The registration is fixed for a session. A subagent spawned inside a running
session inherits its parent's command line, so varying policy by role is the
engine's job through the `agent` clause rather than the launcher's.

## What is not finished

There is no tagged release, so nothing here is a stable interface yet, and the
rule schema is still moving.

Rules can name the agent a request came from, and no shipped rule does, because
which agent types exist is not qwark's to invent. `tag` and the state it implies
are deferred: `rules/40-state.toml` is a worked example rather than a loaded
file, and nothing writes tag state today.

This is the first of a planned three layers, and it is the outermost one. A
sandbox and a per-task manifest of writable files are what close the gap this
one cannot, which is a fixed command line whose meaning lives in a file the
agent may legitimately write.
`docs/DECISIONS/the-end-state-is-three-layers.md` sets out where that goes, and
`NEXT_STEPS.md` is the working detail.

## Documentation

    docs/INSTALLING.md    deploying it, registering the hook, and the log
    docs/RULES.md         what a rule is, the declarations, the shipped set
    docs/PROJECT.md       the layout and how the repository is gated
    docs/DECISIONS/       why it is built this way, one topic per file
    docs/LESSONS/         what the shell turned out to do
    REQUIREMENTS.md       what must be true
    CONTRIBUTING.md       building, testing, and what a change has to carry

The full clause vocabulary lives in the header of `rules/00-structure.toml`,
beside the rules that use it, and
`docs/PATTERNS/the-mechanicals-the-shapes-a-rule-can-be-written-in.md` covers
the shapes a rule takes and which one fits a given intent.

## Building

    just build        the binary
    just test         the suite
    just checks       quality, tests, coverage, secrets
    just --list       the whole interface

`go build -o bin/qwark ./cmd/qwark` and `go test ./...` are the underlying
commands if you would rather not use `just`. CONTRIBUTING.md has the rest.

## Licence

Apache 2.0. See `LICENSE`, and `NOTICE` for the copyright.
