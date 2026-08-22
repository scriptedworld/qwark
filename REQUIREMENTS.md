# qwark — Requirements

Derived 2026-08-19 from the author's statements that day, recorded in
`DESIGN-NOTES.md`. qwark is a `PreToolUse` hook for Claude Code; the first mode
gates the Bash tool.

Requirements are stated as observable properties: each says what is true of a
run, not how the code is arranged. Mechanism appears only where the mechanism is
itself the requirement.

**The chain is meant to be followed in both directions.** A design entry says
why; a requirement says what must be true; a test says `COVERS:` and names the
requirement it discharges. The `traceability` task enforces the second link
mechanically; the first is enforced by review.

**Status markers.** `[A]` traces to an author statement. `[D]` is derived from
one. `[A/D]` is both. `[?]` is an open question — stated so it is not lost,
carrying no test yet, and reported by the gate as context rather than as a
failure.

---

## 1. Reading a command

*Derives from:* **Why a parser rather than a matcher**.

| ID | Requirement | |
|---|---|---|
| FR-1.1 | A command is read as the language of the shell that will actually run it. **FACT 2026-08-20: despite its name, Claude Code's Bash tool runs zsh 5.9 on this machine — `$0` is `/bin/zsh` and `BASH_VERSION` is unset.** The shell is therefore something to establish, never to infer from the tool's name. | [A/D] |
| FR-1.2 | A command that cannot be parsed is reported with the line and column at which parsing stopped, and the reason. A command qwark cannot parse is one it cannot judge, which is a verdict rather than an absence of findings. | [D] |
| FR-1.3 | A command that ends mid-construct is distinguished from one that is malformed. The first indicates truncation, the second a mistake, and they are not the same event. | [D] |
| FR-1.4 | The exact source text of any node is recoverable as written, with its quoting and spacing intact, rather than re-printed from the tree. | [D] |
| FR-1.5 | qwark refuses to run when the shell it parses for is not the shell that will run the command. **FACT 2026-08-20: the mismatch is silent, not loud — of ten zsh constructs, only two are rejected by the bash parser, while four parse cleanly and mean something else** (`**/`, `*(.)`, `$foo[2]`, and the `noglob` precommand modifier). A gate reading the wrong language returns confident wrong answers rather than errors. | [D] |
| FR-1.7 | The permitted shells are declared in the rule files, and qwark refuses every command when nothing declares them. It is a declaration rather than a clause because it does not depend on the command — and because a rule file that omitted a clause would disable the check silently, which is the failure this design exists to close. | [A/D] |
| FR-1.8 | The shell is verified against what the environment reports, and a refusal names both what will run and what is permitted. **This is a consistency check on the best available signal, never proof**: qwark is a child of the process that spawns the shell and cannot observe it directly, so the check is worth exactly as much as the environment declaring it, which must therefore be unwritable like the rule files. | [D] |
| FR-1.9 | Permitted shells are absolute paths, and a relative entry is a configuration error. Comparing names would accept any file called `bash` anywhere, including one written into a directory the agent can reach — and for a gate whose subject can create files, what a program is called is not a property worth checking. | [A/D] |
| FR-1.10 | Both the reported shell and each permitted path are resolved through their symbolic links before comparing. **FACT 2026-08-20: `/bin` is a symlink to `usr/bin` here**, so two spellings name one file and must reach one answer — and a permitted path replaced by a link to something else no longer passes on the strength of its name. | [A/D] |

## 2. What a command establishes

*Derives from:* **Rules are layered by cost**; **Tier one: the command must say
what it does**.

