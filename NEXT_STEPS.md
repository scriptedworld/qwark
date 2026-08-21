# qwark — what is not done

State at 2026-08-20. `REQUIREMENTS.md` states the same ground as requirements;
this file says what to do about it and what is waiting on an answer.

---

## Built, committed, and passing the gate

**FACT 2026-08-20**, measured this session: the jig passes all twelve tasks;
coverage 94.3% with every file above the 80% per-file floor; 207 tests carrying
a `COVERS` line; **127** requirements of which 19 have no test, and every one of
those is deferred.

    bolt -c ../bolt/bolt.go-std-quality.yaml -c bolt.qwark.yaml

*Corrected 2026-08-20: this said 125 requirements when there were 124 —*
*`grep -oE 'FR-[0-9]+\.[0-9]+[a-z]?' REQUIREMENTS.md | sort -u | wc -l`, and*
*bolt's traceability task reports the count independently. It is 127 now:*
*FR-10.10, FR-7.12 and FR-7.13 were added this session.*

**REMOVED 2026-08-20: the `justfile`.** The owner did not add it — it arrived in
`c812a9e`, the scaffold commit, and `CLAUDE.md` then recorded `just checks` as
"the gate" as though that had been decided. Every recipe was either a one-line
wrapper around the command above or a copy of a task bolt already defines, and
its `tests` recipe was **byte-for-byte** bolt's `tests` command — the second
copy free to drift that the no-vendoring rule exists to forbid. It also made
qwark's own repository carry an executor and an agent-writable task definition
inside the blast radius, which is the thing qwark is being built to refuse.

- `internal/shell` — parse as Bash, gather every fact in one walk, and record
  the parser's own vocabularies (node types, operators, statement flags).
- `internal/command` — ordinals, escape resolution, option decomposition
  against a declared table.
- `internal/rules` — TOML loading, validation, ownership checking, and the
  evaluator.
- `internal/hook` — the PreToolUse contract, and the guarantee that every path
  ends in a decision or exit 2.
- `internal/reach`, `internal/repo` — blast-radius containment; branch read
  without running git.
- `internal/cli` — `ast`, `facts`, `rules`, `judge`, `hook`.
- `internal/gate` — the join: a rule set and a request become a decision.
- `rules/` — 50 rules across six files, and one declaration. Read the counts
  out of `./bin/qwark rules rules/` rather than from here.

## Adopting the toolbox jigs

**Owner, 2026-08-20:** the jigs and their supporting files belong to
`../toolbox`; qwark and bolt should both be symlinks of them, and `link-jigs` is
being built to do the symlinking. **`bolt` may still hold a copy from before
that fixing started** — measured, it does.

**FACT 2026-08-20**, measured, not inherited:

- **The definition has been split in toolbox and bolt's copy predates it.**
  `bolt.common-quality.yaml` carries `traceability`, `suppressions`,
  `complexity`; `bolt.go-std-quality.yaml` carries `format`, `tidy`, `build`,
  `vet`, `lint`, `tests`, `coverage`, `vuln`. Eleven between them. Bolt's single
  file still has all twelve in one, `entrypoint` included.
- **qwark passes the newer, stricter traceability today**, run directly against
  it: `108 of 108` requirements held to coverage are covered, and the 19 with no
  test are all `[?]` — *open and exempt*. **This is why bolt was not failing on
  uncovered requirements: it is running the older checker**, which reported them
  as context rather than holding settled ones to a test.

Two things qwark must fix to adopt, both found by reading the toolbox files:

1. **`entrypoint` must be defined in full, not overridden.** It is
   *deliberately* absent from the shared jig — *"a shared definition carries the
   rule and never the subject"*, and a task naming `./cmd/bolt` fails for every
   adopter in a way that looks like the adopter's fault. `bolt.qwark.yaml`
   currently supplies only `command` and says the rest is inherited; under the
   toolbox jig there is nothing to inherit, so it needs `description`, `tags`,
   `requires`, `timeout` and the declared `output` as well.
