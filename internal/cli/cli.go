// Package cli implements qwark's command line. It exists as a package rather
// than living in main so that every path through it is reachable from a test:
// nothing in a test process calls main(), so anything left there is measured
// only by building the binary and running it.
package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/scriptedworld/qwark/internal/rules"
	"github.com/scriptedworld/qwark/internal/shell"
	"mvdan.cc/sh/v3/syntax"
)

// Usage is the help text, and the only description of the command line.
const Usage = `qwark, a parsing gate for Claude Code's Bash tool

usage:
  qwark ast [--debug] [command]   outline the syntax tree of a command
  qwark facts [command]           list the properties a command establishes
  qwark rules PATH...             load rule files and report what they hold
  qwark judge [--agent=T] RULES COMMAND...
                                  judge a command against a rule set, as the
                                  agent type T; default none is the main session
  qwark hook RULES...             run as the PreToolUse hook: read one call
                                  from stdin, judge it, answer on stdout
  qwark help                      this text

With no command argument, ast and facts read the command from stdin. --debug
prints the raw node structs instead of the outline.
`

// Exit statuses. Distinguishing them means a caller can tell "qwark was asked
// for something that does not exist" from "qwark was asked properly and could
// not do it".
const (
	statusOK    = 0
	statusError = 1
	statusUsage = 2
)

// Main runs one invocation and returns the process exit status. Every stream is
// a parameter so a test can drive it without touching the real ones.
//
// Writes to these streams discard their error explicitly. A failed write to
// stderr has nowhere left to be reported, and returning it would replace a
// real diagnostic with the news that printing it did not work.
func Main(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stderr, Usage)
		return statusUsage
	}

	switch args[0] {
	case "ast":
		return outline(args[1:], stdin, stdout, stderr)
	case "facts":
		return list(args[1:], stdin, stdout, stderr)
	case "rules":
		return check(args[1:], stdout, stderr)
	case "judge":
		return judge(args[1:], stdout, stderr)
	case "hook":
		return runHook(args[1:], stdin, stdout, stderr)
	case "help", "-h", "--help":
		_, _ = fmt.Fprint(stdout, Usage)
		return statusOK
	default:
		_, _ = fmt.Fprintf(stderr, "qwark: unknown command %q\n\n%s", args[0], Usage)
		return statusUsage
	}
}

func outline(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	debug := false
	if len(args) > 0 && args[0] == "--debug" {
		debug, args = true, args[1:]
	}

	parsed, status := read(args, stdin, stderr)
	if parsed == nil {
		return status
	}

	var err error
	if debug {
		err = syntax.DebugPrint(stdout, parsed.File)
	} else {
		err = parsed.Inspect(stdout)
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "qwark: %v\n", err)
		return statusError
	}

	return statusOK
}

func list(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	parsed, status := read(args, stdin, stderr)
	if parsed == nil {
		return status
	}

	for _, finding := range parsed.Facts().All() {
		_, _ = fmt.Fprintf(stdout, "%-26s %d:%-3d │ %s\n",
			finding.Fact, finding.Line, finding.Col, finding.Text)
	}

	return statusOK
}

// check loads a rule set and reports what it holds, so a rule file can be
// found wrong before it is the reason every command is failing.
func check(paths []string, stdout, stderr io.Writer) int {
	set, err := rules.Load(paths)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "qwark: %v\n", err)
		return statusError
	}

	_, _ = fmt.Fprintf(stdout, "shells:       %s\n", strings.Join(set.Shell.Allow, ", "))
	_, _ = fmt.Fprintf(stdout, "groups:       %d\n", len(set.Groups))
	_, _ = fmt.Fprintf(stdout, "declarations: %d\n", len(set.Commands))
	_, _ = fmt.Fprintf(stdout, "rules:        %d\n", len(set.Rules))

	counts := map[rules.Action]int{}
	for _, rule := range set.Rules {
		counts[rule.Action]++
	}
	for _, action := range []rules.Action{
		rules.ActionDeny, rules.ActionAsk, rules.ActionAllow,
		rules.ActionTag, rules.ActionUntag,
	} {
		if counts[action] > 0 {
			_, _ = fmt.Fprintf(stdout, "  %-6s      %d\n", action, counts[action])
		}
	}

	return statusOK
}

