# The proxy also asks for intention, and that is the proxy's job

*"the point of the proxy is also to help include that we're asking each tool
usage to include notes regarding the intention"*. Asked whether to build an
intention clause into the Bash gate: **"No, the intent is for the PROXY."**

A scope boundary, recorded here for when the proxy is built.

**An intention is the one input the subject authors freely.** Everything mode one
reads is fixed by the shell grammar, assigned by the dispatcher (`agent_type`), or
read off the filesystem (ownership, resolved paths). A note saying why is
unverified prose written by the thing being judged.

**So it may require, record and shape-check, and it must never permit.** A rule
that allows because the note says "running the standard gate" is defeated by
typing that string, which is `[command.sh]` again in prose. Requiring one is safe
for the opposite reason: nobody defeats a requirement by satisfying it honestly.
This is the reasoning FR-10.6 used to prefer `agent_type` over an environment
variable, at full strength.

**A plausible note can launder a bad action.** If whatever reviews these reads
them credulously, the note is worse than no note, because it supplies a ready-made
justification. The value is in comparing the stated intention against what the
call actually did. The mismatch is the signal, and the note on its own is not
evidence of anything.

What it buys: FR-4.8's log stops being a list of what happened and becomes
checkable; both sides of every decision get recorded, since qwark's refusals
already state a reason and this makes the request state one too; and the task
management process gets the material it needs to sit between the writer and the
runner.

Mode one could do it and does not. Claude Code's Bash tool input carries a
`description` field, and `hook.BashCall` already parses it. It is deliberately
unconsulted, kept because reading the payload faithfully is FR-10.1.
