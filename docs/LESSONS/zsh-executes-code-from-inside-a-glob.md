# zsh executes code from inside a glob, and that defeats tier one

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 384-443. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**FACT 2026-08-20, measured.** zsh has at least two constructs that run code
where nothing that looks like a command appears:

    zsh -c 'print -l *(e:"print EXECUTED >&2":)'   -> EXECUTED
    zsh -c 'x=$(...); print ${(e)x}'                -> executes the contents
    bash -c 'echo ${(e)x}'                          -> bad substitution

The second is a parameter expansion, so tier one already refuses it. **The first
is a glob**, and tier one does not ban globs. Put through qwark as it stands:

    $ qwark facts "rm *(e:'rm -rf /':)"
    glob            1:4   │ *(e:'rm -rf /':)
    glob.extended   1:4   │ *(e:'rm -rf /':)

No substitution, no pipe, no redirection, no logical concatenation. **It passes
every tier-one check**, and under zsh it runs `rm -rf /`. The parser reads it as
a bash `ExtGlob` — a construct with no execution semantics whatsoever — which is
the silent-misreading class in its worst form.

Closing this while remaining on zsh would mean banning globs outright, which is
a large usability cost for one construct. Under bash the construct does not
exist. **This is the reason that settles it**, rather than parser tidiness.

**2026-08-20:** *"zsh has a lot of features I'd rather not have to deal
with."* Stated as a design property: a gate must model every construct its shell
can execute, so the shell's feature count *is* the gate's attack surface. Bash
is not safe, but it is smaller, and smaller is the only property that makes the
modelling tractable.

**Decision, 2026-08-20: force the shell to bash**, rather than teach
qwark zsh. The reasons compound:

- `LangBash` is mature; the package marks `LangZsh` experimental and incomplete.
- It removes the silent-misreading class entirely rather than narrowing it.
- **It subsumes the alias problem.** The 64 aliases and 24 functions come from
  zsh configuration; a bash shell does not load them. The alternative
  suggestion — start the process by unsetting all aliases — is the same idea
  applied downstream, and it is worth noting that the snapshot *already* opens
  with `unalias -a` and then deliberately restores all 64. Removing the source
  beats fighting the restore.
- It is one auditable change, rather than a zsh-awareness qwark carries forever.

**CLAIM, from correlation and not yet confirmed:** the shell follows `$SHELL`,
which is `/bin/zsh` here and matches both `$0` and the snapshot's name. Verifying
it costs one command in a fresh session: `echo $0`.

**What forcing bash does not fix.** Shell functions and PATH substitution work
in bash exactly as they do in zsh, so the shadowing findings below stand
unchanged. Claude Code would snapshot bash's environment instead of zsh's, which
is only an improvement if that environment is bare — worth checking rather than
assuming.

**Therefore qwark verifies rather than assumes.** The parser's variant is a
precondition, and a precondition that is merely hoped for is a bug waiting for a
new machine. qwark declares the shell it parses for and refuses to run when the
environment says otherwise, on the same fail-closed reasoning as everything else
here.