// judge evaluates one command against a rule set and prints the verdict with
// every reason for it.
//
// This exists so a rule can be tried before it is the reason a command failed.
// A rule set that has never judged anything is a policy nobody has run.
func judge(args []string, stdout, stderr io.Writer) int {
	args, agent := agentOf(args)

	paths, spoken := split(args)
	if len(paths) == 0 || len(spoken) == 0 {
		_, _ = fmt.Fprint(stderr, "qwark: judge wants a rules path and a command\n")
		return statusUsage
	}

	set, err := rules.Load(paths)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "qwark: %v\n", err)
		return statusError
	}

	parsed, err := shell.Parse(strings.Join(spoken, " "))
	if err != nil {
		// A command qwark cannot parse is one it cannot judge, and the
		// parser's own message is what gets returned.
		_, _ = fmt.Fprintf(stdout, "deny\n  (engine) unparseable: %v\n", err)
		return statusOK
	}

	outcome := set.Evaluate(parsed, rules.Context{Agent: agent})

	_, _ = fmt.Fprintf(stdout, "%s\n", outcome.Action)
	for _, finding := range outcome.Findings {
		_, _ = fmt.Fprintf(stdout, "  %-34s %s\n", finding.Rule, oneLine(finding.Reason))
		if finding.Cause != "" {
			_, _ = fmt.Fprintf(stdout, "  %-34s caused by: %s\n", "", finding.Cause)
		}
	}
	for _, change := range outcome.Tags {
		_, _ = fmt.Fprintf(stdout, "  tag %-30s set=%v ttl=%d\n",
			change.Name, change.Set, change.TTL)
	}

	return statusOK
}

// agentFlag names the agent a judgement is made as.
const agentFlag = "--agent="

// agentOf takes the agent off the front of the arguments, so a rule set can be
// tried as the agent that will be judged by it rather than only as the session
// running the command.
//
// **The default is the empty agent, and that is the main session rather than a
// missing value**: a main-session call carries no agent type, so `judge` with
// no option already exercises the caller every session has. There is no
// spelling here for "any agent", deliberately: a rule set is judged as somebody,
// because at runtime it always is.
//
// Only leading occurrences are taken. Everything after the rules path may be
// the command being judged, and a gate that ate an argument out of the middle
// of the command would be judging something other than what was typed.
func agentOf(args []string) ([]string, string) {
	agent := ""
	for len(args) > 0 && strings.HasPrefix(args[0], agentFlag) {
		agent = strings.TrimPrefix(args[0], agentFlag)
		args = args[1:]
	}
	return args, agent
}

// split separates the rule paths from the command to judge.
//
// Everything before `--` is a rule path, so several sources can be given at
// once. Without the separator the first argument is the only rule path, which
// is what the common case wants to type.
func split(args []string) (paths, spoken []string) {
	for i, arg := range args {
		if arg == "--" {
			return args[:i], args[i+1:]
		}
	}
	if len(args) == 0 {
		return nil, nil
	}
	return args[:1], args[1:]
}

// oneLine flattens a reason for a single-line report.
func oneLine(reason string) string {
	return strings.Join(strings.Fields(reason), " ")
}

// read takes the command and parses it, reporting to stderr on the way out. A
// nil result means the caller should return the status alongside it.
func read(args []string, stdin io.Reader, stderr io.Writer) (*shell.Parsed, int) {
	src, err := source(args, stdin)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "qwark: %v\n", err)
		return nil, statusError
	}

	parsed, err := shell.Parse(src)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "qwark: %v\n", err)
		return nil, statusError
	}

	return parsed, statusOK
}

// source takes the command from the argument list, or from stdin when none was
// given. Joining the arguments rather than taking only the first means an
// unquoted command still parses as what was meant.
func source(args []string, stdin io.Reader) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}

	src, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("reading command from stdin: %w", err)
	}
	return string(src), nil
}
