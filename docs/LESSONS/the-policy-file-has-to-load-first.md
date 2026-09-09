# The policy file has to load first

`[declarations] required = false` turns off FR-4.16, the refusal of an
undeclared command. It only works if the file carrying it is loaded **before**
the rules it governs. Nothing said so, nothing detected it, and the live rule
set denied 100% of real commands for eleven days because of it.

## Measured

Two files, same content, opposite verdicts:

    00-structure.toml then minimal.toml    ls -la  ->  DENY
    minimal.toml then 00-structure.toml    ls -la  ->  ALLOW

`qwark rules` reports the same 15 rules either way, so nothing is dropped. The
policy is consulted during a sequential load, so a file read after the rules it
governs cannot govern them.

The deployed set was `00-structure.toml` and `06-allow.toml`. Numeric filename
order put the switch second, always, and the whole gate was unusable.

Replaying 1,938 distinct real commands from two hosts' decision logs:

    before   deny 1938   100.0%     including `qwark help`
    after    deny   30     1.5%     pipes, substitution, cd-chains

## The diagnosis that was written down was wrong

`internal/rules/declarations.go` records the symptom accurately and explains it
as a property of the engine's phases: the declaration check *"arrives before
shape decides anything"*, so a structural-only set *"refuses every command
rather than judging the ones it understands"*.

That reads as though only declarations can fix it. Swapping two filenames fixes
it entirely. The comment describes a real measurement from 2026-08-28 and draws
a conclusion the measurement does not support, which is why it survived: it was
right about what happened and wrong about why.

## What was done

The files are renumbered so the policy sorts first, in both copies:

    00-allow.toml       was 06-allow.toml, carries [declarations]
    01-structure.toml   was 00-structure.toml

A rename rather than a code change, deliberately. It is correct under a glob,
which an explicit argument order in the hook registration would not be: anyone
running `qwark judge ~/.config/qwark/rules/*.toml` gets the right answer.

## The general shape, which outlives this fix

**A policy that any file may set, applied during a sequential load, is order
dependent unless the load is two passes.** Gather every file's policy, then
evaluate. One pass cannot do it, because the first rule is judged before the
last file is read.

`load.go` says the declarations table is *"claimed like any other definition, so
two files cannot each say something about whether declarations are required and
leave the answer depending on which was read last."* Claiming stops two files
disagreeing. It does not stop one file being read too late to matter, and that
is the failure that happened.

Anything reimplementing this gate should load in two passes and make the
question unaskable.
