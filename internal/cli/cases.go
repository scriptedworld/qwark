package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/scriptedworld/qwark/internal/suite"
)

// runCases judges a case directory and prints its envelope.
//
// The envelope is JSON, which is also YAML, so a reader of either takes it as
// it is. It is printed whether the suite passed or failed; the exit status says
// only which, and an error that stopped the suite running prints no envelope.
func runCases(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintf(stderr, "qwark cases: want one case directory\n\n%s", Usage)
		return statusUsage
	}

	env, err := suite.Run(args[0])
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "qwark cases: %v\n", err)
		return statusError
	}

	out, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "qwark cases: %v\n", err)
		return statusError
	}
	_, _ = fmt.Fprintf(stdout, "%s\n", out)

	if !env.Success {
		return statusError
	}
	return statusOK
}
