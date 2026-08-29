# qwark

**A `PreToolUse` hook for Claude Code that decides whether a proposed Bash
command may run.** It takes the tool call on stdin, parses the command the way a
shell would, judges it against declared rules, and answers with a decision.

## What it is FOR

*"remember this is ALL about controlling what an Agent can run."*

qwark is a containment tool and its subject is an agent: a process that may be
careless, may be wrong about what a command does, and in the limit may be working
against the person who ran it.

**So when qwark cannot account for something, it refuses.** Unparseable command:
denied. Unparseable rule file: no Bash at all. Undeclared command: denied.
Undeclared option: denied. A word whose value is not fixed by its text: refused,
never guessed at. **A gate that degrades to permissive whenever it is confused is
a gate whose confusion is the way through it.**

Declaring every command is a smaller job than it looks: *"you don't use that many
tools."*

### Where it sits against its siblings

`silo/docs/GLOSSARY.md` is the canonical statement of how bolt, toolbox and anvil
fit together. **qwark is not part of that chain.** It is a consumer of it like any
other project, and it gates the agent doing the work rather than the work itself.

## Layout

    cmd/qwark/          the entry point, one statement
    internal/shell/     parse as Bash; gather every fact in one walk
    internal/command/   ordinals, escape resolution, option decomposition
    internal/rules/     TOML loading, validation, ownership, the evaluator
    internal/gate/      the join: a rule set and a request become a decision
    internal/hook/      the PreToolUse contract; every path ends in a decision
    internal/cli/       ast, facts, rules, judge, hook
    internal/reach/     blast-radius containment
    internal/repo/      which branch is checked out, without running git
    internal/audit/     the decision log, one JSON object per line
    rules/              the shipped rule set, in five files plus declarations
    install/            the settings fragment that registers the hook
    scripts/            what the gate needs and a jig cannot carry generically
    docs/               this file, and one file per decision, lesson, pattern

**`REQUIREMENTS.md` and `SUPPRESSIONS` are deliberately single files at the root,
and not directories.** *The split is pending* below says why.

## The gate

    bolt --definitions go-std-quality go-std-quality .
    bolt common-quality .

**Read `result.yaml` in the stamped run directory. Never the exit status.** bolt
has exited **0** on a run whose artifact said `success: false`, and it exits 1
when it could not carry the run out at all, which is a different claim from a
check having failed.

Both pass: `success: true` in each artifact, 7 tasks and 3.

**Then read the `kind` on each reason, because a failure has two meanings.** A
tool said no, or bolt could not run it, and only the second indicts the gate
rather than the tree. The two vocabularies are disjoint in bolt's source:
`nonzero-exit` is emitted in one place, `src/run.rs`, while folding a task's exit
status, and the refusal kinds (`base-missing`, `jig-unreadable`,
`unknown-placeholder`, `task-without-command` and the rest) live in
`src/error.rs` and never appear there. So the test is which file the kind comes
from rather than whether the message reads sensibly.

A third kind, `time-limit`, is neither, and bolt withholds `nonzero-exit` for it
deliberately: a task its own limit killed never reaches the fold, because that
status is bolt's signal rather than an answer the tool gave. Synthesising one
would report the kill twice, the second time as an exit nobody produced.

**A killed task still fails, and still writes an envelope.** `timed_out` in
`src/run.rs` writes `success: false` with the limit reason first and then
extends it with whatever the partial run had already reported, so a tool that
found forty problems before hanging keeps all forty. A timed-out run cannot
read as a pass, because what it did not reach is exactly what is unknown about
it.

The general form is bolt's FR-6.1a: it reaches a verdict itself only where no
adapter result is available to take, and each case says so where it arises.
Reading the kind is how a caller tells which happened.

**The composition is one jig per run, not an overlay.** `bolt -c a -c b` is gone
with the rebuild; the current CLI is `bolt <jig> <directory>`, and flags come
before the positionals. How the two quality jigs should compose is unsettled and
tracked in `clank/tasks/toolbox/port-the-jigs/10`.

**`bolt.qwark.yaml` is retired**, not ported. It carried exactly one task,
`entrypoint`, and that is now the shared jig's placeholder filled by the
definitions file below, so porting it would have restated something already
homed. An overlay was the right shape while the CLI composed jigs and is not a
shape the CLI has.

**`main()` is measured, not excluded.** Hard rule 5. The shared jig leaves an
`entrypoint` placeholder defaulting to `true`; qwark fills it from
`bolt.go-std-quality.definitions.yaml` with `scripts/cover-entrypoint.sh`, which
builds with `go build -cover`, runs `qwark help`, and converts the profile for
the adapter to merge. A placeholder is one argument and is shell-quoted, which
is why the chain lives in a script and not in the value.

