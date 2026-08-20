# qwark — what is not done

State at 2026-08-19. `REQUIREMENTS.md` states the same ground as requirements;
this file says what to do about it and what is blocked.

---

## Built and passing the gate

The reading half. Nothing here decides anything yet.

- `internal/shell` — parse as Bash, recover any node's exact source, and gather
  every structural fact in one walk.
- `internal/command` — lift a simple command into ordinals, and report a word's
  value only where its text fixes it. Index specifications with `..` ranges and
  negative ordinals.
- `internal/cli` — `qwark ast` and `qwark facts`, so rules are authored against
  a visible vocabulary rather than a remembered one.
- The gate: twelve bolt tasks, all passing.

## Settled, and not to be re-litigated

Recorded here because a question that has been answered should stop being asked.
The reasoning is in `DESIGN-NOTES.md`.

- Actions are `allow`, `deny`, `ask`, `tag`.
- Rules are conjunctions of clauses; `or` becomes sibling rules.
- Tier one bans redirections, substitutions, pipes and logical concatenation —
  and substitutions include parameter expansion, so `$HOME` and `$PWD` go too.
- Writing code or a file through a here-document is refused.
- Ordinals: 0 is the command, negatives count from the end, ranges use `..`.
- Rules and the per-command option table are TOML, in several files aggregated.
- An unparseable rule file makes Bash unusable.
- An unparseable command is denied, with the parser's own message returned.
- A denied command does not advance the tool-usage count.
- Nothing is expanded, ever.

## Waiting on an answer

**These block the engine. They are not to be guessed at.**

1. **When two rules match with different actions, which wins?** `rm -rf` matches
   both the deny on `-f` and the ask on `-r`. Deny beating ask is the obvious
   reading of the owner's example, but that is a guess about whether precedence
   is by action or by order. With several rule files aggregated, order also
   needs defining across files, not only within one.

2. **Where do rule files live, and in what order are they aggregated?** A fixed
   directory, a per-project one, silo, or a declared list. Whether a later file
   may override an earlier file's rule, or only add to it.

3. **What happens to a command with no declared option table?** And to an option
   a declared command does not declare — `rm --wibble`? The candidates are treat
   as unknown and ask, or let only the rules that do match apply. This decides
   qwark's behaviour for every tool not yet described, which is most of them.

4. **How does state survive between calls?** Owner's own open question. An
   appended ephemeral file each run trims and rewrites under a lock, or a
   sideboard process such as Redis. See `DESIGN-NOTES.md` for what bears on it,
   including that the observability log may already answer what a store is for.

5. **Does the tool-usage count count Bash calls only, or every tool call?** "No
   deleting for six commands" means something different under each.

6. **Which environment variables may be logged by value?** Proposed: names
   always, values only where declared, undeclared recorded as present-but-
   withheld. Needs the declared list. Also where the log lives and whether it
   rotates.

7. **Should a prefix assignment be refused too?** `FOO=bar cmd` changes what the
   command does without appearing in it. Detectable already as `assignment`;
   nothing has been said about whether it should be banned alongside `$VAR`.

## Next, once those are answered

- `internal/option` — decompose a command's words against its declared table:
  bundled short options, `--` as terminator, `--long=value`, and long options
  resolved through abbreviation rather than compared as text.
- `internal/rules` — TOML rule files, aggregated, fail-closed on any that will
  not parse (FR-4.5), naming file, line and text (FR-4.6).
- `internal/hook` — the `PreToolUse` contract: read the payload, emit the
  decision as JSON on stdout, **exit 0 regardless**. A non-zero exit reports
  that the hook failed to run, which is a different claim from a denial.
- Evaluation in cost order, with `tag` enriching what later tiers match on.

## Not yet designed

- Modes for tools other than Bash. The shape is meant to take them; nothing has
  been decided about what they gate.
- How qwark is installed and registered. `silo` holds the Claude configuration
  and has a wired-but-empty `hooks/` slot; qwark is a standalone repository that
  silo links to, following silo's own rule that a tool substantial enough to
  stand alone gets its own repository. Registration is a `settings.json` entry
  there, plus a build step of the kind `linux.dotfiles` uses for `milton`.
