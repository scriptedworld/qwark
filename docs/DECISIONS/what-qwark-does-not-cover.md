# What qwark does not cover

Stated as a boundary, in the manner of the predecessor's own header, because a
control whose limits are not written down gets trusted past them.

**Other tools reach the filesystem without passing through qwark.** qwark gates
Bash. `Write` and `Edit` do not go through it, and they can reach the shell
snapshot, `settings.json`, `.git/hooks`, a `justfile`, or a rule file. Every path
qwark protects with a class-three rule needs the other half, a `permissions.deny`
entry, and that file must itself be unwritable.

**Some commands run other commands.** `env`, `xargs`, `sudo`, `nohup`, `timeout`,
`nice`, `watch`, `setsid`, `command`, `exec`, `eval`, `source`, and `find -exec`
all take a command as an operand, so a rule about `rm` never fires on
`env rm -rf x`. Interpreters do the same with inline code: `sh -c`, `python -c`,
`perl -e`, `node -e`, `awk 'BEGIN{system()}'`, GNU `sed` with `e`. These must
simply never be declared.

**Some commands run content the agent wrote.** `go test` executes test files the
agent authored; so do `pytest`, `npm test`, `cargo test`. `git commit` runs
`.git/hooks`. `just`, `make` and `npm run` execute recipes from files in the tree.
A coding agent that can write code and run its tests has arbitrary execution, and
no rule set changes that. What qwark constrains is what is typed, not what the
typed thing goes on to execute.

**A prefix assignment changes which binary runs.** With `PATH=<dir>:$PATH rm …`,
a script named `rm` in that directory ran instead. This is why prefix assignments
are refused, and it also means the undeclared-option denial is beside the point
once the program is not the declared one.
