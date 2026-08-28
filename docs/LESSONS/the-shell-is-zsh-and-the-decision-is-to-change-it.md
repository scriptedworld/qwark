# The shell is zsh, and the decision is to change the shell

The tool is named Bash and runs **zsh 5.9**. `$0` is `/bin/zsh`, `BASH_VERSION`
is unset, `ZSH_VERSION` is 5.9, and Claude Code names its snapshots
`snapshot-zsh-*.sh`. qwark parsed with `LangBash` for a while on the strength of
the tool's name.

**The mismatch is silent.** Put ten zsh constructs through the bash parser and it
rejects two. Of the eight it accepts, four mean something else in zsh:

    **/*.go        two globs in bash; recursive descent in zsh
    *(.)           an extglob group in bash; "regular files only" in zsh
    $foo[2]        $foo then literal [2] in bash; an array element in zsh
    noglob rm *    a command named noglob in bash; a zsh precommand modifier
                   that leaves the glob unexpanded

Being rejected is harmless, because the gate denies whatever it cannot parse.
Parsing under the wrong grammar is the dangerous half: it produces a verdict, and
the verdict is confidently wrong.
