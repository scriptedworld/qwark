# A rule is a conjunction, and `or` is spelled with more rules

A rule is several clauses, *all* of which must match. There is no disjunction
inside one. Alternatives are written as separate, nearly identical rules.

The worked example:

    rm -r -f     forbidden, because of -f
    rm -r        no -f: ask, warning that it is recursive
    rm -f        no -r: forbidden

That is disjunctive normal form, chosen for the reason the archive-guard regex
was kept blunt: **a rule you can check by reading it alone beats a compact one
you cannot.** The cost is duplication between sibling rules, and that cost is
visible. The alternative costs a reader holding a boolean expression in their
head to know what a rule does, and that cost is not.
