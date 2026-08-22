# The proxy also asks for intention, and that is the proxy's job

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 708-747. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**2026-08-20:** *"the point of the proxy is also to help include that
we're asking each tool usage to include notes regarding the intention"* — and,
asked whether to build an intention clause into the Bash gate: **"No, the intent
is for the PROXY."**

So this is a scope boundary rather than a feature waiting to be written. It is
recorded here for when the proxy is built, and because two things about it are
easy to get wrong.

**An intention is the one input the subject authors freely.** Everything mode
one reads is fixed by the shell grammar, assigned by the dispatcher
(`agent_type`), or read off the filesystem (ownership, resolved paths). A note
saying why is unverified prose written by the thing being judged.

**So it may require, record and shape-check — and it must never permit.** A rule
that allows because the note says "running the standard gate" is defeated by
typing that string, which is `[command.sh]` again in prose. Requiring one is
safe for the opposite reason: nobody defeats a requirement by satisfying it
honestly. This is the same reasoning FR-10.6 used to prefer `agent_type` over an
environment variable, at full strength.

**A plausible note can launder a bad action.** If whatever reviews these reads
them credulously, the note is worse than no note, because it supplies a
ready-made justification. The value is in comparing the stated intention against
what the call actually did — the mismatch is the signal, and the note on its own
is not evidence of anything.

What it buys is worth the care: FR-4.8's log stops being a list of what happened
and becomes checkable, both sides of every decision get recorded — qwark's
refusals already state a reason, and this makes the request state one too — and
the task management process gets the material it needs to sit between the writer
and the runner.

**Mode one is not the place for it, and not because it could not.** FACT
2026-08-20: Claude Code's Bash tool input carries a `description` field, and
`hook.BashCall` already parses it. It is deliberately unconsulted — kept because
reading the payload faithfully is FR-10.1, not left as an oversight.