| ID | Requirement | |
|---|---|---|
| FR-2.1 | Every structural fact about a command is gathered in a single traversal, so the cost of the cheapest rule class is set by the size of the tree and not by the number of rules consulting it. | [A/D] |
| FR-2.2 | Facts are hierarchical, and a node records every level it satisfies. A rule may forbid a family by naming the parent or one member by naming the child, without knowing which node types make up the family. | [D] |
| FR-2.3 | A redirection is distinguished by what it does to its target: truncate, append, input, duplicate, here-document. | [D] |
| FR-2.4 | A pipe is distinguished from a logical concatenation, both being the same node type with different operators. A rule forbidding one does not forbid the other. | [D] |
| FR-2.5 | The four substitutions — command, process, parameter, arithmetic — are each named separately as well as collectively. | [A/D] |
| FR-2.6 | A wildcard is reported only where the shell would expand one. The `*` in `grep "*"` is an argument, not a glob. | [D] |
| FR-2.7 | Every fact carries the position and the source text of the node that established it, so a decision can quote what caused it. A denial that names only its rule cannot be checked by the person reading it. | [D] |
| FR-2.8 | Every command form carries a fact, not only a plain invocation. `time`, `coproc`, a declaration keyword (`export`, `declare`, `readonly`, `local`, `typeset`) and an arithmetic command (`((…))`, `let`) each have one. | [D] |
| FR-2.9 | A statement with no usable command name remains addressable by a rule. **FACT 2026-08-20: `time rm x` puts `rm` at ordinal zero, and `((x=1))` and `let x=1` put nothing there** — so a rule reaching such a statement must do so by fact, and before FR-2.8 there was no fact to reach it by. | [D] |

## 3. The command line

*Derives from:* **Why a parser rather than a matcher** — a rule names node
types, so the vocabulary has to be visible before a rule is written.

| ID | Requirement | |
|---|---|---|
| FR-3.1 | The syntax tree of a command can be printed as an outline, one node per line, naming node types exactly as a rule file must spell them. | [D] |
| FR-3.2 | The facts a command establishes can be listed, with the position and text of each. | [D] |
| FR-3.3 | A command is taken from the argument list, or from standard input when no argument is given. | [D] |
| FR-3.4 | An unknown subcommand is an error, never an empty success. | [D] |

## 4. Deciding

*Derives from:* **The decision model**; **Rules are layered by cost**;
**The strictest action wins**; **Tags have lifetimes**; **Configuration**.

The loading half of this section is built and tested. The evaluating half —
FR-4.22 onward, and the rules-in-cost-order of FR-4.2 — is not.

