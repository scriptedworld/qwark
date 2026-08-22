# A clause states one of three forms, and says which reading it tests

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 772-817. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**PREFERENCE, 2026-08-20.** A clause states what it tests for in one of
three forms, and may say which reading of the word it tests.

    value   = "rm"          the whole word, exactly
    partial = ".claude"     anywhere within the word
    pattern = "rm|rmdir"    a regular expression over the whole word

**Naming `partial` is the point of having three rather than two.** An earlier
draft here had only exact-and-pattern, with "contains" written `.*\.claude.*`.
That works, and it hides the breadth of the rule inside a regex the reader has
to parse. A form named `partial` announces itself in the key.

It also settles an argument the two-form design could not. `pattern` is anchored
to the whole value — otherwise every pattern is quietly a partial, and the broad
reading becomes the one obtained by accident. That is exactly the predecessor's
mistake: `archive-guard.sh` matched the substring `.archive` and blocked
`web.archive.org`, costing a legitimate research route. **Nothing here prevents
an author choosing that breadth** — `partial = ".archive"` does the same thing.
What changed is that choosing it is now a visible act rather than a default.

**Exactly one form per clause.** None is an error rather than a clause matching
everything; several is an error rather than a precedence order nobody would
remember. An empty `partial` is refused for the same reason — every string
contains the empty string. An empty `value` stands, being precise.

**The interpreted reading is the default.**

    reading = "interpreted"   what the shell will pass       (default)
    reading = "written"       the source, escapes intact

Testing what was written is what lets `/home/ancient/.cl\aude/x` past a rule
about `.claude`. The written reading is still worth having — a rule about *how*
something was spelled is a real thing to want, and the alias ban is one — but it
has to be asked for.

**A word with no interpreted value is not matched.** Nothing is expanded, so
`$HOME` has none; a clause reading it finds nothing to test rather than testing
the empty string, which `partial` and `.*` would both match.

**CLAIM, and a property worth keeping.** Go's regexp is RE2: no backtracking,
linear in the input. A rule file cannot carry a pattern that a crafted command
makes pathological. For a program that runs before every shell command that is
worth more than backreferences, which no clause about a command word needs.

