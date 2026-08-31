# Rule files must not be writable by the agent

> **REVERSED. The reasoning below is sound and the mechanism it chose was
> withdrawn.** FR-4.17, FR-4.17a and FR-4.17b are retired, `CheckOwnership` is
> deleted, and the live rule set is user-owned at `~/.config/qwark/rules`.
>
> **Why:** enforcing ownership means a root-owned live set, so every change to a
> rule needs a root command. A session cannot run `sudo`, so the gate could not
> be developed by the session it gates, and iterating on rules stopped being
> possible at all. That cost was paid in full on the day it was measured: the
> gate had to be wedged and unwedged by hand before a single rule could change.
>
> **What holds the property now**, and it is weaker on purpose, in three parts
> that were expected to be jointly sufficient and are not:
>
> the estate standing rule that an agent does not edit these files without
> a person in the conversation. Policy rather than mechanism, and currently
> carrying most of the weight.
>
> The live set remaining a separate deliberate copy rather than the source tree.
> This holds, and nothing compares the two, so a drift between them is undetected
> rather than prevented.
>
> The `permissions.deny` twin in the registration. **Measured, it does not stop
> a write to the live rules**: `rm` through Bash and `Write` through the tool
> both reached `~/.config/qwark/rules` past it. What it demonstrably does is
> remove the documented escape, since naming `settings.local.json` itself locks
> a wedged session out of unwedging, which needed a person to undo. So it bites
> in one direction and not the one it was relied on for.
>
> The Bash half is now covered by qwark's own `no-touching-qwark`, which had been
> guarding two abandoned paths while the live rules sat elsewhere. That is source
> only; this phase loads neither file.
>
> **The return condition**, recorded in `REQUIREMENTS.md` under `## Retired`: a
> deployment where qwark runs as a user other than the one being gated. Ownership
> becomes enforceable there without costing the ability to change a rule, because
> the two users are no longer the same user.
>
> Read the rest as the argument for a property still worth wanting, not as a
> description of what runs.

Rule files must be unwritable at runtime.

This is the finding the predecessor ended on, reached from the other direction.
`archive-guard.sh` concluded that a tool-layer gate cannot be the guarantee,
because it never sees a child process's syscalls. The mirror of that is that
qwark gates Bash, and the Write and Edit tools reach the filesystem without
passing through it. An agent that can edit the rules is not constrained by them,
and it needs no shell to do it.

So the control has to be filesystem ownership, exactly as `chmod 000` was for the
sealed tree: rule files owned by another user, not writable by the one the agent
runs as.

The directory matters as much as the files. A writable directory permits
unlink-and-replace, which defeats an unwritable file completely. Both are checked.

qwark verifies this at load and refuses to run otherwise, so the property holds
itself up instead of being a convention somebody has to remember after a fresh
clone or a `chmod -R` that meant well.
