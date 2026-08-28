# Rule files must not be writable by the agent

Rule files must be unwritable at runtime.

This is the finding the predecessor ended on, reached from the other direction.
`archive-guard.sh` concluded that a tool-layer gate cannot be the guarantee,
because it never sees a child process's syscalls. The mirror of that is that
qwark gates Bash, and the Write and Edit tools reach the filesystem without
passing through it. An agent that can edit the rules is not constrained by them,
and it needs no shell to do it.

So the control has to be filesystem ownership, exactly as `chmod 000` was for the
sealed tree: rule files owned by another user, not writable by the one the agent
runs as.

The directory matters as much as the files. A writable directory permits
unlink-and-replace, which defeats an unwritable file completely. Both are checked.

qwark verifies this at load and refuses to run otherwise, so the property holds
itself up instead of being a convention somebody has to remember after a fresh
clone or a `chmod -R` that meant well.
