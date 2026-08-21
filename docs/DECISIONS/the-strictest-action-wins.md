# The strictest action wins, so order never matters

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 166-183. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**PREFERENCE, owner, 2026-08-19.** Where several rules match one command, the
verdict is the strictest of them: deny over ask over allow.

The property this buys is worth naming. **No rule can be weakened by where it
sits**, in its file or among the files. A rule set assembled from several
sources cannot be defeated by arranging for a permissive file to be read last,
and a reader establishing what a command will do never has to know what came
before. Under first-match-wins, every rule's meaning depends on all the rules
above it — which is the property DNF was chosen to avoid, reintroduced at the
level of the file.

The cost is real and should be stated: a narrow exception cannot override a
broad deny. An exception has to be written into the deny rule as a clause that
excludes it. That is more to type, and it keeps the exception where a reader of
that rule will see it, rather than in another file that quietly outranks it.

