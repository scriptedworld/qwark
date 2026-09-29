# Removing a file names what it removes

> bare `rm` is fine, but `rm -f` or `rm -r` or `rm -rf` is NOT

`rm -f`, `rm -r` and `rm -rf` are all denied. Bare `rm <path>` is explicitly
permitted, not merely undenied: it has an allow rule of its own,
`allow-removing-a-named-file`.

## Why recursion is a denial and not an ask

The obvious middle ground makes `rm -r` an `ask`, warning that it is recursive,
and denies only the two cases carrying `-f`. An earlier form of this ruling did
exactly that, and the distinction does not survive contact with what the two
options actually do.

Recursion and a glob fail in the same way. What `rm -r <dir>` deletes is decided
by what the tree holds at the moment it runs, not by anything the command says,
which is the exact property tier one refuses a wildcard for. An `ask` on it puts
a person in front of a question they cannot answer from the text they are shown:
the command names a directory and the consequence is its contents, which are not
in the command.

Force is refused for its own reason. It suppresses the report of what was
missing, so the command stops being able to tell you it did something other than
what you meant.

## What this costs

Emptying a directory means naming its files. There is no permitted spelling of
"remove this tree", and that is deliberate, not an omission waiting to be
filled. A person at a terminal is not what the gate is for.

## What the worked example in `30-options.toml` teaches

Two rules cover three spellings, because `rm -rf` matches both. With both set to
`block` that demonstrates composition, not escalation: `rm -rf` is refused with
both reasons given under FR-4.34, so its author learns that the force is a
problem and the recursion is a problem, instead of being sent round twice.

A comment that explains a rule is part of that rule. Changing an action and
leaving the explanation is how a file comes to document something that is not
there, and the next reader believes the comment.

## Why it is a rule and not a practice

Spoken, it did not hold. An `rm -r` deleted a resolved inbox entry within the
hour of the ruling, on a tree that was ungated, and nothing caught it because
nothing was there to catch it. That is the argument for a boundary over care,
and why the rule lives in `30-options.toml` and not only in a paragraph.
