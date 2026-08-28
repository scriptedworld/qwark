# qwark, what is not done

`REQUIREMENTS.md` states the same ground as requirements. This file says what to
do about it, and what is waiting on an answer.

---

## Built, committed, and passing the gate

The jig passes all twelve tasks; coverage 94.3% with every file above the 80%
per-file floor; 207 tests carrying a `COVERS` line; **127** requirements, of which
19 have no test and every one of those is deferred.

    bolt -c ../bolt/bolt.go-std-quality.yaml -c bolt.qwark.yaml

Count the requirements independently with
`grep -oE 'FR-[0-9]+\.[0-9]+[a-z]?' REQUIREMENTS.md | sort -u | wc -l`. bolt's
traceability task reports the same number from its own reading.

**There is no `justfile` here, and there should not be one.** Every recipe one
would carry is either a wrapper around the command above or a copy of a task bolt
already defines, which is the second copy free to drift that the no-vendoring
rule forbids. It also puts an executor and an agent-writable task definition
inside the blast radius, which is the thing qwark is being built to refuse.

- `internal/shell`: parse as Bash, gather every fact in one walk, and record the
  parser's own vocabularies (node types, operators, statement flags).
- `internal/command`: ordinals, escape resolution, option decomposition against a
  declared table.
- `internal/rules`: TOML loading, validation, ownership checking, and the
  evaluator.
- `internal/hook`: the PreToolUse contract, and the guarantee that every path ends
  in a decision or exit 2.
- `internal/reach`, `internal/repo`: blast-radius containment, and the branch read
  without running git.
- `internal/cli`: `ast`, `facts`, `rules`, `judge`, `hook`.
- `internal/gate`: the join, where a rule set and a request become a decision.
- `rules/`: the shipped rule set, in several files plus one declaration. Read the
  counts out of `./bin/qwark rules rules/` and not from here.

## Adopting the toolbox jigs

The jigs and their supporting files belong to `../toolbox`; qwark and bolt should
both be symlinks of them, and `link-jigs` is being built to do the symlinking.
bolt still holds a copy from before that fixing started.

- **The definition has been split in toolbox, and bolt's copy predates the
  split.** `bolt.common-quality.yaml` carries `traceability`, `suppressions` and
  `complexity`; `bolt.go-std-quality.yaml` carries `format`, `tidy`, `build`,
  `vet`, `lint`, `tests`, `coverage` and `vuln`. Eleven between them. Bolt's
  single file still has all twelve in one, `entrypoint` included.
- **qwark passes the newer, stricter traceability today**, run directly against
  it: `108 of 108` requirements held to coverage are covered, and the 19 with no
  test are all `[?]`, open and exempt. bolt was not failing on uncovered
  requirements because it runs the older checker, which reported them as context.

Adopting turned up two things. The first is still open:

1. **`entrypoint` must be defined in full, not overridden.** It is *deliberately*
   absent from the shared jig, since *"a shared definition carries the rule and
   never the subject"*, and a task naming `./cmd/bolt` fails for every adopter in
   a way that looks like the adopter's fault. `bolt.qwark.yaml` currently supplies
   only `command` and says the rest is inherited; under the toolbox jig there is
   nothing to inherit, so it needs `description`, `tags`, `requires`, `timeout`
   and the declared `output` as well.
2. **The `.gitignore` carries the linked set**: `bin/`, covering both
   `bin/qwark` and the two linked checker scripts, alongside
   `bolt.common-quality.yaml`, `bolt.go-std-quality.yaml`, `bolt.secrets.yaml`,
   `adapters/`, `config/`, `coverage.out` and `.bolt*/`. **`bolt.qwark.yaml` stays
   tracked**, because the overlay is this project's own content and is exactly the
   part a shared definition must not carry.

   A tracked symlink stores its target path as content, which encodes where
   `../toolbox` sits, and committing the jig at all is vendoring. Adoption is a
   step run per clone, like fetching modules, and a missing link makes bolt fail
   loudly on a path that is not there. The jig files are also baked into the anvil
   images, so there they arrive from the layer beneath and a copy committed here
   would be a third statement of the same thing.

