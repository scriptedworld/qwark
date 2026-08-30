# qwark documentation

    INSTALLING.md   deploying it, registering the hook, and the log
    RULES.md        what a rule is, the declarations, the shipped set
    PROJECT.md      the layout, and how this repository is gated

`../REQUIREMENTS.md` states what must be true, and `../NEXT_STEPS.md` carries
what is not done and what is waiting on an answer.

## Decisions, `DECISIONS/`

The reasoning that would otherwise survive only in a commit message.

**The shape of a rule**
`why-a-parser-rather-than-a-matcher` ·
`the-decision-model` ·
`a-rule-is-a-conjunction` ·
`the-strictest-action-wins` ·
`a-clause-states-one-of-three-forms` ·
`addressing-a-command` ·
`nothing-is-expanded`

**What is refused, and why**
`tier-one-the-command-must-say-what-it-does` ·
`rules-are-layered-by-cost` ·
`writing-files-through-a-here-document` ·
`what-it-costs-to-detect-force` ·
`what-qwark-does-not-cover`

**Who may change the rules**
`rule-files-are-named-on-the-command-line` ·
`rule-files-must-not-be-writable-by-the-agent` ·
`a-declaration-is-a-permission` ·
`separation-of-duties-belongs-in-the-engine`

**Where this is going**
`the-end-state-is-three-layers` ·
`the-proxy-is-a-layer-and-mode-two-is-the-audit` ·
`intention-notes-belong-to-the-proxy` ·
`where-the-command-surface-is-heading`

**State, which is deferred**
`tags-have-lifetimes` ·
`the-leaking-bucket-has-no-honest-home-in-mode-one` ·
`redis-with-lua-and-the-update-is-what-ticks`

**Running it**
`configuration-is-toml-and-fails-closed` ·
`observability-and-withheld-environment-values`

## Lessons, `LESSONS/`

What the shell turned out to do, each found the hard way. These are what a gate
learns from having to read a command the same way the shell will, which is a
narrower thing than knowing how shells behave.

`zsh-executes-code-from-inside-a-glob`, the finding that settles the shell
question and the reason `no-glob` is in tier one ·
`no-spelling-of-a-command-is-immune-to-shadowing` ·
`aliases-reach-the-bash-tool` ·
`the-shell-is-zsh-and-the-decision-is-to-change-it` ·
`an-escape-defeats-a-path-rule` ·
`a-hook-change-takes-effect-immediately-and-can-lock-you-out` ·
`a-recursive-grep-here-cannot-see-your-working-files`

## Patterns, `PATTERNS/`

`the-mechanicals-the-shapes-a-rule-can-be-written-in`, the six shapes a rule
takes with the worked example for each, and the two that look right and do not
work. Read it before writing a rule.
