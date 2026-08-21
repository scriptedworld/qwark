# Configuration

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 1093-1109. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**PREFERENCE, owner, 2026-08-19.** TOML, in multiple rule files that are
aggregated. `github.com/BurntSushi/toml` — already the house library
(`dotfiles/go/internal/manifest` — that repository was named `linux.dotfiles`
when this was written, and took over the `dotfiles` name on 2026-08-20), and its
`ParseError.ErrorWithPosition`
renders a source excerpt with line and column, which this design needs:

**If any rule file is unparseable, Bash is unusable.** Fail-closed. A gate that
degrades to permissive when its own configuration is broken is a gate that
reports success while guarding nothing.

The cost is that a typo denies every Bash command until it is fixed, so the
denial has to name the file, the line, and the text — and the escape route must
not itself require Bash. Editing the rule file with the Edit tool does not.

