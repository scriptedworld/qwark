# Removing a file names what it removes

Ruled 2026-08-28, revising the ruling of 2026-08-19.

> bare `rm` is fine, but `rm -f` or `rm -r` or `rm -rf` is NOT

## What changed

The 2026-08-19 ruling was three cases:

    rm -r -f   forbidden, because of -f
    rm -r      no -f: ask, warning that it is recursive
    rm -f      no -r: forbidden

`rm -r` was an `ask`. It is now a `deny`. Everything else is unchanged, and
bare `rm <path>` is explicitly permitted rather than merely undenied: it has an
allow rule of its own, `allow-removing-a-named-file`.

## Why the middle case moved

Recursion and a glob fail in the same way. What `rm -r <dir>` deletes is decided
by what the tree holds at the moment it runs, not by anything the command says,
which is the exact property tier one refuses a wildcard for. An `ask` on it puts
a person in front of a question they cannot answer from the text they are shown:
the command names a directory and the consequence is its contents, which are not
in the command.

Force is refused for its own reason and that reason has not changed. It
suppresses the report of what was missing, so the command stops being able to
tell you it did something other than what you meant.

## What this costs, stated rather than discovered

Emptying a directory means naming its files. There is no permitted spelling of
"remove this tree", and that is deliberate rather than an omission waiting to be
filled. A person at a terminal is not what the gate is for.

## The worked example was rewritten in the same change

`30-options.toml` taught the strictest-wins precedence argument through these
three cases: two rules rather than three, because `rm -rf` matches both and the
deny takes it, so neither rule has to know the other exists.

That argument is still true and is still what the two rules demonstrate. It is
no longer visible in their **actions**, because both are now `deny`, so the
comment would have taught a rule the file no longer holds. It now demonstrates
composition rather than escalation: `rm -rf` is refused with both reasons given
under FR-4.25, so its author learns the force is a problem and the recursion is
a problem, instead of being sent round twice.

**A comment that explains a rule is part of that rule.** Changing the action and
leaving the explanation is how a file comes to document something that is not
there, and the next reader believes the comment.

## Recorded first-hand

I heard this ruling directly and am writing it down here, in this project's own
decision record, because a ruling relayed between sessions is not recorded. It
had already been repeated second-hand twice before this file existed.

**And it did not hold while it was only spoken.** I used `rm -r` to delete a
resolved inbox entry within the hour, against a tree that was ungated at the
time, having received the ruling in that same conversation. Nothing caught it
because nothing was there to catch it. That is the argument for a boundary
rather than for care, and it is the reason the rule now exists in
`30-options.toml` rather than in a paragraph.
