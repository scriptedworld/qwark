# The proxy is another layer, and mode two is the audit of usage

*Extracted verbatim from `DESIGN-NOTES.md` on 2026-08-21, where it was
lines 612-707. Split because a 1151-line document is a list that
outgrew its file: one topic per file keeps diffs surgical and stops
concurrent sessions colliding.*


**2026-08-20:**

> The next layer will be more than this as a gate, and will be an MCP that
> becomes the proxy for the tools & mechanicals we want to allow. Then the
> situation gets much more simple. … Once we have the proxy, then we can have
> these kinds of rules for the various tools per agent type.

> **Mode Two will continue to be the audit of usage … still a different thing
> than the PROXY** and how things will interact with it.

**CORRECTED 2026-08-20: an earlier draft of this section called the proxy the
second mode. It is not.** qwark's second mode is the audit of usage — the
observability half, FR-4.8 to FR-4.9a. The proxy is a separate component that
qwark interacts with, not a further thing qwark becomes.

The distinction is worth holding, because a gate and an audit are opposite in
almost every respect that matters:

    a gate     synchronous, per call, must decide, FAILS CLOSED
    an audit   after the fact, across calls, must not lose a record,
               and blocking nothing is the point

**Their failure modes point in opposite directions.** A gate that breaks must
refuse — which is why exit 2 exists and why every path here ends in a decision.
An audit that breaks must not refuse, because an audit that can stop work will
be turned off; what it must not do is silently lose records, since a log with
gaps reads as a clean history.

An audit also sees what no gate can: a pattern across calls that no single call
would trip. That is the same argument tags were built on, arriving from the
other end — and it is why the log is worth having even where a store is not.

**This is also where an intention note would land.** The proxy collects the
stated reason; mode two is where it is compared against what the call actually
did. Neither half is worth much alone: the note is unverified prose, and the
record of effects does not say what was meant.

---

Back to the proxy itself, which is a layer rather than a mode. The engine does
not care what it is judging. A rule is clauses that must all hold; whether a
clause selects a word of a command line or a field of a tool call is
an adapter question, not a design one.

**It turns denying into enumerating, and that is the whole gain.** qwark denies
by default across the infinite space of things somebody might type. A proxy
exposes a finite set of operations, so what exists is what was written down.
Every hard problem in mode one is a consequence of the space being infinite and
the text being ambiguous: quoting, escapes, aliases, shell functions, `PATH`,
wrappers, interpreters, globs, substitutions. **A typed call cannot hide its own
effect the way a command line can**, which is what the whole of tier one exists
to force.

The mechanicals mostly become API design. "Allowed as a word, refused in a
shape" stops being a rule and becomes a parameter that is not offered — a
`reflog` operation with no `expire`. "Refused unless" becomes a required
argument. That is a better place for the constraint, because an operation that
cannot be named cannot be attempted, and a refusal never has to be understood.

**What carries over is the part worth keeping**: rules as conjunctions with no
disjunction inside one, the strictest action winning so order never matters,
deny as the engine's default rather than a rule's, a declaration granting
understanding and not permission, groups so a class is a list rather than twenty
rules, and a refusal that states its reason.

**This is what makes FR-7.12 and FR-7.13 foundational rather than interim.**
"These kinds of rules for the various tools per agent type" *is* the agent
clause: the discriminator that is awkward to arrange in the plumbing is the
natural one here, and it is the same clause either way.

**A proxy per agent type is the stronger form**, and raised as an
alternative or an addition: *"if we encode a PROXY for each of the agent types …
which would likewise limit everything"*. It is stronger for the same reason
enumeration beats denial — **an unexposed tool cannot be called at all**, while
a rule refuses a call that was already formed.

Its cost is the question already open under **where this is heading**: N proxies
are N surfaces to keep in step with each other and with the agent prompts, which
is the duplication problem in another dress. One proxy with agent-scoped rules
keeps the policy in one readable file; a proxy per agent type puts it in the
wiring. Unless the surfaces are generated from a single source — and the rule
files are the obvious candidate, which is the "those end up referencing these
rule files" branch named.

That also says what the agent clause is for once proxies exist: not the primary
control, but a check on the wiring rather than the wiring itself — and the
policy for whatever Bash surface remains.

**What the proxy does not dissolve** is the residue everything else left. If the
proxy exposes an operation that runs `just checks`, the `justfile` still decides
what that does — the call is typed and its meaning is still in a file in the
tree. The proxy has to own the recipe, or the recipe has to sit outside what the
agent may write. Same residue as the sandbox, same answer: the manifest.

