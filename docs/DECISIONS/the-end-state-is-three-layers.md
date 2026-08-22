# The end state is three layers, and qwark is the outermost

Two statements that belong together:

> When we have the tool chain in place, the agents will be in a sandbox, and
> these files won't be in there, and with that the blast radius rules will
> prevent a good chunk of the possible issues.

> Likewise, if/when we have the manifest files, those will tighten things down
> even better.

So the destination is not a longer deny list. It is:

    1. a sandbox         the file is not there to be reached
    2. the blast radius  a write must land inside the directory the agent
                         was started in                        (FR-9.1 - FR-9.6)
    3. the manifest      which files may be read and which written, named
                         by the task management process        (FR-9.7)

That changes what the path groups in 20-paths.toml are FOR. **Four of the six sit
outside a project-rooted sandbox and are absorbed by layer one**: qwark's own
rules, Claude Code's configuration and snapshot, the shell startup files, and the
PATH directories. Under a sandbox those rules guard against something that cannot
happen, and they survive only as the un-sandboxed case and as defence in depth.

**Two of the six sit INSIDE it, and no sandbox removes them.**

    repository-hooks    .git/hooks/, .git/config, .githooks/
    task-definition     justfile, Makefile, Taskfile.yml, pyproject.toml,
                        package.json, bolt.*.yaml, .pre-commit-config.yaml

These are in the project, which is the one place the agent is supposed to be able
to write. Layer two does not help either: the blast radius says a write must land
inside the project, and every one of these already is.

So the residue after the sandbox is exactly the worry that prompted this section.
`just checks` and `bolt run` are fixed command lines whose meaning lives in a file
the agent may legitimately write. **The layer that closes it is the manifest**,
because the manifest is the only one of the three that discriminates between
files inside the blast radius. Not "the project is writable" but "these files are
writable, and a task definition is not one of them".

That puts the priority somewhere other than where it looks. FR-9.6 and FR-9.7 are
both `[?]` and unbuilt, and between them they are the whole of layer three.
Meanwhile the executors are all denied today, so the exposure is not live. It
becomes live the moment one of them is permitted, and permitting one is what
everybody will want as soon as the agent needs to run the gate.

What no layer reaches, stated so it is not rediscovered: a coding agent that can
write a test file and run the test runner has arbitrary execution. The manifest
would have to say `*_test.go` is writable, because writing tests is the job. That
is irreducible, and it is why `go test` sits in the executor group instead of
being quietly allowed.
