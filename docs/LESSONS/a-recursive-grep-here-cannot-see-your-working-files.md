# A recursive grep here cannot see your working files

FACT, measured 2026-08-28.

## The measurement

Same binary, same string, same file:

    \grep -n   "THE GATE IS ARMED" ~/.projects/qwark/START_HERE.md   ->  line 8
    \grep -rln "THE GATE IS ARMED" ~/.projects/qwark/                ->  no match

A recursive search does not reach `START_HERE.md`, and a direct one does.
`START_HERE.md` is line 14 of this repository's `.gitignore`. Everything under
`.ephemera/` behaves the same way.

## What it is

`grep` here resolves to something gitignore-aware when given `-r`. **A backslash
did not fix it**, which is how it was identified: `\` suppresses alias expansion
and not a shell function, so this is a function rather than an alias.

That is FR-4.18 word for word, in this project's own requirements:

> A backslash suppresses alias expansion but not a shell function, and both zsh
> and bash accept a function named `/usr/bin/ls` which shadows the binary.

FR-4.18 is `[?]`, carries no test, and was deferred as defence in depth. It is
not theoretical. It is the shell this session was running in.

## What it cost

A false clean on a real check.

Asked whether this repository cited any of the clank SHAs killed by that night's
history rewrite, a recursive grep answered no. `START_HERE.md` held three, on
lines 52 and 118. Had the handoff gone out on that answer, the next session would
have inherited three dead references in the one file it is told to read first.

The blind spot is **exactly the tier of file a session keeps its working state
in**: the handoff, the scratch notes, the commit messages, the evidence under
`.ephemera/`. Tracked source is visible and everything about the session's own
work is not.

## What to do instead

**Name the path when the answer has to include untracked files.**

    \grep -n <pattern> <explicit-path>          reliable
    find <dir> -type f -exec grep -Hn … {} +    reliable, and reaches everything
    \grep -rn <pattern> <dir>                   TRACKED FILES ONLY

**Check a negative result before believing it.** A recursive grep returning
nothing means "not in any tracked file", which is a narrower claim than "not
here" and reads identically. Where the answer matters, re-run it against one
path you know contains the string; if that matches and the recursive one did
not, the recursion is lying to you.

**Treat any finding derived from a recursive grep as having a gitignore-shaped
hole.** Several in this repository were, including the search for citations of a
retired requirement. That one happened to find what it needed in tracked files.

## Why this belongs to qwark specifically

qwark exists because a command's name does not reliably say what will run. This
is that, in the tool the project uses to check its own claims, discovered by
being bitten rather than by reading the requirement that predicted it.

It is also the argument for building FR-4.18 rather than deferring it further:
the requirement was written from reasoning, and this is the incident.
