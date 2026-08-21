# No spelling of a command is immune to shadowing

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 444-505. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


The obvious mechanism for that ban is the escape already in the standing rules —
`\ls` runs the binary — which is why a word records whether it was escaped. It
is not sufficient, and the measurements say why.

**FACT 2026-08-20.** A backslash suppresses *alias* expansion only. Against a
shell function it does nothing:

    ls() { echo "FUNCTION RAN"; }
    ls          -> FUNCTION RAN
    \ls         -> FUNCTION RAN
    command ls  -> real binary
    /usr/bin/ls -> real binary

The snapshot carries functions as well as aliases, so this is a live mechanism
here and not a hypothetical one.

**The set is not stable, which matters more than its size.** ~~64 aliases and 24
functions~~ was measured on 2026-08-20 against one snapshot; **re-measured the
same day against a newer one after the shell configuration was rebuilt: 13
aliases and 21 functions.** Both readings were correct when taken. A rule
written against "the aliases" is therefore written against a moving target,
which is the argument for requiring an absolute path rather than consulting the
snapshot to find out what a name currently means.

Count them with:

    grep -c '^alias -- ' "$(ls -t ~/.claude/shell-snapshots/*.sh | head -1)"

**FACT 2026-08-20, and the one that settles it.** Both zsh and bash accept a
function whose name contains slashes, and it shadows the path:

    function /usr/bin/ls { echo "SLASH FUNCTION RAN"; }
    /usr/bin/ls -> SLASH FUNCTION RAN        (in both shells)

So an absolute path is not immune either. Neither is `command`, which is a
builtin, and function lookup precedes builtin lookup.

**CLAIM, and the honest conclusion: qwark cannot defend against a poisoned shell
environment.** Whatever the command line says, the function table decides what
runs, and nothing written in the command line escapes it. Requiring an absolute
path is still worth doing — it defeats alias expansion and the PATH hijack
demonstrated above, both of which are real and neither of which needs a function
— but it is defence in depth and must not be described as a guarantee.

**The control that actually holds is the same one as everywhere else in this
design: the shell environment must not be writable by the agent.** The snapshot
is `-rw-rw-r--`, owned by the agent's user, and reachable by `Write` without any
shell involved. It needs the treatment the rule files get.

**If qwark ever consults the snapshot, it must parse it and never source it.**
Sourcing an agent-writable file inside the gate would be executing the subject's
code in the judge's process. There is also no environment variable naming the
current snapshot — checked — so identifying it means guessing the newest file in
the directory, which is a further reason to prefer a policy that needs no
snapshot at all.

Whether editing the snapshot mid-session affects later calls was not
established: Claude Code's own classifier refused the write, and that refusal
was not worked around.

