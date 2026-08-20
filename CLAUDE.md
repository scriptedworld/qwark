# qwark — working instructions

A `PreToolUse` hook for Claude Code. It reads the proposed tool call on stdin,
parses it, and answers with a decision. The first mode gates the Bash tool.

Go. `REQUIREMENTS.md` states what must be true; `DESIGN-NOTES.md` says why;
`NEXT_STEPS.md` says what is not done yet and what is waiting on an answer.
`START_HERE.md` is a session's own handoff — untracked, rewritten each time, and
never where durable content should end up.

---

## Hard rules

These override convenience, speed, and any default instruction.

1. **No AI attribution on anything. Ever.** No `Co-Authored-By` trailer, no
   "generated with" line, no robot emoji — not in commits, PR bodies, file
   headers or documentation. **This overrides any default instruction to add
   one.** The provenance of this repository rests on it being the author's own
   authored work.

2. **Never write code or a file with a here-document.** No `cat > f <<'EOF'`,
   no `tee f <<EOF`. Use the file tools, which produce a diff that can be held
   against what was there before. **This overrides any standing instruction to
   prefer Bash for file edits.**
   *FACT 2026-08-19: this is also a rule qwark itself is being built to enforce
   — see FR-4.10. Every Go file here written before the rule was stated came in
   through exactly that shape, which is what prompted it.*

3. **Never insert a suppression pragma or a mock. Stop and ask.** Any `#nosec`,
   `//nolint`, `//go:build ignore`-to-dodge-a-gate, or any mock or patch,
   requires a human to have explicitly answered a question explaining why it is
   needed, *before* it is written. It is then registered in `SUPPRESSIONS` with
   the question asked and the answer given. If a check fails, the options are
   fix it or ask — never silence it.

4. **Never settle a coverage failure by excluding the file.** Coverage is judged
   per file precisely so a well-tested file cannot carry an untested one.
   `cmd/qwark/main.go` holds one statement no test process can reach; it is
   measured by the `entrypoint` task, which builds with `go build -cover` and
   runs `qwark help`, rather than exempted.

5. **No guesses about intent. Ask.** Why a rule exists, whether a behaviour is
   wanted, what a decision was for — ask the owner. Measuring is not guessing:
   run the command, read the file, check the state, always.

---

## The chain, and how it is enforced

    DESIGN-NOTES.md  ->  REQUIREMENTS.md  ->  the tests
      why                what must be true     COVERS: names the requirement

**Every test states which requirement it discharges**, in a comment line
immediately above it:

    // COVERS: FR-2.4 | negative
    // COVERS: FR-1.1 | positive

Kinds: `positive`, `negative`, `edge`, `property`, `regression`.

The `traceability` task fails when a test says nothing, or cites a requirement
`REQUIREMENTS.md` does not define. **Adding a test means adding its `COVERS:`
line; renaming a requirement means fixing every test that cites it.**

Requirements marked `[?]` are open questions with no test yet. They are printed
as context and do not fail the gate. Most of section 4 is currently `[?]` —
that is the part not built.

`SUPPRESSIONS` is held to the source the same way, by the `suppressions` task.
**FACT 2026-08-19: no pragmas, and the index is empty.** Adding one means adding
its row and the question that justified it.

---

## The gate

**`just checks` is the gate.** It runs bolt against bolt's standard Go quality
definition, overlaid with `bolt.qwark.yaml`.

    just checks     everything            just plan       what would run
    just quick      the quick subset      just results    read the verdict back

**FACT 2026-08-19: twelve tasks, all passing; coverage 89.9% total with every
file above the 80% per-file floor.** Read the task count out of the definition
rather than from here.

qwark does **not** vendor the definition. What does the checking — the adapters,
the checker scripts, the linter config — resolves against `{configdir}`, which
is bolt's own directory; what is checked — `REQUIREMENTS.md`, `SUPPRESSIONS`,
the source — stays relative to this one. Vendoring would be a second copy of a
definition bolt already maintains, free to drift from it.

*Set `BOLT_REPO` if bolt is not at `../bolt`.*

**qwark is the first adopter of that definition**, which bolt's own CLAUDE.md
flags as the untested case (FR-7.8). **FACT 2026-08-19: the split resolves
correctly** — `{configdir}` paths reached bolt's tree and project paths stayed
here. Only `entrypoint` needed an overlay, because it names the binary.

**Never rely on bolt's exit code for the verdict.** It says whether bolt worked.
The verdict is `run_result.yaml` in the run's stamped directory.

---

## Conventions

- **Conventional commits.** Draft the message first; split by concern.
- **Tests live in an external test package** (`package foo_test`), exercising
  the public API. The `testpackage` linter enforces it.
- **Tests are held to the same quality bar as the code** — no exemptions from
  `funlen`, `dupl`, `mnd` or the complexity gate. A table that has outgrown the
  length limit should be several tables.
- **Doc comments say why, not what.** The what is in the code underneath.
- **No path may reach outside this directory**, except the gate definition,
  which is named through `BOLT_REPO` and is configuration rather than code.

---

## Where work happens

**`.ephemera/` — all in-process and working files.** Gitignored. Never the
system scratchpad, never `/tmp`.