2. **The `.gitignore` needs the linked set added.** Not `bin/` — that one is
   already right.

   **Owner, 2026-08-20:** *the things linked from the other repo should be
   gitignored, as should any Go executable we generate — it's obviously
   regenerateable.* So `bin/` covering both `bin/qwark` and the two linked
   checker scripts is the intended state rather than a collision.

   *Corrected from an earlier note here that called the ignored `bin/` a problem
   and suggested handing it to `link-jigs` as a failure to refuse. It is not a
   failure. A tracked symlink would store its target path as content, which
   encodes where `../toolbox` sits — and committing the jig at all is vendoring,
   which is the thing the no-vendoring rule already forbids. Adoption is a step
   run per clone, like fetching modules; a missing link makes bolt fail loudly on
   a path that is not there, which is the safe direction.*

   What still needs entries, since only `bin/`, `coverage.out` and `.bolt*/` are
   ignored today: `bolt.common-quality.yaml`, `bolt.go-std-quality.yaml`,
   `adapters/`, and `config/go-std-quality.golangci.yml`. **`bolt.qwark.yaml`
   stays tracked** — the overlay is this project's own content, and is exactly
   the part a shared definition must not carry.

   **Owner, 2026-08-20: the jig files also get baked into the anvil images**, so
   a project's overlay refers to them there when its own file builds on top of
   the anvil. That makes the symlinks the local-development route and the image
   the built one, with the same set reached either way — and it is why ignoring
   them is right rather than merely tolerable: in the image they come from the
   layer beneath, so a copy committed here would be a third statement of the
   same thing.

   Reading `anvil/README.md` and `toolbox/GLOSSARY.md` rather than inferring:
   bolt runs the checks, toolbox holds the jigs, anvil is what you run them on,
   and none of the three depends on the other two in a circle. **An image
   installs exactly the `requires:` of its matching jig** — *"the image manifest
   is the jig, not a second list"*, which is the no-second-copy rule again, one
   layer down.

## Done 2026-08-20: the stated denials, and git classified

Three things the project had already said it never wanted, which no rule
enforced. Each was measured before and after.

- **Globs.** `fact = "glob"` was computed by the engine and consumed by no
  rule, so `rm *`, `rm ?.txt` and `rm *(e:'rm -rf /':)` were all allowed. Now
  `no-glob` in tier one, where the property it defends is stated. Owner's
  ruling: deny, no exception.
- **Undeclared options.** FR-6.7 held in `internal/command` and did not reach
  the verdict — `rm -Z x` was allowed. `internal/rules/evaluate.go` now
  consults `Options.Faults`, and reports every one of them rather than the
  first.
- **`git config`.** Refused only by deny-by-default, saying nothing, while
  `git -c` was denied by name. The persistent spelling was the unguarded one.

**git is classified across all 64 porcelain commands**, checked mechanically
against `git --list-cmds` rather than by eye. Nine groups, each carrying its own
reason; overlaps are intended and every reason is collected. Read-only is
allowed per the owner's ruling, and that allowance is narrow because
`05-declarations.toml` omits the dangerous options, not because a rule names
them.

## Next: per-agent command surfaces

**Owner, 2026-08-20**, and this resets the target rather than extending it:

> Eventually the goal is that the agents will **never** be executing git
> commands, and instead have a very specific command surface they are allowed.
> The more specialised the agent, the more the list can be narrowed. One would
> expect that either the list ends up a duplication of a series of allowed
> commands in the agent text or supporting files, or those end up referencing
> these rule files.

- **CORRECTED 2026-08-20: the mechanism does *not* already exist.** This said
  the external process chooses rule files per agent (FR-10.6a) and that
  narrowing needed more files rather than more machinery. That holds only when
  every specialised agent is its own session. **The registration is fixed for a
  session, so a subagent inherits its parent's command line** and a partition
  chosen by the launcher collapses. FR-10.6a is revised; see below.
