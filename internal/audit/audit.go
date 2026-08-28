// Package audit records what qwark decided, so that a rule set can be changed
// on evidence rather than on recollection.
//
// # Why this is not optional to the design
//
// The plan qwark is being introduced under is to gate structurally first, log
// everything, and add rules while watching what comes through. Without a record
// the second half of that sentence has nothing to work from: a refusal exists
// only in the transcript of the session that was refused, and that goes away
// when the session does. A gate that denies and remembers nothing teaches
// nobody.
//
// # What a log write must never do
//
// **A failure to record does not change a verdict.** The decision is already
// made by the time it reaches here, and turning a full disk into a refusal
// would make the audit trail a way to stop the machine. The write failure is
// reported on stderr, where the hook contract sends it back, and the verdict
// stands.
//
// That is the permissive direction, and it is chosen deliberately against this
// project's usual instinct. It is also a real hole: somebody who can fill the
// disk can stop the recording without stopping the commands. The honest fix is
// the one FR-8.7 and the leaking-bucket note already describe for tag state,
// which is a writer the subject is not, and that arrives with the proxy rather
// than here. Recorded as an open question rather than solved quietly.
package audit

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// An Entry is one decision, as it will be read back.
//
// Field names are short because there will be one of these per Bash call and
// the file is read by machine first. Every field that could be absent is
// omitempty, so a reader can tell "no agent type" from "an empty agent type"
// without the file carrying a column of nulls.
type Entry struct {
	At        time.Time `json:"at"`
	RuleSet   string    `json:"rule_set"`
	Decision  string    `json:"decision"`
	Tool      string    `json:"tool"`
	Command   string    `json:"command,omitempty"`
	Rules     []string  `json:"rules,omitempty"`
	Agent     string    `json:"agent,omitempty"`
	Cwd       string    `json:"cwd,omitempty"`
	SessionID string    `json:"session,omitempty"`
	Env       []Var     `json:"env,omitempty"`
}

// A Var is one environment variable as recorded.
//
// **A withheld value is recorded as withheld, never omitted** (FR-4.9). A
// reader must be able to tell that a variable was present and its value kept
// back from the variable not being there at all, because those are different
// facts and only one of them is about secrecy.
type Var struct {
	Name     string `json:"name"`
	Value    string `json:"value,omitempty"`
	Withheld bool   `json:"withheld,omitempty"`
}

// A Recorder appends entries to a log.
type Recorder struct {
	path string
	out  io.Writer
}

// DefaultPath is where the log lives when nothing says otherwise.
//
// Under XDG state rather than beside the rules, because the rules are policy
// and this is history: one is read to decide and the other is written to
// remember, and mixing them means an install overwrites a record. It matches
// where the rest of this machine keeps per-tool state.
func DefaultPath() string {
	if state := os.Getenv("XDG_STATE_HOME"); state != "" {
		return filepath.Join(state, "qwark", "decisions.jsonl")
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "state", "qwark", "decisions.jsonl")
}

// The log is the record of what a gate decided, so it is readable by the user
// qwark runs as and by nobody else. The directory is walked to reach the file
// and holds nothing else, hence execute rather than read on the owner.
const (
	logDirMode  = 0o700
	logFileMode = 0o600
)

// To opens a recorder writing to a file, creating the directory if needed.
//
// **It does not truncate.** The file is opened for append, so a restart adds to
// the record rather than replacing it: a log with earlier entries missing reads
// as a clean history, which is worse than no log.
func To(path string) (*Recorder, error) {
	// Cleaned before it is used rather than after it is opened. The path comes
	// from a flag or from DefaultPath, so it is not the subject's to choose,
	// but a traversal reaching this far would be opened for append with the
	// record's own permissions, and the log is the one file whose contents are
	// the evidence that anything was judged at all.
	path = filepath.Clean(path)

	if err := os.MkdirAll(filepath.Dir(path), logDirMode); err != nil {
		return nil, fmt.Errorf("creating the log directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, logFileMode)
	if err != nil {
		return nil, fmt.Errorf("opening the log: %w", err)
	}

	return &Recorder{path: path, out: file}, nil
}

// Writing sends entries to an already-open writer, which is what a test wants.
func Writing(out io.Writer) *Recorder {
	return &Recorder{out: out}
}

// Record appends one entry.
//
// One JSON object per line. The format is chosen so that a partly written final
// line is one unreadable record rather than a file that will not parse, which
// is what a single JSON array would give after a kill.
func (r *Recorder) Record(entry Entry) error {
	if r == nil || r.out == nil {
		return nil
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encoding the log entry: %w", err)
	}

	if _, err := r.out.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("writing to %s: %w", r.path, err)
	}

	return nil
}

// Close releases the file, where there is one.
func (r *Recorder) Close() error {
	if r == nil {
		return nil
	}
	if closer, ok := r.out.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return fmt.Errorf("closing %s: %w", r.path, err)
		}
	}
	return nil
}