| ID | Requirement | |
|---|---|---|
| FR-4.1 | A rule's action is one of `allow`, `deny`, `ask`, `tag` or `untag`. The tagging actions decide nothing; they attach or clear a name for later rules to match. | [A] |
| FR-4.2 | Rules are evaluated in order of cost: node presence, then conjunctions, then context such as whether paths fall within a given tree, then ongoing state. **Deferred to a later version, and correctly so: ordering cannot change a verdict, because the strictest action wins.** It changes only how much work is done before the answer is known. A seam is left for it — `order` in the evaluator, today the identity — so that adding it later does not mean restructuring. | [?] |
| FR-4.3 | Redirections, substitutions, pipes, logical concatenation **and globs** can be forbidden as one property: that the command's effect is determined by its own text. A wildcard belongs in that list because what it matches is decided by the directory at the moment it runs, and because a glob gives a rule about where a command reaches a word that denotes no path — so every path rule is silent on it. **FACT 2026-08-20: `rm *(e:'rm -rf /':)` carries no substitution, pipe, redirection or concatenation, satisfies every other tier-one rule, and zsh executes the quoted command as a glob qualifier.** | [A/D] |
| FR-4.4 | Rules are read from TOML, from several files aggregated. | [A] |
| FR-4.5 | If any rule file cannot be parsed, no Bash command is permitted. A gate that becomes permissive when its own configuration is broken reports success while guarding nothing. | [A] |
| FR-4.6 | A denial caused by an unreadable rule file names the file, the line and the text, and the route to fixing it does not itself require Bash. | [A] |
| FR-4.7 | A tag may persist across commands and decay after a stated number of them. | [?] |
| FR-4.8 | Every command is logged with the detail of its environment that bears on the decision. | [?] |
| FR-4.9 | Environment variable values are logged except where a rule file says not to. The declaration says what is **withheld**, not what is permitted, and a withheld variable is recorded as present-but-withheld rather than omitted, so the log never implies a variable was absent. | [?] |
| FR-4.9a | A value is also withheld when its **name matches a declared secret-shaped pattern** — token, secret, key, password, credential and the like. Naming variables one at a time fails open: a credential added tomorrow is logged until somebody remembers it exists. Matching by shape catches the ones nobody thought of, which is the only kind that gets written to a durable file by accident. | [?] |
| FR-4.10 | Writing code or a file through a here-document is refused. It is stated separately from the redirection ban because its reason is separate — such content was never a diff and leaves nothing to review — and that separateness is only real if the separate reason reaches the reader, which is what is tested. | [A] |
| FR-4.11 | A rule is a set of clauses, all of which must match for it to apply. There is no disjunction within a rule: alternatives are written as separate rules, so each one can be checked by reading it alone. | [A] |
| FR-4.12 | A command that cannot be parsed is denied, and the reason returned is the parser's own message. | [A] |
| FR-4.13 | The count a TTL is measured in is **allowed or approved Bash commands**. A denied command does not advance it, so a countdown is not spent by commands that never ran, and no other tool advances it either. | [?] |
| FR-4.13a | A definition belongs to the file that created it. A rule file may create declarations and groups of its own, and may not redefine one another file created. Collision is a configuration error rather than a precedence order, so no file can quietly weaken another's definition by being read later. | [A] |
| FR-4.13b | A declaration grants eligibility; it does not grant permission. An explicit deny rule outranks any declaration, so a file that declares a command still cannot run it past a deny rule in another file. This is what makes FR-4.13a safe and is why wrappers are denied explicitly rather than left undeclared. | [D] |
| FR-4.14 | When several rules match one command, the strictest action wins: deny over ask over allow. Rule order therefore never changes a verdict, so no rule can be weakened by where it sits or which file it arrived in. | [A] |
| FR-4.15 | Rule files are named on the command line. A path naming a directory contributes every rule file in it; a path naming a file contributes that file. The policy in force is thus readable where qwark is invoked. | [A] |
| FR-4.16 | A command qwark has no declaration for is refused. Nothing runs unless it has been described. | [A] |
| FR-4.16a | **The rules are consulted before the declaration is checked**, because most of them can answer without one: a clause naming node types, operators, flags or a fact needs no table. Letting the declaration check short-circuit them made a refusal say only "this is undescribed" about a command that also redirected and wrote through a here-document. The verdict is unchanged; what changes is whether the reader is told everything that was wrong or one thing that was. | [D] |
| FR-4.17 | Rule files must not be writable by the user qwark runs as, and neither may the directory holding them. qwark refuses to run otherwise. A writable directory permits unlink-and-replace, which is the same defeat as a writable file. | [A] |
| FR-4.17a | A refusal names the path and the route by which it is writable — world-writable, owner-writable, or through a group this user is in — because "a rule file is writable" without saying which bit is a message that cannot be acted on. | [D] |
| FR-4.17b | Running as root is reported rather than passed over. No arrangement of permission bits makes a file unwritable by root, so "not writable by this user" is not a property root can have, and a check that quietly succeeded there would be worse than none. | [D] |
| FR-4.18 | Using a name the shell may resolve to something other than the intended program is refused. **FACT 2026-08-20: a backslash suppresses alias expansion but not a shell function, and both zsh and bash accept a function named `/usr/bin/ls` which shadows the binary.** This requirement is therefore defence in depth, not a guarantee: it defeats alias expansion and PATH substitution, and it does not defeat a poisoned function table. | [?] |
| FR-4.19 | qwark never executes any file it reads, including a shell snapshot consulted to learn what a name resolves to. Sourcing an agent-writable file inside the gate would run the subject's code in the judge's process. | [D] |
| FR-4.20 | A command form qwark does not model is denied, on the same reasoning as one it cannot parse. An unrecognised construct still executes something, and finding no command to check is not the same as finding nothing to check. | [D] |
| FR-4.21 | Command wrappers are denied by an explicit rule, not merely by being undeclared. An explicit denial states why — which stops the same command being retried in other spellings — records that the command was considered rather than forgotten, and, because the strictest action wins, survives someone later declaring it for one harmless flag. An absence provides none of the three. | [A/D] |
| FR-4.22 | **Deny is the default, and it is the engine's, not a rule's.** A command is denied unless a rule allowed it: being in the allowed list *means* an allow rule matched. There is no state in which a command is understood, unobjected-to, and permitted by nothing — so a rule set containing no allow rules permits nothing, which is the correct reading of an empty policy. | [A] |
| FR-4.23 | A declaration grants **understanding, not permission**. It says how a command's options decompose, so that rules can be written about them; it permits nothing by existing. A rule set may therefore describe a command in order to deny it precisely. | [A/D] |
| FR-4.24 | **A denied command has no effect of any kind.** It sets and clears no tags, advances no TTL, and changes no state — it did not happen. The record of the decision is not an exception to this: that is a record of what qwark did, not of what the command did. | [A] |
| FR-4.25 | **A deny settles the verdict, and every deny reason is still collected.** Once one rule denies, no later rule can change the outcome — but evaluation continues so that the refusal can list everything wrong rather than sending its reader round three times. Tags are applied only after permission is decided, and never on a denial. | [A] |
| FR-4.25a | There are three layers of verdict, not four. `ask` is the refusal a person can lift and `deny` is the one nobody can, so every deny is a hard block and none is overridable. A rule able to override another rule would put that power in configuration, which sits far closer to the subject than a person does. | [A] |
| FR-4.26 | **More than one command in a call is an instant deny.** One command at a time, always. This is the engine's, not a rule's: it cannot be omitted from a rule file, and it holds however the commands were joined — by a sequence, a pipe, a logical concatenation, or a substitution carrying one of its own. | [A] |
| FR-4.27 | A clause that cannot be evaluated does not match. An option clause against a command with no declaration has nothing to test, and a clause that cannot be tested must never be counted as satisfied — which in an allow rule is what stops it permitting on the strength of qwark's own ignorance. | [D] |
| FR-4.28 | A verdict names the rule that produced it, states that rule's reason, and quotes the part of the command that satisfied it. A decision nobody can check is one nobody can correct. | [D] |