- **The read-only git allowance is a waypoint.** It stands because it was ruled
  on, and the direction above says the eventual answer is narrower. It is the
  first thing to remove once the specific surfaces exist.
- **The duplication is settled, 2026-08-20.** The proxies hold the details of
  what they expose, so the exposed surface is both the statement of what an
  agent may do and the thing that enforces it — one artifact, nothing to keep in
  step. How much it settles depends on how narrow the tools are: a surface says
  *which operations*, not *with what values*, so whatever argument-level
  constraint remains is what rules are still for.

### The engine carries the separation, and it is buildable now

**Owner, 2026-08-20:** *"The point of the rules is using the engine to support
that separation of duties."* The answer to an agent writing a `justfile` and
then running `just` is not an unwritable file — it is that **the agent that can
write those files is not the agent allowed to run them.**

The owner ruled out doing it in the plumbing: the base session has no
`agent_type`, so a launcher-side partition means *"actively managing symlinks or
something else … some form of ENV VAR that will have to be actively managed …
which feels rickety"*. FR-10.6 had already chosen the payload over an
environment variable, for the reason that **the subject can reach an environment
variable and cannot set its own `agent_type`.**

**BUILT 2026-08-20.** Two requirements state the shape and both now carry tests:

- **FR-7.12** — a clause may name the agent the request came from, compared
  whole. `agent = "gate-runner"`.
- **FR-7.13** — **absence is a role.** A main-session call reliably carries no
  agent type, so `agent = ""` names it exactly — while stating no `agent` at all
  remains the distinct case that covers every caller. The clause records whether
  the key was stated, not only what it said, so the two cannot collapse.

FR-7.13 is what removes the ricketiness: **one rule set, named once in
`settings.json`, carrying every role's policy inside it.** No symlink swapping,
no environment variable, nothing to keep in step outside the file being read.

Try it:

    ./bin/qwark judge --agent=gate-runner rules/ -- git status

It composed rather than adding a mechanism: an `agent` clause is a clause, rules
stay conjunctions, strictest still wins, and a role cannot grant itself anything
because deny outranks allow. An agent allowance also reaches only the command
its rule names — the clause narrows a rule rather than attaching a permission
to a role.

**Still to do.** `rules/` carries no agent-scoped rule, because which agent
types exist is not qwark's to invent — the vocabulary in `00-structure.toml`
documents the clause and the policy waits on the roles being named.

## Mode one runs: `qwark hook`

**Owner, 2026-08-20:** *"Mode One is the most useful here & now because I don't
have the rest of the system built, so that development needs further quality
controls."*

**BUILT 2026-08-20.** `internal/hook.Run` had been written and tested with
nothing calling it, which meant every other subcommand was a way of asking
qwark questions rather than a gate. `qwark hook RULES...` reads one call from
stdin, judges it, and answers on stdout — the subcommand
`install/settings-fragment.json` has been naming all along.

    printf '{"hook_event_name":"PreToolUse","tool_name":"Bash",
             "tool_input":{"command":"git status"}}' | ./bin/qwark hook rules/

The payload's `agent_type` reaches `rules.Context`, so the agent clause is fed
rather than merely expressible.

Every path was exercised: a decision exits 0 with the verdict in the JSON; a
truncated payload exits 2; **so does invoking it with no rules path**, which
reads oddly for a usage error and is the only correct answer, since every other
non-zero status is a `non_blocking_error` that lets the command run. A rule set
that will not load denies with the file named and points at the Edit tool — the
way out must not need the thing just taken away. A tool qwark does not model is
refused rather than waved through, so a matcher wide enough to send Write here
blocks loudly instead of judging nothing while looking installed.

