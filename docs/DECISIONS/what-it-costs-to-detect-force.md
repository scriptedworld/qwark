# What it costs to detect `-f`

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 313-338. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**Requirement, owner, 2026-08-19.** Block `--force` and `-f` — but only where
`-f` means force. It means file to `tar`, so the meaning cannot come from the
spelling. **The table of what each command's options mean, and which take a
value, is declared in TOML beside the rules.**

Two measurements say why a clause cannot simply compare strings.

**FACT 2026-08-19.** GNU long options accept unambiguous abbreviations. Each of
these deleted nothing and exited 0, which is only possible if force took effect:

    rm --force  --forc  --fo  --f     all accepted

So a clause matching the text `--force` is bypassed by three other spellings of
it. Long options must be resolved against the command's declared option set.

**FACT 2026-08-19,** from the tree:

    rm -rf x    ->  Word "rm"  Word "-rf"  Word "x"     no -f word exists
    rm -- -f    ->  Word "rm"  Word "--"   Word "-f"    -f is a filename here

Short options bundle into one word, and `--` turns the rest into operands. A
matcher that looked for an argument equal to `-f` would miss the first and
wrongly deny the second.

