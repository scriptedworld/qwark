# The shell is zsh, and the decision is to change the shell

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 363-383. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**FACT 2026-08-20.** The tool is called Bash and runs **zsh 5.9**: `$0` is
`/bin/zsh`, `BASH_VERSION` is unset, `ZSH_VERSION` is 5.9, and Claude Code's
snapshots are named `snapshot-zsh-*.sh`. qwark had been parsing with `LangBash`
on the strength of the tool's name, which is exactly the kind of inference this
project exists to stop making.

**The mismatch is silent, which is what makes it serious.** Measured over ten
zsh constructs, the bash parser rejected two and parsed eight — and four of
those eight mean something different in zsh:

    **/*.go        two globs in bash; recursive descent in zsh
    *(.)           an extglob group in bash; "regular files only" in zsh
    $foo[2]        $foo then literal [2] in bash; an array element in zsh
    noglob rm *    a command named noglob in bash; a zsh precommand modifier
                   that leaves the glob unexpanded

A rejection is harmless here, because the gate denies what it cannot parse. A
construct that parses under the wrong grammar produces a confident wrong verdict.

