# What the shipped rules decide

`qwark cases rules/testdata` judges every file here and prints a result
envelope. `just rules-suite` runs it as part of `just checks`. A run fails on a
case that gets the wrong verdict, on any rule a set includes that no case
triggers, and on any rule no case holds quiet, each named as a reason. So a
rule added without a case fails the gate, and so does one added without a
near miss: a case naming it under `quiet:` whose command sits beside the one
the rule exists for, the same command with the tag unset, the word as an
operand, a quoted literal, a neighbouring subcommand. Nothing checks how near
the miss is; that is for whoever reads the case.

    <set>/set.txt
    <set>/<verdict>/<name>.cmd

- `<set>/set.txt` names the set's rule files, one per line, relative to the
  set's directory. `live` is what `~/.config/qwark/rules` holds; `full` is all
  of `rules/`.
- `<verdict>` is `block`, `allow`, `ask` or `deny`, and every file in the
  directory has to get it.
- The file holds one command, as a session would send it.
- `<name>` up to any `--` is the rule that has to fire. For a tag rule it has
  to set or clear its tag instead. `engine` says the engine itself refuses it,
  and `default` says no rule decided it. What follows `--` tells cases apart.

Header lines before the command, each optional:

    # tags: post-rebase           tags live when the command is judged
    # fires: no-forcing-anything  further rules that have to fire
    # quiet: no-git-staging       rules that must not be among the reasons
    # cwd: /srv/project           where the call came from, this by default
