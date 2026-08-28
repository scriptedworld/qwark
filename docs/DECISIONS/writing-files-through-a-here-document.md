# Writing code or files through a here-document

**qwark enforces this rule on its own subject.** It is FR-4.10 in
`REQUIREMENTS.md` and `no-heredoc-write` in `rules/00-structure.toml`, so the gate
refuses of an agent exactly what the standing rules refuse of whoever works here.

Writing code or files with a here-document is not allowed.

Tier one's ban on redirections already subsumes this, and it earns its own rule
because the reason is different. A redirection is banned for being redundant. A
here-document write is
banned for going around the tools that make a change reviewable: content arriving
through `cat > file <<'EOF'` was never a diff, was never held against the file it
replaced, and leaves nothing to inspect but the command that produced it.

The shape is conjunctive, which makes it a tier-two rule rather than a tier-one
one: a here-document *and* a truncating redirect to a path. Measured, the two
facts are reported separately and both present:

    $ qwark facts "cat > f.go <<EOF
    package main
    EOF"
    redirect.truncate          1:5   │ > f.go
    redirect.heredoc           1:12  │ <<EOF package main EOF

The rule was asked for while the failure was happening: every Go source file in
this repository up to that point had been written by exactly that command shape,
by the assistant, under a standing instruction to prefer Bash for file edits.
