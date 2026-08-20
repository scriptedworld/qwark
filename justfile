# qwark — task recipes.
#
# Recipe names follow the cross-language contract: plural for collections
# (`checks`, `tests`), verbs for actions (`build`, `run`, `format`).
#
# The gate is bolt's standard Go quality definition, overlaid with
# bolt.qwark.yaml. qwark does not vendor a copy: what does the checking — the
# adapters, the checker scripts, the linter config — belongs to the definition
# and resolves against its own directory, while what is checked stays relative
# to this one. Vendoring would be a second copy of the definition, free to
# drift from the one bolt maintains.

BOLT_REPO := env("BOLT_REPO", justfile_directory() / ".." / "bolt")
BOLT := BOLT_REPO / "bin" / "bolt"
STD_GATE := BOLT_REPO / "bolt.go-std-quality.yaml"
LINE_LENGTH := "100"

# Show available recipes.
default:
    @just --list --unsorted

alias test := tests
alias check := checks
alias fmt := format

# Everything a pipeline runs, through bolt.
#
# bolt's own exit status says whether bolt worked, never whether the checks
# passed. The verdict is run_result.yaml, which `just results` reads back.
checks: build
    {{ BOLT }} -c {{ STD_GATE }} -c bolt.qwark.yaml

# The quick subset, by tag.
quick: build
    {{ BOLT }} -c {{ STD_GATE }} -c bolt.qwark.yaml quick

# What the gate would run, without running it.
plan:
    {{ BOLT }} plan -c {{ STD_GATE }} -c bolt.qwark.yaml

# Read the most recent run's merged verdict.
results:
    {{ BOLT }} results

# Rewrite formatting in place.
format:
    golines --max-len={{ LINE_LENGTH }} --shorten-comments --write-output .
    gofmt -w .
    go mod tidy

# Compiler-adjacent checks that need no configuration.
vet:
    go vet ./...

# The test suite, with race detection, shuffling and a coverage profile.
#
# -covermode=atomic is required alongside -race: the default `set` mode writes
# the counters non-atomically, so the race detector reports on the
# instrumentation itself.
tests:
    CGO_ENABLED=1 go test -race -shuffle=on -covermode=atomic \
        -coverpkg=./... -coverprofile=coverage.out ./...

# Build the binary into ./bin.
build:
    go build -o bin/qwark ./cmd/qwark

# Run qwark from source. `just run ast 'git status'`
run *ARGS:
    go run ./cmd/qwark {{ ARGS }}

# Install the tools these recipes need but the machine may not have.
tools:
    go install golang.org/x/vuln/cmd/govulncheck@latest
    go install github.com/segmentio/golines@latest

# Remove build output and artifacts.
clean:
    rm -rf bin coverage.out .bolt-*
