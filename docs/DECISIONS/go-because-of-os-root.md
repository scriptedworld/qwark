# Go, because of `os.Root`

qwark is written in Go. The estate's other tools are Rust and Python, and bolt
was moved off Go deliberately, so this is a choice rather than a default.

## What decided it

**`os.Root`**, and it is not used yet.

Go 1.24 added directory-scoped filesystem access: a `*os.Root` opened on a
directory can only reach inside it, and the kernel enforces that rather than the
caller remembering to. Symlinks pointing out, `..` walking up and absolute paths
are all refused by construction.

That is the shape of qwark's hardest problem. Every path rule today reasons
about **the text of a command**: `rules/20-paths.toml` asks where a command
reaches by inspecting its words, and `docs/LESSONS/an-escape-defeats-a-path-rule.md`
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

## The alternative that was live

Rust, which is what bolt runs and what wrench measured fastest per document.
`cap-std` offers the same capability-oriented filesystem access, so the
containment argument is not unique to Go. Go won on being the language this
estate should keep a real program in, and qwark is that program.

`docs/DECISIONS/the-end-state-is-three-layers.md` sets out where the containment
work goes. This decision is upstream of it.
