# qwark — why it is built this way

`REQUIREMENTS.md` states what must be true. This file says why, and is the
place a decision goes when the reasoning behind it would otherwise survive only
in a commit message.

Marks follow the silo convention: **FACT** measured, with the command that
produced it; **CLAIM** asserted or inferred; **PREFERENCE** chosen on taste.

---

## What qwark is

A `PreToolUse` hook for Claude Code. It reads the proposed tool call on stdin
and answers with a decision. The first mode gates the Bash tool; the shape is
meant to take other tools later.

**Owner, 2026-08-19:** *"remember this is ALL about controlling what an Agent
can run."*

That sentence decides most of what follows, so it is first. qwark is a
containment tool, and its subject is an agent — which is not a careful colleague
who occasionally mistypes, but a process that may be careless, may be wrong
about what a command does, and in the limit may be working against the person
who ran it. Every awkward choice recorded below is the same choice: **when qwark
cannot account for something, it refuses.**

- Unparseable command: denied.
- Unparseable rule file: no Bash at all.
- Undeclared command: denied.
- Undeclared option: denied.
- A word whose value is not fixed by its text: refused, never guessed at.

Each of those, read as ergonomics, looks needlessly harsh. Read as containment,
they are the only settings that mean anything: a gate that degrades to
permissive whenever it is confused is a gate whose confusion is the way through
it.

The corollary, and the owner's own answer to a worry raised here that declaring
every command would be a large job: **it is not.** *"you don't use that many
tools."* An agent's working set is small, and a table covering it is a morning's
work, not a project.

## Why a parser rather than a matcher

**FACT 2026-08-19**, read from `~/.projects/dotfiles`, commit `0e6ea10`: the
predecessor was `claude/hooks/archive-guard.sh`, a `grep -E` over the raw
command string, retired on 2026-08-14. Its own header records the failure,
demonstrated 2026-08-02:

> `bin/repos status` in the dotfiles repo walked from a configured root,
> reached the tree and enumerated it, and never contained the literal string —
> so this hook passed it.

**CLAIM.** That is not a bug in the regex, and no regex fixes it. The hook was
asked a question about *what the command would do* and answered a question
about *what the command said*. Those are the same question only for commands
whose effect is determined by their own text — and shell syntax exists largely
to break that correspondence.

So the gate has to work on structure. `mvdan.cc/sh/v3/syntax` gives a typed
tree, no cgo, and round-trips; tree-sitter-bash was the alternative and loses
on both counts.

**It still does not make the gate a guarantee.** The archive-guard header's
conclusion stands unchanged: a tool-layer gate cannot stop a program that names
a path only at runtime, because the deny happens before the child process
exists. qwark catches the explicit case early and says why. Where a real
boundary is needed, it belongs in the filesystem.

## The decision model

A rule's action is one of four:

    allow   auto-approve; the command runs without a prompt
    deny    block it
    ask     force the normal permission prompt
    tag     decide nothing; attach a name to the evaluation

`tag` is the one that makes the rest compose. A tag rule enriches the context
that later, more expensive rules match against, so cheap structural rules can
annotate and expensive rules can stay expressed over names rather than
re-deriving the tree.

## Rules are layered by cost

**PREFERENCE, owner, 2026-08-19.** Four classes, cheapest first, so a decision
that can be reached cheaply is:

1. **Node presence.** Certain nodes in the tree are an instant rule.
2. **Conjunctive.** Several elements, *all* of which must match.
3. **Context.** Whether the paths involved fall inside a given directory tree.
4. **State.** An ongoing tracker across commands.

Class 1 costs nothing per rule: every structural fact is gathered in a single
walk (`internal/shell/facts.go`), so rules consulting them are set lookups and
adding one is free. Cost is set by the size of the tree, not the rule count.

## Tier one: the command must say what it does

The first rule bans **redirections, substitutions, pipes, and logical
concatenation**. Two separate reasons, both the owner's, 2026-08-19:

- **Redirections buy nothing.** The tool call already returns stdout and stderr
  separately, so redirecting to capture output is redundant — the harness hands
  it back regardless. Banning it costs no capability and removes file
  truncation as a side effect.

- **Substitutions make the answer relative.** A command should always be clear
  as to what it is doing. A substitution makes the text a recipe whose result
  depends on runtime state, rather than a statement of what will happen.

**CLAIM, and the reason this tier comes first.** These four are one property,
not four bans: *the command's effect is determined by its own text.* Every tier
above depends on it. Deciding which paths a command reaches is unsound the
moment a `$(…)` can produce a path at runtime — which is precisely how
`bin/repos status` got past the predecessor. Tier one is not merely a danger
list; it is what makes tiers two and three decidable.

## Writing code or files through a here-document

**Requirement, owner, 2026-08-19.** Writing code or files with a here-document
is not allowed.

Tier one's ban on redirections already subsumes this, but it earns its own rule
because the reason is different, and reasons are what survive a policy being
rewritten. A redirection is banned for being redundant. A here-document write is
banned for going around the tools that make a change reviewable: content that
arrives through `cat > file <<'EOF'` was never a diff, was never held against
the file it replaced, and leaves nothing to inspect but the command that
produced it.

The shape is conjunctive, which makes it a tier-two rule rather than a tier-one
one: a here-document *and* a truncating redirect to a path. Measured, the two
facts are reported separately and both present:

    $ qwark facts "cat > f.go <<EOF
    package main
    EOF"
    redirect.truncate          1:5   │ > f.go
    redirect.heredoc           1:12  │ <<EOF package main EOF

**FACT 2026-08-19, and the reason this rule was asked for.** Every Go source
file in this repository up to that point was written by exactly that command
shape, by the assistant, under a standing instruction to prefer Bash for file
edits. The rule was stated while it was happening.

## A rule is a conjunction, and `or` is spelled with more rules

**PREFERENCE, owner, 2026-08-19.** A rule is several clauses, *all* of which must
match. There is no disjunction inside one. Where alternatives are wanted, they
are written as separate, nearly identical rules.

The worked example, in the owner's words:

    rm -r -f     forbidden, because of -f
    rm -r        no -f: ask, warning that it is recursive
    rm -f        no -r: forbidden