**Not yet installed.** `qwark` is not on `PATH` as a gate and `settings.json`
carries no hook entry, so nothing is gated yet. That is the next step, and it is
the one that turns this from a program into a control.

**Two limits to know before relying on it.** Two *main sessions* are
indistinguishable — if writer and runner are both top-level launches, no clause
tells them apart and the launcher must still differ. And a partition does not
stop a chain: writer writes, runner runs, neither breaks its own rules.
**Owner, 2026-08-20: the task management process is what sits between them** —
the same process that produces the manifest of FR-9.7. So the manifest and the
partition are not alternatives; the manifest is what makes the partition mean
something.

Useful when working on it:

- `./bin/qwark rules rules/` for the counts; `./bin/qwark judge rules/ -- <cmd>`
  to try a rule before trusting it.
- A declaration grants understanding, not permission — but it *is* the
  eligibility surface, so adding one is a wider change than adding a rule.
  `05-declarations.toml` says so at the top and lists what is deliberately left
  out.
- `.ephemera/demo-allow.toml` holds throwaway declarations used to exercise the
  chain. It is not part of the rule set and is gitignored.

## The horizon: a proxy, and what survives it

**Owner, 2026-08-20:** the next layer is an MCP server that becomes the proxy
for the tools and mechanicals to be allowed — *"then the situation gets much
more simple … once we have the proxy, then we can have these kinds of rules for
the various tools per agent type."*

`REQUIREMENTS.md` already says **the *first* mode gates the Bash tool**, so this
is the second rather than a replacement. It matters for what to invest in now:

- **The engine carries over.** Conjunctions, strictest-wins, deny-by-default,
  declarations, groups, reasons that explain themselves — none of that is about
  shells.
- **The shell half is mode-one adapter.** Tier one exists because a command line
  can hide its own effect; a typed tool call cannot, so quoting, aliases,
  functions, `PATH`, wrappers and globs stop being problems rather than being
  solved.
- **FR-7.12 and FR-7.13 are foundational, not interim.** "Rules for the various
  tools per agent type" is the agent clause. Build them.
- **The mechanicals become API design.** Allowed-as-a-word-refused-in-a-shape
  becomes a parameter that is not offered.
- **The residue survives.** A proxy operation that runs `just checks` still has
  its meaning in the `justfile`. The proxy owns the recipe, or the manifest
  keeps it out of the agent's write surface.

## Two contradictions inside the rule set

Both found by running the rules rather than by reading them.

1. **`commit-must-be-signed` fires on a signed commit.** `--gpg-sign` is not
   declared, so the `absent = true` clause holds and the rule tells somebody who
   signed that they must sign. The verdict fails safe; the message does not.
   **A "refused unless" rule is only honest when the option it excepts is
   declared** — written up as shape 3 in DESIGN-NOTES. Moot while `git commit`
   is denied by class, live the moment it is not.
2. **The post-rebase tag machinery cannot fire.** `git rebase` is denied, and a
   denied command has no effect of any kind (FR-4.24), so `note-rebase` never
   sets the tag and everything in 40-state.toml that depends on it is
   unreachable. That file is illustrative and tags are deferred, so this is not
   a fault — but it does mean the worked example is not exercised by anything.

   The related break in it *was* real and is fixed: `git reflog` had been denied
   as history-rewriting, which made `no-deleting-after-a-rebase`'s own
   instruction impossible to follow. The word is allowed and `expire`, `delete`
   and `drop` are denied at ordinal 2.

## Waiting on an answer

1. **How tag state survives between calls.** Deferred by the owner on
   2026-08-20 — the shape is settled and the foundation is in place, but there
   will be no store until there are concrete scenarios worth limiting this way.
   Nine requirements sit behind it (FR-4.7, 4.13, and section 8).

   **The blocker is not the shape, it is the writer.** `40-state.toml` requires
   that tag state not be writable by the user qwark runs as — and qwark runs as
   the agent's user, so any file it maintains the agent can rewrite with the
   Write tool. The leading candidate, an appended file trimmed each run, is
   precisely the option that fails that. Mode two's log inherits it, and worse:
   a log with entries removed reads as a clean history. **The proxy is the way
   out** — a long-lived process the agent reaches only by typed call is a writer
   the subject is not. See **The leaking bucket has no honest home in mode one**.
