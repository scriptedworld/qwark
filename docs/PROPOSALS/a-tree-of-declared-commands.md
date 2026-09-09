# A tree of declared commands

**Status: proposal, for discussion. Nothing here is built and no rule file has
been changed.** Hard rule 4a wants a change to qwark's rules described and
agreed in words before it is written, and this is the description.

## What stays

The structural rules. `00-structure.toml` is deployed, works, and is not in
question: parsing a command into a syntax tree, establishing what its words are,
and refusing what cannot be parsed. Everything below sits on top of it.

## What is replaced

The declaration model. `05-declarations.toml`, `10-commands.toml`,
`20-paths.toml`, `30-options.toml` and `40-state.toml` are 1,778 lines that are
**not deployed**, because incomplete declarations denied everything.

That is not drift. It is a deliberate partial deployment: ship the half that
works, hold back the half that would stop every session. The live set is 391
lines and the two files in it are the structural rules and the allow rules.

**And the live set denies everything today.** Measured 2026-09-09 by replaying
1,938 distinct real commands from two hosts' decision logs:

    verdicts: {'deny': 1938}   100.0%

Including `qwark help`. The reason is in the refusal itself:

    (engine) declared commands only    This command has no declaration, so
                                       qwark cannot account for what its
                                       options mean or which of its words are
                                       paths.

So the deployed set is not "structural only". It is structural plus a
declaration requirement with no declarations. Anything that wires the hook up
has to resolve that first.

## The shape

A multi-branch tree used as an INDEX, not a parse. Navigate to a node, then look
each observed token up among that node's children. Order does not matter because
nothing is walked as a path.

    git
      commit
        -m [message]
        -F [file]
        -a
        [paths]
      push
        --force
      branch

`-m`, `-F` and `-a` are siblings: they are the parallel options of `git commit`.
`git commit -F x -a` and `git commit -a -F x` reach the same three lookups.

### A node carries more than its name

**Arity.** `-F [file]` consumes the next token; `-a` does not. Without it the
splitter cannot tell where a flag stops and its argument begins.

**Type**, which is the part that pays for itself twice over:

    path      resolve and canonicalise, then apply containment
    string    opaque, do not path-check it
    number    cheap sanity check
    enum      a fixed set of values
    pattern   a glob, so it may match more than it says
    program   an argument that IS a command

**Spelling collapse.** `-m`, `--message`, `--message=x` and `--message x` are one
child, or the tree grows a branch per typing habit.

### Option style is a property of the command

Bundling is a convention and enough real tools break it that it cannot be
assumed. Measured in the corpus:

    grep -rn, ls -la, git -C     posix bundling, and common
    grep -A12, git log -7        argument FUSED to the flag with no space
    go test -run X               Go's flag package: single dash, long name,
                                 and it does NOT bundle. -abc is one flag
    dd if=x of=y                 key=value
    tar xzvf                     bare first word, no dash

So `style` is one field on the command node, not a global rule and not a
property of each option.

## Verdicts

**Anything not listed is deny.** Absence is refusal.

Explicit deny nodes exist anyway, and they earn their place through the message
rather than the outcome. `CLAUDE.md` puts it exactly: a `reason` is what a
refused agent is shown, so it is the only thing standing between a denial and a
session that does not understand why.

    --force absent              "not declared"        the agent guesses
    --force: deny, with reason  names the hazard      the agent knows

## Types are what let path rules stop matching text

`docs/LESSONS/an-escape-defeats-a-path-rule.md` records that path rules reason
about the TEXT of a command, so `rm /home/user/.cl\aude/x` reaches `.claude` and
a rule matching the literal string does not see it.

Once a node says an argument is a path, qwark can unescape, expand and
canonicalise it before judging, because it knows which words deserve it. Today
every word looks the same.

**This is also the missing input to `os.Root`.** `go-because-of-os-root.md`
chose the language for kernel-enforced containment and says the feature is not
used yet. A root can only be opened on something known to be a path, and the
tree is what knows.

## Discovery is separated from execution

