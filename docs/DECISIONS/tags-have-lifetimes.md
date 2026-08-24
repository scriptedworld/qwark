# Tags have lifetimes

Two kinds:

- **Ephemeral.** Derived from this command's tree, and live for one evaluation.
- **Sticky.** Written by a rule, persisted, and decaying over subsequent
  commands. The worked example: after a rebase, deletion is denied for the next
  six commands.

Sticky tags key on the `session_id` in the hook payload.

**How that state survives between calls is undecided.** The shape under
consideration is an ephemeral file appended to on each call, which each run then
trims and rewrites, with locking so concurrent calls do not collide.

Several facts bear on the locking, none of them settling it:

- Contention has two sources: several Bash calls running in parallel inside one
  session, and separate Claude Code sessions running at once on this machine.
  Keying the file by `session_id` removes the second entirely, leaving only the
  first for a lock to cover.
- A sideboard process holding the state, Redis being the one raised, with my
  own reservation that some would call it heavy, moves the problem out of the
  filesystem but adds a daemon that must be up. That interacts with the
  fail-closed rule in **Configuration**: if an unparseable rule file makes Bash
  unusable, then by the same reasoning an unreachable state store does too, and a
  daemon outage becomes a Bash outage. A file has fewer ways to be absent.
- Weigh this before adding any store at all. If every command is already logged,
  "was there a rebase in the last six commands" is a question the log answers.
  That is not an argument against a store, but a store has to earn its place
  against a tail read of a file that exists anyway.
- Append and compaction are not one problem. Appends are frequent, small and
  independent; the trim-and-rewrite is rare and needs every other writer held
  off. A scheme that locks both alike pays the expensive price on the common
  path.
