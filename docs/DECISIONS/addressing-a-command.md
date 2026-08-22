# Addressing a command

A clause names the position it applies to. Ordinal 0 is the command, arguments
run from 1, and negative ordinals count from the end with -1 as the last word.
An index may name several at once: a comma-separated list of ordinals and ranges.

Ranges are written `..`. The obvious `-` collides with negative ordinals, since
`-3--1` cannot be read, and a separator that works in only one direction is one
the writer has to think about every time. `1..-1` is every argument, whatever the
count.

A caution for whoever writes the rules: bundling moves the positive ordinals.
`rm -r -f x` puts the operand at 3 and `rm -rf x` puts it at 2, and the two
commands are identical to the shell. Negative ordinals are stable against that,
so name an operand from the end.

An ordinal the command does not have selects nothing rather than failing. A rule
about the third argument is not a rule about a command with one argument, and
that is not an error in either direction.