## 5. Addressing a command's words

*Derives from:* **A clause names a position**; **Nothing is expanded**.

| ID | Requirement | |
|---|---|---|
| FR-5.1 | The words of a simple command are addressed by ordinal: 0 is the command name, and its arguments run from 1. | [A] |
| FR-5.2 | A negative ordinal counts from the end, -1 being the last word. | [A] |
| FR-5.3 | One index may name several ordinals, as a comma-separated list of ordinals and ranges. | [A] |
| FR-5.4 | A range is written `..`, which reads the same whichever sign its endpoints carry. `1..-1` is every argument, whatever the count. | [A] |
| FR-5.5 | An ordinal the command does not have selects nothing rather than failing. A rule about the third argument does not apply to a command with one, which is not an error in either. | [D] |
| FR-5.6 | A malformed index is a configuration error, named with the text that could not be read. | [D] |
| FR-5.7 | A word's value is reported only where its text alone fixes it. Nothing is expanded — which is why substitutions are refused outright rather than resolved — so a word containing one is reported as undetermined and never guessed at. | [A/D] |
| FR-5.8 | Ordinals address the words of a simple command only. A pipeline, subshell or loop is a structure containing simple commands, and is described by facts rather than by position. | [D] |
| FR-5.9 | A word's value has its backslash escapes resolved the way the shell resolves them, which differs inside double quotes from outside. **FACT 2026-08-19, from bash: `a\ b` passes `a b`, `a\qb` passes `aqb`, `"a\$b"` passes `a$b`, and `"a\qb"` passes `a\qb`.** An unresolved value is a string the shell will never produce, and comparing a path against one lets `/home/ancient/.cl\aude/x` past a rule about `.claude`. | [D] |
| FR-5.10 | A word records whether it carried an escape outside quotes. A leading escape suppresses alias expansion, so `ls` and `\ls` name one file and run two different programs, and only this distinguishes them. | [A/D] |
| FR-5.11 | One end of a range may be left off. **An omitted start is 1 and an omitted end is -1**, so `1..` runs to the last word and `..3` covers the first three arguments. An open end never reaches the command: arguments do not start at 0, the command does. | [A] |
| FR-5.12 | A clause stating no index addresses the arguments. **Ordinal 0 is reachable only by naming it.** A test written without an index asks about what the command was given, and the command is not one of the things it was given — including it would make `value = "rm"` true of `echo rm`. | [A] |
| FR-5.13 | A range with neither end, `..`, is a configuration error. It names every argument, and so does stating no index at all — and one meaning with two spellings costs every reader who meets the unfamiliar one a trip to the documentation to learn it meant the other. | [A] |

## 6. Options

*Derives from:* **What it costs to detect `-f`**; **Controlling what an agent
can run**.

