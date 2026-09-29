// Package suite judges the rule-set cases under a testdata directory and
// reports the result as a result envelope.
//
// The layout is <set>/<verdict>/<name>.cmd, one command per file, with each
// set's rule files named in <set>/set.txt. rules/testdata/README.md describes
// it for the people writing cases.
//
// A run fails when any case gets a verdict other than its directory's, when
// any rule the sets include is triggered by no case, and when any is held
// quiet by no case. A rule nothing triggers is a rule nobody has seen decide
// anything, and one that can never fire reads exactly like one that is
// working. A rule nothing holds quiet is one nobody has seen stay out of the
// command beside it, and one that fires on everything reads the same way.
package suite

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/scriptedworld/qwark/internal/rules"
	"github.com/scriptedworld/qwark/internal/shell"
)

// Reason kinds, so a consumer tells a failing case from an untested rule
// without reading the message.
const (
	KindCaseFailed       = "case-failed"
	KindRuleNotTriggered = "rule-not-triggered"
	KindRuleNotQuieted   = "rule-not-quieted"
)

// Names a case can take that stand for no rule: the engine's own refusal, and
// the verdict for a command no rule decided.
const (
	nameEngine  = "engine"
	nameDefault = "default"
)

// DefaultCwd is where a case's call comes from when it names nowhere.
const DefaultCwd = "/srv/project"

// Layout errors: the directory cannot be read as a suite at all, which is a
// different claim from a suite that ran and failed.
var (
	ErrNoSets    = errors.New("no <set>/set.txt under the case directory")
	ErrNoCases   = errors.New("no <set>/<verdict>/*.cmd under the case directory")
	ErrUnknown   = errors.New("case names a set with no set.txt")
	ErrNoVerdict = errors.New("case directory is not a verdict")
)

// A Reason is one entry in an envelope's reasons.
type Reason struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// Statistics are what a run counted.
type Statistics struct {
	Cases          int `json:"cases"`
	CasesFailed    int `json:"cases_failed"`
	Rules          int `json:"rules"`
	RulesTriggered int `json:"rules_triggered"`
	RulesQuieted   int `json:"rules_quieted"`
}

// Metadata carries the statistics and what the run read.
type Metadata struct {
	Statistics Statistics        `json:"statistics"`
	Evidence   map[string]string `json:"evidence"`
}

// An Envelope is the result of one run, in the shape every producer writes.
type Envelope struct {
	Success  bool     `json:"success"`
	Reasons  []Reason `json:"reasons,omitempty"`
	Metadata Metadata `json:"metadata"`
}

// A Case is one file: the command, and what it has to be decided as.
type Case struct {
	Path    string
	Set     string
	Verdict rules.Action
	Name    string
	Command string
	Headers map[string][]string
}

// Run judges every case under dir and returns the envelope. An error means the
// directory could not be read as a suite, not that the suite failed.
func Run(dir string) (Envelope, error) {
	root := os.DirFS(dir)

	sets, err := loadSets(root, dir)
	if err != nil {
		return Envelope{}, err
	}
	cases, err := loadCases(root, sets)
	if err != nil {
		return Envelope{}, err
	}

	env := Envelope{Metadata: Metadata{Evidence: map[string]string{"cases": dir}}}
	triggered := map[string]bool{}
	quieted := map[string]bool{}
	for _, c := range cases {
		problems := judge(sets[c.Set], c, triggered, quieted)
		for _, problem := range problems {
			env.Reasons = append(env.Reasons, Reason{KindCaseFailed, c.Path + ": " + problem})
		}
		if len(problems) > 0 {
			env.Metadata.Statistics.CasesFailed++
		}
	}

	included := includedRules(sets)
	for _, id := range included {
		if !triggered[id] {
			env.Reasons = append(env.Reasons, Reason{
				KindRuleNotTriggered, id + ": no case triggered this rule",
			})
		}
		if !quieted[id] {
			env.Reasons = append(env.Reasons, Reason{
				KindRuleNotQuieted, id + ": no case held this rule quiet",
			})
		}
	}

	env.Metadata.Statistics.Cases = len(cases)
	env.Metadata.Statistics.Rules = len(included)
	env.Metadata.Statistics.RulesTriggered = len(included) - countMissing(included, triggered)
	env.Metadata.Statistics.RulesQuieted = len(included) - countMissing(included, quieted)
	env.Success = len(env.Reasons) == 0
	return env, nil
}

// loadSets reads every <set>/set.txt and loads the rule paths it names,
// relative to that set's directory.
func loadSets(root fs.FS, dir string) (map[string]*rules.Set, error) {
	listings, err := fs.Glob(root, "*/set.txt")
	if err != nil || len(listings) == 0 {
		return nil, ErrNoSets
	}
	sets := make(map[string]*rules.Set, len(listings))
	for _, listing := range listings {
		name := path.Dir(listing)
		body, err := fs.ReadFile(root, listing)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", listing, err)
		}
		var paths []string
		for _, line := range strings.Split(string(body), "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				paths = append(paths, filepath.Join(dir, name, line))
			}
		}
		set, err := rules.Load(paths)
		if err != nil {
			return nil, fmt.Errorf("set %s: %w", name, err)
		}
		sets[name] = set
	}
	return sets, nil
}