That is disjunctive normal form, chosen for the same reason the archive-guard
regex was kept blunt: **a rule that can be checked by reading it alone is worth
more than a compact one that cannot.** The cost is duplication between sibling
rules, which is visible. The cost of the alternative is a reader having to hold
a boolean expression in their head to know what a rule does, which is not.

## The strictest action wins, so order never matters

**PREFERENCE, owner, 2026-08-19.** Where several rules match one command, the
verdict is the strictest of them: deny over ask over allow.

The property this buys is worth naming. **No rule can be weakened by where it
sits**, in its file or among the files. A rule set assembled from several
sources cannot be defeated by arranging for a permissive file to be read last,
and a reader establishing what a command will do never has to know what came
before. Under first-match-wins, every rule's meaning depends on all the rules
above it — which is the property DNF was chosen to avoid, reintroduced at the
level of the file.

The cost is real and should be stated: a narrow exception cannot override a
broad deny. An exception has to be written into the deny rule as a clause that
excludes it. That is more to type, and it keeps the exception where a reader of
that rule will see it, rather than in another file that quietly outranks it.

## Rule files are named on the command line

**PREFERENCE, owner, 2026-08-19.** Rule files are given as arguments: a path
naming a directory contributes every rule file in it, a path naming a file
contributes that one.

The gain is that **the policy in force is readable where qwark is invoked** —
in the `settings.json` entry that registers the hook — rather than being implied
by which files happen to be sitting in a directory qwark knows about.

## Separation of duties belongs in the engine, not in the plumbing

**Owner, 2026-08-20.** The answer to an agent writing a `justfile` and then
running `just` is not to make the file unwritable. It is that **the agent which
can write those files is not the agent allowed to run them**:

> The point of the rules is using the engine to support that separation of
> duties.

And the reason it has to be the engine rather than the launcher:

> The base session doesn't get an "agent type" … so we can't as easily manage
> those rules without ACTIVELY managing symlinks or something else … so at the
> moment the concern is EITHER something wired into subagents, or some form of
> ENV VAR that will have to be actively managed … which feels rickety.

That is correct, and it also settles a contradiction already in the
requirements. **FR-10.6** says `agent_type` arriving in the payload "is what
makes per-agent scoping implementable from the payload, rather than through an
environment variable the agent might itself reach". **FR-10.6a** then says the
scoping "does not need solving inside qwark" because an external process picks
the files. Both cannot stand: the external route is precisely the env-var and
symlink management called rickety above, and FR-10.6 named the payload as the
better source for a stated security reason — *the subject cannot set its own
`agent_type`, and it can reach an environment variable.*

**Resolution: FR-10.6 stands, FR-10.6a is wrong as written.**

### Absence is a role, and that is what makes it work

The obstacle is real: a main-session call carries no `agent_type`, so identity
cannot simply be looked up. But **no identity is itself an identity.** The main
session is the one caller with no agent type, reliably and by construction, so a
rule can name that case exactly as it names any other — the schema already has
the spelling for it, because `absent = true` is how a clause says "this is not
there".

    [[rule.clause]]          # applies to a subagent of this type
    agent = "gate-runner"

    [[rule.clause]]          # applies to the main session, and only it
    agent  = ""
    absent = true

What that buys is the whole point: **one rule set, named once in
`settings.json`, carrying every role's policy inside it.** No symlink swapping
between launches, no environment variable to manage, and nothing to keep in step
outside the file being read. The policy in force stays readable where qwark is
invoked, which is what FR-4.15 was for.

It also composes with everything already here rather than adding a mechanism. An
`agent` clause is a clause: rules stay conjunctions, the strictest action still
wins, and a role cannot grant itself anything, because deny outranks allow and
no rule can override another.

### What this does not fix, stated plainly

**Two main sessions are indistinguishable.** If both the writer and the runner
are top-level launches, they carry no agent type and no clause can tell them
apart — the launcher must still give them different rule files. Engine-side
scoping solves the subagent case completely and the main-session case not at
all. That is worth knowing before it is relied on, and it is an argument for the
specialised agents being subagents rather than separate launches.

**A partition does not stop a chain.** Writer writes the `justfile`, runner runs
it: two agents, neither breaking its own rules, and the effect composes into the
attack the partition was meant to prevent. Separation of duties is only a
control if something sits between the two.

**Owner, 2026-08-20: that something is the task management process** — the same
process that produces the manifest of FR-9.7. It sees what the writer changed
before any runner is dispatched against it. So the manifest and the partition
are not alternatives; the manifest is what makes the partition mean something,
and the task process is the reviewer both of them assume.

## Rule files must not be writable by the agent

**Requirement, owner, 2026-08-19.** Rule files must be unwritable at runtime.

This is the same finding the predecessor ended on, arriving from the other
direction. `archive-guard.sh` concluded that a tool-layer gate cannot be the
guarantee, because it never sees a child process's syscalls. The mirror of that
is: **qwark gates Bash, and the Write and Edit tools reach the filesystem
without passing through it.** An agent that can edit the rules is not
constrained by them, and it does not need a shell to do it.

So the control has to be filesystem ownership, exactly as `chmod 000` was for
the sealed tree. Rule files owned by another user, not writable by the one the
agent runs as.

**The directory matters as much as the files.** A writable directory permits
unlink-and-replace, which defeats an unwritable file completely. Both are
checked.

qwark verifies this at load and refuses to run otherwise, so the property is
self-enforcing rather than a convention someone has to remember after a restore,
a fresh clone, or a `chmod -R` that meant well.

## Addressing a command

**PREFERENCE, owner, 2026-08-19.** A clause names the position it applies to.
Ordinal 0 is the command, arguments run from 1, and negative ordinals count from
the end with -1 as the last word. An index may name several: a comma-separated
list of ordinals and ranges.

**Ranges are written `..`.** The obvious `-` collides with negative ordinals --
`-3--1` cannot be read -- and a separator that only works in one direction is
one the writer has to think about every time. `1..-1` is every argument
regardless of count.