## git, classified

**git is classified across all 64 porcelain commands**, checked mechanically
against `git --list-cmds`. Nine groups, each carrying its own reason; overlaps are
intended and every reason is collected. Read-only is allowed by my ruling, and
that allowance is narrow because `05-declarations.toml` omits the dangerous
options, not because a rule names them.

## Next: per-agent command surfaces

This resets the target instead of extending it:

> Eventually the goal is that the agents will **never** be executing git
> commands, and instead have a very specific command surface they are allowed.
> The more specialised the agent, the more the list can be narrowed. One would
> expect that either the list ends up a duplication of a series of allowed
> commands in the agent text or supporting files, or those end up referencing
> these rule files.

- **The mechanism does not already exist.** Choosing rule files per agent from an
  external process (FR-10.6a) holds only when every specialised agent is its own
  session. **The registration is fixed for a session, so a subagent inherits its
  parent's command line**, and a partition chosen by the launcher collapses.
  FR-10.6a is revised; see below.
- **The read-only git allowance is a waypoint.** It stands because it was ruled
  on, and it is the first thing to remove once the specific surfaces exist.
- **The duplication is settled.** The proxies hold the details of what they
  expose, so the exposed surface is both the statement of what an agent may do and
  the thing that enforces it: one artifact, with nothing to keep in step. A
  surface says *which operations* and not *with what values*, so whatever
  argument-level constraint remains is what rules are still for.

### The engine carries the separation, and it is buildable now

*"The point of the rules is using the engine to support that separation of
duties."* The answer to an agent writing a `justfile` and then running `just` is
not an unwritable file. It is that **the agent that can write those files is not
the agent allowed to run them.**

Doing it in the plumbing was ruled out. The base session has no `agent_type`, so a
launcher-side partition means *"actively managing symlinks or something else …
some form of ENV VAR that will have to be actively managed … which feels
rickety"*. FR-10.6 had already chosen the payload over an environment variable,
because the subject can reach an environment variable and cannot set its own
`agent_type`.

Two requirements state the shape, and both now carry tests:

- **FR-7.12.** A clause may name the agent the request came from, compared whole.
  `agent = "gate-runner"`.
- **FR-7.13. Absence is a role.** A main-session call reliably carries no agent
  type, so `agent = ""` names it exactly, while stating no `agent` at all remains
  the distinct case that covers every caller. The clause records whether the key
  was stated, not only what it said, so the two cannot collapse.

FR-7.13 is what removes the ricketiness: **one rule set, named once in
`settings.json`, carrying every role's policy inside it.** No symlink swapping, no
environment variable, nothing to keep in step outside the file being read.

Try it:

    ./bin/qwark judge --agent=gate-runner rules/ -- git status

It composed instead of adding a mechanism. An `agent` clause is a clause, rules
stay conjunctions, strictest still wins, and a role cannot grant itself anything
because deny outranks allow. An agent allowance also reaches only the command its
rule names.

**Still to do.** `rules/` carries no agent-scoped rule, because which agent types
exist is not qwark's to invent. The vocabulary in `00-structure.toml` documents
the clause, and the policy waits on the roles being named.

## Mode one runs: `qwark hook`

*"Mode One is the most useful here & now because I don't have the rest of the
system built, so that development needs further quality controls."*

`qwark hook RULES...` reads one call from stdin, judges it, and answers on stdout:
the subcommand `install/settings-fragment.json` has been naming all along.

    printf '{"hook_event_name":"PreToolUse","tool_name":"Bash",
             "tool_input":{"command":"git status"}}' | ./bin/qwark hook rules/

The payload's `agent_type` reaches `rules.Context`, so the agent clause is fed and
not merely expressible.

