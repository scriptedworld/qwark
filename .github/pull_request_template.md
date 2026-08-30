## What changed

One concern per pull request, and the subject line says which.

## How it was checked

    go test ./...

Paste what that printed. If the change touches the rules, also paste the
`qwark judge` output for a command the rule should refuse and one it should not,
because a rule that has judged nothing is a policy nobody has run.

## What it traces to

The requirement in `REQUIREMENTS.md` this discharges, and the test that cites it:

    // COVERS: FR-0.0 | positive

A new requirement needs a row of its own. A change to why something is required
needs a file in `docs/DECISIONS/`. `CONTRIBUTING.md` has the rest.
