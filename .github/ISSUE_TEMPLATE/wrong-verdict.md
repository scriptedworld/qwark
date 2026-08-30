---
name: A command was judged wrongly
about: qwark refused something it should have allowed, or allowed something it should have refused
---

**Do not report a bypass here.** A command that gets past a rule which should
have refused it goes through private vulnerability reporting instead. `SECURITY.md`
says what counts.

## The command

The exact command line, as typed.

## What qwark answered, and what you expected

Paste the verdict. If it came from the hook rather than from `judge`, say so.

    qwark judge <your rules> -- <the command>

## The rule set

Which files were loaded, and the output of:

    qwark rules <your rules>

If the rules are your own rather than the shipped set, the rule you believe
should have fired, or the one that fired and should not have.

## How qwark read the command

    qwark ast "<the command>"
    qwark facts "<the command>"

This is usually the whole answer: a verdict that surprises you almost always
comes from the command parsing into something other than what it looks like.

## Version and platform

The commit you built from, your operating system, and the shell Claude Code
actually invokes. `echo $0` inside the Bash tool, not the shell you use yourself.
