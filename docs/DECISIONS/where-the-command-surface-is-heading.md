# Where this is heading, and it is not more git rules

Asked which git commands should be reclaimed as guarded allows, the answer was
none, and the reason resets the target:

> Eventually the goal is that the agents will **never** be executing git
> commands, and instead have a very specific command surface they are allowed.
> The more specialised the agent, the more the list can be narrowed. One would
> expect that either the list ends up a duplication of a series of allowed
> commands in the agent text or supporting files, or those end up referencing
> these rule files.

Three things follow.

**The allowed surface is per agent, not per machine.** That is already the
mechanism: rule files are named on the command line (FR-4.15), and an external
process chooses which files a given agent gets (FR-10.6a). Narrowing by
specialisation needs no new machinery, only more files.

**The read-only git allowance is a waypoint.** It stands because it was ruled on,
and the direction above says the eventual answer is narrower. It is the first
thing to remove once the specific surfaces exist.

**The duplication was the open question, and the proxy settles it.** An agent's
prompt saying what it may run and a rule file deciding what it may run are two
statements of one fact, and two statements of one fact drift. Generating the
prompt from the rules and having the prompt reference them were both on the table,
and the answer came out a third way:

> The proxies then ALSO hold the details on what they expose, meaning they
> include the details on what those agents can do … so we aren't repeating those
> details any more (once we're there).

One artifact does both jobs: **the exposed surface is simultaneously the statement
of what an agent may do and the thing that enforces it.** A capability list that
is documentation somewhere and configuration somewhere else has a wrong version; a
tool that is simply not exposed has no second version to be wrong.

**How much it settles depends on how narrow the tools are**, and that is knowable
in advance. A proxy exposing `run_command(cmd)` is Bash with extra steps and moves
nothing. A proxy exposing `git_log(ref)` still needs something to say which `ref`,
because a surface says *which operations* and not *with what values*. So the
duplication disappears to the extent the tools are narrow, and whatever
argument-level constraint remains is what rules are still for.
