# A hook change takes effect immediately, and it can lock you out

Measured in the session that first ran with qwark armed.

## The question this answers

A session started after `.claude/settings.local.json` is written is gated from
its first command, and **a change to an existing registration takes effect on
the very next command**, with no restart and no approval.

That needed establishing because an earlier observation had suggested otherwise:
a registration was written, an `ls` ran normally afterwards, and arming appeared
not to take effect mid-session. The two cases are different. Creating a
registration where none existed is not the same as changing one, and treating
them as one claim shaped a whole task around restarting Claude Code to find out
something already answerable.

## How it was measured

Mid-session, the hook command was edited from

    qwark hook /etc/qwark/rules || exit 2

to name a different rules path. The next Bash call was refused by the new path's
policy, quoting a file that had not existed a minute earlier. Nothing was
restarted between the edit and the call.

## The second half, which cost more than the first

The same edit added the `permissions.deny` twin FR-10.10 asks for:

    "Edit(/home/user/.projects/qwark/.claude/settings.local.json)"
    "Write(/home/user/.projects/qwark/.claude/settings.local.json)"

That is correct in intent. An agent that can edit the file declaring its own
constraint has no constraint, and the deny closes it.

**It also removes the documented escape.** `PROJECT.md` said, and every note
here repeated, that getting out again needs no Bash: delete the `hooks` key with
the Edit tool. Once the deny names the settings file itself, that route is gone
for the session and available only to a person.

Both halves fired at once. The gate refused every Bash command, because the
binary on PATH still contained an ownership check the source had already
dropped, and the way to undo the registration was denied. The tree was wedged
until a person moved the file aside.

## What to do instead

**Rebuild before repointing.** Source is not the binary. A code change that
relaxes what the gate refuses does nothing until the binary the hook invokes
carries it, and the hook on this machine invoked `/usr/local/bin/qwark` while
the fixed build sat in `bin/qwark`.

**Change one of the two at a time.** Repoint the hook, confirm the gate still
answers, then add the deny twin. Adding both in one write means a failure in
either leaves no way back.

**Keep one lever outside the deny.** Either leave `settings.local.json`
editable and accept that an agent can lift its own gate, or deny it and accept
that only a person can unwedge the tree. Both are defensible; what is not
defensible is choosing the second without knowing it, because it reads as a
tightened bolt rather than as a door with no handle on this side.

**A moved registration needs a note.** A parked
`.ephemera/settings.local.json.wedged` is indistinguishable from a broken one to
the next session, which will read an ungated tree as normal.

## What it says about the gate more broadly

**A gate strict enough to be worth having is strict enough to block its own
development.** When this was written an armed qwark denied `qwark` itself, so
the project's own recommended way to try a rule before trusting it,
`qwark judge rules -- <cmd>`, was unavailable from inside the repository the
gate protects.

`allow-trying-a-rule` closes that case, on the grounds that qwark parses and
reports: what it is given is described rather than executed, and nothing it is
given is written. The general shape does not close with it. Every command the
project needs to build, test and gate itself is a command some rule has a good
reason to refuse, and each one has to be decided rather than assumed.