A decision exits 0 with the verdict in the JSON; a truncated payload exits 2;
**so does invoking it with no rules path**, which reads oddly for a usage error
and is the only correct answer, since every other non-zero status is a
`non_blocking_error` that lets the command run. A rule set that will not load
denies with the file named and points at the Edit tool, because the way out must
not need the thing just taken away. A tool qwark does not model is refused rather
than waved through, so a matcher wide enough to send Write here blocks loudly
instead of judging nothing while looking installed.

**Installed, and it has gated a live session.** `qwark` is on `PATH` at
`/usr/local/bin/qwark`, and `.claude/settings.local.json` registers
`qwark hook /etc/qwark/rules || exit 2` on `PreToolUse` for Bash. It gated a
whole session in this tree on 2026-08-28, refusing `ls`, `cat`, `grep`, `go`,
`bolt`, `qwark` itself and both mandated commit forms, while allowing
`git status` and `git log`.

**The registration is parked right now**, at `.ephemera/settings.local.json.wedged`,
so this tree is ungated. `START_HERE.md` says why and what restores it.

**`/usr/local/bin/qwark` is root-owned and stale.** It still contains the
ownership check retired in `fa9c9cd`. `bin/qwark` is user-owned and current, and
the no-root direction means the hook should name that instead. Filed at
`clank/inbox/qwark/binary-is-installed-to-a-root-path`.

**Two limits to know before relying on it.** Two *main sessions* are
indistinguishable: if writer and runner are both top-level launches, no clause
tells them apart and the launcher must still differ. And a partition does not stop
a chain, since writer writes, runner runs, and neither breaks its own rules. The
task management process is what sits between them, the same process that produces
the manifest of FR-9.7, so the manifest is what makes the partition mean
something.

Useful when working on it:

- `./bin/qwark rules rules/` for the counts; `./bin/qwark judge rules/ -- <cmd>`
  to try a rule before trusting it.
- A declaration grants understanding, not permission, but it *is* the eligibility
  surface, so adding one is a wider change than adding a rule.
  `05-declarations.toml` says so at the top and lists what is deliberately left
  out.
- `.ephemera/demo-allow.toml` holds throwaway declarations used to exercise the
  chain. It is not part of the rule set, and is gitignored.

## The horizon: a proxy, and what survives it

The next layer is an MCP server that becomes the proxy for the tools and
mechanicals to be allowed: *"then the situation gets much more simple … once we
have the proxy, then we can have these kinds of rules for the various tools per
agent type."*

`REQUIREMENTS.md` already says **the *first* mode gates the Bash tool**, so this
is the second and not a replacement. It matters for what to invest in now:

- **The engine carries over.** Conjunctions, strictest-wins, deny-by-default,
  declarations, groups, reasons that explain themselves: none of that is about
  shells.
- **The shell half is mode-one adapter.** Tier one exists because a command line
  can hide its own effect; a typed tool call cannot, so quoting, aliases,
  functions, `PATH`, wrappers and globs stop being problems instead of being
  solved.
- **FR-7.12 and FR-7.13 are foundational, not interim.** "Rules for the various
  tools per agent type" is the agent clause. Build them.
- **The mechanicals become API design.** Allowed-as-a-word-refused-in-a-shape
  becomes a parameter that is not offered.
- **The residue survives.** A proxy operation that runs `just checks` still has
  its meaning in the `justfile`. Either the proxy owns the recipe, or the manifest
  keeps it out of the agent's write surface.

## Three contradictions inside the rule set

1. **`commit-must-be-signed` fires on a signed commit.** `--gpg-sign` is not
   declared, so the `absent = true` clause holds and the rule tells somebody who
   signed that they must sign. The verdict fails safe; the message does not. **A
   "refused unless" rule is only honest when the option it excepts is declared**,
   written up as shape 3 in
   `docs/PATTERNS/the-mechanicals-the-shapes-a-rule-can-be-written-in.md`. Moot
   while `git commit` is denied by class, and live the moment it is not.
