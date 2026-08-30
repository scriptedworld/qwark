# qwark's own layer, overriding base.just and lang.just.
#
# ONE RECIPE, AND IT EXISTS TO PASS `--definitions`.
#
# `_verdict` in base.just runs `bolt <jig> .` with no other arguments. That is
# right for every adopter that fills no placeholder, and wrong here: the
# go-std-quality jig leaves `entrypoint` defaulted to `true`, and qwark supplies
# `scripts/cover-entrypoint.sh` for it through
# `bolt.go-std-quality.definitions.yaml`.
#
# Without the flag the placeholder stays `true`, the entry point is never run,
# and `cmd/qwark/main.go` reports 0.0% against an 80% floor. Measured on
# adoption: `just checks` failed there, and the jig was correct to fail. Hard
# rule 5 says a coverage failure is never settled by excluding the file, so the
# fix is to pass the definitions rather than to exempt main.go.
#
# THE ARTIFACT-READING BELOW IS A SECOND COPY OF `_verdict` AND SHOULD NOT
# SURVIVE. It is here because a just recipe cannot add an argument to another
# recipe's command line. The fix belongs in the shared base, as a `_verdict`
# that takes definitions and passes them when given:
#
#     _verdict jig definitions="":
#
# When that lands, delete this file and let lang.just's `checks` stand.
# Reported to factory on adoption.

# everything: quality, tests, coverage, secrets
checks:
    @just _verdict common-quality
    @just _go-quality
    @just leak-scan

# go-std-quality, with qwark's placeholder values supplied
_go-quality:
    #!/usr/bin/env bash
    set -euo pipefail
    out=$(bolt --definitions go-std-quality go-std-quality .)
    result=$(printf '%s\n' "$out" | tail -1)
    if [ ! -f "$result" ]; then
        echo "REFUSED: bolt wrote no result for go-std-quality" >&2
        printf '%s\n' "$out" >&2
        exit 1
    fi
    if grep -q '"success": *true' "$result"; then
        echo "pass  go-std-quality  $result"
    else
        echo "FAIL  go-std-quality  $result" >&2
        exit 1
    fi
