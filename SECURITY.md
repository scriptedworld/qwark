# Security policy

## Reporting a vulnerability

Report it privately, through GitHub's private vulnerability reporting on this
repository, rather than by opening an issue. A bypass is worth more to an
attacker than to anyone else while it is public.

Include the command line, the rule set you were running, and the verdict you
got. `qwark ast` and `qwark facts` output for the command is the fastest thing
to read.

## What counts as a vulnerability

Anything that gets a command past a rule that should have refused it.

- A parse that differs from what the shell will actually do, so a rule looks at
  something other than what runs.
- An option that decomposes to the wrong meaning, or not at all, so a rule
  naming that meaning does not fire.
- A quoting or escaping case where a path rule fails to see the path.
- A clause that loads and is then silently ignored, since a rule nobody
  enforces reads as protection that is not there.
- An input to `qwark hook` that ends in anything other than a decision on
  stdout or exit 2. Every other exit lets the command run.
- A rule file that loads when it should have been refused, or a load failure
  that does not stop Bash.

## What is not a vulnerability

These are the documented limits, and
`docs/DECISIONS/what-qwark-does-not-cover.md` states them in full. A control
whose limits are not written down gets trusted past them.

**qwark gates Bash and nothing else.** The Write and Edit tools reach every path
a rule protects without passing through it. That half is covered by the
`permissions.deny` list in the registration, which is maintained by hand and is
wrong wherever it is incomplete.

**An agent that can write a file and run its tests has arbitrary execution.**
`go test` runs code written a moment ago, and `just`, `make` and `npm run`
execute recipes from files in the tree. qwark constrains what is typed, not what
the typed thing goes on to execute.

**Some commands run other commands.** `env`, `xargs`, `sudo`, `timeout`,
`find -exec` and every interpreter take a command as an operand. The answer is
that they are never declared, so they are refused for being unaccountable. A
deployment that declares one has widened its own surface deliberately.

**A rule set that judges by shape alone is not containment.** With
`required = false`, any command word runs if its shape is clean. The two
structural files are shipped that way on purpose, as an observation phase.
`docs/RULES.md` shows both answers side by side.

**A failure to record does not change a verdict.** Somebody who can fill the
disk can stop the logging without stopping the commands. That direction is
chosen deliberately, because the alternative makes a full disk a way to stop
every command on the machine.

## What a deployment has to hold up

qwark assumes all of this, and cannot check most of it.

The rule files are not writable by the user the agent runs as, and neither is
the directory holding them, because a writable directory permits unlink and
replace.

The registration is not writable by that user either, and every path a rule
protects has a `permissions.deny` twin. Otherwise the rule is enforced against a
shell and against nothing else.

The hook command ends in `|| exit 2`. Without it, a qwark that crashes lets the
command through.

The shell qwark parses for is the shell that will run the command. The mismatch
is silent rather than loud: of ten zsh constructs the bash parser rejects only
two, while four parse cleanly and mean something else. qwark verifies the
declared shell against what the environment reports, which is worth exactly as
much as that environment being unwritable.
