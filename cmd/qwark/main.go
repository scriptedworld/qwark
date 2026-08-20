// Command qwark gates Claude Code's Bash tool. It runs as a PreToolUse hook,
// parses the proposed command, and answers with a decision.
//
// Everything is in internal/cli. This file holds the one statement a test
// process cannot reach, because nothing in a test calls main().
package main

import (
	"os"

	"github.com/scriptedworld/qwark/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