**CLAIM, and a caution for whoever writes the rules.** Bundling moves the
positive ordinals: `rm -r -f x` puts the operand at 3, `rm -rf x` puts it at 2,
and the two commands are identical to the shell. Negative ordinals are stable
against that, so an operand should be named from the end.

An ordinal the command does not have selects nothing rather than failing. A rule
about the third argument is simply not about a command with one, and that is not
an error in either direction.

## What it costs to detect `-f`

**Requirement, owner, 2026-08-19.** Block `--force` and `-f` — but only where
`-f` means force. It means file to `tar`, so the meaning cannot come from the
spelling. **The table of what each command's options mean, and which take a
value, is declared in TOML beside the rules.**

Two measurements say why a clause cannot simply compare strings.

**FACT 2026-08-19.** GNU long options accept unambiguous abbreviations. Each of
these deleted nothing and exited 0, which is only possible if force took effect:

    rm --force  --forc  --fo  --f     all accepted

So a clause matching the text `--force` is bypassed by three other spellings of
it. Long options must be resolved against the command's declared option set.

**FACT 2026-08-19,** from the tree:

    rm -rf x    ->  Word "rm"  Word "-rf"  Word "x"     no -f word exists
    rm -- -f    ->  Word "rm"  Word "--"   Word "-f"    -f is a filename here

Short options bundle into one word, and `--` turns the rest into operands. A
matcher that looked for an argument equal to `-f` would miss the first and
wrongly deny the second.

## The containment surface, and where it ends

Findings from an adversarial pass on 2026-08-19/20. Each is measured.

### Aliases reach the Bash tool, and they are not allowed

**FACT.** The Bash tool's shell is a **login, non-interactive** zsh — `$-` is
`569JOXYl`, carrying no `i`, and `[[ -o interactive ]]` is off. It therefore
never reads `.zshrc`, and `.zshenv`, `.zprofile` and `.zlogin` do not exist on
this machine.

**The aliases arrive anyway, from Claude Code's own shell snapshot**,
`~/.claude/shell-snapshots/snapshot-zsh-*.sh`, which replays them:

    3: unalias -a 2>/dev/null || true
  449: alias -- ls='eza --long --header --icons --git …'

So a rule about `ls` would describe a program that is not being invoked, and
seven options qwark never sees would be added to every call. That is the
`ss`→`rg` hazard already recorded in the standing rules, now load-bearing for a
security control.

**Decision, owner, 2026-08-19: using an alias is not allowed.**

### The shell is zsh, and the decision is to change the shell

**FACT 2026-08-20.** The tool is called Bash and runs **zsh 5.9**: `$0` is
`/bin/zsh`, `BASH_VERSION` is unset, `ZSH_VERSION` is 5.9, and Claude Code's
snapshots are named `snapshot-zsh-*.sh`. qwark had been parsing with `LangBash`
on the strength of the tool's name, which is exactly the kind of inference this
project exists to stop making.

