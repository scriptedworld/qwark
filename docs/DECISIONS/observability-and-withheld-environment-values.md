# Observability

Every command is logged with the relevant detail from its environment: cwd,
environment variables, and whatever else bears on the decision.

JSONL, following bolt's precedent. One object per command, appended, so the
record is greppable and a partial write costs one line rather than the file.

The log is also the state tracker. Tier four asks questions of the form "was
there a rebase in the last six commands?", which the log already answers. A
separate store maintained alongside it would be a second copy of the same
history, free to disagree with the first. Reading the tail of the log instead
means the record that explains a decision *is* the record that produced it.

### Environment variables are a disclosure risk, and this log is durable

Claude Code spawns the hook as a subprocess, so the hook inherits its whole
environment, which on this machine routinely includes API tokens. Recording
`os.Environ()` verbatim writes those to a file that persists, is grepped later,
and may be read while diagnosing something unrelated. That is the failure
`secret-scan` exists to catch, arriving through a door qwark would have opened
itself.

**Proposed, and waiting on an answer.** Record every variable *name*, so the
shape of the environment is visible and a change in it is detectable. Record
*values* only for names a rule file declares. Anything undeclared is recorded as
present-but-withheld rather than omitted, so the log never silently implies a
variable was absent.