2. **The post-rebase tag machinery cannot fire.** `git rebase` is denied, and a
   denied command has no effect of any kind (FR-4.24), so `note-rebase` never sets
   the tag and everything in 40-state.toml that depends on it is unreachable. That
   file is illustrative and tags are deferred, so this is not a fault, but it does
   mean the worked example is exercised by nothing.

   The related break in it was real and is fixed. `git reflog` had been denied as
   history-rewriting, which made `no-deleting-after-a-rebase`'s own instruction
   impossible to follow. The word is allowed, and `expire`, `delete` and `drop`
   are denied at ordinal 2.
3. **`no-touching-qwark` guards two paths that hold nothing, and neither of the
   two that hold everything.** `group.qwark-control` names `/etc/qwark/` and
   `/var/lib/qwark/`. The first is abandoned and the second does not exist. The
   live rule set is at `~/.config/qwark/rules` and the decision log at
   `~/.local/state/qwark/`, and `match = "partial"` compares fragments, so
   nothing covers them. Judged against `rules/`, `cp` over `00-structure.toml`
   and `rm` of `decisions.jsonl` are both **allow**, while `ls /etc/qwark/rules`
   is refused.

   The group was written when `/etc/qwark/rules` was the install target;
   `gate/30-install-to-a-user-owned-path` moved the set and left the guard
   pointing at the old address. This is the third of the three things FR-4.17's
   retirement leaned on, after the `permissions.deny` twin, that was assumed to
   hold and does not.

   Not applied here: hard rule 4a wants a person to agree a rule change in words
   first. Written up with the measurement and a repro in
   `clank/tasks/qwark/rules/40-the-path-group-guards-dead-paths.ready/`.

## What installing the source set costs, measured

The structural-only phase is being validated in live sessions now. The question
for the phase after it is whether a session can still do the work, and it is
answered rather than predicted.

FACT 2026-08-28. 84 unique commands, judged against both sets with `bin/qwark`
built from `18e4f8a`: every command in the live decision log, plus a floor of
what this repository cannot be worked in without. Live allows 79 of 84, source
allows 70. `.ephemera/can-work-continue.py` regenerates it; the results are kept
in `clank/tasks/qwark/rules/40-the-path-group-guards-dead-paths.ready/evidence/`.

**Nine commands are allowed today and refused by the source set. Four of them
are how qwark gets built and gated:**

    go test ./...                no-go-execution   compiles and runs this tree
    bolt -c bolt.qwark.yaml      no-executors      runs a recipe from the tree
    python3 <script>             no-interpreters   runs code given as an argument
    sed -n 1,40p FILE            no-interpreters   same class, though this reads

Each denial is correct about the general case and each stops the project
developing itself. `78e0410` built the declaration table, and it does not help:
these are deny rules, which fire whatever is declared. So the gap is not
declaration coverage, and no amount of it closes this.

The remaining five are wanted, or nearly. `rm -rf` and reaching a PATH directory
are deliberate. `ls` on the rules directory is the guard in contradiction 3,
firing on the dead path, and `CLAUDE.md` rule 4a explicitly permits reading
either copy, so refusing a read is over-broad even once the group is corrected.

**The mechanism is built.** `a85b1b9` adds a clause selecting on `cwd`, which
arrives in the payload on the same footing as `agent_type` and which the subject
cannot set. `go test` and `bolt` can stay refused everywhere and run inside this
tree. Measured against a probe: allow at the root, allow in a subdirectory, deny
from another repository, deny from a neighbour whose name shares a prefix, and
deny when the request carries no directory at all.

**Writing the scope is not what the reflex suggests.** There is no overridable
deny, so a scoped allow beside the existing deny leaves both live and the
command refused. The scope goes inside the deny rule as an inverted `cwd`
clause, which also fails closed. Shape 6 in
`docs/PATTERNS/the-mechanicals-the-shapes-a-rule-can-be-written-in.md`.

**What is left is the policy, and it is a rules change.** Which denials become
tree-scoped, and to which trees, wants an answer in words before anything is
written. The four that block this repository are the obvious first set:
`no-go-execution`, `no-executors`, and `no-interpreters` for `python3` and for
`sed`. The last is worth separating: `sed -n 1,40p FILE` reads a file and is
caught by a rule about running code supplied as an argument.

