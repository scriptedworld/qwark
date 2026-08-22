# Tags have lifetimes

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 818-854. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


Two kinds:

- **Ephemeral** — derived from this command's tree, live for one evaluation.
- **Sticky** — written by a rule, persisted, and decaying over subsequent
  commands. **Example, 2026-08-19:** after a rebase, deletion is denied
  for the next six commands.

Sticky tags key on the `session_id` in the hook payload.

**UNDECIDED, 2026-08-19.** How the state survives between calls is open.
The shape under consideration is an ephemeral file appended to on each call,
which each run then trims and rewrites, with locking so concurrent calls do not
collide.

Two facts bear on the locking, neither of them settling it:

- Contention has two sources — several Bash calls running in parallel within
  one session, and separate Claude Code sessions running at once on this
  machine. Keying the file by `session_id` removes the second entirely, leaving
  only the first for a lock to cover.
- A sideboard process holding the state — Redis was raised, with the
  own reservation that some would call it heavy — moves the problem out of the
  filesystem but adds a daemon that must be up. That interacts with the
  fail-closed rule above: if an unparseable rule file makes Bash unusable, then
  by the same reasoning an unreachable state store does too, and a daemon
  outage becomes a Bash outage. A file has fewer ways to be absent.
- Worth weighing before adding any store at all: if every command is already
  logged, "was there a rebase in the last six commands" is a question the log
  answers. That is not an argument against a store, but it does mean a store
  has to earn its place against a tail read of a file that exists anyway.
- Append and compaction are not the same problem. Appends are frequent, small
  and independent; the trim-and-rewrite is rare and needs every other writer
  held off. A scheme that locks both alike pays the expensive price on the
  common path.

