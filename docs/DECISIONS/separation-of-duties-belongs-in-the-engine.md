# Separation of duties belongs in the engine, not in the plumbing

The answer to an agent writing a `justfile` and then running `just` is not to make
the file unwritable. It is that the agent which can write those files is not the
agent allowed to run them:

> The point of the rules is using the engine to support that separation of
> duties.

And the reason it has to be the engine and not the launcher:

> The base session doesn't get an "agent type" … so we can't as easily manage
> those rules without ACTIVELY managing symlinks or something else … so at the
> moment the concern is EITHER something wired into subagents, or some form of
> ENV VAR that will have to be actively managed … which feels rickety.

FR-10.6 says `agent_type` arriving in the payload "is what makes per-agent
scoping implementable from the payload, rather than through an environment
variable the agent might itself reach". FR-10.6a puts the scoping in the engine
for the same reason. Scoping from outside qwark, with an external process picking
the files, is precisely the env-var and symlink management called rickety above,
and the subject cannot set its own `agent_type` while it can reach an
environment variable.

### Absence is a role

A main-session call carries no `agent_type`, so identity cannot simply be looked
up. No identity is itself an identity: the main session is the one caller with no
agent type, reliably and by construction, so a rule can name that case exactly as
it names any other. The schema already has the spelling, because `absent = true`
is how a clause says "this is not there".

    [[rule.clause]]          # applies to a subagent of this type
    agent = "gate-runner"

    [[rule.clause]]          # applies to the main session, and only it
    agent  = ""
    absent = true

One rule set, named once in `settings.json`, carries every role's policy inside
it. Nothing outside the file being read needs keeping in step: no symlinks
swapped between launches and no environment variable to manage. The policy in force stays
readable where qwark is invoked, which is what FR-4.15 was for.

It composes with everything already here instead of adding a mechanism. An `agent`
clause is a clause: precedence still decides, and a role cannot grant itself
past a block, because nothing outranks one. Against a deny, an allow scoped to
a role is exactly how one role is given what another is not.

### What this does not fix

**Two main sessions are indistinguishable.** If both the writer and the runner are
top-level launches, they carry no agent type and no clause can tell them apart, so
the launcher must still give them different rule files. Engine-side scoping solves
the subagent case completely and the main-session case not at all. Take it as an
argument for the specialised agents being subagents, not separate launches.

**A partition does not stop a chain.** Writer writes the `justfile`, runner runs
it: two agents, neither breaking its own rules, and the effect composes into the
attack the partition was meant to prevent. Separation of duties is only a control
if something sits between the two.

That something is the task management process, the same process that produces the
manifest of FR-9.7. It sees what the writer changed before any runner is
dispatched against it, so the manifest is what makes the partition a control.
