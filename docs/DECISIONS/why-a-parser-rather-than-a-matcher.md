# Why a parser rather than a matcher

The predecessor was `claude/hooks/archive-guard.sh` in a dotfiles repository
since retired: a `grep -E` over the raw command string. That tree has been
deleted, so what follows is the only surviving record of the failure its own
header described.

> `bin/repos status` in the dotfiles repo walked from a configured root,
> reached the tree and enumerated it, and never contained the literal string —
> so this hook passed it.

That is not a bug in the regex, and no regex fixes it. The hook was asked what
the command would *do* and answered what the command *said*. Those are the same
question only for a command whose effect is fixed by its own text, and shell
syntax exists largely to break that correspondence.

So the gate works on structure. `mvdan.cc/sh/v3/syntax` gives a typed tree, needs
no cgo, and round-trips; tree-sitter-bash was the alternative and loses on both
counts.

It still does not make the gate a guarantee. The archive-guard header's
conclusion stands unchanged: a tool-layer gate cannot stop a program that names a
path only at runtime, because the denial happens before the child process exists.
qwark catches the explicit case early and says why. Where a real boundary is
needed, it belongs in the filesystem.