**The mismatch is silent, which is what makes it serious.** Measured over ten
zsh constructs, the bash parser rejected two and parsed eight — and four of
those eight mean something different in zsh:

    **/*.go        two globs in bash; recursive descent in zsh
    *(.)           an extglob group in bash; "regular files only" in zsh
    $foo[2]        $foo then literal [2] in bash; an array element in zsh
    noglob rm *    a command named noglob in bash; a zsh precommand modifier
                   that leaves the glob unexpanded

A rejection is harmless here, because the gate denies what it cannot parse. A
construct that parses under the wrong grammar produces a confident wrong verdict.

### zsh executes code from inside a glob, and that defeats tier one

**FACT 2026-08-20, measured.** zsh has at least two constructs that run code
where nothing that looks like a command appears:

    zsh -c 'print -l *(e:"print EXECUTED >&2":)'   -> EXECUTED
    zsh -c 'x=$(...); print ${(e)x}'                -> executes the contents
    bash -c 'echo ${(e)x}'                          -> bad substitution

The second is a parameter expansion, so tier one already refuses it. **The first
is a glob**, and tier one does not ban globs. Put through qwark as it stands:

    $ qwark facts "rm *(e:'rm -rf /':)"
    glob            1:4   │ *(e:'rm -rf /':)
    glob.extended   1:4   │ *(e:'rm -rf /':)

No substitution, no pipe, no redirection, no logical concatenation. **It passes
every tier-one check**, and under zsh it runs `rm -rf /`. The parser reads it as
a bash `ExtGlob` — a construct with no execution semantics whatsoever — which is
the silent-misreading class in its worst form.

Closing this while remaining on zsh would mean banning globs outright, which is
a large usability cost for one construct. Under bash the construct does not
exist. **This is the reason that settles it**, rather than parser tidiness.

**Owner, 2026-08-20:** *"zsh has a lot of features I'd rather not have to deal
with."* Stated as a design property: a gate must model every construct its shell
can execute, so the shell's feature count *is* the gate's attack surface. Bash
is not safe, but it is smaller, and smaller is the only property that makes the
modelling tractable.

**Decision, owner, 2026-08-20: force the shell to bash**, rather than teach
qwark zsh. The reasons compound:

- `LangBash` is mature; the package marks `LangZsh` experimental and incomplete.
- It removes the silent-misreading class entirely rather than narrowing it.
- **It subsumes the alias problem.** The 64 aliases and 24 functions come from
  zsh configuration; a bash shell does not load them. The owner's alternative
  suggestion — start the process by unsetting all aliases — is the same idea
  applied downstream, and it is worth noting that the snapshot *already* opens
  with `unalias -a` and then deliberately restores all 64. Removing the source
  beats fighting the restore.
- It is one auditable change, rather than a zsh-awareness qwark carries forever.

**CLAIM, from correlation and not yet confirmed:** the shell follows `$SHELL`,
which is `/bin/zsh` here and matches both `$0` and the snapshot's name. Verifying
it costs one command in a fresh session: `echo $0`.

**What forcing bash does not fix.** Shell functions and PATH substitution work
in bash exactly as they do in zsh, so the shadowing findings below stand
unchanged. Claude Code would snapshot bash's environment instead of zsh's, which
is only an improvement if that environment is bare — worth checking rather than
assuming.

**Therefore qwark verifies rather than assumes.** The parser's variant is a
precondition, and a precondition that is merely hoped for is a bug waiting for a
new machine. qwark declares the shell it parses for and refuses to run when the
environment says otherwise, on the same fail-closed reasoning as everything else
here.

### No spelling of a command is immune to shadowing

The obvious mechanism for that ban is the escape already in the standing rules —
`\ls` runs the binary — which is why a word records whether it was escaped. It
is not sufficient, and the measurements say why.

**FACT 2026-08-20.** A backslash suppresses *alias* expansion only. Against a
shell function it does nothing:

    ls() { echo "FUNCTION RAN"; }
    ls          -> FUNCTION RAN
    \ls         -> FUNCTION RAN
    command ls  -> real binary
    /usr/bin/ls -> real binary

The snapshot carries functions as well as aliases, so this is a live mechanism
here and not a hypothetical one.

**The set is not stable, which matters more than its size.** ~~64 aliases and 24
functions~~ was measured on 2026-08-20 against one snapshot; **re-measured the
same day against a newer one after the shell configuration was rebuilt: 13
aliases and 21 functions.** Both readings were correct when taken. A rule
written against "the aliases" is therefore written against a moving target,
which is the argument for requiring an absolute path rather than consulting the
snapshot to find out what a name currently means.

Count them with:

    grep -c '^alias -- ' "$(ls -t ~/.claude/shell-snapshots/*.sh | head -1)"

**FACT 2026-08-20, and the one that settles it.** Both zsh and bash accept a
function whose name contains slashes, and it shadows the path:

    function /usr/bin/ls { echo "SLASH FUNCTION RAN"; }
    /usr/bin/ls -> SLASH FUNCTION RAN        (in both shells)

So an absolute path is not immune either. Neither is `command`, which is a
builtin, and function lookup precedes builtin lookup.

**CLAIM, and the honest conclusion: qwark cannot defend against a poisoned shell
environment.** Whatever the command line says, the function table decides what
runs, and nothing written in the command line escapes it. Requiring an absolute
path is still worth doing — it defeats alias expansion and the PATH hijack
demonstrated above, both of which are real and neither of which needs a function
— but it is defence in depth and must not be described as a guarantee.

**The control that actually holds is the same one as everywhere else in this
design: the shell environment must not be writable by the agent.** The snapshot
is `-rw-rw-r--`, owned by the agent's user, and reachable by `Write` without any
shell involved. It needs the treatment the rule files get.

**If qwark ever consults the snapshot, it must parse it and never source it.**
Sourcing an agent-writable file inside the gate would be executing the subject's
code in the judge's process. There is also no environment variable naming the
current snapshot — checked — so identifying it means guessing the newest file in
the directory, which is a further reason to prefer a policy that needs no
snapshot at all.

Whether editing the snapshot mid-session affects later calls was not
established: Claude Code's own classifier refused the write, and that refusal
was not worked around.

### An escape defeats a path rule, unless it is resolved

**FACT.** The parser keeps escapes in a literal's value, so `a\ b` arrives as
`a\ b` while bash passes `a b`. Written out, `rm /home/ancient/.cl\aude/x`
reaches `.claude` and a rule comparing the unresolved text does not match. The
resolution rules differ inside double quotes from outside, and both were read
off bash rather than recalled.

### A declaration is a permission

**CLAIM, and a caution for the project-level rules file.** Under strictest-wins,
an agent that adds an `allow` rule changes nothing: deny still beats it. But an
agent that adds a *declaration* moves a command from "undeclared, therefore
denied" to eligible. `[command.sh]` is a complete bypass written as
configuration.

So a project-level file that may add declarations needs the same protection as
the global set. A lower-trust project file is possible, but only if it may add
**rules** and never **declarations**.

### What qwark does not cover

Stated as a boundary, in the manner of the predecessor's own header, because a
control whose limits are not written down gets trusted past them.

**Other tools reach the filesystem without passing through qwark.** qwark gates
Bash. `Write` and `Edit` do not go through it, and they can reach the shell
snapshot, `settings.json`, `.git/hooks`, a `justfile`, or a rule file. Every
path qwark protects with a class-three rule needs the other half — a
`permissions.deny` entry — and that file must itself be unwritable.

**Some commands run other commands.** `env`, `xargs`, `sudo`, `nohup`,
`timeout`, `nice`, `watch`, `setsid`, `command`, `exec`, `eval`, `source`, and
`find -exec` all take a command as an operand, so a rule about `rm` never fires
on `env rm -rf x`. Interpreters do the same with inline code: `sh -c`,
`python -c`, `perl -e`, `node -e`, `awk 'BEGIN{system()}'`, GNU `sed` with `e`.
These must simply never be declared.

**Some commands run content the agent wrote.** `go test` executes test files the
agent authored; so do `pytest`, `npm test`, `cargo test`. `git commit` runs
`.git/hooks`. `just`, `make` and `npm run` execute recipes from files in the
tree. **A coding agent that can write code and run its tests has arbitrary
execution, and no rule set changes that.** What qwark constrains is what is
typed, not what the typed thing goes on to execute.

**A prefix assignment changes which binary runs.** Demonstrated: with
`PATH=<dir>:$PATH rm …`, a script named `rm` in that directory ran instead. This
is why prefix assignments are refused, and it also means the undeclared-option
denial is beside the point when the program is not the declared one.

### The end state is three layers, and qwark is the outermost

**Owner, 2026-08-20**, in two statements that belong together:

> When we have the tool chain in place, the agents will be in a sandbox, and
> these files won't be in there, and with that the blast radius rules will
> prevent a good chunk of the possible issues.

> Likewise, if/when we have the manifest files, those will tighten things down
> even better.

So the destination is not a longer deny list. It is:

    1. a sandbox         the file is not there to be reached
    2. the blast radius  a write must land inside the directory the agent
                         was started in                        (FR-9.1 - FR-9.6)
    3. the manifest      which files may be read and which written, named
                         by the task management process        (FR-9.7)

This is worth writing down because it changes what the path groups in
20-paths.toml are FOR. **Four of the six sit outside a project-rooted sandbox
and are absorbed by layer one**: qwark's own rules, Claude Code's configuration
and snapshot, the shell startup files, and the PATH directories. Under a
sandbox those rules guard against something that cannot happen, and they remain
worth keeping only as the un-sandboxed case and as defence in depth.

**Two of the six sit INSIDE it, and no sandbox removes them.**

    repository-hooks    .git/hooks/, .git/config, .githooks/
    task-definition     justfile, Makefile, Taskfile.yml, pyproject.toml,
                        package.json, bolt.*.yaml, .pre-commit-config.yaml

These are in the project, which is the one place the agent is supposed to be
able to write. **Layer two does not help either**: the blast radius says a write
must land inside the project, and every one of these already is.

So the residue after the sandbox is exactly the worry that prompted this
section — `just checks` and `bolt run` are fixed command lines whose meaning
lives in a file the agent may legitimately write. **The layer that closes it is
the manifest**, because the manifest is the only one of the three that
discriminates between files inside the blast radius. Not "the project is
writable" but "these files are writable, and a task definition is not one of
them".

That puts the priority somewhere other than where it looks. FR-9.6 and FR-9.7
are both `[?]` and unbuilt, and between them they are the whole of layer three.
Meanwhile the executors are all denied today, so the exposure is not live —
it becomes live the moment one of them is permitted, and permitting one is what
everybody will want as soon as the agent needs to run the gate.

**What no layer reaches**, stated so it is not rediscovered: a coding agent that
can write a test file and run the test runner has arbitrary execution. The
manifest would have to say `*_test.go` is writable, because writing tests is the
job. That is irreducible, and it is the reason `go test` is in the executor
group rather than quietly allowed.

### The proxy is another layer, and mode two is the audit of usage

**Owner, 2026-08-20:**

> The next layer will be more than this as a gate, and will be an MCP that
> becomes the proxy for the tools & mechanicals we want to allow. Then the
> situation gets much more simple. … Once we have the proxy, then we can have
> these kinds of rules for the various tools per agent type.

> **Mode Two will continue to be the audit of usage … still a different thing
> than the PROXY** and how things will interact with it.

**CORRECTED 2026-08-20: an earlier draft of this section called the proxy the
second mode. It is not.** qwark's second mode is the audit of usage — the
observability half, FR-4.8 to FR-4.9a. The proxy is a separate component that
qwark interacts with, not a further thing qwark becomes.

The distinction is worth holding, because a gate and an audit are opposite in
almost every respect that matters:

    a gate     synchronous, per call, must decide, FAILS CLOSED
    an audit   after the fact, across calls, must not lose a record,
               and blocking nothing is the point

**Their failure modes point in opposite directions.** A gate that breaks must
refuse — which is why exit 2 exists and why every path here ends in a decision.
An audit that breaks must not refuse, because an audit that can stop work will
be turned off; what it must not do is silently lose records, since a log with
gaps reads as a clean history.

An audit also sees what no gate can: a pattern across calls that no single call
would trip. That is the same argument tags were built on, arriving from the
other end — and it is why the log is worth having even where a store is not.

**This is also where an intention note would land.** The proxy collects the
stated reason; mode two is where it is compared against what the call actually
did. Neither half is worth much alone: the note is unverified prose, and the
record of effects does not say what was meant.

---

Back to the proxy itself, which is a layer rather than a mode. The engine does
not care what it is judging. A rule is clauses that must all hold; whether a
clause selects a word of a command line or a field of a tool call is
an adapter question, not a design one.

**It turns denying into enumerating, and that is the whole gain.** qwark denies
by default across the infinite space of things somebody might type. A proxy
exposes a finite set of operations, so what exists is what was written down.
Every hard problem in mode one is a consequence of the space being infinite and
the text being ambiguous: quoting, escapes, aliases, shell functions, `PATH`,
wrappers, interpreters, globs, substitutions. **A typed call cannot hide its own
effect the way a command line can**, which is what the whole of tier one exists
to force.

The mechanicals mostly become API design. "Allowed as a word, refused in a
shape" stops being a rule and becomes a parameter that is not offered — a
`reflog` operation with no `expire`. "Refused unless" becomes a required
argument. That is a better place for the constraint, because an operation that
cannot be named cannot be attempted, and a refusal never has to be understood.

**What carries over is the part worth keeping**: rules as conjunctions with no
disjunction inside one, the strictest action winning so order never matters,
deny as the engine's default rather than a rule's, a declaration granting
understanding and not permission, groups so a class is a list rather than twenty
rules, and a refusal that states its reason.

**This is what makes FR-7.12 and FR-7.13 foundational rather than interim.**
"These kinds of rules for the various tools per agent type" *is* the agent
clause: the discriminator that is awkward to arrange in the plumbing is the
natural one here, and it is the same clause either way.

**A proxy per agent type is the stronger form**, and the owner raised it as an
alternative or an addition: *"if we encode a PROXY for each of the agent types …
which would likewise limit everything"*. It is stronger for the same reason
enumeration beats denial — **an unexposed tool cannot be called at all**, while
a rule refuses a call that was already formed.

Its cost is the question already open under **where this is heading**: N proxies
are N surfaces to keep in step with each other and with the agent prompts, which
is the duplication problem in another dress. One proxy with agent-scoped rules
keeps the policy in one readable file; a proxy per agent type puts it in the
wiring. Unless the surfaces are generated from a single source — and the rule
files are the obvious candidate, which is the "those end up referencing these
rule files" branch the owner named.

That also says what the agent clause is for once proxies exist: not the primary
control, but a check on the wiring rather than the wiring itself — and the
policy for whatever Bash surface remains.

**What the proxy does not dissolve** is the residue everything else left. If the
proxy exposes an operation that runs `just checks`, the `justfile` still decides
what that does — the call is typed and its meaning is still in a file in the
tree. The proxy has to own the recipe, or the recipe has to sit outside what the
agent may write. Same residue as the sandbox, same answer: the manifest.

### The proxy also asks for intention, and that is the proxy's job

**Owner, 2026-08-20:** *"the point of the proxy is also to help include that
we're asking each tool usage to include notes regarding the intention"* — and,
asked whether to build an intention clause into the Bash gate: **"No, the intent
is for the PROXY."**

So this is a scope boundary rather than a feature waiting to be written. It is
recorded here for when the proxy is built, and because two things about it are
easy to get wrong.

**An intention is the one input the subject authors freely.** Everything mode
one reads is fixed by the shell grammar, assigned by the dispatcher
(`agent_type`), or read off the filesystem (ownership, resolved paths). A note
saying why is unverified prose written by the thing being judged.

**So it may require, record and shape-check — and it must never permit.** A rule
that allows because the note says "running the standard gate" is defeated by
typing that string, which is `[command.sh]` again in prose. Requiring one is
safe for the opposite reason: nobody defeats a requirement by satisfying it
honestly. This is the same reasoning FR-10.6 used to prefer `agent_type` over an
environment variable, at full strength.

**A plausible note can launder a bad action.** If whatever reviews these reads
them credulously, the note is worse than no note, because it supplies a
ready-made justification. The value is in comparing the stated intention against
what the call actually did — the mismatch is the signal, and the note on its own
is not evidence of anything.

What it buys is worth the care: FR-4.8's log stops being a list of what happened
and becomes checkable, both sides of every decision get recorded — qwark's
refusals already state a reason, and this makes the request state one too — and
the task management process gets the material it needs to sit between the writer
and the runner.

**Mode one is not the place for it, and not because it could not.** FACT
2026-08-20: Claude Code's Bash tool input carries a `description` field, and
`hook.BashCall` already parses it. It is deliberately unconsulted — kept because
reading the payload faithfully is FR-10.1, not left as an oversight.

## Nothing is expanded

**Owner, 2026-08-19:** *"we don't expand anything, that's why we block those."*

A word's value is reported only where its own text fixes it. Anything containing
a substitution is reported as undetermined, never guessed.

**FACT 2026-08-19, measured, and the reason this is hand-written rather than
delegated.** `expand.Literal` with a nil config refuses command substitution
properly — it does not execute anything — but it resolves `$HOME` to the empty
string and returns *no error*:

    $(echo EXECUTED)   err="unexpected command substitution"   (safe)
    $QWARK_PROBE       value=""   err=<nil>                    (silent)
    $((2+2))           value="4"  err=<nil>                    (silent)

A caller cannot tell a fixed word from one that was quietly guessed at. Reasoning
about `rm -rf /x` while the shell acts on `rm -rf /home/ancient/x` is the same
class of error as the predecessor's, reached by a different route.

**Environment variables are eliminated from command lines too** (owner,
2026-08-19), so the substitution ban covers parameter expansion — `$HOME`,
`$PWD` and the rest — and not only the other three.

## A clause states one of three forms, and says which reading it tests

**PREFERENCE, owner, 2026-08-20.** A clause states what it tests for in one of
three forms, and may say which reading of the word it tests.

    value   = "rm"          the whole word, exactly
    partial = ".claude"     anywhere within the word
    pattern = "rm|rmdir"    a regular expression over the whole word

**Naming `partial` is the point of having three rather than two.** An earlier
draft here had only exact-and-pattern, with "contains" written `.*\.claude.*`.
That works, and it hides the breadth of the rule inside a regex the reader has
to parse. A form named `partial` announces itself in the key.

It also settles an argument the two-form design could not. `pattern` is anchored
to the whole value — otherwise every pattern is quietly a partial, and the broad
reading becomes the one obtained by accident. That is exactly the predecessor's
mistake: `archive-guard.sh` matched the substring `.archive` and blocked
`web.archive.org`, costing a legitimate research route. **Nothing here prevents
an author choosing that breadth** — `partial = ".archive"` does the same thing.
What changed is that choosing it is now a visible act rather than a default.

**Exactly one form per clause.** None is an error rather than a clause matching
everything; several is an error rather than a precedence order nobody would
remember. An empty `partial` is refused for the same reason — every string
contains the empty string. An empty `value` stands, being precise.

**The interpreted reading is the default.**

    reading = "interpreted"   what the shell will pass       (default)
    reading = "written"       the source, escapes intact

Testing what was written is what lets `/home/ancient/.cl\aude/x` past a rule
about `.claude`. The written reading is still worth having — a rule about *how*
something was spelled is a real thing to want, and the alias ban is one — but it
has to be asked for.

**A word with no interpreted value is not matched.** Nothing is expanded, so
`$HOME` has none; a clause reading it finds nothing to test rather than testing
the empty string, which `partial` and `.*` would both match.

**CLAIM, and a property worth keeping.** Go's regexp is RE2: no backtracking,
linear in the input. A rule file cannot carry a pattern that a crafted command
makes pathological. For a program that runs before every shell command that is
worth more than backreferences, which no clause about a command word needs.

## Tags have lifetimes

Two kinds:

- **Ephemeral** — derived from this command's tree, live for one evaluation.
- **Sticky** — written by a rule, persisted, and decaying over subsequent
  commands. **Owner's example, 2026-08-19:** after a rebase, deletion is denied
  for the next six commands.

Sticky tags key on the `session_id` in the hook payload.

**UNDECIDED, owner, 2026-08-19.** How the state survives between calls is open.
The shape under consideration is an ephemeral file appended to on each call,
which each run then trims and rewrites, with locking so concurrent calls do not
collide.

Two facts bear on the locking, neither of them settling it:

- Contention has two sources — several Bash calls running in parallel within
  one session, and separate Claude Code sessions running at once on this
  machine. Keying the file by `session_id` removes the second entirely, leaving
  only the first for a lock to cover.
- A sideboard process holding the state — Redis was raised, with the owner's
  own reservation that some would call it heavy — moves the problem out of the
  filesystem but adds a daemon that must be up. That interacts with the
  fail-closed rule above: if an unparseable rule file makes Bash unusable, then
  by the same reasoning an unreachable state store does too, and a daemon
  outage becomes a Bash outage. A file has fewer ways to be absent.
- Worth weighing before adding any store at all: if every command is already
  logged, "was there a rebase in the last six commands" is a question the log
  answers. That is not an argument against a store, but it does mean a store
  has to earn its place against a tail read of a file that exists anyway.
- Append and compaction are not the same problem. Appends are frequent, small
  and independent; the trim-and-rewrite is rare and needs every other writer
  held off. A scheme that locks both alike pays the expensive price on the
  common path.

### The leaking bucket has no honest home in mode one

**Owner, 2026-08-20:** *"there are also still things that are going to need a
way to deal with this leaking bucket for the ttl tracking."*

There is a contradiction between the options above and a constraint already
stated in `40-state.toml`, and it is worth naming rather than discovering later:

> Tag state must not be writable by the user qwark runs as. A countdown the
> subject can reset is not a constraint on the subject.

**qwark runs as the agent's user.** It is a hook the agent's own session spawns,
with the agent's uid. So anything qwark can write, the agent can write —
directly, with the Write tool, without a shell. The leading candidate above, an
appended file the run trims, is exactly the option that cannot satisfy the
constraint: qwark needs to write it, therefore the subject can rewrite it,
therefore the countdown is advisory.

Nor does deriving it from the log escape this. A log qwark appends to is written
with the same uid, so "was there a rebase in the last six commands" is answered
from a file the subject can edit. **Mode two inherits the same problem**, and it
is sharper there: a log with selectively removed entries reads as a clean
history rather than as a damaged one.

**So the bucket needs a writer the subject is not**, and that is a deployment
fact rather than a thing more code can fix. Two ways out, and the second is
already coming:

- A store owned by another uid — the sideboard process above. Its cost was
  named as a daemon that must be up, which under fail-closed means a daemon
  outage becomes a Bash outage.
- **The proxy is that process.** Once tool calls go through a long-lived server
  the agent reaches only by typed call and never through the filesystem, the
  natural place for the count is inside it. The daemon objection loses most of
  its force at that point, because the daemon is already there and Bash is
  already narrowed to whatever the proxy does not cover.

Two properties the count must keep wherever it lands, both already decided and
easy to lose in a reimplementation: **a denied command advances nothing**
(FR-4.24 — it did not happen), and **tags do not stack**, so setting one that is
live replaces it, TTL and all.

### Redis with Lua, and the update is what ticks

**Owner, 2026-08-20**, thinking it through in three steps:

> I'm considering redis … we can just use some embedded lua or such for
> gathering the tags per session ID.

> The goal of using Lua within Redis was that it makes the execution of the
> entire Lua script an atomic action.

> After we have a result and a list of tags, we can then issue the update
> command that then updates the state for the session … **the update is what
> ticks the counts.**

**Atomicity is the whole argument, and it deletes most of the discussion above.**
A Lua script runs to completion inside Redis with nothing interleaved, so
reading the tags, applying the changes and decrementing every TTL is one
indivisible step. That removes the lock, the append-versus-compaction split, and
the worry about paying the expensive price on the common path — none of those
problems exist once the state transition is a single script rather than a
read-modify-write against a file.

**The update ticking, rather than the read, is the load-bearing choice.** It
makes FR-4.24 true by construction: a denied command issues no update, so it
advances no counter and sets no tag, and nothing has to remember to check the
verdict first. Had the read ticked, every refused command would have spent a
tick — and refused commands are the ones a constrained agent produces most.

Two things it does not settle.

**`ask` is unresolved, and qwark cannot resolve it.** FR-4.13 counts *allowed or
approved* commands. qwark returns `ask` and never hears whether the person
approved it, so at update time it cannot know which happened. Not ticking is the
safe direction — the tag then lives longer than it strictly should, which is
more restriction rather than less — but the honest answer is that **only mode two
sees what actually ran**, so the tick for an approved command belongs with the
audit rather than with the gate.

**The read-evaluate-update gap is not atomic, even though each step is.** Claude
Code can issue Bash calls in parallel, so a second call may read state taken
before the first call's update landed, and be judged without a tag that had just
been set. **This is the unsafe direction**, unlike a lost tick, and it is worth
saying plainly rather than being reassured by the word atomic. Closing it means
either serialising a session's calls, or optimistic concurrency — the read
returns a version, the update script refuses if the version moved, and qwark
re-evaluates.

## The mechanicals — the shapes a rule can be written in

**PREFERENCE, owner, 2026-08-20**, asked what a "library of useful mechanicals"
should be: a catalogue of the shapes, not a file of rules. What follows is that
catalogue. Nothing here is a new mechanism — each is the existing schema used a
particular way — and the point of naming them is that the next person writing a
rule reaches for a shape that already works instead of inventing a fifth one.

**Every shape below is a conjunction of clauses.** That is not one of the
options; it is the only thing a rule is. What varies is which selectors the
clauses use and where they point.

### 1. Refused by class

The plainest shape and the one most rules should be. A group names the members,
one rule names the group, and the reason belongs to the class rather than to
any member.

    [group.git-network]
    members = ["fetch", "pull", "push", "clone", …]

    [[rule.clause]]
    index = "0"
    value = "git"
    [[rule.clause]]
    index = "1"
    group = "git-network"

Adding a member is a one-line edit that does not touch the rule. **Classes are
expected to overlap**: `push` runs hooks and reaches the network, and under
FR-4.25 both reasons are collected, so a command in three classes is refused
with all three stated. Put a command in a class when the class's reason is true
of it, not when no other class claimed it first.

### 2. Allowed as a word, refused in a shape

For a command that reads when bare and writes when given a particular
subcommand. The word stays available; the destructive form does not.

    [[rule.clause]]
    index = "0"
    value = "git"
    [[rule.clause]]
    index = "1"
    value = "reflog"
    [[rule.clause]]
    index = "2"
    group = "git-reflog-writes"

The worked example is `no-git-destroying-the-reflog`, and it exists because
denying the word broke something: 40-state.toml tells a reader to look at
`git reflog` before deleting after a rebase, and clears the tag when they do.
**A denied command has no effect of any kind (FR-4.24)**, so denying the word
made the instruction impossible to follow and the tag impossible to clear. A
denial whose own message names a refused command is the smell this shape fixes.

### 3. Refused unless

A conditional refusal, written as one deny rule with an inverted clause rather
than as two rules where one outranks the other. The exception lives inside the
rule it modifies, so a reader of the denial sees the way out of it.

    [[rule.clause]]
    option = "gpg-sign"
    absent = true

**FACT 2026-08-20, and it is the trap in this shape: an inverted clause is
satisfied by absence, including the absence caused by qwark not understanding
the option.** `commit-must-be-signed` fires on `git commit --gpg-sign -m x`,
because `--gpg-sign` is not in the declaration, so "the signing option is not
there" is true. The verdict fails safe. The *message* does not: it tells
somebody who signed that they must sign.

So a "refused unless" rule is only honest when **the option it excepts is
declared**. Writing one without declaring the option produces a rule that reads
as conditional and behaves as unconditional.

### 4. Refused because it cannot be accounted for

Not a shape anyone writes — it is what happens when they write nothing. An
undeclared command is refused (FR-4.16) and an option the declaration does not
name is refused (FR-6.7), so **omission is a denial** and the safe state is the
default one.

This is what actually holds the read-only git allowance narrow. `git help -w`
opens a browser and `git log --ext-diff` runs an external program, and neither
is named by any rule: they are refused because `05-declarations.toml` does not
list those options. Nobody had to enumerate the dangerous flags, and forgetting
one costs a refusal rather than a hole.

The inverse is the thing to be careful about. **Declaring an option is what
makes it reachable**, so the declaration file is the eligibility surface and an
addition to it is a wider change than an addition to a rule file.

### 5. Set a condition, then judge against it

The tag shape, deferred to a later version and documented in 40-state.toml. One
rule establishes a condition with a lifetime; other rules name it as a clause
without knowing how it was established. Its cost is that it is the only shape
whose answer depends on anything beyond the command in front of it.

### Where this is heading, and it is not more git rules

**Owner, 2026-08-20**, asked which git commands should be reclaimed as guarded
allows — the answer was none, and the reason resets the target:

> Eventually the goal is that the agents will **never** be executing git
> commands, and instead have a very specific command surface they are allowed.
> The more specialised the agent, the more the list can be narrowed. One would
> expect that either the list ends up a duplication of a series of allowed
> commands in the agent text or supporting files, or those end up referencing
> these rule files.

Three things follow, and they are worth separating.

**The allowed surface is per agent, not per machine.** That is already the
mechanism: rule files are named on the command line (FR-4.15), and an external
process chooses which files a given agent gets (FR-10.6a). Narrowing by
specialisation needs no new machinery, only more files.

**The read-only git allowance is a waypoint, not the destination.** It stands
because it was ruled on this session, and the direction above says the eventual
answer is narrower, not wider. It is the first thing to remove when the specific
surfaces exist.

**The duplication was the open question, and the proxy settles it.** An agent's
prompt saying what it may run and a rule file deciding what it may run are two
statements of one fact, and two statements of one fact drift. The owner first
named both directions — generate the prompt from the rules, or have the prompt
reference them — and then **resolved it a third way, 2026-08-20:**

> The proxies then ALSO hold the details on what they expose, meaning they
> include the details on what those agents can do … so we aren't repeating those
> details any more (once we're there).

That is the strongest available answer, because it is not two artifacts kept in
step but one artifact doing both jobs: **the exposed surface is simultaneously
the statement of what an agent may do and the thing that enforces it.** Nothing
can drift from itself. A capability list that is documentation somewhere and
configuration somewhere else has a wrong version; a tool that is simply not
exposed has no second version to be wrong.

**How much it settles depends on how narrow the tools are**, and that is worth
knowing in advance. A proxy exposing `run_command(cmd)` is Bash with extra
steps and moves nothing. A proxy exposing `git_log(ref)` still needs something
to say which `ref` — a surface says *which operations*, not *with what values*.
So the duplication disappears to the extent the tools are narrow, and whatever
argument-level constraint remains is what rules are still for.

## Configuration

**PREFERENCE, owner, 2026-08-19.** TOML, in multiple rule files that are
aggregated. `github.com/BurntSushi/toml` — already the house library
(`dotfiles/go/internal/manifest` — that repository was named `linux.dotfiles`
when this was written, and took over the `dotfiles` name on 2026-08-20), and its
`ParseError.ErrorWithPosition`
renders a source excerpt with line and column, which this design needs:

**If any rule file is unparseable, Bash is unusable.** Fail-closed. A gate that
degrades to permissive when its own configuration is broken is a gate that
reports success while guarding nothing.

The cost is that a typo denies every Bash command until it is fixed, so the
denial has to name the file, the line, and the text — and the escape route must
not itself require Bash. Editing the rule file with the Edit tool does not.

## Observability

**Requirement, owner, 2026-08-19.** Every command is logged with the relevant
detail from its environment — cwd, environment variables, and what else bears
on the decision.

JSONL, following bolt's precedent: one object per command, appended, so the
record is greppable and a partial write costs one line rather than the file.

**The log is also the state tracker.** Tier four asks questions of the form
"was there a rebase in the last six commands?" — which the log already answers.
A separate store maintained alongside it would be a second copy of the same
history, free to disagree with the first. Reading the tail of the log instead
means the record that explains a decision *is* the record that produced it.

### Environment variables are a disclosure risk, and this log is durable

**CLAIM.** Claude Code spawns the hook as a subprocess, so the hook inherits
its whole environment — which on this machine routinely includes API tokens.
Recording `os.Environ()` verbatim writes those to a file that persists, is
grepped later, and may be read while diagnosing something unrelated. That is
the failure `secret-scan` exists to catch, arriving through a door qwark would
have opened itself.

**Proposed, pending the owner's answer.** Record every variable *name*, so the
shape of the environment is visible and a change in it is detectable. Record
*values* only for names a rule file declares. Anything undeclared is recorded
as present-but-withheld rather than omitted, so the log never silently implies
a variable was absent.

## Open questions

Recorded rather than guessed at; see `NEXT_STEPS.md`.

- Does a *denied* command decrement a sticky tag's countdown, or only one that
  actually ran? `PreToolUse` cannot tell; `PostToolUse` can.
- `substitution.parameter` covers a bare `$HOME` as well as `${HOME}`, so
  banning the family bans every variable reference including `$PWD`. Is that
  intended, or should the ban name the other three?
- Verdict for a command qwark cannot parse.
- Which environment variables may be logged by value; where the log lives,
  and whether it rotates.
