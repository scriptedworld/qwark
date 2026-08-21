# The mechanicals — the shapes a rule can be written in

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 944-1044. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**PREFERENCE, owner, 2026-08-20**, asked what a "library of useful mechanicals"
should be: a catalogue of the shapes, not a file of rules. What follows is that
catalogue. Nothing here is a new mechanism — each is the existing schema used a
particular way — and the point of naming them is that the next person writing a
rule reaches for a shape that already works instead of inventing a fifth one.

**Every shape below is a conjunction of clauses.** That is not one of the
options; it is the only thing a rule is. What varies is which selectors the
clauses use and where they point.

### 1. Refused by class

The plainest shape and the one most rules should be. A group names the members,
one rule names the group, and the reason belongs to the class rather than to
any member.

    [group.git-network]
    members = ["fetch", "pull", "push", "clone", …]

    [[rule.clause]]
    index = "0"
    value = "git"
    [[rule.clause]]
    index = "1"
    group = "git-network"

Adding a member is a one-line edit that does not touch the rule. **Classes are
expected to overlap**: `push` runs hooks and reaches the network, and under
FR-4.25 both reasons are collected, so a command in three classes is refused
with all three stated. Put a command in a class when the class's reason is true
of it, not when no other class claimed it first.

### 2. Allowed as a word, refused in a shape

For a command that reads when bare and writes when given a particular
subcommand. The word stays available; the destructive form does not.

    [[rule.clause]]
    index = "0"
    value = "git"
    [[rule.clause]]
    index = "1"
    value = "reflog"
    [[rule.clause]]
    index = "2"
    group = "git-reflog-writes"

The worked example is `no-git-destroying-the-reflog`, and it exists because
denying the word broke something: 40-state.toml tells a reader to look at
`git reflog` before deleting after a rebase, and clears the tag when they do.
**A denied command has no effect of any kind (FR-4.24)**, so denying the word
made the instruction impossible to follow and the tag impossible to clear. A
denial whose own message names a refused command is the smell this shape fixes.

### 3. Refused unless

A conditional refusal, written as one deny rule with an inverted clause rather
than as two rules where one outranks the other. The exception lives inside the
rule it modifies, so a reader of the denial sees the way out of it.

    [[rule.clause]]
    option = "gpg-sign"
    absent = true

**FACT 2026-08-20, and it is the trap in this shape: an inverted clause is
satisfied by absence, including the absence caused by qwark not understanding
the option.** `commit-must-be-signed` fires on `git commit --gpg-sign -m x`,
because `--gpg-sign` is not in the declaration, so "the signing option is not
there" is true. The verdict fails safe. The *message* does not: it tells
somebody who signed that they must sign.

So a "refused unless" rule is only honest when **the option it excepts is
declared**. Writing one without declaring the option produces a rule that reads
as conditional and behaves as unconditional.

### 4. Refused because it cannot be accounted for

Not a shape anyone writes — it is what happens when they write nothing. An
undeclared command is refused (FR-4.16) and an option the declaration does not
name is refused (FR-6.7), so **omission is a denial** and the safe state is the
default one.

This is what actually holds the read-only git allowance narrow. `git help -w`
opens a browser and `git log --ext-diff` runs an external program, and neither
is named by any rule: they are refused because `05-declarations.toml` does not
list those options. Nobody had to enumerate the dangerous flags, and forgetting
one costs a refusal rather than a hole.

The inverse is the thing to be careful about. **Declaring an option is what
makes it reachable**, so the declaration file is the eligibility surface and an
addition to it is a wider change than an addition to a rule file.

### 5. Set a condition, then judge against it

The tag shape, deferred to a later version and documented in 40-state.toml. One
rule establishes a condition with a lifetime; other rules name it as a clause
without knowing how it was established. Its cost is that it is the only shape
whose answer depends on anything beyond the command in front of it.