// loadCases reads every case file, and refuses one placed where the layout
// gives it no meaning.
func loadCases(root fs.FS, sets map[string]*rules.Set) ([]Case, error) {
	files, err := fs.Glob(root, "*/*/*.cmd")
	if err != nil || len(files) == 0 {
		return nil, ErrNoCases
	}
	cases := make([]Case, 0, len(files))
	for _, file := range files {
		body, err := fs.ReadFile(root, file)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", file, err)
		}
		c := parseCase(file, string(body))
		if _, known := sets[c.Set]; !known {
			return nil, fmt.Errorf("%w: %s", ErrUnknown, file)
		}
		if !c.Verdict.Decides() {
			return nil, fmt.Errorf("%w: %s", ErrNoVerdict, file)
		}
		cases = append(cases, c)
	}
	return cases, nil
}

// headerPrefix opens a header line in a case file.
const headerPrefix = "# "

// parseCase takes what a case must decide from its path, and its headers and
// command from its body.
func parseCase(file, body string) Case {
	parts := strings.Split(file, "/")
	name, _, _ := strings.Cut(strings.TrimSuffix(parts[2], ".cmd"), "--")
	c := Case{
		Path: file, Set: parts[0], Verdict: rules.Action(parts[1]),
		Name: name, Headers: map[string][]string{},
	}
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	for len(lines) > 0 && strings.HasPrefix(lines[0], headerPrefix) {
		key, value, _ := strings.Cut(strings.TrimPrefix(lines[0], headerPrefix), ":")
		c.Headers[key] = strings.Fields(value)
		lines = lines[1:]
	}
	c.Command = strings.Join(lines, "\n")
	return c
}

// judge evaluates one case, records every rule it triggered and every rule it
// held quiet, and returns each way the outcome differs from the case.
func judge(set *rules.Set, c Case, triggered, quieted map[string]bool) []string {
	parsed, err := shell.Parse(c.Command)
	if err != nil {
		return []string{"does not parse: " + err.Error()}
	}
	ctx := rules.Context{Cwd: DefaultCwd, Tags: map[string]bool{}}
	if cwd := c.Headers["cwd"]; len(cwd) == 1 {
		ctx.Cwd = cwd[0]
	}
	for _, tag := range c.Headers["tags"] {
		ctx.Tags[tag] = true
	}
	outcome := set.Evaluate(parsed, ctx)

	ids := fired(outcome)
	acted := append(ids, tagRules(set, outcome)...)
	for _, id := range acted {
		triggered[id] = true
	}
	for _, id := range c.Headers["quiet"] {
		if !slices.Contains(acted, id) {
			quieted[id] = true
		}
	}
	return compare(c, outcome, ids, acted)
}

// compare lists every way an outcome differs from its case.
func compare(c Case, outcome rules.Outcome, ids, acted []string) []string {
	var problems []string
	if outcome.Action != c.Verdict {
		problems = append(problems,
			fmt.Sprintf("%s, want %s; reasons %v", outcome.Action, c.Verdict, ids))
	}
	switch c.Name {
	case nameDefault:
	case nameEngine:
		if !slices.ContainsFunc(ids, func(id string) bool { return strings.HasPrefix(id, "(engine)") }) {
			problems = append(problems, fmt.Sprintf("the engine did not refuse it; reasons %v", ids))
		}
	default:
		if !slices.Contains(acted, c.Name) {
			problems = append(problems, fmt.Sprintf("%s did not fire; reasons %v", c.Name, ids))
		}
	}
	for _, want := range c.Headers["fires"] {
		if !slices.Contains(ids, want) {
			problems = append(problems, fmt.Sprintf("%s did not fire; reasons %v", want, ids))
		}
	}
	for _, unwanted := range c.Headers["quiet"] {
		if slices.Contains(acted, unwanted) {
			problems = append(problems, unwanted+" fired and should not have")
		}
	}
	return problems
}

// fired lists the rules among the reasons for a verdict.
func fired(outcome rules.Outcome) []string {
	ids := make([]string, 0, len(outcome.Findings))
	for _, finding := range outcome.Findings {
		ids = append(ids, finding.Rule)
	}
	return ids
}

// tagRules lists the tag rules whose change the outcome carries. A tag rule
// decides nothing, so it is never among the reasons; setting or clearing its
// tag is what it does.
func tagRules(set *rules.Set, outcome rules.Outcome) []string {
	var ids []string
	for _, change := range outcome.Tags {
		action := rules.ActionUntag
		if change.Set {
			action = rules.ActionTag
		}
		for _, rule := range set.Rules {
			if rule.Action == action && rule.Tag == change.Name {
				ids = append(ids, rule.ID)
			}
		}
	}
	return ids
}

// includedRules is every rule any set loads, once each and in order.
func includedRules(sets map[string]*rules.Set) []string {
	seen := map[string]bool{}
	for _, set := range sets {
		for _, rule := range set.Rules {
			seen[rule.ID] = true
		}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func countMissing(ids []string, triggered map[string]bool) int {
	missing := 0
	for _, id := range ids {
		if !triggered[id] {
			missing++
		}
	}
	return missing
}
