# What the two parsers actually give

The Rust rewrite rests on a premise nobody had checked: that tree-sitter-bash
builds a better and more specific tree for judging a command. This record
measures it.

**It does not, for the thing qwark needs most, and it is worse in three places
that currently carry deny rules.** The language ruling stands on its other
grounds; this is the premise underneath it coming out the other way, which
`silo/docs/DECISIONS/what-language-each-component-is-written-in.md` names itself
as the place to record.

Evidence, with a script that regenerates it:
`clank/tasks/qwark/rewrite/20-measure-the-two-parsers.*/evidence/`.

    mvdan.cc/sh/v3   v3.13.1   through `qwark ast`
    tree-sitter      de98c6c   2026-09-09
    tree-sitter-bash a06c2e4   2025-12-02

## Neither parser decomposes options, and that was the whole hope

    grep -rn foo bar        mvdan: Word/Lit  ×4      tree-sitter: word ×3
    go test -run TestX      mvdan: Word/Lit  ×4      tree-sitter: word ×3
    dd if=x of=y            mvdan: Word/Lit  ×3      tree-sitter: word ×2
    tar xzvf a.tgz          mvdan: Word/Lit  ×3      tree-sitter: word ×2
    grep -A12 pat file      mvdan: Word/Lit  ×4      tree-sitter: word ×3

Both hand back a flat list of words. Neither knows `-rn` is two options, that
`-A12` fuses an argument, that `if=x` is a key-value pair, or that `xzvf` has no
dash and is still options. Option decomposition is qwark's own work in either
parser, which is exactly what `the-format-for-phases-one-and-two.md` specifies
the tree for, and it is no reason to prefer one parser over the other.

Same for the two execution vectors. `sed 's/x/echo RAN/e'` is `SglQuoted` to
mvdan and `raw_string` to tree-sitter: an opaque quoted blob to both, so only an
option-level rule catches it. `git -c alias.qq=!echo RAN qq` is a bare `word` to
both.

## Three node types have no tree-sitter counterpart, and all three carry rules

    mvdan          tree-sitter                    consequence
    TimeClause     command, command_name=time     rule selects nothing
    LetClause      command, command_name=let      rule selects nothing
    CoprocClause   command, command_name=coproc   rule selects nothing

`01-structure.toml` denies all three by node type. Under tree-sitter they are
ordinary commands with an ordinary command word, so the rules that name them
match nothing at all.

`time rm -rf /` is the case to look at. Denied today:

    deny  no-time-prefix   `time` is not permitted. It is a keyword rather than
                           a command, so the word at ordinal 0 is the command
                           underneath it and `time` itself is unaddressable by
                           name.

That reason is a statement about `mvdan.cc/sh`, not about bash. Under
tree-sitter `time` is addressable by name, `rm` is a plain word at ordinal 1,
and with the current permissive allow rule the command would be **allowed**.
`env rm -rf /` already demonstrates the failure the other way round: it is
allowed today, and `10-commands.toml` puts the general form as *the command word
is not the command*.

So the swap moves `time`, `let` and `coproc` out of the structural phase and
into the wrapper group, which is a rule change nobody proposed and which follows
from the parser alone. `10-commands.toml` says outright that `time` is not in
the wrapper group *because* it is a keyword and never a command word. That
sentence stops being true when the parser changes.

## `compound_statement` collides two things qwark rules on separately

    { ls; }     compound_statement, children '{' … '}'
    ((x=1))     compound_statement, children '((' … '))'

mvdan gives `Block` and `ArithmCmd`. qwark denies grouping with
`nodes = ["Subshell", "Block"]` and arithmetic separately. Under tree-sitter one
node name covers both, and telling them apart means inspecting anonymous
children instead of reading a type.

That is survivable, and it is a shape of defect to name: a clause that still
loads, still matches, and now matches more than it says.

## Where tree-sitter is genuinely better

Assignments are typed. `x=1 rm -rf /` gives `variable_assignment` →
`variable_name`, `'='`, `number`, where mvdan gives `Assign` with a name field
and a `Word`. More decomposed, and useful if a rule ever reasons about assigned
values.

Operators are addressable nodes. `&&`, `;`, `|`, `>` and `((` are anonymous
nodes in the tree. qwark's `ops` clause maps onto them directly, where mvdan
carries them as fields on typed nodes.

Expansions are separated by kind. `simple_expansion` for `$HOME` and
`arithmetic_expansion` for `$((1+2))`, against mvdan's `ParamExp` and
`ArithmExp`. A wash, but a clean one.

Backgrounding is a node, not a flag. mvdan sets `Background` on the statement;
tree-sitter emits a sibling `'&'`. mvdan's is easier to test, which is why
`flags` exists as a clause kind at all.

## The mapping, for the sixteen that do map

    CmdSubst     command_substitution      ProcSubst    process_substitution
    ParamExp     simple_expansion          ArithmExp    arithmetic_expansion
    Redirect     file_redirect             Subshell     subshell
    Block        compound_statement        ArithmCmd    compound_statement  (!)
    FuncDecl     function_definition       ForClause    for_statement
    WhileClause  while_statement           IfClause     if_statement
    CaseClause   case_statement            TestClause   test_command
    Assign       variable_assignment       DeclClause   declaration_command
    Background   '&'                       TimeClause   none
    LetClause    none                      CoprocClause none

`Redirect` also changes shape: tree-sitter wraps the whole thing in
`redirected_statement` and puts `file_redirect` inside, where mvdan hangs
`Redirect` off the statement beside the `CallExpr`.

## What this changes

Not the language. Rust was ruled on tree-sitter being a C library and on the two
grounds for `mvdan.cc/sh` being spent, and neither of those depends on the tree
being better.

It does change what the rewrite has to carry. Three deny rules stop working and
have to be rewritten as command-word rules, one clause starts matching two
things, and the existing requirement that an unknown node name is a
configuration error at load is what makes the remap safe and not silent. That
requirement is now the most load-bearing line in the format.

And it retires the claim. `docs/DECISIONS/why-a-parser-rather-than-a-matcher.md`
said tree-sitter-bash loses on a typed tree and no cgo. Measured, it does not
lose on the typed tree; it wins on assignments and operators and loses on three
keyword forms. Neither parser is better at the job qwark most needs done,
because that job is not parsing.
