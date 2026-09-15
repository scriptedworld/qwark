# Go, because of `os.Root`

This decision is superseded. qwark is being rewritten in Rust, and `os.Root`
goes with it. The parser choice drove the language: tree-sitter-bash is a C
library, which is cgo in Go and an ordinary crate in Rust, and the two grounds
that chose `mvdan.cc/sh` were round-tripping, which this gate never uses, and
no-cgo, which was an argument for a pure-Go parser inside a Go program.

`silo/docs/DECISIONS/what-language-each-component-is-written-in.md` carries the
ruling and states the cost: `cap-std` is a dependency where `os.Root` is
standard library, for a tool whose whole job is containment. The section below
is why that cost is real, so it is kept.

Two of the costs below expire with the language. Branch coverage is measurable
in Rust, and the one-statement `main` exists because a Go test process cannot
reach `main`. The last section already said so.

qwark is written in Go. The estate's other tools are Rust and Python, and bolt
was moved off Go deliberately, so this is a choice and not a default.

## What decided it

`os.Root`, and it is not used yet.

Go 1.24 added directory-scoped filesystem access: a `*os.Root` opened on a
directory can only reach inside it, and the kernel enforces that; the caller does
not have to remember to. Symlinks pointing out, `..` walking up and absolute
paths are all refused by construction.

That is the shape of qwark's hardest problem. Every path rule today reasons
about the text of a command: `rules/20-paths.toml` asks where a command reaches
by inspecting its words, and `docs/LESSONS/an-escape-defeats-a-path-rule.md`
records what that costs, since `rm /home/user/.cl\aude/x` reaches `.claude` and a
rule matching the literal string does not see it.

Textual containment can always be spelled around. Kernel containment cannot. So
the language was picked for where the third layer has to go, not for what the
first layer needed.

## What that means today

**Nothing in qwark calls `os.Root`.** The parser, the rule engine and the hook
are ordinary Go and would have been ordinary anything. Judged on what is built,
the choice is unexercised.

It is recorded because the reason is invisible from the code, and because a
reader who finds a Go program in an estate that moved its other Go program to
Rust is owed the answer.

## What Go costs

A decision naming only its upside is not one, so here are the costs.

Branch coverage is reachable and not measurable. Go reports statements, so the
toolchain can say what ran and not which way a condition went: an `if` with no
`else` has a false path it cannot see.

Covering every edge is still ordinary work. A table-driven test with a case per
branch does it, and a test hierarchy that follows the seams gets you there by
construction instead of by hunting. What is missing is the instrument that
confirms it. Nothing reports which branches went unvisited, so the guarantee
rests on discipline and review where the statement number rests on measurement,
and only one of those degrades quietly.

The jig gates on statements and says so. It does not report a guarantee nothing
established.

`main` cannot be reached by a test. Nothing in a test process calls it, so it is
uncovered by construction, and covering it means building with
`go build -cover`, running the binary, and appending a second profile to the
first. That is a whole extra mechanism for one function.

So `main` holds exactly one statement, and this is a requirement, not a style:
`cmd/qwark/main.go` calls `cli.Main` and exits. Everything real is in
`internal/cli`, which a test reaches normally. The binary's own `help` then
costs almost nothing to run and proves the one unreachable statement executed.

Rust needs none of this: an integration test runs the binary and the profile is
one artifact. The entry-point pattern exists because of the language, not
because it is better design, so when you meet it, know which of the two you are
looking at.

## The alternative that was live

Rust, which is what bolt runs and what wrench measured fastest per document.
`cap-std` offers the same capability-oriented filesystem access, so the
containment argument is not unique to Go. Go won on being the language this
estate should keep a real program in, and qwark is that program.

`docs/DECISIONS/the-end-state-is-three-layers.md` sets out where the containment
work goes. This decision is upstream of it.