| ID | Requirement | |
|---|---|---|
| FR-6.1 | A command's options are decomposed against a table declaring which options it has, whether each takes a value, and what each means. | [A] |
| FR-6.2 | A clause matches an option by its declared meaning rather than its spelling, so every way of saying the same thing satisfies the same clause. | [D] |
| FR-6.3 | A long option is resolved against the declared set and accepts any unambiguous abbreviation, an exact name winning outright. **FACT 2026-08-19: `rm --force`, `--forc`, `--fo` and `--f` all take effect.** | [D] |
| FR-6.4 | Bundled short options are decomposed into the options they are. `rm -rf` contains no `-f` word. | [D] |
| FR-6.5 | After `--`, every remaining word is an operand. In `rm -- -f` the `-f` is a filename, not a request to force. | [D] |
| FR-6.6 | An option that takes a value takes it from the remainder of its bundle, from after `=`, or from the following word. | [D] |
| FR-6.7 | An option the table does not declare is refused, as is an abbreviation matching more than one declared option, and an option whose required value is missing. | [A/D] |
| FR-6.8 | Every fault in a command's options is reported, not only the first, so one denial says everything that is wrong. | [D] |
| FR-6.9 | A lone `-` is an operand, being the conventional name for standard input rather than an option. | [D] |
| FR-6.10 | A declaration states what an option's value and what the command's operands denote — a path, text, or another command — so a rule about where a command reaches knows which words to read. `rm`'s operands are paths; the message in `git commit -m "…"` is not. | [A] |
| FR-6.11 | A word whose kind was never declared has no kind, and a rule needing one does not apply to it. Guessing which arguments are paths would be reading the command as text again. | [D] |

## 7. Clauses

*Derives from:* **A rule is a conjunction**; **A clause states a string or a
pattern**.

| ID | Requirement | |
|---|---|---|
| FR-7.1 | A clause may state what it tests for as `value` — the whole word, exactly. `value = "rm"` does not match `rmdir`. | [A] |
| FR-7.2 | A clause may state it as `partial` — anywhere within the word. This is the broad form, and it is named rather than reached by accident: **the predecessor matched the substring `.archive` and thereby blocked `web.archive.org`.** Nothing prevents that here; naming the form is what makes the breadth a decision the author made and a reader can see. | [A/D] |
| FR-7.3 | A clause may state it as `pattern` — a regular expression matched against the whole word. Anchored, because an unanchored pattern would make every pattern quietly partial when partial has its own name. | [A/D] |
| FR-7.4 | A pattern that will not compile is a configuration error. A rule carrying one matches nothing, which reads as a clean run while leaving a hole. | [D] |
| FR-7.5 | An empty `partial` is refused. Every string contains the empty string, so such a clause matches every command — over-blocking in a deny rule and a hole in an allow rule. An empty `value` stands, being precise rather than universal. | [D] |
| FR-7.6 | A clause states exactly one of the three. Stating none is an error rather than a clause matching everything; stating several is an error rather than a precedence nobody would remember. A match that was never stated matches nothing. | [D] |
| FR-7.7 | A message about a rule quotes its match as the author wrote it, without anchoring or other machinery this program added. | [D] |
| FR-7.8 | A clause states which reading of the word it tests: `interpreted`, the value the shell will pass, or `written`, the source with quoting and escapes intact. `interpreted` is the default, because testing what was written is what lets `/home/ancient/.cl\aude/x` past a rule about `.claude`. | [A/D] |
| FR-7.9 | A word with no interpreted value — one containing a substitution, since nothing is expanded — is not matched by a clause reading it, rather than being matched as the empty string, which `partial` and `.*` otherwise would. | [D] |
| FR-7.10 | No pattern can make the gate slow. Go's regexp is RE2 — no backtracking, linear in the input — so no rule file can carry a pattern that a crafted command turns pathological, in a program that runs before every shell command. | [D] |
| FR-7.11 | A test with no selector is a complete clause, not an empty one. The index narrows a clause rather than being what makes it one, so `value = "rm"` alone asks whether some argument is `rm`. Only a clause saying nothing at all is refused. | [A] |
| FR-7.12 | **A clause may name the agent the request came from**, so one rule set can carry a different policy per role and the writer of a file need not be the runner of it. The value is the `agent_type` from the payload, which the subject cannot set — unlike an environment variable, which it can reach. It is compared whole: an agent type is a name a dispatcher assigned, not a path, so there is nothing for a prefix to be right about. | [A/D] |
| FR-7.13 | **A main-session call is named by `agent = ""`**, since it reliably carries no agent type. Absence is a role rather than a gap, so one rule set covers every caller without the launcher varying what it passes. **Stating no `agent` at all is the distinct case that covers every caller**, and the two must not collapse into one: a rule meant for the main session that silently became a rule for everybody would hand out permission nobody granted, which is why the clause records whether the key was stated rather than only what it said. | [A/D] |

