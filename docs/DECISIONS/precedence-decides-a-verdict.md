# Precedence decides a verdict, so order never matters

Where several rules match one command, the verdict is the one with the highest
precedence: block over allow over ask over deny. FR-4.32 and FR-4.33.

## Why

A block is a refusal nothing lifts. An allow lifts any number of denies, and
an ask lifts denies too, handing a narrow case to a person. Deleting a branch
is denied; deleting your own is asked; a task agent may be allowed to delete
task branches and nothing else.

Under strictest-wins, the alternative, a narrow exception has to be written into
the broad refusal as a clause excluding it. Every exception makes the refusal
longer, and a set of similar refusals become permutations of each other. Under
precedence the broad refusal is a `deny` and the exception is its own rule.

Precedence keeps what strictest-wins was for: **no verdict depends on where a
rule sits.** Precedence is a total order over actions, so a rule set assembled
from several files cannot be defeated by arranging for one to be read last.

## What it gives up, and what covers it

A rule able to override another puts that power in configuration, which sits
closer to the agent than a person does. Strictest-wins avoids that risk; here it
rests on the live rules being a separate copy the agent cannot write, which `permissions.deny` enforces for the file tools and
the hook for Bash. Anything that must hold whatever a later file says is a
`block`.

Each rule that refused under strictest-wins is a `block` under precedence, so
the two engines agree. Replaying 901 logged commands through both on 2026-09-28
gave 666 allowed and 235 refused under each, with no command changing verdict.
