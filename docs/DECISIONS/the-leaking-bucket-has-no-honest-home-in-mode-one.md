# The leaking bucket has no honest home in mode one

*"there are also still things that are going to need a way to deal with this
leaking bucket for the ttl tracking."*

The options in **Tags have lifetimes** contradict a constraint already stated in
`40-state.toml`:

> Tag state must not be writable by the user qwark runs as. A countdown the
> subject can reset is not a constraint on the subject.

**qwark runs as the agent's user.** It is a hook the agent's own session spawns,
with the agent's uid. So anything qwark can write, the agent can write: directly,
with the Write tool, without a shell. The leading candidate, an appended file the
run trims, is exactly the option that cannot satisfy the constraint. qwark needs
to write it, therefore the subject can rewrite it, therefore the countdown is
advisory.

Deriving it from the log does not escape this either. A log qwark appends to is
written with the same uid, so "was there a rebase in the last six commands" is
answered from a file the subject can edit. **Mode two inherits the same problem**,
and it is sharper there: a log with entries selectively removed reads as a clean
history rather than as a damaged one.

So the bucket needs a writer the subject is not, and that is a deployment fact
rather than something more code can fix. Two ways out, and the second is already
coming:

- A store owned by another uid, which is the sideboard process from **Tags have
  lifetimes**. Its cost was named as a daemon that must be up, and under
  fail-closed a daemon outage becomes a Bash outage.
- **The proxy is that process.** Once tool calls go through a long-lived server
  the agent reaches only by typed call and never through the filesystem, the
  natural place for the count is inside it, and the daemon objection loses most of
  its force: the daemon is already there.

Two properties the count must keep wherever it lands, both already decided and
easy to lose in a reimplementation: a denied command advances nothing (FR-4.24,
it did not happen), and tags do not stack, so setting one that is live replaces
it, TTL and all.
