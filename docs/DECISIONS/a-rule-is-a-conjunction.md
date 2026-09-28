# A rule states how its clauses combine, and all is the default

Superseded 2026-09-27. A rule carries `match`, `all` or `any`, defaulting
to `all`. FR-4.30 and FR-4.30a.

## What changed

Conjunction only meant a rule refusing one of several unlike things had to be
written once per thing. Similar rules became permutations of each other, and
permutations drift. `match = "any"` states the alternatives in one rule.

`all` stays the default, so every rule written before `match` existed still
means what it says.

`any` is refused at load on `allow` and `untag`. Under `any` each clause fires
the rule by itself, so one clause written too broadly is enough. On a refusal
that refuses too much, which fails safe. On a permit it permits too much, which
does not.

## What it replaced

A rule was several clauses, *all* of which had to match, with alternatives
written as separate, nearly identical rules:

    rm -r -f     forbidden, because of -f
    rm -r        no -f: ask, warning that it is recursive
    rm -f        no -r: forbidden

That was chosen because a rule you can check by reading it alone beats a
compact one you cannot. `any` keeps that property for a flat list of
alternatives. What it still leaves out is a mix, such as "git, and `-C` or
`--git-dir`": a flat `any` would let the `git` clause fire alone. Where the
alternatives are the same kind of clause, a group expresses them inside one
clause instead. Nesting is left out until a real rule needs it.
