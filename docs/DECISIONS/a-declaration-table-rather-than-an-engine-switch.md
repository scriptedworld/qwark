# A declaration table, rather than a switch that makes declarations optional

> **The switch this rejected was later built, and is in use.** `06-allow.toml`
> carries `[declarations] required = false` and `accounted = false`, which is
> what makes the observation phase below possible after all.
>
> **What survives is the reasoning about the destination.** The table is still
> the route to a working gate, and the objection to the switch is still the
> reason it is temporary rather than a setting: it makes "qwark cannot account
> for this command" configurable, and that property is the whole of what qwark
> is. So the switch is documented in `06-allow.toml` with the way back written
> beside it, deleting the table restores FR-4.16 and FR-6.7 as written, and no
> shipped set is meant to keep it.
>
> **What was wrong here was the necessity, not the principle.** This file
> concluded that a structural-only phase could not exist. It can; it needed a
> code change and a ruling, and both happened.

## The plan this replaced

The intent was to arm qwark in stages: purely structural rules first, denying
only the shapes tier one is about, logging everything, then introducing rules
while watching what came through. Deny little, observe, treat the workarounds as
data.

That plan does not survive contact with FR-4.16. **An undeclared command is
refused by the engine**, at `internal/rules/evaluate.go`, unconditionally.
Omitting `05-declarations.toml` does not produce a gate that judges by shape
alone; it produces a gate that refuses everything, because the declaration check
fires before shape ever decides anything.

Measured against a rule set holding only `00-structure.toml` and one catch-all
allow rule:

    git add -N docs/PROJECT.md    deny  (engine) declared commands only  git
    rm -rf ~/.projects/qwark      deny  (engine) declared commands only  rm
    ls -la /etc                   deny  (engine) declared commands only  ls

The rule set was sound. `qwark rules` reported 15 rules, 14 deny and 1 allow,
and the allow rule provably fired the moment any declaration existed. The engine
simply answered first.

## The decision

**FR-4.16 stays as written, and the route to a working gate is a declaration
table.** The alternative on the table was a rule-set-level switch, something like
`[declarations] required = false`, gating that block so a set could opt out.

Rejected. The switch would make "qwark cannot account for this command" a
configurable opinion, and that property is the whole of what qwark is: when it
cannot account for something, it refuses. A gate whose confusion is the way
through it is the failure this project exists to avoid, and a switch is that
failure written as configuration rather than reached by accident.

Declaring commands is the work the design always implied. It was described early
as a smaller job than it looks, *"you don't use that many tools"*, and the
corpus bears that out once the words a native tool already replaces are taken
out of the count.

## What it costs, and this is the part that is not obvious

**Declaring a command means declaring every option it is used with.** FR-6.7
refuses an option the table does not name, so a half-declared command is refused
in exactly the shapes people actually type, which reads as the gate being broken
rather than as the table being incomplete.

Measured with `git` declared and nothing else changed:

    git add -N <path>       deny, -N undeclared
    git log --oneline -10   deny, -1 and -0 undeclared, because shorts bundle

So the unit of work is a command **plus its option set**, never a word. Eighteen
commands is not eighteen lines; it is a little over two hundred.

That cost is the reason the table stays deliberately short and fails closed.
Leaving an option out is not a gap, it is a refusal, and nobody has to enumerate
the dangerous flags in order to be safe from them.

## What was decided about which commands

Both tiers were declared. The nine where no native tool does the job (`git`,
`go`, `gofmt`, `rm`, `mkdir`, `cp`, `mv`, `readlink`, `qwark`), and the
read-only set that Grep, Read and Glob already replace (`ls`, `cat`, `head`,
`tail`, `wc`, `grep`, `find`, `sort`, `cut`).

**Declaring the second tier is a decision against the measurement**, taken
knowingly. The corpus says those are 32,462 occurrences, 36.1% of everything
Bash was asked to do here, and recommends the behaviour should stop rather than
be blessed. They are declared because a gate nobody can work behind gets
switched off, and because tier one still refuses the shapes that made them
dangerous: `grep -rn x .` runs, `grep x *.md` is a glob, and `grep x f | head`
is a pipe. Revisit when the proxy lands, which is the point at which the native
tools replace these rather than competing with them.

## Recorded first-hand

I heard this ruling directly, in the conversation where the structural-only plan
was abandoned, and am writing it here because a ruling that reaches other
sessions by relay is not recorded anywhere that binds.
