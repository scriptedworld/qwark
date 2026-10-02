# Rules are layered by cost

Four classes, cheapest first, so a decision reachable in an early class never
pays for a later one:

1. Node presence. Certain nodes in the tree are an instant rule.
2. Conjunctive. Several elements, *all* of which must match.
3. State. An ongoing tracker across commands.
4. Context. What the paths a command names are: where they resolve, whether
   they fall inside a given tree, their type, owner, mode, size, content type
   and mount.

Class 1 costs nothing per rule. Every structural fact is gathered in a single
walk (`internal/shell/facts.go`), so a rule consulting them is a set lookup and
adding one is free. The cost is set by the size of the tree, not by the number of
rules.

State comes before context because its cost is flat and context's grows with
the command. Measured on ext4, as medians over 2000 runs, with qwark starting a
fresh process for every command:

    state: lock, read, decode, rename the session file      about 30 us
    context, per path word: stat, resolve, containment,      6 to 8 us
      type, owner, mode, size, a 512-byte content sniff
    context, the mount table: one /proc/self/mountinfo       65 to 80 us
      parse, up to 300 us on a first read

So context passes state at three or four path words, and a mount lookup alone
costs more than the state file. Two choices keep the numbers this way. The
state file is not fsynced, since an fsync costs about 6,400 us, and `rename`
already guarantees a reader sees a whole file; the exclusive lock still
serialises every writer in a session, its subagents included. And the mount
table is read only when a loaded rule tests a mount.
