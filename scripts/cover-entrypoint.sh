#!/bin/sh
# Measure the statements in main() that no test process can reach.
#
# Hard rule 5: a coverage failure is never settled by excluding the file. main()
# is unreachable from `go test` because nothing in a test calls it, so it is
# measured instead: built with -cover, run once, and its profile merged with the
# test one by the adapter.
#
# Called by the `tests` task of bolt.go-std-quality.yaml through the {entrypoint}
# placeholder, which is one argument by design, so this takes the work directory
# and nothing else. The task has already written cover-entry.out holding a bare
# mode line; this overwrites it.
#
# `help` is the smallest safe invocation qwark has: it reads no rules, opens no
# log, judges nothing, and exits 0.
set -eu

work="${1:?the work directory is the only argument}"

profile="$work/cover-entry.out"
binary="$work/qwark"
covdata="$work/covdata"

mkdir -p "$covdata"

go build -cover -covermode=atomic -coverpkg=./... -o "$binary" ./cmd/qwark

# The binary writes one coverage file per run into GOCOVERDIR, in a binary
# format, so it is converted to the text profile the adapter merges.
GOCOVERDIR="$covdata" "$binary" help >/dev/null

go tool covdata textfmt -i="$covdata" -o="$profile"

# An empty conversion leaves a file with no mode line, which merges as a broken
# profile rather than as nothing. Restore the no-op form instead.
if [ ! -s "$profile" ]; then
    printf 'mode: atomic\n' > "$profile"
fi
