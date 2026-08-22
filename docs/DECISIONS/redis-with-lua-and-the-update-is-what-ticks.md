# Redis with Lua, and the update is what ticks

Thought through in three steps:

> I'm considering redis … we can just use some embedded lua or such for
> gathering the tags per session ID.

> The goal of using Lua within Redis was that it makes the execution of the
> entire Lua script an atomic action.

> After we have a result and a list of tags, we can then issue the update
> command that then updates the state for the session … **the update is what
> ticks the counts.**

**Atomicity is the whole argument, and it deletes most of the discussion in
Tags have lifetimes.** A Lua script runs to completion inside Redis with nothing
interleaved, so reading the tags, applying the changes and decrementing every TTL
is one indivisible step. That removes the lock, the append-versus-compaction
split, and the worry about paying the expensive price on the common path. None of
those problems exist once the state transition is a single script instead of a
read-modify-write against a file.

**The update ticking, rather than the read, is the load-bearing choice.** It
makes FR-4.24 true by construction: a denied command issues no update, so it
advances no counter and sets no tag, and nothing has to remember to check the
verdict first. Had the read ticked, every refused command would have spent a
tick, and refused commands are the ones a constrained agent produces most.

Two things it does not settle.

**`ask` is unresolved, and qwark cannot resolve it.** FR-4.13 counts *allowed or
approved* commands. qwark returns `ask` and never hears whether the person
approved it, so at update time it cannot know which happened. Not ticking is the
safe direction, since the tag then lives longer than it strictly should, which is
more restriction and not less. The honest answer is that only mode two sees what
actually ran, so the tick for an approved command belongs with the audit and not
with the gate.

**The read-evaluate-update gap is not atomic, even though each step is.** Claude
Code can issue Bash calls in parallel, so a second call may read state taken
before the first call's update landed, and be judged without a tag that had just
been set. This is the unsafe direction, unlike a lost tick, and it is worth
saying plainly instead of being reassured by the word atomic. Closing it means
either serialising a session's calls, or optimistic concurrency: the read returns
a version, the update script refuses if the version moved, and qwark
re-evaluates.