## 8. Tags — deferred to a later version

**2026-08-20: deferred.** The shape is settled and the foundation is in
place — the evaluator emits tag changes and a clause can test a tag — but there
is no store behind it and there will not be one until there are concrete
scenarios worth limiting this way. Wanting a mechanism is not the same as
knowing what to point it at.

Everything below is therefore marked `[?]`: decided in principle, carrying no
test, and not waiting on anybody.

*Derives from:* **Tags have lifetimes**; **The decision model**.

| ID | Requirement | |
|---|---|---|
| FR-8.1 | A rule whose action is `tag` decides nothing. It attaches a name to the evaluation, which later rules match against. This is what lets cheap structural rules annotate and expensive rules stay expressed over names. | [A] |
| FR-8.2 | A tag is set with a TTL, measured in commands. **Example: after a rebase, deletion is denied for the next six.** | [?] |
| FR-8.3 | A tag is a clause criterion for any other rule, on the same footing as a node, an option or a path. | [A] |
| FR-8.4 | A denied command does not advance any TTL, so a countdown is not spent by commands that never ran. | [?] |
| FR-8.5 | A tag is **set or unset by a rule, never toggled**. A rule states which it does, so a rule's effect never depends on what the tag already was. | [?] |
| FR-8.6 | **Tags do not stack.** Setting a tag that is already set replaces it, TTL and all — six commands after the most recent rebase, not twelve after two. There is no count of how many rules set it and no order in which they must be unset. | [?] |
| FR-8.6a | A tag with no TTL lives for the current evaluation only. An unbounded tag would be indistinguishable from a command permanently changing policy. | [?] |
| FR-8.7 | Tag state must not be writable by the user qwark runs as. A countdown the subject can reset is not a constraint on the subject — the same reasoning as the rule files, one level down, and reachable by `Write` without any shell. | [?] |
| FR-8.8 | A decision taken because of a tag says which tag, when it was set, and what remains of its TTL. A denial that cites invisible state cannot be checked by the person reading it. | [?] |
| FR-8.9 | A tag may be a live calculation rather than a stored one, evaluated afresh from the world on each command — which branch a repository has checked out being the first of them. A detached head, a worktree redirection and the absence of any repository are each reported distinctly, because a rule about `main` silently ceasing to apply is the failure to avoid. | [A/D] |
| FR-8.10 | A live tag carries a value, matched by a clause like any other word, so `main` and `master` are one group rather than two rules. | [?] |
| FR-8.11 | A live calculation reads files and never runs the tool that owns them. **FACT 2026-08-20, demonstrated: an alias in a repository's own `.git/config` executes on a plain `git` invocation**, and that file needs no shell to write — so asking git a question would run the subject's code inside the judge. | [?] |
| FR-8.12 | The live calculations are a fixed set, enumerated in qwark. A rule file names one; it can never supply a command to run, which would reintroduce by configuration exactly what FR-8.11 removes. | [?] |

## 9. The blast radius

*Derives from:* **The agent has a directory it was started in**; **A manifest
states what may be read and written**.

| ID | Requirement | |
|---|---|---|
| FR-9.1 | A path can be placed inside or outside the directory the agent was started in — its blast radius. | [A] |
| FR-9.2 | Containment is decided by path components, never by text. **Compared as strings, `/home/x/project` is inside `/home/x/proj`** — one is a prefix of the other as text and neither contains the other as a path — which would permit every write to a neighbouring directory. Climbing out with `..` is likewise resolved before comparing. | [D] |
| FR-9.3 | A relative path is resolved against the directory the command will run in, which is not the directory qwark is running in. | [D] |
| FR-9.4 | Symbolic links are followed as far as the path exists, and whatever does not exist yet is reattached. A rule about writing is asked about files not yet created, so the leaf usually cannot be resolved — while a link is far likelier to be a directory in the middle of the path than the file at the end of it. A lexical check would call `<radius>/link/x` contained while the shell wrote through the link to somewhere else. | [D] |
| FR-9.5 | A relative blast radius, or a relative directory to resolve against, is refused rather than resolved against qwark's own working directory, which has nothing to do with where the agent was started. | [D] |
| FR-9.6 | Any path given to a command that writes must be inside the blast radius. | [?] |
| FR-9.7 | A manifest created by the task management process, read at runtime, states which files may be read and which may be written. | [?] |

