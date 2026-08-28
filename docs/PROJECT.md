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
    rules/              the shipped rule set, in five files plus declarations
    install/            the settings fragment that registers the hook
    docs/               this file, and one file per decision, lesson, pattern

**`REQUIREMENTS.md` and `SUPPRESSIONS` are deliberately single files at the root,
and not directories.** *The split is pending* below says why.

## The gate

    bolt -c bolt.common-quality.yaml -c bolt.go-std-quality.yaml -c bolt.qwark.yaml

**Read `run_result.yaml` in the stamped run directory. Never the exit status.**
bolt has exited **0** on a run whose artifact said `success: false`.

**That command currently FAILS, and not because of anything in qwark.** `coverage`
reports `below_minimum: 1` because `entrypoint` now runs *after* it:

    linked jigs   … 09_tests  10_coverage  11_vuln  12_entrypoint
    bolt's old    … 09_tests  10_entrypoint  11_coverage  12_vuln

`entrypoint` appends `cmd/qwark/main.go`'s profile to `coverage.out`, which only
helps if it runs first. The shared jig removed it correctly, since it names a
project's own main package, and an overlay's tasks are appended last, so no
adopter can put it back in place. bolt's FR-2.9 pins execution to declaration
order, and there is no ordering key to override that with.

**Filed as `clank/inbox/toolbox/entrypoint-runs-after-coverage/`**, with the
evidence and a repro. The ruling on it: *"entrypoint should be part of the
standard for go projects … if it's my project, it will follow that pattern, and
therefore need that test."* So it goes back into the shared jig, written
generically: `go list ./cmd/...` returns one package in every Go project here and
its basename is the binary name.

**When that lands, delete the `entrypoint` task from `bolt.qwark.yaml`.**

Until then the older invocation still passes, and measures the pre-split checkers:

    bolt -c ../bolt/bolt.go-std-quality.yaml -c bolt.qwark.yaml    # 12/12, older rules

Against the linked jigs: 11 of 12 tasks pass; coverage 94.2% with one file under
the floor for the reason above; 207 tests carrying a `COVERS` line; 127
requirements of which 19 have no test, and all 19 are `[?]`.

### Adopter status

**All ten links of the `go` set resolve.** qwark is the first repository to have
adopted cleanly, because it never carried a vendored jig; `link-jigs` refused four
links in bolt for exactly that reason.

    for f in bin/*.py adapters/*/*.py config/*.yml bolt.*.yaml; do
      [ -e "$f" ] && echo "ok   $f" || echo "BROKEN $f"
    done

The links are gitignored. The same files arrive from the anvil layer in a built
image, so committing a copy here would make a third statement of one thing.
**`bolt.qwark.yaml` is tracked**, because the overlay is this project's own
content, and is the part a shared definition must never carry.

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

**Two properties of the registration, both measured that day.** A change to an
existing hook takes effect on the very next command, with no restart. And
`permissions.deny` naming `settings.local.json` itself removes the documented
escape from anything but a person: deleting the `hooks` key with the Edit tool
stops being available to the session that needs it.

Getting out needs no Bash and no root: move `.claude/settings.local.json` aside,
which is a person's command when the gate is wedged and the session's own Edit
otherwise.
