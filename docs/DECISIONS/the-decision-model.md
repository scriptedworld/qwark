# The decision model

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 71-84. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


A rule's action is one of four:

    allow   auto-approve; the command runs without a prompt
    deny    block it
    ask     force the normal permission prompt
    tag     decide nothing; attach a name to the evaluation

`tag` is the one that makes the rest compose. A tag rule enriches the context
that later, more expensive rules match against, so cheap structural rules can
annotate and expensive rules can stay expressed over names rather than
re-deriving the tree.

