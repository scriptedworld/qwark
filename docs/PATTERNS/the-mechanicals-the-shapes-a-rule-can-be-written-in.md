# The mechanicals, the shapes a rule can be written in

Asked what a "library of useful mechanicals" should be, the answer was a catalogue
of shapes and not a file of rules. None of it is new mechanism; every shape is the
existing schema pointed a particular way.

**Every shape below is a conjunction of clauses.** What varies is which selectors
the clauses use, and where they point.

### 1. Refused by class

The plainest shape, and what most rules should be. A group names the members and
one rule names the group, so the reason belongs to the class and not to any member
of it.

    [group.git-network]
    members = ["fetch", "pull", "push", "clone", …]

    [[rule.clause]]
    index = "0"
    value = "git"
    [[rule.clause]]
    index = "1"
    group = "git-network"

Adding a member is a one-line edit that leaves the rule alone. **Classes are
expected to overlap.** `push` runs hooks and it reaches the network; FR-4.25
collects both reasons, so a command in three classes is refused with all three
stated. Put a command in a class whenever the class's reason is true of it, and
not only when no other class has claimed it.

### 2. Allowed as a word, refused in a shape

For a command that reads when bare and writes when handed a particular subcommand.
The word stays available and the destructive form does not.

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
denying the whole word broke something. 40-state.toml tells a reader to look at
`git reflog` before deleting after a rebase, and it clears the tag when they do.
**A denied command has no effect of any kind (FR-4.24)**, so denying the word left
the instruction impossible to follow and the tag impossible to clear. Watch for a
denial whose own message names a refused command; that is the smell this shape
fixes.

### 3. Refused unless

A conditional refusal, written as one deny rule carrying an inverted clause, and
not as two rules where one outranks the other. The exception lives inside the rule
it modifies, so whoever reads the denial also reads the way out of it.

    [[rule.clause]]
    option = "gpg-sign"
    absent = true

**The trap in this shape: an inverted clause is satisfied by absence, including
the absence caused by qwark not understanding the option.**
`commit-must-be-signed` fires on `git commit --gpg-sign -m x`, because
`--gpg-sign` is missing from the declaration, which makes "the signing option is
not there" true. The verdict fails safe. The *message* does not, and tells
somebody who signed that they must sign.

So a "refused unless" rule is only honest when **the option it excepts is
declared**. Write one without declaring the option and you get a rule that reads
as conditional and behaves as unconditional.

### 4. Refused because it cannot be accounted for

Nobody writes this shape. It is what happens when they write nothing at all. An
undeclared command is refused (FR-4.16), and so is an option the declaration does
not name (FR-6.7), so **omission is a denial** and the safe state is the default
one.

It is what actually holds the read-only git allowance narrow. `git help -w` opens
a browser, `git log --ext-diff` runs an external program, and no rule names either
of them; they are refused because `05-declarations.toml` lists neither option.
Nobody had to enumerate the dangerous flags, and forgetting one buys a refusal
instead of a hole.

The inverse is the part to be careful about. **Declaring an option is what makes
it reachable**, so the declaration file is the eligibility surface, and an
addition to it is a wider change than an addition to a rule file.

### 5. Set a condition, then judge against it

The tag shape, deferred to a later version and documented in 40-state.toml. One
rule establishes a condition with a lifetime, and other rules name that condition
in a clause without knowing how it came to hold. It costs more than the rest,
being the only shape whose answer depends on anything beyond the command in front
of it.

### 6. Refused everywhere except where it belongs

The scoping shape, and it is written **inside the deny rule**, never as an allow
beside it:

    [[rule]]
    id     = "no-go-execution-outside-its-own-tree"
    action = "deny"

      [[rule.clause]]
      index = "0"
      value = "go"

      [[rule.clause]]
      index = "1"
      group = "go-executes"

      [[rule.clause]]
      cwd    = "/home/user/.projects/qwark"
      absent = true

**The reflex is to write a scoped allow, and it does not work.** There is no
overridable deny in this engine: the strictest action wins, so an allow naming a
directory cannot lift a deny that already fired. Adding one leaves both rules
live and the command refused, which reads as the clause being broken.

So the exception goes where shape 3 puts every exception, in the rule it
modifies, and for the same reason: a reader of the deny rule sees the whole of
it. Measured 2026-08-28, judging a probe under `.ephemera/` with the scoped
rules alongside the originals; the originals fired and the scope looked inert.

The same holds for the agent clause, which shares the shape. What differs is
what the two are good for: `agent` names a role a dispatcher assigned, and `cwd`
names a tree, so a policy that varies by repository has only one of them to be
written with.

**An inverted clause in a deny rule is the safe direction, and it is worth
saying why.** Inversion is satisfied by absence, including absence qwark caused
by not understanding something. Here that means a request whose directory could
not be established does not satisfy `cwd`, so `absent = true` holds, so the
denial stands. The rule fails closed on ignorance, which is what a deny rule
must do and what the same clause in an allow rule would not.
