# zsh executes code from inside a glob, and that defeats tier one

zsh has at least two constructs that run code where nothing resembling a command
appears:

    zsh -c 'print -l *(e:"print EXECUTED >&2":)'   -> EXECUTED
    zsh -c 'x=$(...); print ${(e)x}'                -> executes the contents
    bash -c 'echo ${(e)x}'                          -> bad substitution

The second is a parameter expansion, so tier one already refuses it. **The first
is a glob**, and tier one does not ban globs. Put through qwark as it stands:

    $ qwark facts "rm *(e:'rm -rf /':)"
    glob            1:4   │ *(e:'rm -rf /':)
    glob.extended   1:4   │ *(e:'rm -rf /':)

No substitution, no pipe, no redirection, no logical concatenation. **It passes
every tier-one check**, and under zsh it runs `rm -rf /`. The parser reads it as a
bash `ExtGlob`, a construct with no execution semantics whatsoever.

Closing this while staying on zsh would mean banning globs outright, a large
usability cost for one construct. Under bash the construct does not exist. That,
rather than parser tidiness, is what settles it:

*"zsh has a lot of features I'd rather not have to deal with."*

A gate must model every construct its shell can execute, so the shell's feature
count *is* the gate's attack surface. Bash is not safe, only smaller, and smaller
is what makes the modelling tractable.

**The decision is to force the shell to bash**, and not to teach qwark zsh. The
reasons compound:

- `LangBash` is mature; the package marks `LangZsh` experimental and incomplete.
- It removes the silent-misreading class entirely instead of narrowing it.
- **It subsumes the alias problem.** The 64 aliases and 24 functions come from zsh
  configuration, and a bash shell does not load them. The alternative suggestion,
  starting the process by unsetting all aliases, is the same idea applied
  downstream, and the snapshot *already* opens with `unalias -a` and then
  deliberately restores all 64. Removing the source beats fighting the restore.
- It is one auditable change, instead of a zsh-awareness qwark carries forever.

From correlation, and not yet confirmed: the shell follows `$SHELL`, which is
`/bin/zsh` here and matches both `$0` and the snapshot's name. Verifying it costs
one command in a fresh session, `echo $0`.

**What forcing bash does not fix.** Shell functions and PATH substitution work in
bash exactly as they do in zsh, so the findings in **No spelling of a command is
immune to shadowing** stand unchanged. Claude Code would snapshot bash's
environment instead of zsh's, which is only an improvement if that environment is
bare, and that wants checking, not assuming.

**So qwark verifies rather than assumes.** It declares the shell it parses for and
refuses to run when the environment says otherwise, on the same fail-closed
reasoning as everything else here.
