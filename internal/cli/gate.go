package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/scriptedworld/qwark/internal/audit"
	"github.com/scriptedworld/qwark/internal/gate"
	"github.com/scriptedworld/qwark/internal/hook"
	"github.com/scriptedworld/qwark/internal/rules"
)

// runHook runs qwark as the hook itself: read one proposed call from stdin,
// judge it, and answer on stdout.
//
// This is the subcommand `install/settings-fragment.json` names. It is what
// makes qwark a gate; the other subcommands only ask it questions, and
// `internal/hook.Run` does nothing unless this calls it.
//
// A usage error here exits 2, which blocks. That reads oddly for a usage
// error and is the only correct answer: qwark invoked without rule paths has
// not decided anything, and every other non-zero status is a
// `non_blocking_error` that lets the command run.
func runHook(paths []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(paths) == 0 {
		_, _ = fmt.Fprint(stderr,
			"qwark: hook wants at least one rules path\n"+
				"Without one there is no policy, and qwark has not decided anything.\n")
		return statusUsage
	}

	set, refusals := preflight(paths)
	if len(refusals) > 0 {
		return hook.Run(stdin, stdout, stderr, refusing(refusals))
	}

	log, err := audit.To(audit.DefaultPath())
	if err != nil {
		// A gate that cannot open its log still judges. Refusing here would
		// make an unwritable directory a way to stop every command.
		_, _ = fmt.Fprintf(stderr, "qwark could not open its log: %v\n", err)
	}
	defer func() { _ = log.Close() }()

	return hook.Run(stdin, stdout, stderr, recording(log, set, stderr))
}

// recording wraps a decider so every judgement is written down.
//
// The decision is made first and recorded second, and the recording cannot
// change it. A failed write is reported on stderr and the verdict stands. The
// alternative, refusing when the log is unwritable, would turn a full disk into
// a refusal of every command in every session.
//
// That is the permissive direction and it is a real hole: somebody who can fill
// the disk can stop the recording without stopping the commands. Closing it
// needs a writer the subject is not, which is the same answer the leaking-bucket
// note reaches for tag state, and it arrives with the proxy, not here.
func recording(log *audit.Recorder, set *rules.Set, errOut io.Writer) hook.Decider {
	judged := gate.Judged(set)

	return func(request hook.Request) (hook.Decision, string) {
		judgement := judged(request)

		entry := audit.Entry{
			At:        time.Now(),
			RuleSet:   set.Digest,
			Decision:  string(judgement.Decision),
			Action:    string(judgement.Action),
			Tool:      request.ToolName,
			Rules:     judgement.Rules,
			Agent:     request.AgentType,
			Cwd:       request.Cwd,
			SessionID: request.SessionID,
		}
		if call, err := request.Bash(); err == nil {
			entry.Command = call.Command
		}

		if err := log.Record(entry); err != nil {
			_, _ = fmt.Fprintf(errOut, "qwark could not record its decision: %v\n", err)
		}

		return judgement.Decision, judgement.Reason
	}
}

// preflight settles whether this rule set can be believed, before any command
// is judged against it.
//
// It reports a list, not one error, because a refusal names everything wrong
// at once, on the same reasoning that has a verdict list every rule that
// objected and not only the first. Only loading can fail today.
//
// Whether the running user could rewrite the rule set is deliberately not
// asked. The check cost more than it bought: enforcing it means a root-owned
// live set, so every change to a rule needs a root command and the gate cannot
// be developed under its own gating. What
// holds the property now sits outside qwark, in the rule that an agent does not
// edit these files without a person, and in the `permissions.deny` twin that
// keeps the Write and Edit tools off them. See FR-4.17, retired, in
// REQUIREMENTS.md, which records the condition for bringing it back.
func preflight(paths []string) (*rules.Set, []string) {
	var refusals []string

	set, err := rules.Load(paths)
	if err != nil {
		refusals = append(refusals, unloadable(err))
	}

	return set, refusals
}

// refusing answers every request with the same refusal, whatever it asks.
//
// A gate that cannot believe its own rule set has one thing to say and should
// say it to everything, instead of judging commands against a policy it has
// already reported as untrustworthy.
func refusing(refusals []string) hook.Decider {
	reason := joinReasons(refusals)
	return func(hook.Request) (hook.Decision, string) {
		return hook.DecisionDeny, reason
	}
}

func joinReasons(refusals []string) string {
	joined := ""
	for i, refusal := range refusals {
		if i > 0 {
			joined += "\n"
		}
		joined += refusal
	}
	return joined
}

// unloadable is the reason a broken rule set gives for permitting nothing.
//
// A gate that becomes permissive when its own configuration is broken reports
// success while guarding nothing, so the answer is a refusal.
// The cost is that a typo denies every command until it is fixed, which
// is why the message carries the parser's position and names a way out that
// does not itself need Bash: editing the rule file with the Edit tool.
func unloadable(err error) string {
	return fmt.Sprintf(
		"qwark's rule set will not load, so nothing is permitted:\n  %v\n"+
			"Fix the rule file with the Edit tool; that does not require Bash.", err)
}
