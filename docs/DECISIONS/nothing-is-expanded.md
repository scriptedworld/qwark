# Nothing is expanded

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 748-771. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**Owner, 2026-08-19:** *"we don't expand anything, that's why we block those."*

A word's value is reported only where its own text fixes it. Anything containing
a substitution is reported as undetermined, never guessed.

**FACT 2026-08-19, measured, and the reason this is hand-written rather than
delegated.** `expand.Literal` with a nil config refuses command substitution
properly — it does not execute anything — but it resolves `$HOME` to the empty
string and returns *no error*:

    $(echo EXECUTED)   err="unexpected command substitution"   (safe)
    $QWARK_PROBE       value=""   err=<nil>                    (silent)
    $((2+2))           value="4"  err=<nil>                    (silent)

A caller cannot tell a fixed word from one that was quietly guessed at. Reasoning
about `rm -rf /x` while the shell acts on `rm -rf /home/ancient/x` is the same
class of error as the predecessor's, reached by a different route.

**Environment variables are eliminated from command lines too** (owner,
2026-08-19), so the substitution ban covers parameter expansion — `$HOME`,
`$PWD` and the rest — and not only the other three.