`bolt secrets .` **fails and never passed here**: `detect-secrets` wants a
`.secrets.baseline` this repository has never had. Creating one records the
current findings as accepted, which is suppression-shaped, so it waits on a
ruling rather than being generated.

Measured 2026-08-28 from the passing runs: 129 requirement rows, every one held
to coverage has a test at **109 of 109**, and 17 open questions are exempt.
Every test cites a requirement the document defines.

### Adopter status

**All ten links of the `go` set resolve.** qwark is the first repository to have
adopted cleanly, because it never carried a vendored jig; `link-jigs` refused four
links in bolt for exactly that reason.

    for f in bin/*.py adapters/*/*.py config/*.yml bolt.*.yaml; do
      [ -e "$f" ] && echo "ok   $f" || echo "BROKEN $f"
    done

The links are gitignored. The same files arrive from the anvil layer in a built
image, so committing a copy here would make a third statement of one thing.
**`bolt.go-std-quality.definitions.yaml` and `scripts/` are tracked**, because
the placeholder values and the script they name are this project's own content,
and are the part a shared definition must never carry.

### The split is pending, and this is why

`test-traceability.py --requirements <DIR>` dies with an unhandled
`IsADirectoryError`, and `suppression-register.py` takes a single path. Both gate
this repository.

So `docs/REQUIREMENTS/` and `docs/SUPPRESSIONS/` are not here yet, because
splitting a gated file into a directory turns a passing task into a crashing one.
`LESSONS`, `PATTERNS` and `DECISIONS` gate nothing, and they use the directory
layout already.

`docs/MOCKS/` is absent because nothing is mocked, and `SUPPRESSIONS` carries no
pragmas: `grep -rn 'nolint\|#nosec' --include='*.go' .` returns nothing.

## The chain, and how it is enforced

    docs/DECISIONS/  ->  REQUIREMENTS.md  ->  the tests
      why                what must be true     COVERS: names the requirement

**Every test states which requirement it discharges**, in a comment immediately
above it:

    // COVERS: FR-2.4 | negative

Kinds: `positive`, `negative`, `edge`, `property`, `regression`. The
`traceability` task fails a test that cites nothing, and one that cites a
requirement `REQUIREMENTS.md` never defined, so renaming a requirement means
fixing every test that cites it.

A requirement marked `[?]` is an open question and carries no test. The newer
checker in toolbox holds *settled* requirements to a test and exempts `[?]`;
qwark passes it at **108 of 108**, with 19 open and exempt.

## Conventions particular to this repository

- **Tests live in an external test package** (`package foo_test`), exercising the
  public API. `testpackage` enforces it.
- **Tests are held to the same bar as the code.** No exemption from `funlen`,
  `dupl`, `mnd` or the complexity gate. A table that has outgrown the length limit
  wants splitting into several.
- **Doc comments say why.** The what is in the code underneath them.
- **No path reaches outside this directory** except the jigs, which are
  configuration rather than code.

## What is not done

`NEXT_STEPS.md` carries the detail.

**qwark has gated a live session, and the result is recorded rather than
predicted.** 2026-08-28: the hook registered in `.claude/settings.local.json`
was active for a whole session in this tree. Under the shipped rules `git status`
and `git log` ran; `ls`, `cat`, `find`, `grep`, `go`, `bolt`, `qwark` itself,
`git add -N` and `git commit -F` were all refused, most of them at
`declared commands only` because `git` is the one declared command.

**Arming it makes this repository nearly unusable, by design, and that is now
measured rather than expected.** A session cannot build qwark, test it, run the
gate, list a directory, or commit. It also cannot run `qwark judge`, which is
this project's own way of trying a rule before trusting it.

**The live set is now two files of shape only, and the tree is workable.** That
was `78e0410`'s declaration table plus `required = false`; a session commits,
builds and runs the gate, and only compound shapes are refused. What the full
set would still cost is measured in `NEXT_STEPS.md` under *What installing the
source set costs*: nine commands lost, four of them how qwark is built and
gated. Those four are deny rules, which no amount of declaration reaches, and
the `cwd` clause is the mechanism that resolves it.

**Two properties of the registration, both measured that day.** A change to an
existing hook takes effect on the very next command, with no restart. And
`permissions.deny` naming `settings.local.json` itself removes the documented
escape from anything but a person: deleting the `hooks` key with the Edit tool
stops being available to the session that needs it.

Getting out needs no Bash and no root: move `.claude/settings.local.json` aside,
which is a person's command when the gate is wedged and the session's own Edit
otherwise.
