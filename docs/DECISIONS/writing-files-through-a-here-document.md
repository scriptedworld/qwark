# Writing code or files through a here-document

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 120-147. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**qwark enforces this rule on its own subject.** It is FR-4.10 in
`REQUIREMENTS.md` and `no-heredoc-write` in `rules/00-structure.toml`, so the
gate refuses of an agent exactly what the standing rules refuse of whoever works
here. *That cross-reference used to live in this repository's `CLAUDE.md`, which
was deleted on 2026-08-21 because its rules duplicated the global ones; the
reference was the only part of it that was qwark's own.*

**Requirement, owner, 2026-08-19.** Writing code or files with a here-document
is not allowed.

Tier one's ban on redirections already subsumes this, but it earns its own rule
because the reason is different, and reasons are what survive a policy being
rewritten. A redirection is banned for being redundant. A here-document write is
banned for going around the tools that make a change reviewable: content that
arrives through `cat > file <<'EOF'` was never a diff, was never held against
the file it replaced, and leaves nothing to inspect but the command that
produced it.

The shape is conjunctive, which makes it a tier-two rule rather than a tier-one
one: a here-document *and* a truncating redirect to a path. Measured, the two
facts are reported separately and both present:

    $ qwark facts "cat > f.go <<EOF
    package main
    EOF"
    redirect.truncate          1:5   │ > f.go
    redirect.heredoc           1:12  │ <<EOF package main EOF

**FACT 2026-08-19, and the reason this rule was asked for.** Every Go source
file in this repository up to that point was written by exactly that command
shape, by the assistant, under a standing instruction to prefer Bash for file
edits. The rule was stated while it was happening.

