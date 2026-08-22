# Rules are layered by cost

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 85-98. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**PREFERENCE, 2026-08-19.** Four classes, cheapest first, so a decision
that can be reached cheaply is:

1. **Node presence.** Certain nodes in the tree are an instant rule.
2. **Conjunctive.** Several elements, *all* of which must match.
3. **Context.** Whether the paths involved fall inside a given directory tree.
4. **State.** An ongoing tracker across commands.

Class 1 costs nothing per rule: every structural fact is gathered in a single
walk (`internal/shell/facts.go`), so rules consulting them are set lookups and
adding one is free. Cost is set by the size of the tree, not the rule count.

