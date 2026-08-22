# Tier one: the command must say what it does

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 99-119. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


The first rule bans **redirections, substitutions, pipes, and logical
concatenation**. Two separate reasons, both the, 2026-08-19:

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