2. **The observability log**: where it lives, whether it rotates, and the list
   of environment variables whose values are withheld. Three requirements
   (FR-4.8, 4.9, 4.9a). The withhold model is a denylist by the owner's choice,
   with pattern matching added because naming secrets one at a time fails open.
3. **Which commands write.** FR-9.6 says any path given to a writing command
   must stay in the blast radius, and nothing yet says which commands write.
   That is a declaration question: a `writes` flag per command, or per option.
4. **The manifest** (FR-9.7) — created by the task management process, read at
   runtime, saying which files may be read and which written.

**Owner, 2026-08-20: 3 and 4 are the priority, and not for the reason they look
like.** The end state is three layers — a sandbox, the blast radius, then the
manifest. The sandbox absorbs four of the six path groups in `20-paths.toml`,
because those files are simply not in it. **Two are inside the sandbox and no
sandbox removes them**: `repository-hooks` and `task-definition`. The blast
radius does not help there either, since a `justfile` is already inside the
project — which is the one place the agent must be able to write.

So the manifest is the only layer of the three that discriminates between files
inside the blast radius, and it is therefore the layer that answers "if they can
write a new file, they can get the agent to approve anything". Both of its
requirements are `[?]` and unbuilt. See the design note **The end state is three
layers**.

## Known limits, written down so they are not rediscovered

- **qwark gates Bash only.** The Write and Edit tools reach the rule files, the
  shell snapshot, `.git/hooks` and `settings.json` without passing through it.
  Every class-three rule needs a `permissions.deny` twin plus ownership.
- **A coding agent that can write files and run its tests has arbitrary
  execution** regardless of qwark. `go test` runs code the agent just wrote.
  What qwark constrains is what is typed, not what the typed thing executes.
- **The hook registration is fixed for a session.** An external process can
  choose rule files per session launch, but a subagent spawned inside a running
  session gets the same command line. Varying policy per subagent would need
  qwark reading `agent_type` after all.
- **`SHELL` decides the Bash tool's shell at session start** and cannot be
  changed by a hook afterwards. Forcing bash means exporting it before
  launching Claude Code. **FACT 2026-08-20: this machine runs zsh 5.9 there.**

## Deferred to a later version

- **Tags** (section 8) — foundation in place, no store, no scenarios yet.
- **Cost ordering** (FR-4.2) — cannot change a verdict, only how much work
  happens before it is known. The seam is `order` in the evaluator, today the
  identity.

## The open questions DESIGN-NOTES used to carry

Moved here on 2026-08-21 when that document was split, and **three of the four
had been settled without the list being updated** — which is the argument for
keeping open questions in one place rather than at the end of a design record.

| Question | State |
|---|---|
| Does a *denied* command decrement a sticky tag's countdown? | **SETTLED.** FR-4.24: a denied command has no effect of any kind. The Redis shape makes it structural rather than remembered — a denied command issues no update, so it cannot tick. |
| `substitution.parameter` bans `$HOME` and `$PWD` along with the rest. Intended? | **SETTLED, and intended.** `rules/00-structure.toml` says so outright: *"command, process, arithmetic and parameter alike, so $HOME and $PWD are included."* |
| Verdict for a command qwark cannot parse. | **SETTLED.** FR-4.12 — denied, with the parser's own message, which carries the line and column. |
| Which environment variables may be logged by value; where the log lives; whether it rotates. | **STILL OPEN.** FR-4.8, FR-4.9, FR-4.9a, all `[?]`. This is mode two, the audit, and the same store question the leaking bucket runs into. |
