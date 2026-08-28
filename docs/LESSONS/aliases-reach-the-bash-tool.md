# The containment surface, and where it ends

Findings from an adversarial pass.

### Aliases reach the Bash tool, and they are not allowed

The Bash tool's shell is a login, non-interactive zsh: `$-` is `569JOXYl`,
carrying no `i`, and `[[ -o interactive ]]` is off. It therefore never reads
`.zshrc`, and `.zshenv`, `.zprofile` and `.zlogin` do not exist on this machine.

**The aliases arrive anyway, from Claude Code's own shell snapshot**,
`~/.claude/shell-snapshots/snapshot-zsh-*.sh`, which replays them:

    3: unalias -a 2>/dev/null || true
  449: alias -- ls='eza --long --header --icons --git …'

So a rule about `ls` would describe a program that is not being invoked, and seven
options qwark never sees would be added to every call. That is the `ss`→`rg`
hazard the standing rules already record, now load-bearing for a security control.

**Using an alias is not allowed.**
