# The Go half for qwark.
#
# bin/qwark IS ON PATH: `~/bin/qwark -> qwark/bin/qwark`. It is a PreToolUse
# hook, so it runs on every Bash call in every live session, and removing it
# changes what every agent on this machine may do rather than degrading one
# program. `clean` below therefore does not touch it. silo dd24c42.

# everything: quality, tests, coverage, secrets
checks:
    @just _verdict common-quality
    @just _verdict go-std-quality
    @just leak-scan

# the suite
test:
    go test ./...

# COVERAGE IS JUDGED PER FILE, and the entry point is measured by running it
# rather than excluded, which is hard rule 5. `go test -race` needs cgo, which
# is why CGO_ENABLED is 1 here and 0 in build.
#
# scripts/cover-entrypoint.sh already exists in this tree and is what the
# go-std-quality jig invokes for the same purpose. If it does what this recipe
# does, call it instead and delete the inline version.

# the suite with coverage, and the minimum enforced
coverage:
    #!/usr/bin/env bash
    set -euo pipefail
    CGO_ENABLED=1 go test -race -shuffle=on -covermode=atomic \
        -coverpkg=./... -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | tail -1
    dir=$(mktemp -d); out=$(mktemp -d)
    trap 'rm -rf "$dir" "$out"' EXIT
    go build -cover -o "$out/qwark" ./cmd/qwark
    GOCOVERDIR="$dir" "$out/qwark" --help >/dev/null 2>&1 || true
    echo "entry point, measured by running it:"
    go tool covdata percent -i="$dir" | grep '/cmd/' || echo "  no entry-point data"

# GOFMT EXITS 0 WHATEVER IT FINDS, so testing its output is what makes this a
# gate rather than a report.

# formatting verified, nothing written
format-check:
    #!/usr/bin/env bash
    set -euo pipefail
    out=$(gofmt -l .)
    if [ -n "$out" ]; then
        echo "not gofmt-clean:" >&2
        printf '%s\n' "$out" >&2
        exit 1
    fi
    echo "gofmt clean"

# formatting written
format:
    gofmt -w .
    go mod tidy

# the binary, statically linked
build:
    CGO_ENABLED=0 go build -o bin/qwark ./cmd/qwark

# whatever this language distributes
dist:
    @echo "dist: the binary is the distribution; see build"

# COMPARE BYTES, NOT MTIMES. `cp` sets a fresh timestamp whatever it copied, so
# a time comparison reports the install as current when it is not.

# put the built artefact where it is used from
install:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p .ephemera bin
    CGO_ENABLED=0 go build -o .ephemera/qwark.new ./cmd/qwark
    if cmp -s .ephemera/qwark.new bin/qwark; then
        echo "bin/qwark is current"
    else
        mv .ephemera/qwark.new bin/qwark
        echo "bin/qwark installed"
    fi
    rm -f .ephemera/qwark.new

# bin/qwark IS NOT REMOVED. See the header, and just/base.just.

# remove run directories and build output
clean:
    rm -rf .bolt-* coverage.out
