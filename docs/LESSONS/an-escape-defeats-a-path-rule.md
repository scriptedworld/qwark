# An escape defeats a path rule, unless it is resolved

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 506-513. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**FACT.** The parser keeps escapes in a literal's value, so `a\ b` arrives as
`a\ b` while bash passes `a b`. Written out, `rm /home/ancient/.cl\aude/x`
reaches `.claude` and a rule comparing the unresolved text does not match. The
resolution rules differ inside double quotes from outside, and both were read
off bash rather than recalled.

