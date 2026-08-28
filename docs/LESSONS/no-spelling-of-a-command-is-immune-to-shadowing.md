# No spelling of a command is immune to shadowing

The alias ban has an obvious mechanism: the escape already in the standing rules,
where `\ls` runs the binary. That is why a word records whether it was escaped. It
is not sufficient.

A backslash suppresses *alias* expansion only. Against a shell function it does
nothing:

    ls() { echo "FUNCTION RAN"; }
    ls          -> FUNCTION RAN
    \ls         -> FUNCTION RAN
    command ls  -> real binary
    /usr/bin/ls -> real binary

The snapshot carries functions as well as aliases, so this is a live mechanism
here and not a hypothetical one.

**The set is not stable, and that matters more than its size.** One snapshot gave
64 aliases and 24 functions; a newer one, taken after the shell configuration was
rebuilt, gave 13 aliases and 21 functions. Both readings were correct when taken.
A rule written against "the aliases" is written against a moving target, which is
the argument for requiring an absolute path instead of consulting the snapshot to
find out what a name currently means.

Count them with:

    grep -c '^alias -- ' "$(ls -t ~/.claude/shell-snapshots/*.sh | head -1)"

And the measurement that settles it: both zsh and bash accept a function whose
name contains slashes, and it shadows the path.

    function /usr/bin/ls { echo "SLASH FUNCTION RAN"; }
    /usr/bin/ls -> SLASH FUNCTION RAN        (in both shells)

So an absolute path is not immune either. Neither is `command`, which is a
builtin, and function lookup precedes builtin lookup.

**qwark cannot defend against a poisoned shell environment.** Whatever the command
line says, the function table decides what runs, and nothing written in the
command line escapes it. Requiring an absolute path still earns its place, since
it defeats alias expansion and the PATH hijack demonstrated above, both of which
are real and neither of which needs a function. It is defence in depth and must
not be described as a guarantee.

**The control that actually holds is the one that holds everywhere else in this
design: the shell environment must not be writable by the agent.** The snapshot is
`-rw-rw-r--`, owned by the agent's user, and reachable by `Write` with no shell
involved. It needs the treatment the rule files get.

**If qwark ever consults the snapshot, it must parse it and never source it.**
Sourcing an agent-writable file inside the gate would run the subject's code in
the judge's process. There is also no environment variable naming the current
snapshot, which was checked, so identifying it means guessing the newest file in
the directory. That is a further reason to prefer a policy needing no snapshot at
all.

Whether editing the snapshot mid-session affects later calls is unestablished.
Claude Code's own classifier refused the write, and that refusal was not worked
around.
