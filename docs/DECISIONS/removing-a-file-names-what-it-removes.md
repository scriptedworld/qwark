# Removing a file names what it removes

> bare `rm` is fine, but `rm -f` or `rm -r` or `rm -rf` is NOT

`rm -f`, `rm -r` and `rm -rf` are all denied. Bare `rm <path>` is explicitly
permitted rather than merely undenied: it has an allow rule of its own,
`allow-removing-a-named-file`.

## Why recursion is a denial and not an ask

An earlier form of this ruling made `rm -r` an `ask`, warning that it was
recursive, and denied only the two cases carrying `-f`. That distinction did not
survive contact with what the two options actually do.

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

## What the worked example in `30-options.toml` now teaches

Two rules cover three spellings, because `rm -rf` matches both. With both set to
`deny` that demonstrates composition rather than escalation: `rm -rf` is refused
with both reasons given under FR-4.25, so its author learns that the force is a
problem and the recursion is a problem, instead of being sent round twice.

**A comment that explains a rule is part of that rule.** Changing an action and
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