**A tool that finds things does not run things.** `find` produces a list; a list
goes into a file; a separate, judged step drives execution from that file. The
file between the two is the point: it can be read before anything acts on it,
which a pipeline cannot.

So a class of options is denied wherever it appears, because each one turns a
data tool into an execution vector:

    find      -exec -execdir -ok -okdir      run a command per result
              -delete                        act rather than report
              -fprintf -fls                  write an arbitrary file
    xargs     the whole command              its purpose is to execute
    sed       s///e                          GNU sed executes the pattern space
              -i                             writes in place rather than reporting
    git       -c alias.X=!cmd                config that runs a shell command
              -c core.pager=...
    ssh       host COMMAND                   execution, remotely
    docker    run, exec                      execution, in a container

The `find` list is not invented here: Claude Code's own Bash tool already
auto-allows `find` while blocking exactly `-delete`, `-exec`, `-execdir`, `-ok`,
`-okdir`, `-fprint*`, `-fls` and `-files0-from`. qwark agreeing with that is
consistency rather than a new policy.

The two non-obvious ones were confirmed against the real binaries here on
2026-09-09, because a rule written on a guess is worse than no rule:

    sed 's/x/echo RAN/e' file           printed RAN
    git -c alias.qq='!echo RAN' qq      printed RAN

Both execute. `sed` is worth dwelling on: it is in the read-only set of every
allowlist in this estate, including the one written into
`.claude/settings.json` today, and `s///e` makes it an execution vector. An
option-level rule is the only thing that catches that; a command-level allow
cannot.

The general rule the tree should carry: **an option whose effect is to run
something is denied on a command whose declared purpose is to report
something.** Where execution is genuinely wanted, it is a recipe, which is the
next section.

## The mechanical principle

**Anything to be generally executable goes behind a Justfile recipe.**

That makes the declared surface the recipe set rather than the command set.
Recipes are versioned, reviewed and gated code; a command line is improvised.
Adding a capability becomes a reviewable diff rather than a rule change.

**And the `just` branch generates itself.** `just --dump --dump-format json`
gives every recipe with its parameters and whether it is private, so that
subtree is derived per project and is never incomplete. The failure that put the
last model in a drawer cannot happen on the half of the tree that is generated.

## What the corpus says the tree needs

2,266 real invocations, from this host's decision log and oslo's mirror,
2026-08-28 to 2026-09-05.

    declared: just, bolt, git, and the read-only tools    1,412   62.3%
    left over                                               854   37.7%, 51 distinct

The leftover is four kinds, and only one of them is a design problem:

    sh 199, python3 151, bash 8     program-carrying, 16% of everything
    qwark 97                        your own tool, declare it
    docker, magick, omarchy, herdr  real tools with no recipe, about 150
    cp 39, rm 31, mv 17, mkdir 12   file operations, 107
    go 35, gofmt, golangci-lint,
      cargo                         58 calls, 2.6%, and these are the ones
                                    a recipe already wraps

**The compiler and linter calls are 2.6%.** Declaring `just` covers them.

## What this does not solve

**The 16% carrying a program.** `sh -c` and `python3 -c` are ad hoc by
definition and no recipe wraps them. The `program` type means qwark can recurse
into the argument and judge it with the same tree, which is better than treating
it as an opaque string, but it is the case that needs the most care.

**The recipes that do not exist.** `docker`, `magick`, `omarchy` and the rest
are a list rather than a design question, and they are the "piles of little
things" lost from the old system.

**Whether `verdict` inherits.** If `git commit` is allowed, is `git commit
--amend` allowed by default? Inheriting is usable; not inheriting is safe. Not
decided here.

## The order this can be built in

1. Resolve the deployed set so structure alone can allow, since nothing can be
   wired until it stops denying `qwark help`.
2. Generate the `just` and `bolt` subtrees from the Justfiles and jigs.
3. Declare `git`, `qwark`, and the read-only tools by hand. Small and stable.
4. Declare the file operations with path types, which is where containment
   starts paying.
5. The `program` type and recursive judgement, last, because it is hardest.
