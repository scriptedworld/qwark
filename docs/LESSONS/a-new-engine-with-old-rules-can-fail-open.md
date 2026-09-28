# A new engine with old rules can fail open

When the verdict order changed to block over allow over ask over deny, the
engine and the rule files had to change together, and the two halves are
installed separately: `just install` writes the binary, a copy writes
`~/.config/qwark/rules`.

Before either was installed, the new binary judging the old live rules allowed
`ls > out.txt`. The old rules still said `deny` and
still carried a catch-all allow, and under the new order an allow lifts every
deny. The structural gate was gone, and nothing reported it.

The other mismatch fails closed. The old binary cannot load a rule file saying
`block`, so every Bash call is refused, including the one that would install
the new binary.

So an engine change that alters what a rule file means is installed as a pair,
in the order that fails closed: rules first, then the binary, from a terminal.
A session gated by the hook it is replacing cannot take that order, because
the first step locks it out; it installs the binary first, and the one command
it runs in between is the copy.

Before installing either half, judge the new binary against the live rules as
they stand:

    .ephemera/qwark-new judge --cwd=$PWD ~/.config/qwark/rules -- 'ls > out.txt'

Anything but a refusal means the pair has to move together.