## Waiting on an answer

1. **How tag state survives between calls.** The shape is settled and the
   foundation is in place, but there will be no store until there are concrete
   scenarios worth limiting this way. Nine requirements sit behind it (FR-4.7,
   4.13, and section 8).

   **The blocker is not the shape, it is the writer.** `40-state.toml` requires
   that tag state not be writable by the user qwark runs as, and qwark runs as the
   agent's user, so any file it maintains the agent can rewrite with the Write
   tool. The leading candidate, an appended file trimmed each run, is precisely
   the option that fails that. Mode two's log inherits the problem, and worse: a
   log with entries removed reads as a clean history. **The proxy is the way
   out**, because a long-lived process the agent reaches only by typed call is a
   writer the subject is not. See **The leaking bucket has no honest home in mode
   one**.
2. **The observability log:** where it lives, whether it rotates, and the list of
   environment variables whose values are withheld. Three requirements (FR-4.8,
   4.9, 4.9a). The withhold model is a denylist by deliberate choice, with pattern
   matching added because naming secrets one at a time fails open.
3. **Which commands write.** FR-9.6 says any path given to a writing command must
   stay in the blast radius, and nothing yet says which commands write. That is a
   declaration question: a `writes` flag per command, or per option.
4. **The manifest** (FR-9.7), created by the task management process, read at
   runtime, saying which files may be read and which written.

**3 and 4 are the priority.** The end state is three layers: a sandbox, the blast
radius, then the manifest. The sandbox absorbs four of the six path groups in
`20-paths.toml`, because those files are simply not in it. **Two are inside the
sandbox and no sandbox removes them**: `repository-hooks` and `task-definition`.
The blast radius does not help there either, since a `justfile` is already inside
the project, which is the one place the agent must be able to write.

So the manifest is the only layer of the three that discriminates between files
inside the blast radius, and it is therefore the layer that answers "if they can
write a new file, they can get the agent to approve anything". Both of its
requirements are `[?]` and unbuilt. See the design note **The end state is three
layers**.

## Known limits, written down so they are not rediscovered

- **FR-4.18 has a live instance, and it was found by being bitten.** That
  requirement says using a name the shell may resolve to something other than
  the intended program is refused, and notes that a backslash suppresses alias
  expansion but not a shell function. Measured 2026-08-28 on this machine:

      \grep -n   "THE GATE IS ARMED" qwark/START_HERE.md   ->  line 8
      \grep -rln "THE GATE IS ARMED" qwark/                 ->  no match

  Same binary, same string, same file. Recursive `grep` silently skips
  gitignored files, so `START_HERE.md` and everything under `.ephemera/` are
  invisible to it, and the backslash does not restore the real binary. It
  produced a false clean on a real check: a recursive search reported no dead
  clank SHAs in this repository while `START_HERE.md` held three.

  FR-4.18 is `[?]` and carries no test. This is the evidence that it is worth
  building rather than deferring, and it is also a caution about qwark's own
  measurements: any finding here derived from a recursive `grep` has a blind
  spot the size of the gitignore.
- **The `[shell]` policy is parsed and never consulted.** `ShellPolicy.Verify`
  is defined at `internal/rules/shell.go:78` and called from nothing but
  `shell_test.go`, and no code reads `SHELL` from the environment. So FR-1.5,
  FR-1.7, FR-1.8, FR-1.9 and FR-1.10 describe a check with no caller, while
  `00-structure.toml` declares `allow = ["/bin/bash", "/usr/bin/bash"]` and
  reads as though bash were enforced. Confirmed 2026-08-28. Filed at
  `clank/inbox/qwark/shell-policy-is-parsed-and-never-consulted`, which also
  measures that the Bash tool's shell is zsh carrying the user's aliases.
