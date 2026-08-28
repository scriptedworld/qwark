# Separation of duties belongs in the engine, not in the plumbing

The answer to an agent writing a `justfile` and then running `just` is not to make
the file unwritable. It is that **the agent which can write those files is not the
agent allowed to run them**:

> The point of the rules is using the engine to support that separation of
> duties.

And the reason it has to be the engine rather than the launcher:

> The base session doesn't get an "agent type" … so we can't as easily manage
> those rules without ACTIVELY managing symlinks or something else … so at the
> moment the concern is EITHER something wired into subagents, or some form of
> ENV VAR that will have to be actively managed … which feels rickety.

That settles a contradiction already in the requirements. **FR-10.6** says
`agent_type` arriving in the payload "is what makes per-agent scoping
implementable from the payload, rather than through an environment variable the
agent might itself reach". **FR-10.6a** then put the scoping outside qwark, on the
grounds that an external process picks the files. Both cannot stand: the external
route is precisely the env-var and symlink management called rickety above, and
the subject cannot set its own `agent_type` while it can reach an environment
variable.

**Resolution: FR-10.6 stands, and the external route FR-10.6a rested on does
not.**

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

**One rule set, named once in `settings.json`, carrying every role's policy inside
it.** No symlink swapping between launches, no environment variable to manage, and
nothing to keep in step outside the file being read. The policy in force stays
readable where qwark is invoked, which is what FR-4.15 was for.

It composes with everything already here instead of adding a mechanism. An `agent`
clause is a clause: rules stay conjunctions, the strictest action still wins, and
a role cannot grant itself anything, because deny outranks allow and no rule can
override another.

### What this does not fix

**Two main sessions are indistinguishable.** If both the writer and the runner are
top-level launches, they carry no agent type and no clause can tell them apart, so
the launcher must still give them different rule files. Engine-side scoping solves
the subagent case completely and the main-session case not at all. Take it as an
argument for the specialised agents being subagents instead of separate launches.

**A partition does not stop a chain.** Writer writes the `justfile`, runner runs
it: two agents, neither breaking its own rules, and the effect composes into the
attack the partition was meant to prevent. Separation of duties is only a control
if something sits between the two.

That something is the task management process, the same process that produces the
manifest of FR-9.7. It sees what the writer changed before any runner is
dispatched against it. So the manifest and the partition are not alternatives; the
manifest is what makes the partition mean something.
