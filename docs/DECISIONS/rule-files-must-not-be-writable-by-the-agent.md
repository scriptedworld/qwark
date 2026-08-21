# Rule files must not be writable by the agent

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 269-291. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**Requirement, owner, 2026-08-19.** Rule files must be unwritable at runtime.

This is the same finding the predecessor ended on, arriving from the other
direction. `archive-guard.sh` concluded that a tool-layer gate cannot be the
guarantee, because it never sees a child process's syscalls. The mirror of that
is: **qwark gates Bash, and the Write and Edit tools reach the filesystem
without passing through it.** An agent that can edit the rules is not
constrained by them, and it does not need a shell to do it.

So the control has to be filesystem ownership, exactly as `chmod 000` was for
the sealed tree. Rule files owned by another user, not writable by the one the
agent runs as.

**The directory matters as much as the files.** A writable directory permits
unlink-and-replace, which defeats an unwritable file completely. Both are
checked.

qwark verifies this at load and refuses to run otherwise, so the property is
self-enforcing rather than a convention someone has to remember after a restore,
a fresh clone, or a `chmod -R` that meant well.

