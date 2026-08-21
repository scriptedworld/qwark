# A declaration is a permission

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 514-525. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**CLAIM, and a caution for the project-level rules file.** Under strictest-wins,
an agent that adds an `allow` rule changes nothing: deny still beats it. But an
agent that adds a *declaration* moves a command from "undeclared, therefore
denied" to eligible. `[command.sh]` is a complete bypass written as
configuration.

So a project-level file that may add declarations needs the same protection as
the global set. A lower-trust project file is possible, but only if it may add
**rules** and never **declarations**.

