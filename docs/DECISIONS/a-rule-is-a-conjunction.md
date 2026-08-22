# A rule is a conjunction, and `or` is spelled with more rules

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 148-165. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**PREFERENCE, 2026-08-19.** A rule is several clauses, *all* of which must
match. There is no disjunction inside one. Where alternatives are wanted, they
are written as separate, nearly identical rules.

The worked example, in the words used at the time:

    rm -r -f     forbidden, because of -f
    rm -r        no -f: ask, warning that it is recursive
    rm -f        no -r: forbidden

That is disjunctive normal form, chosen for the same reason the archive-guard
regex was kept blunt: **a rule that can be checked by reading it alone is worth
more than a compact one that cannot.** The cost is duplication between sibling
rules, which is visible. The cost of the alternative is a reader having to hold
a boolean expression in their head to know what a rule does, which is not.

