# Why a parser rather than a matcher

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 44-70. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**FACT 2026-08-19**, read from `~/.projects/dotfiles`, commit `0e6ea10`: the
predecessor was `claude/hooks/archive-guard.sh`, a `grep -E` over the raw
command string, retired on 2026-08-14. Its own header records the failure,
demonstrated 2026-08-02:

> `bin/repos status` in the dotfiles repo walked from a configured root,
> reached the tree and enumerated it, and never contained the literal string —
> so this hook passed it.

**CLAIM.** That is not a bug in the regex, and no regex fixes it. The hook was
asked a question about *what the command would do* and answered a question
about *what the command said*. Those are the same question only for commands
whose effect is determined by their own text — and shell syntax exists largely
to break that correspondence.

So the gate has to work on structure. `mvdan.cc/sh/v3/syntax` gives a typed
tree, no cgo, and round-trips; tree-sitter-bash was the alternative and loses
on both counts.

**It still does not make the gate a guarantee.** The archive-guard header's
conclusion stands unchanged: a tool-layer gate cannot stop a program that names
a path only at runtime, because the deny happens before the child process
exists. qwark catches the explicit case early and says why. Where a real
boundary is needed, it belongs in the filesystem.

