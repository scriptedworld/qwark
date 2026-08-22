# qwark, why it is built this way

The reasoning behind qwark, one topic per file under `docs/`. What qwark is and
what it is for is in [`docs/PROJECT.md`](docs/PROJECT.md). `REQUIREMENTS.md`
states what must be true, and stays a single file while the traceability checker
takes a single path, for the reason under *The split is pending* in
`docs/PROJECT.md`. Open questions are in `NEXT_STEPS.md`.

## Decisions, `docs/DECISIONS/`

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

## Lessons, `docs/LESSONS/`

What the shell turned out to do, each found the hard way.

`zsh-executes-code-from-inside-a-glob`, the finding that settles the shell
question and the reason `no-glob` is in tier one ·
`no-spelling-of-a-command-is-immune-to-shadowing` ·
`aliases-reach-the-bash-tool` ·
`the-shell-is-zsh-and-the-decision-is-to-change-it` ·
`an-escape-defeats-a-path-rule`

Some of these are general, and their general form lives in silo:
`silo/docs/LESSONS/cli-aliases.md` carries what is true of any shell on this
machine. What is here is what only qwark's work teaches, which is what it means
to be a gate that has to read a command the same way the shell will.

## Patterns, `docs/PATTERNS/`

`the-mechanicals-the-shapes-a-rule-can-be-written-in`, the five shapes a rule
takes with the worked example for each. Read it before writing a rule.
