# The format for phases one and two

Status: a proposal, for discussion. **Nothing is built and no rule file has
been changed.** Hard rule 4a wants a change to qwark's rules described and
agreed in words first, and this is that description. It becomes `docs/SPEC.md`
when agreed.

`a-tree-of-declared-commands.md` settled the *shape* of the declaration model.
This settles the *format*: what a rule file is, in what serialisation, and how
phase one and phase two share one document without one being able to weaken the
other.

## The phases, and what each is allowed to say

    1  structure    what the command's syntax is, judged without knowing any
                    command word. Refuses substitution, redirection, pipes,
                    globs, backgrounding, multiple statements.
    2  the tree     what each word means: is it a command, a verb, an option,
                    an operand; what its arity is; what type its value has.
    3  paths        what a value may refer to: file types, containment,
                    metadata. Deferred, and specified only far enough here to
                    be sure phase two does not block it.

Phase one needs nothing from phase two. Phase two needs phase one, because a
word's meaning is only decidable once the command's effect is fixed by its own
text. Phase three needs phase two, which is why phase two gets built before
anyone wants it: a path rule can only fire on a word something has already said
is a path.

The phase number is data in the file, not a filename convention. Numeric
filename order is what put `[declarations]` second and made the deployed set
refuse `qwark help` for eleven days,
`docs/LESSONS/the-policy-file-has-to-load-first.md`.

### A deployment states which phases it runs

The live set at `~/.config/qwark/rules` is two of the seven files in `rules/`,
deliberately, and **nothing on the machine can tell that from the live set
alone.** The drift check in `CLAUDE.md` iterates the live directory,
so five absent files read as no drift. A partial deployment and a botched one
are the same bytes.

So the deployment carries a manifest naming the phases it intends to run, and
loading refuses a file whose `phase` the manifest does not list, and refuses a
manifest naming a phase no file supplies. An incomplete install then fails at
load instead of gating quietly with half a policy.

## The serialisation is YAML, and 47% of the rule set does not survive it

`silo/docs/DECISIONS/yaml-everywhere-validated-against-the-decoded-structure.md`
is estate-wide and wrench no longer ships a TOML codec. So the format is YAML,
validated by JSON Schema against the decoded structure.

Across `rules/*.toml`:

    total        2,196 lines
    comment      1,022 lines   47%
    data           916 lines

Nearly half the rule set is prose in comments, and canonical YAML has no
comments. The same conversion destroyed all 76 lines of reasoning in
`dotfiles/repos.*.toml` and cost a rewrite, recorded in
`clank/inbox/palette-print/toml-is-retired-so-the-palettes-move-to-yaml/`.

The remedy there was to make the prose data, and here it is not merely a rescue.
The comments are carrying two different things and only one of them is a
comment:

`reason` is what a refused agent is shown. It already exists and is already
data. `CLAUDE.md` 4a: it is the only thing standing between a denial and a
session that does not understand why.

`why` is why the rule exists and why not to weaken it. Today this is a
comment, so it reaches a maintainer reading the file and nothing else.

Making `why` data buys something the comment cannot. `10-commands.toml` carries
a list headed DELIBERATELY NOT DECLARED: `--exec-path=`, `--git-dir=`,
`--ext-diff`, `-O`, `-w` and the rest, each with the hazard written out. Every
one of those is a comment, so the refusal an agent actually receives is *"not
declared"*, and the reasoning sits in a file it never reads. As data they become deny nodes that
name the hazard.

    field       audience                    when it is shown
    reason      the agent being refused     at the decision
    why         the next maintainer         `qwark explain <id>`
    about       the reader of the file      never; documentation

Comments stay legal in a hand-maintained file. qwark reads rule files and
never writes them, so nothing rewrites them canonically. The rule is narrower
than "no comments": anything a decision or an audit has to carry is data,
because a comment cannot reach either. Generated subtrees are the exception and
are canonical, below.

## Phase one: a rule is still a conjunction

The structural model is deployed, works, and is not in question. It is
translated, not redesigned. `docs/DECISIONS/a-rule-is-a-conjunction.md` and
`the-strictest-action-wins.md` both stand.

    "phase": 1
    "about": "Structural denials. Judged without knowing any command word."
    "shell":
      "allow": ["/bin/bash", "/usr/bin/bash"]
    "rule":
      - "id": "no-substitution"
        "action": "deny"
        "reason": |
          A command whose words are produced at runtime cannot be judged by
          reading it. Name the value.
        "why": |
          Everything above this depends on the command's effect being fixed by
          its own text. Deciding which paths a command reaches is unsound the
          moment a substitution can produce one.
        "clause":
          - "nodes": ["command_substitution", "process_substitution"]

Two things change and neither is a redesign.

The node vocabulary is tree-sitter-bash's, not `mvdan.cc/sh`'s. `CmdSubst`
becomes `command_substitution`. This is a remap of a closed set, and the existing
requirement that an unknown node name is a configuration error at load is what
makes it safe: a clause naming a node the grammar does not have fails the load
instead of quietly matching nothing. That requirement is load-bearing and
carries over verbatim.

`index`, `group`, `option` and `kind` clauses move to phase two or three.
Phase one keeps `nodes`, `flags`, `ops` and `fact`, which are the four that need
no declaration. The rest select on things only the tree can say, and a phase-one
file naming them is a load error.

## Phase two: the tree

