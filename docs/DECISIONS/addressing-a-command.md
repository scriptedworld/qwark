# Addressing a command

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 292-312. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**PREFERENCE, 2026-08-19.** A clause names the position it applies to.
Ordinal 0 is the command, arguments run from 1, and negative ordinals count from
the end with -1 as the last word. An index may name several: a comma-separated
list of ordinals and ranges.

**Ranges are written `..`.** The obvious `-` collides with negative ordinals --
`-3--1` cannot be read -- and a separator that only works in one direction is
one the writer has to think about every time. `1..-1` is every argument
regardless of count.

**CLAIM, and a caution for whoever writes the rules.** Bundling moves the
positive ordinals: `rm -r -f x` puts the operand at 3, `rm -rf x` puts it at 2,
and the two commands are identical to the shell. Negative ordinals are stable
against that, so an operand should be named from the end.

An ordinal the command does not have selects nothing rather than failing. A rule
about the third argument is simply not about a command with one, and that is not
an error in either direction.