## 10. The hook contract

*Derives from:* **Read out of the installed binary**, Claude Code 2.1.233 — not
recalled and not inferred.

| ID | Requirement | |
|---|---|---|
| FR-10.1 | The request is read as Claude Code sends it: `session_id`, `transcript_path`, `cwd`, `prompt_id`, `permission_mode`, `agent_id`, `agent_type`, `effort`, `hook_event_name`, `tool_name`, `tool_input`, `tool_use_id`. | [D] |
| FR-10.2 | A payload that cannot be read is a refusal, never an empty request. Decoding a truncated pipe into a zero value would turn a broken connection into an approval. | [D] |
| FR-10.3 | **When qwark has decided, it exits 0 and the decision travels in the JSON.** A non-zero exit reports that the hook failed to run, which is a different claim from a refusal. | [D] |
| FR-10.3a | **When qwark cannot decide, it exits 2.** **FACT 2026-08-20, confirmed in the binary and the documentation: any other non-zero exit is a `non_blocking_error` and the tool proceeds, and exit 0 without JSON is no decision at all, so the normal permission flow proceeds too.** Both of the obvious failure exits are therefore permissive. Exit 2 is the only one that blocks, which makes it the only fail-closed answer available to a gate that has broken. | [D] |
| FR-10.3b | qwark emits a decision or exits 2 — never neither. A panic, an unreadable payload, a rule set that will not load, or a stdout it cannot write are all cases where a gate that simply died would let the command through. | [D] |
| FR-10.4 | The reply names the event it answers. Claude Code validates a reply against the event it asked about, so answering a different one is not a partial answer but none. | [D] |
| FR-10.5 | **qwark never returns `defer`.** The decision has four values, not three: alongside allow, deny and ask, `defer` means the hook declines to decide and the dispatcher continues past it. Deciding nothing is the one outcome this design exists to prevent. | [D] |
| FR-10.6 | `agent_id` and `agent_type` are carried through, so a rule set may differ by which agent is asking. **This is what makes per-agent scoping implementable from the payload**, rather than through an environment variable the agent might itself reach. | [A/D] |
| FR-10.6a | Those two fields are present only for a subagent, so a main-session call carries neither. **REVISED 2026-08-20, and the first reading was wrong.** It said the scoping "does not need solving inside qwark" because an external process chooses the rule files per agent. That holds only when every specialised agent is its own session: the registration is fixed for a session, so a subagent inherits its parent's command line and the partition collapses. The the judgement on the external route — *"actively managing symlinks or something else … some form of ENV VAR that will have to be actively managed … which feels rickety"* — also runs against FR-10.6, which chose the payload over an environment variable for a stated security reason. **The engine carries the scoping** (FR-7.12, FR-7.13); the launcher is what cannot be relied on. | [A] |
| FR-10.7 | A hook may also return `updatedInput` and rewrite the tool call. qwark does not: rewriting the subject's command would make qwark the author of what runs, and a gate that edits what it judges can no longer be said to have judged it. | [D] |
| FR-10.8 | **qwark runs in one goroutine.** `recover` catches a panic in the goroutine that deferred it and in no other, so the guarantee that every path ends in a decision or a refusal holds only while there is one. This is an architectural invariant, not a preference, and it is enforced by a test that walks the source for `go` statements rather than left to be remembered. | [D] |
| FR-10.9 | The registration wraps qwark so that any death becomes exit 2. In-process recovery cannot catch a fatal runtime error, an out-of-memory kill or a signal, and every one of those otherwise exits non-zero-and-not-two, which lets the command run. Hook commands are executed through a shell, so `qwark … \|\| exit 2` closes it. | [D] |
| FR-10.10 | **The registration carries the deny list qwark cannot enforce itself.** qwark gates Bash, and the Write and Edit tools reach the rule files, the shell snapshot, `.git/hooks`, `settings.json` and a task definition without passing through it — so every path a rule protects needs a `permissions.deny` twin in the same registration, or the rule is enforced against a shell and against nothing else. **A task definition is on that list for a reason of its own: `just checks` is a fixed, familiar command line whose meaning lives in a file in the working tree, so permitting the command permits whatever that file says next.** | [A/D] |
