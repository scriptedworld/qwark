# Tier one: the command must say what it does

The first rule bans redirections, substitutions, pipes, and logical
concatenation. Two separate reasons, both Jeff's:

- **Redirections buy nothing.** The tool call already returns stdout and stderr
  separately, so redirecting to capture output is redundant: the harness hands it
  back regardless. Banning it costs no capability and removes file truncation as
  a side effect.

- **Substitutions make the answer relative.** A command should always be clear
  as to what it is doing. A substitution makes the text a recipe whose result
  depends on runtime state, rather than a statement of what will happen.

Those four are one property rather than four separate bans: the command's effect
is determined by its own text. Every tier above depends on it. Deciding which
paths a command reaches is unsound the moment a `$(…)` can produce a path at
runtime, which is precisely how `bin/repos status` got past the predecessor. Tier
one does more than list dangers; it is what makes tiers two and three decidable,
and that is why it comes first.
