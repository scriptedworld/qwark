# Installing qwark

Putting the gate in front of a live Claude Code session: build it, deploy a rule
set, register the hook, and know how to get out again.

## Build

    go build -o bin/qwark ./cmd/qwark

`just build` does the same with `CGO_ENABLED=0`, and `just install` puts the
binary where the registration expects it, comparing bytes rather than timestamps
so a second run reports it current instead of rewriting it.

**Rewriting that binary changes what a live session may do**, immediately and
with no restart, because it is the program every Bash call is already passing
through. Nothing warns you.

## Deploy a rule set

    install -d ~/.config/qwark/rules
    install -m 0644 rules/*.toml ~/.config/qwark/rules/

The rules directory is an argument to `qwark hook`, so it can live anywhere the
registration can name and a machine can keep several. Copying the shipped set
rather than pointing at this tree is deliberate: the live policy and the policy
under development are then separate things, and changing one does not change the
other by accident. Nothing compares the two for you.

A deployment names which files it loads, and the shipped set is not obliged to
be the loaded set. `docs/RULES.md` says what each file holds, which is what you
choose between.

**The rule files must not be writable by the user the agent runs as, and neither
may the directory holding them.** A writable directory permits unlink and
replace, which defeats an unwritable file.
`docs/DECISIONS/rule-files-must-not-be-writable-by-the-agent.md` carries what
that costs and when it is worth paying.

## Register the hook

`install/settings-fragment.json` is the registration to merge into
`settings.json`. It carries a `_comment` key for a reader, which
`settings.json` has no syntax for and which must not survive into the real file.

    "command": "qwark hook ~/.config/qwark/rules || exit 2"

The fragment ships this pointing at `/etc/qwark/rules`. Change it to wherever
you installed the rules.

### `|| exit 2` is the only fail-closed exit

Claude Code treats exit 0 with no JSON as no decision, and every non-zero status
other than 2 as a `non_blocking_error`. The command proceeds in both cases. Only
exit 2 blocks.

So a qwark that segfaults, is killed for running out of memory, or exits 1 the
way a Unix program usually reports failure, lets the command through. qwark
recovers from its own panics and runs in a single goroutine so that it can, and
every path through the hook ends in a decision or in exit 2. What it cannot
catch from inside its own process is what the shell wrapper is for.

### The deny list is the other half, and it is not optional

qwark gates Bash. The Write and Edit tools reach every path a rule protects
without passing through it, so a rule enforced against a shell and nothing else
is worth very little. Each path a rule protects needs a twin entry in
`permissions.deny`, which is one control written twice because two mechanisms
enforce the two halves.

The fragment carries one representative of each class the shipped rules protect:
qwark's own rules and state, Claude Code's configuration and hooks, shell
startup files, git hooks and config, and task definitions such as `Justfile`,
`Makefile` and `package.json`. Keep it in step with the groups in
`rules/20-paths.toml` by hand. Nothing checks that for you.

Two things decide whether an entry works at all.

**A bare path is relative to the settings file's root**, the project root for
project settings and `~/.claude` for user settings, while `//` is absolute and
`~/` is home. That splits the list in two: the absolute and `~` entries belong
in user settings, and the bare task-definition entries belong in project
settings, where they resolve against the project root. In user settings a bare
`Justfile` would mean `~/.claude/Justfile` and match nothing.

**Resolve symlinks on the machine you install on.** Whether Claude Code resolves
a symlink before matching is unverified, so list both spellings where the target
is known, and derive them with `readlink -f` rather than copying somebody else's
paths. On the machine these were first written, three of four moved within a
day.

## Check it before you rely on it

`judge` takes the same rule paths and the same request fields the hook does, so
the policy can be exercised as the caller that will meet it.

    qwark judge ~/.config/qwark/rules -- git status
    qwark judge --agent=test-runner ~/.config/qwark/rules -- go test ./...

The hook itself reads one call from stdin and answers on stdout:

    printf '{"hook_event_name":"PreToolUse","tool_name":"Bash",
             "tool_input":{"command":"git status"}}' | qwark hook ~/.config/qwark/rules

A decision exits 0 with the verdict in the JSON. A truncated payload exits 2,
and so does invoking it with no rules path, which reads oddly for a usage error
and is the only safe answer available: every other non-zero status lets the
command run.

A rule set that will not load denies with the file named and points at the Edit
tool, because the way out must not need the thing just taken away. A tool qwark
does not model is refused rather than waved through, so a matcher wide enough to
send Write here blocks loudly instead of judging nothing while looking
installed.

## The decision log

Every decision is appended to `$XDG_STATE_HOME/qwark/decisions.jsonl`, falling
back to `~/.local/state/qwark/decisions.jsonl`. One JSON object per line:

    {"at":"...","rule_set":"47d8fcfa576a62b0","decision":"allow","tool":"Bash",
     "command":"git status","rules":["allow-reading-the-repository"],
     "agent":"","cwd":"/srv/project","session":"..."}

`rule_set` is a digest of the rules that judged the call, which is what lets
entries made under different policies be told apart rather than compared with
each other. A field that could be absent is omitted rather than written null, so
`agent` missing means no agent type reached the hook.

The file is opened for append and never truncated. A log with earlier entries
missing reads as a clean history and answers "what happened" confidently and
wrongly.

**A failure to record does not change a verdict.** The decision is made before it
reaches the log, and refusing when the log is unwritable would turn a full disk
into a way of stopping every command. That is the permissive direction, chosen
deliberately: somebody who can fill the disk can stop the recording without
stopping the commands.

## Getting out

**A change to a registered hook takes effect on the very next command, with no
restart.** That is what makes a bad rule set immediate, and it is also what
makes the repair immediate.

Move the settings file aside. It needs no shell and no root, which matters
because the shell is what has just been taken away.

    mv .claude/settings.local.json .claude/settings.local.json.wedged

If `permissions.deny` names the settings file itself, the Edit tool cannot do
this and a person at a terminal has to.
`docs/LESSONS/a-hook-change-takes-effect-immediately-and-can-lock-you-out.md`
is the account of finding that out.