- **The observation phase is running.** It was blocked on FR-4.16: the engine
  denied an undeclared command unconditionally, so a rule set omitting
  declarations denied everything rather than judging by shape. `f39b70b` settled
  that the enforcement stays and gave it a switch, and the live `06-allow.toml`
  sets `required = false`. Measured 2026-08-28: against the live set `ls -la`,
  `cat`, `grep`, `go build` and `git commit -F` all run, and only shape is
  refused.
- **qwark gates Bash only.** The Write and Edit tools reach the rule files, the
  shell snapshot, `.git/hooks` and `settings.json` without passing through it.
  Every class-three rule needs a `permissions.deny` twin. **A twin naming
  `settings.local.json` itself removes the escape hatch from the session**, so
  the documented way out, deleting the `hooks` key with the Edit tool, stops
  being available to anyone but a person. Measured 2026-08-28.

  **The twin is written and does not hold.** `.claude/settings.local.json`
  carries four `deny` entries covering `Write` and `Edit` on the live rules and
  on the registration itself. Measured 2026-08-28: `rm` through Bash and `Write`
  through the tool both reached `~/.config/qwark/rules` past them. So the
  property FR-4.17 dropped is carried by neither the twin nor, until
  contradiction 3 is fixed, by qwark's own path group. **Nothing mechanical
  protects the live rules today**, and hard rule 4a, an instruction to an agent,
  is the whole of it.

- **No drift check runs.** The live set and the source set are two copies with
  nothing comparing them, and the figure in `CLAUDE.md` describing them has gone
  stale twice. There is no `qwark verify` subcommand; `qwark --help` lists `ast`,
  `facts`, `rules`, `judge` and `hook`. A gate that cannot check its own
  deployment is the candidate this wants, and the earlier objection that `diff`
  and `sha256sum` were themselves refused no longer applies: both run under the
  live set.
- **The registration is project-scoped, so looking for it at user scope finds
  nothing.** It lives in `qwark/.claude/settings.local.json` and gates sessions
  in this tree only. `~/.claude/settings.json` symlinks into silo and holds
  `SessionStart` and `SessionEnd` and no `PreToolUse`, which is correct rather
  than a fault: no Bash command **outside this repository** is judged, and that
  is the phase. An inbox entry filed 2026-08-28 read the user-scope file and
  concluded qwark had never been registered anywhere. Check both scopes.

- **A coding agent that can write files and run its tests has arbitrary
  execution** regardless of qwark. `go test` runs code the agent just wrote. What
  qwark constrains is what is typed, not what the typed thing executes.
- **The hook registration is fixed for a session.** An external process can choose
  rule files per session launch, but a subagent spawned inside a running session
  gets the same command line. Varying policy per subagent would need qwark reading
  `agent_type` after all.
- **`SHELL` decides the Bash tool's shell at session start** and cannot be changed
  by a hook afterwards. Forcing bash means exporting it before launching Claude
  Code. This machine runs zsh 5.9 there.

## Deferred to a later version

- **Tags** (section 8). The foundation is in place; there is no store and no
  scenario yet.
- **Cost ordering** (FR-4.2). It cannot change a verdict, only how much work
  happens before the verdict is known. The seam is `order` in the evaluator, today
  the identity.

## Open questions that used to sit in DESIGN-NOTES

| Question | State |
|---|---|
| Does a *denied* command decrement a sticky tag's countdown? | **SETTLED.** FR-4.24: a denied command has no effect of any kind. The Redis shape makes it structural rather than remembered, since a denied command issues no update and so cannot tick. |
| `substitution.parameter` bans `$HOME` and `$PWD` along with the rest. Intended? | **SETTLED, and intended.** `rules/00-structure.toml` says so outright: *"command, process, arithmetic and parameter alike, so $HOME and $PWD are included."* |
| Verdict for a command qwark cannot parse. | **SETTLED.** FR-4.12: denied, with the parser's own message, which carries the line and column. |
| Which environment variables may be logged by value; where the log lives; whether it rotates. | **STILL OPEN.** FR-4.8, FR-4.9, FR-4.9a, all `[?]`. This is mode two, the audit, and the same store question the leaking bucket runs into. |
