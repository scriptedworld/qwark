# Nothing is expanded

*"we don't expand anything, that's why we block those."*

A word's value is reported only where its own text fixes it. Anything containing
a substitution is reported as undetermined, never guessed.

This is hand-written rather than delegated, and one measurement says why.
`expand.Literal` with a nil config refuses command substitution properly, in that
it executes nothing, but it resolves `$HOME` to the empty string and returns *no
error*:

    $(echo EXECUTED)   err="unexpected command substitution"   (safe)
    $QWARK_PROBE       value=""   err=<nil>                    (silent)
    $((2+2))           value="4"  err=<nil>                    (silent)

A caller cannot tell a fixed word from one that was quietly guessed at. Reasoning
about `rm -rf /x` while the shell acts on `rm -rf /home/user/x` is the
predecessor's class of error, reached by a different route.

Environment variables are eliminated from command lines too, so the substitution
ban covers parameter expansion, `$HOME` and `$PWD` and the rest, and not only the
other three.
