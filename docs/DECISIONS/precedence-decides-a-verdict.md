# Precedence decides a verdict, so order never matters

Where several rules match one command, the verdict is the one with the highest
precedence: block over allow over ask over deny. FR-4.32 and FR-4.33.

## Why

A block is a refusal nothing lifts. An allow lifts any number of denies, and
an ask lifts denies too, handing a narrow case to a person. Deleting a branch
is denied; deleting your own is asked; a task agent may be allowed to delete
task branches and nothing else.

Under strictest-wins, a narrow exception had to be written into the broad
refusal as a clause excluding it. Every exception made the refusal longer, and
a set of similar refusals became permutations of each other. Now the broad
refusal is a `deny` and the exception is its own rule.

What stays true is the point of the original: **no verdict depends on where a
rule sits.** Precedence is a total order over actions, so a rule set assembled
from several files cannot be defeated by arranging for one to be read last.

## What it gives up, and what covers it

The original held that a rule able to override another puts that power in
configuration, which sits closer to the agent than a person does. That is
still the risk, and it now rests on the live rules being a separate copy the
agent cannot write, which `permissions.deny` enforces for the file tools and
the hook for Bash. Anything that must hold whatever a later file says is a
`block`.

Every rule that was a `deny` under strictest-wins became a `block` when the
order changed, so no verdict moved. Replaying 901 logged commands through both
engines on 2026-09-28 gave 666 allowed and 235 refused under each, with no
command changing verdict.
