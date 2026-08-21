# qwark

**A `PreToolUse` hook for Claude Code that decides whether a proposed Bash
command may run.** It reads the tool call on stdin, parses the command as a
shell would, judges it against declared rules, and answers with a decision.

Marks follow the silo convention: **FACT** measured this session with the
command that produced it; **CLAIM** asserted, inherited or inferred;
**PREFERENCE** chosen, with nothing to be right about.

---

## What it is FOR

**Owner, 2026-08-19:** *"remember this is ALL about controlling what an Agent
can run."*

That sentence decides most of the design. qwark is a containment tool and its
subject is an agent — not a careful colleague who occasionally mistypes, but a
process that may be careless, may be wrong about what a command does, and in the
limit may be working against the person who ran it.

**So when qwark cannot account for something, it refuses.** Unparseable command:
denied. Unparseable rule file: no Bash at all. Undeclared command: denied.
Undeclared option: denied. A word whose value is not fixed by its text: refused,
never guessed at.

Read as ergonomics each of those looks needlessly harsh. Read as containment
they are the only settings that mean anything: **a gate that degrades to
permissive whenever it is confused is a gate whose confusion is the way through
it.**

The owner's answer to the obvious worry — that declaring every command is a
large job — is that it is not. *"you don't use that many tools."* An agent's
working set is small.

### Where it sits against its siblings

`toolbox/GLOSSARY.md` is the canonical statement of how the toolchain fits
together; read it there rather than here. In one line each: **bolt** runs the
jigs, **toolbox** holds them, **anvil** is what you run them on, and **qwark**
gates what an agent may type at a shell.

**qwark is not part of that chain** — it is a consumer of it like any other
project, and gates the agent doing the work rather than the work itself.

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

**`REQUIREMENTS.md` and `SUPPRESSIONS` are single files at the root, not
directories, and that is deliberate.** See *The split is pending* below.

## The gate

    bolt -c bolt.common-quality.yaml -c bolt.go-std-quality.yaml -c bolt.qwark.yaml

**Read `run_result.yaml` in the stamped run directory. Never the exit status.**
FACT 2026-08-21: bolt exited **0** on a run whose artifact said `success: false`.

**FACT 2026-08-21: that command currently FAILS, and not because of anything in
qwark.** `coverage` reports `below_minimum: 1` because `entrypoint` now runs
*after* it:

    linked jigs   … 09_tests  10_coverage  11_vuln  12_entrypoint
    bolt's old    … 09_tests  10_entrypoint  11_coverage  12_vuln

`entrypoint` appends `cmd/qwark/main.go`'s profile to `coverage.out`, so it has
to run first. The shared jig removed `entrypoint` — correctly, since it names a
project's own main package — and an overlay's tasks are appended last, so no
adopter can put it back in place. bolt's FR-2.9 fixes execution to declaration
order and no ordering key exists.

**Filed as `clank/inbox/toolbox/entrypoint-runs-after-coverage/`**, with the
evidence and a repro. Until it is resolved, the older invocation still passes and
measures the pre-split checkers:

    bolt -c ../bolt/bolt.go-std-quality.yaml -c bolt.qwark.yaml    # 12/12, older rules

**FACT 2026-08-21, against the linked jigs:** 11 of 12 tasks pass; coverage
94.2% with one file under the floor for the reason above; 207 tests carrying a
`COVERS` line; 127 requirements of which 19 have no test and all 19 are `[?]`.

### Adopter status

**FACT 2026-08-21: all ten links of the `go` set resolve.** qwark is the first
repository to have adopted cleanly, because it never carried a vendored jig —
`link-jigs` refused four links in bolt for exactly that reason.

    for f in bin/*.py adapters/*/*.py config/*.yml bolt.*.yaml; do
      [ -e "$f" ] && echo "ok   $f" || echo "BROKEN $f"
    done

The links are gitignored. In a built image the same files arrive from the anvil
layer, so a committed copy would be a third statement of one thing.
**`bolt.qwark.yaml` is tracked** — the overlay is this project's own content, and
is the part a shared definition must never carry.

### The split is pending, and this is why

**FACT 2026-08-20, measured:** `test-traceability.py --requirements <DIR>` dies
with an unhandled `IsADirectoryError`, and `suppression-register.py` takes a
single path. Both gate this repository.

So `docs/REQUIREMENTS/` and `docs/SUPPRESSIONS/` do not exist here yet:
splitting a gated file into a directory turns a passing task into a crashing
one. `LESSONS`, `PATTERNS` and `DECISIONS` gate nothing and use the directory
layout now.

`docs/MOCKS/` is absent because nothing is mocked, and `SUPPRESSIONS` carries no
pragmas. **FACT 2026-08-21: `grep -rn 'nolint\|#nosec' --include='*.go' .`
returns nothing.** Those are claimed states, not gaps.

## The chain, and how it is enforced

    docs/DECISIONS/  ->  REQUIREMENTS.md  ->  the tests
      why                what must be true     COVERS: names the requirement

**Every test states which requirement it discharges**, in a comment immediately
above it:

    // COVERS: FR-2.4 | negative

Kinds: `positive`, `negative`, `edge`, `property`, `regression`. The
`traceability` task fails when a test cites nothing, or cites a requirement
`REQUIREMENTS.md` does not define. **Adding a test means adding its `COVERS:`
line; renaming a requirement means fixing every test that cites it.**

Requirements marked `[?]` are open questions with no test yet. The newer checker
in toolbox holds *settled* requirements to a test and exempts `[?]`; qwark passes
it at **108 of 108**, with 19 open and exempt.

## Conventions particular to this repository

- **Tests live in an external test package** (`package foo_test`), exercising
  the public API. The `testpackage` linter enforces it.
- **Tests are held to the same bar as the code** — no exemption from `funlen`,
  `dupl`, `mnd` or the complexity gate. A table that outgrew the length limit
  should be several tables.
- **Doc comments say why, not what.** The what is in the code underneath.
- **No path reaches outside this directory** except the jigs, which are
  configuration rather than code.

## What is not done

`NEXT_STEPS.md` carries the detail. In short: **qwark has never gated a live
session.** The hook is registered in `.claude/settings.local.json` and was not
active in the session that wrote this; whether that needs a restart or an
approval is unestablished.

**Arming it makes this repository nearly unusable, by design.** Under the shipped
rules `git status` and `git log` are allowed and `ls`, `cat`, `grep`, `go build`,
`git add` and `git commit` are denied. The way out needs no Bash: delete the
`hooks` key from `.claude/settings.local.json` with the Edit tool.
