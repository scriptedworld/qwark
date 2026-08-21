# Rule files are named on the command line

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 184-193. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**PREFERENCE, owner, 2026-08-19.** Rule files are given as arguments: a path
naming a directory contributes every rule file in it, a path naming a file
contributes that one.

The gain is that **the policy in force is readable where qwark is invoked** —
in the `settings.json` entry that registers the hook — rather than being implied
by which files happen to be sitting in a directory qwark knows about.

