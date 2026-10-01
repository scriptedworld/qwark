# Why a parser rather than a matcher

The predecessor is `claude/hooks/archive-guard.sh`, from a dotfiles tree that no
longer exists: a `grep -E` over the raw command string. This record is the only
surviving copy of the failure its own header described.

> `bin/repos status` in the dotfiles repo walked from a configured root,
> reached the tree and enumerated it, and never contained the literal string —
> so this hook passed it.

The regex had no bug, and no regex fixes it. The hook was asked what
the command would *do* and answered what the command *said*. Those are the same
question only for a command whose effect is fixed by its own text, and shell
syntax exists largely to break that correspondence.

So the gate works on structure. `mvdan.cc/sh/v3/syntax` gives a typed tree, needs
no cgo, and round-trips.

Round-tripping is not a property this gate uses: it judges a command and never
writes one back out. No-cgo holds for as long as qwark is Go, and it stays Go;
`go-because-of-os-root.md` says why.

tree-sitter-bash is the alternative. `what-the-two-parsers-actually-give.md`
measures the two parsers against each other: tree-sitter-bash does not build a
better tree for what qwark judges, and is worse in three places that carry deny
rules.

The rest of this record is about parsing versus matching and not about which
parser. The predecessor's failure, the glob hybrid, and the limit that a
tool-layer gate cannot see a path named at runtime are all parser-independent.

## One tier-one rule is not a tree question, and it is a hybrid, not text

`no-glob` selects `fact = "glob"` and no node type, because a wildcard has no
node of its own. That is true of `mvdan.cc/sh` and of tree-sitter-bash alike:
`cat rules/*.toml` carries one word whose text happens to contain a
metacharacter.

The tree still does the half that matters. `recordGlob` walks a `Word`'s own
parts and tests only the literal ones, so quoting decides the answer:

    grep *.toml        glob
    grep "*" notes.txt no glob, and the command is allowed

A predicate over the whole word's characters gets the first right and the second
wrong. The tree is what says which characters are eligible to be tested, and
`pattern.HasMeta` then answers over those. So the rule divides as structure
choosing the subject and text answering the question, which is also the shape
the path and filename rules take.

The decision log exercises this constantly. Commands like
`find internal -name '*_test.go'` and `grep -oE 'FR-[0-9]+\.[0-9]+[a-z]?' …`
carry a metacharacter inside quotes and establish no glob fact, so the
distinction is load-bearing on ordinary traffic and not only on a constructed
case. A replay against an independent implementation that also tests only the
literal parts reproduced every verdict in the log, which is agreement on a
corpus that does separate a whole-word predicate from a parts-aware one.

It still does not make the gate a guarantee. The archive-guard header's
conclusion holds: a tool-layer gate cannot stop a program that names a
path only at runtime, because the denial happens before the child process exists.
qwark catches the explicit case early and says why. Where a real boundary is
needed, it belongs in the filesystem.
