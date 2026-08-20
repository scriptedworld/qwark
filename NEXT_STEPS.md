# qwark — what is not done

State at 2026-08-20. `REQUIREMENTS.md` states the same ground as requirements;
this file says what to do about it and what is waiting on an answer.

---

## Built, committed, and passing the gate

**FACT 2026-08-20**, measured this session: `just checks` passes all twelve
tasks; coverage 94.1% with every file above the 80% per-file floor; 186 tests
carrying a `COVERS` line; 125 requirements of which 19 are deferred and **every
testable one has a test**.

- `internal/shell` — parse as Bash, gather every fact in one walk, and record
  the parser's own vocabularies (node types, operators, statement flags).
- `internal/command` — ordinals, escape resolution, option decomposition
  against a declared table.
- `internal/rules` — TOML loading, validation, ownership checking, and the
  evaluator.
- `internal/hook` — the PreToolUse contract, and the guarantee that every path
  ends in a decision or exit 2.
- `internal/reach`, `internal/repo` — blast-radius containment; branch read
  without running git.
- `internal/cli` — `ast`, `facts`, `rules`, `judge`.
- `rules/` — 39 draft rules across five files. Illustrative, not final.

## Next: the actual command set

**Owner, 2026-08-20.** The next piece of work is the concrete list of commands
to let through, and the guards over git.

> **MOST git commands are NOT okay for the agent to execute.**

That is the starting position, not a detail. `rules/10-commands.toml` currently
denies a handful of git subcommands (`git-executes`) and says nothing about the
rest, which under deny-by-default means the rest are refused anyway — but
refused for the wrong reason, with a message that says "undeclared" rather than
saying why. Per FR-4.21 the refusals worth having are explicit.

Useful facts for that work:

- `declarations: 0` today. Nothing is declared, so nothing runs. Check with
  `./bin/qwark rules rules/`.
- Try a rule before trusting it: `./bin/qwark judge rules/ -- git commit -m x`.
- A declaration grants understanding, not permission. Declaring git is what
  lets a rule say something precise about `git push --force`; it permits
  nothing on its own.
- `.ephemera/demo-allow.toml` holds throwaway declarations and allow rules used
  to exercise the chain. It is not part of the rule set and is gitignored.

## Waiting on an answer

1. **How tag state survives between calls.** Deferred by the owner on
   2026-08-20 — the shape is settled and the foundation is in place, but there
   will be no store until there are concrete scenarios worth limiting this way.
   Nine requirements sit behind it (FR-4.7, 4.13, and section 8).
2. **The observability log**: where it lives, whether it rotates, and the list
   of environment variables whose values are withheld. Three requirements
   (FR-4.8, 4.9, 4.9a). The withhold model is a denylist by the owner's choice,
   with pattern matching added because naming secrets one at a time fails open.
3. **Which commands write.** FR-9.6 says any path given to a writing command
   must stay in the blast radius, and nothing yet says which commands write.
   That is a declaration question: a `writes` flag per command, or per option.
4. **The manifest** (FR-9.7) — created by the task management process, read at
   runtime, saying which files may be read and which written.

## Known limits, written down so they are not rediscovered

- **qwark gates Bash only.** The Write and Edit tools reach the rule files, the
  shell snapshot, `.git/hooks` and `settings.json` without passing through it.
  Every class-three rule needs a `permissions.deny` twin plus ownership.
- **A coding agent that can write files and run its tests has arbitrary
  execution** regardless of qwark. `go test` runs code the agent just wrote.
  What qwark constrains is what is typed, not what the typed thing executes.
- **The hook registration is fixed for a session.** An external process can
  choose rule files per session launch, but a subagent spawned inside a running
  session gets the same command line. Varying policy per subagent would need
  qwark reading `agent_type` after all.
- **`SHELL` decides the Bash tool's shell at session start** and cannot be
  changed by a hook afterwards. Forcing bash means exporting it before
  launching Claude Code. **FACT 2026-08-20: this machine runs zsh 5.9 there.**

## Deferred to a later version

- **Tags** (section 8) — foundation in place, no store, no scenarios yet.
- **Cost ordering** (FR-4.2) — cannot change a verdict, only how much work
  happens before it is known. The seam is `order` in the evaluator, today the
  identity.
