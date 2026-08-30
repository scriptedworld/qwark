# Contributing to qwark

## Building and testing

Go 1.26.6 or newer, which is what `go.mod` requires. Nothing else is needed to
build or to run the suite.

    go build -o bin/qwark ./cmd/qwark
    go test ./...

`just build` and `just test` are the same two commands. `just --list` prints the
whole interface.

**`just checks` needs more than a clone provides.** It runs the project's
quality gate through `bolt`, and the jig definitions, adapters and checker
scripts arrive as symlinks into a sibling checkout rather than as files in this
tree. They are gitignored, because committing a copy would make a second
statement of one thing that is free to drift. Neither `bolt` nor those
definitions is published yet, so `go test ./...` is the gate an outside
contributor can run, and a change is reviewed against the rest.

## What a change has to carry

The chain runs from why, to what must be true, to the test.

`docs/DECISIONS/` holds the reasoning, one topic per file. `REQUIREMENTS.md`
states each requirement as an observable property of a run: what is true, not
how the code is arranged. Every test names the requirement it discharges, in a
comment directly above it.

    // COVERS: FR-4.4 | property

The kinds are `positive`, `negative`, `edge`, `property` and `regression`. The
traceability check fails a test that cites nothing and a test that cites a
requirement `REQUIREMENTS.md` does not define, so renaming a requirement means
fixing every test that cites it.

A requirement marked `[?]` is an open question. It carries no test and is
reported as context rather than as a failure.

**A requirement can be retired or superseded, and its ID is never reused.**
Reuse silently rewrites what every existing reference to that ID means, and
nothing about the new row looks wrong. A `## Retired` section records where each
one went. Retiring a requirement leaves its `COVERS:` marks pointing at nothing,
and they are repointed or removed in the same change.

## Conventions particular to this repository

Tests live in an external test package, `package foo_test`, and exercise the
public API. The `testpackage` linter enforces it.

Tests are held to the same bar as the code, with no exemption from the length,
duplication or complexity limits. A table that has outgrown the length limit
wants splitting into several.

Doc comments say why. The what is in the code underneath them.

Command functionality does not live in `main.go`. The entry point is one
statement; everything it needs is in `internal/cli`.

**Never add a suppression pragma.** `SUPPRESSIONS` is the register, and it is
currently empty. The gate compares it against the source in both directions and
fails on a pragma with no entry, an entry with no pragma, or a file whose count
has changed. If a check fails, fix it or ask; silencing it is not an option
available here.

## Changing the rules

The rules are policy, not code, and they are read by whoever has just been
refused. Three things follow from that.

**Try a rule before trusting it.** `qwark judge` takes the same rule paths and
request fields the hook does, so a rule can be exercised as the caller that will
meet it. A rule set that has never judged anything is a policy nobody has run.

    qwark judge rules -- git push --force origin main
    qwark judge --agent=test-runner rules -- go test ./...

**Write the reason for the reader who has just hit it.** It is what a refused
agent is shown, and a denial nobody can act on gets routed around rather than
understood.

**Adding a declaration is a wider change than adding a rule.** An undeclared
command is refused outright, so `05-declarations.toml` is the surface that
decides what is eligible to be judged at all. Read its header before adding to
it: an option belongs there only if it means the same thing on every subcommand
that can reach it, and is harmless on all of them.

`docs/RULES.md` is the reference for the rule language, and
`docs/PATTERNS/the-mechanicals-the-shapes-a-rule-can-be-written-in.md` catalogues
the shapes a rule can take, including the two that look right and do not work.

## Commit messages

Conventional commits, and one concern per commit. The subject says what changed.
The body says what it cost, in counts and verdicts, and where the rest lives.

The reasoning belongs in the file the commit changed, and the requirement in
`REQUIREMENTS.md`. A message that restates either has written one thing twice,
and the log is the copy nobody can correct later.

A message can only cite backwards. Resolve a SHA before writing it.