A map keyed by the token, at every level, because the point of the tree is that
a lookup is a lookup. `a-tree-of-declared-commands.md`: navigate to the node,
then look each observed token up among that node's children. Nothing is walked
as a path and order never matters.

    "phase": 2
    "node":
      "git":
        "kind": "command"
        "style": ["posix", "fused"]
        "why": "..."
        "child":
          "commit":
            "kind": "verb"
            "operand": {"type": "path", "arity": "many"}
            "child":
              "-F":
                "kind": "option"
                "spell": ["--file"]
                "arity": 1
                "value": {"type": "path"}
              "--amend":
                "kind": "option"
                "arity": 0
                "verdict": "ask"
                "reason": |
                  Amending rewrites a commit another session may already have
                  pushed or built on.

### What a node carries

    kind      command | verb | option | operand      required
    child     the map of tokens reachable from here
    spell     alternate spellings collapsed into this node
    arity     tokens this option consumes: 0, 1, or n
    value     the type of what it consumes
    operand   the type and arity of non-option arguments
    style     option syntax, on a command node only
    verdict   deny | ask                             optional
    reason    shown when verdict fires               required with verdict
    why       why this node is here                  optional

`kind` is explicit, not inferred from a leading dash, because `tar xzvf`
and `dd if=x` have options with no dash at all.

Spelling collapse is a load-time expansion. `-F` with `spell: ["--file"]` is
one node reachable by two keys. The loader builds the lookup table and refuses
a collision, which is the check that stops a second declaration of `-n`
silently shadowing the first. `--file=x` is split by `style` before lookup, so
it is not a third spelling.

`style` is per command and it is a list, because real tools mix conventions.
Measured in the corpus:

    posix       grep -rn, ls -la, git -C          bundling
    fused       grep -A12, git log -7             argument attached, no space
    go-flag     go test -run X                    single dash, long name, and
                                                  -abc is ONE flag, not three
    key-value   dd if=x of=y
    bare        tar xzvf                          first word, no dash

### There is no `allow`

**A node's presence is its permission.** `docs/DECISIONS/a-declaration-is-a-permission.md`
already establishes this: under strictest-wins an added `allow` rule changes
nothing, while an added declaration moves a command from denied to eligible.
Writing `verdict: allow` on every node would restate that thousands of times and
raise a question with no safe answer, which is whether it inherits.

So `verdict` takes only values that narrow: `deny` and `ask`. Absence of a
verdict means accounted for. Absence of the node means denied, with the engine's
own reason.

A verdict inherits down the subtree, and this is safe in one direction only.
Denying `git` denies every verb under it; nothing beneath can re-open it,
because there is no value that widens. The inheritance question the earlier
proposal left open is not answered so much as deleted.

An explicit deny node earns its place through the message. `--force` absent
is refused as "not declared" and the agent guesses. `--force` present with a
reason names the hazard.

### The generated subtrees are separate files

`just --dump --dump-format json` gives every recipe with its parameters, so the
`just` subtree is derived per project, not maintained. Same for `bolt`
from the jigs.

A generated file is canonical YAML, carries `generated: <the command>`, is
gitignored where it is per-project, and **the loader refuses a generated file
that also carries hand-written nodes.** Mixing the two is how a generated file
acquires an edit that the next regeneration silently drops.

## Loading is two passes, so order cannot matter

Pass one reads the manifest and every file's `phase` and policy. Pass two reads
rules and nodes. Nothing a file says about policy can be affected by which file
loaded first, which removes the whole class of bug
`the-policy-file-has-to-load-first.md` records, not just the one instance.

Both passes fail closed. An unreadable file, an unknown phase, a duplicate
spelling, a generated file with hand-written nodes, a verdict with no reason:
each is a load error, and a load error is no Bash at all.

## What phase three adds, and what it may not change

Phase three reintroduces `20-paths.toml` and `40-state.toml` as rules that
select on what the tree assigned:

    "clause":
      - "value-type": "path"
        "within": "~/.config/qwark"

It adds no node fields and changes no verdict semantics. `value: {type: path}`
in phase two is already the hook it needs, and it is the reason phase two
declares types for values it currently does nothing with.

`docs/LESSONS/an-escape-defeats-a-path-rule.md` is what this buys: a path rule
today reasons about the text of a command, so `rm /home/user/.cl\aude/x` reaches
`.claude` past a rule matching the literal string. Once a node says an argument
is a path, it can be unescaped and canonicalised before judging, because
something knows which words deserve it.

## Decisions this proposal makes

1. YAML, with `reason`, `why` and `about` as data. The 47% that is comment
   today is not all prose to be preserved; the part that has to reach a decision
   or an audit becomes data, and the rest may stay a comment.
2. A tree node map keyed by token, `spell` collapsing alternates, with
   collision refused at load.
3. No `allow` verdict. Presence permits, `deny` and `ask` narrow, and a
   verdict inherits down the subtree.
4. `style` is a list on the command node.
5. Two-pass load, and a manifest naming the phases a deployment runs.
6. Phase one keeps `nodes`, `flags`, `ops`, `fact` and nothing else.

## What is not settled

The node vocabulary remap. Every phase-one clause is written in
`mvdan.cc/sh` node names and has to be re-expressed. The claim that tree-sitter
builds a better tree for this is written down nowhere and the only comparison on
record says the opposite, `docs/DECISIONS/why-a-parser-rather-than-a-matcher.md`.
Parsing the same commands with both and diffing the node structure is ten
minutes and it should happen before the remap is specified, not after.

`program`-typed values. 16% of the corpus is `sh -c` and `python3 -c`.
Recursive judgement through the same tree is the answer the earlier proposal
gives, and nothing here specifies it. The interpreters-behind-a-Justfile rule
may delete the problem instead. Left to phase two's build, not its format.

Where the schema lives. wrench holds `definitions`, `envelope`, `jig` and
`manifest`. A `rules` schema is either a fifth there or qwark's own, and that is
wrench's call.
