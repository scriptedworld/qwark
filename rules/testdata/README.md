# What the shipped rules decide

`internal/rules/cases_test.go` judges every file here and fails on a verdict
that differs, on a rule that should have fired and did not, and on a shipped
rule that no file exercises. A rule change shows up as a failing case before
it is installed.

    <set>/<verdict>/<name>.cmd

- `<set>` is `live`, the four files installed today, or `full`, all of
  `rules/`.
- `<verdict>` is `block`, `allow`, `ask` or `deny`, and every file in the
  directory has to get it.
- The file holds one command, as a session would send it.
- `<name>` up to any `--` is the rule that has to fire. For a tag rule it has
  to set or clear its tag instead. A name starting `default` or `engine` says
  no named rule decided it. What follows `--` tells cases apart.

Header lines before the command, each optional:

    # tags: post-rebase           tags live when the command is judged
    # fires: no-forcing-anything  further rules that have to fire
    # quiet: no-git-staging       rules that must not be among the reasons
    # cwd: /srv/project           where the call came from, this by default
