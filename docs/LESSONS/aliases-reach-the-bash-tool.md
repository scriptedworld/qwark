# The containment surface, and where it ends

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 339-362. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


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

