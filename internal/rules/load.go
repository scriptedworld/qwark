package rules

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/scriptedworld/qwark/internal/command"
)

// Everything that can be wrong with a rule set. Every one of them is fatal:
// **if any rule file cannot be read, no command is permitted.** A gate that
// degrades to permissive when its own configuration is broken reports success
// while guarding nothing.
//
// The cost is that a typo denies every command until it is fixed, which is why
// each of these carries the file and, where the format allows it, the line. The
// way out must also not require a shell: editing the file with the file tools
// does not.
var (
	ErrNoRuleFiles   = errors.New("no rule files given")
	ErrNoRulesFound  = errors.New("no rule files found")
	ErrUnreadable    = errors.New("rule file could not be read")
	ErrSyntax        = errors.New("rule file is not valid TOML")
	ErrRedefined     = errors.New("definition already created by another file")
	ErrDuplicateRule = errors.New("rule id already used")
	ErrUnknownAction = errors.New("not an action")
	ErrNoClauses     = errors.New("rule states no clauses")
	ErrUnknownGroup  = errors.New("clause names a group nothing declares")
	ErrEmptyGroup    = errors.New("group has no members")
	ErrClauseEmpty   = errors.New("clause selects nothing")
	ErrTagMissing    = errors.New("rule tags nothing")
	ErrUnknownNode   = errors.New("clause names a node type the parser does not have")
	ErrUnknownFlag   = errors.New("clause names a statement flag that does not exist")
	ErrGroupMatch    = errors.New("a group compares by value or partial, nothing else")
)

// ruleFileSuffix is what a directory contributes. A directory named on the
// command line contributes every rule file in it, and nothing else in it.
const ruleFileSuffix = ".toml"

// A Set is the aggregated rule set: every file's contents, merged, with each
// definition remembering which file created it.
//
// **A definition belongs to the file that created it.** A file may create
// declarations and groups of its own and may not redefine another file's.
// Collision is an error rather than a precedence order, so no file can quietly
// weaken another's definition by being read later.
//
// That is safe because a declaration grants eligibility, not permission: an
// explicit deny rule outranks any declaration, so a file adding a command still
// cannot run it past a deny stated elsewhere.
type Set struct {
	Shell    ShellPolicy
	Groups   map[string]Group
	Commands map[string]command.Declaration
	Rules    []Rule

	// origin remembers which file created each definition, so a collision can
	// name both files rather than only the one that lost.
	origin map[string]string
}

// Load reads every rule file named, in the order given.
//
// A path naming a directory contributes every `.toml` file directly in it, read
// in lexical order; a path naming a file contributes that file. Order affects
// only which collision is reported first: the strictest action wins, so no
// verdict depends on the order files were read in.
func Load(paths []string) (*Set, error) {
	if len(paths) == 0 {
		return nil, ErrNoRuleFiles
	}

	files, err := expand(paths)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%w in %s", ErrNoRulesFound, strings.Join(paths, ", "))
	}

	set := &Set{
		Groups:   map[string]Group{},
		Commands: map[string]command.Declaration{},
		origin:   map[string]string{},
	}

	for _, path := range files {
		parsed, err := read(path)
		if err != nil {
			return nil, err
		}
		if err := set.merge(path, parsed); err != nil {
			return nil, err
		}
	}

	if err := set.validate(); err != nil {
		return nil, err
	}
	return set, nil
}

// expand turns the paths given into the list of files to read.
func expand(paths []string) ([]string, error) {
	var files []string

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrUnreadable, path, err)
		}
		if !info.IsDir() {
			files = append(files, path)
			continue
		}

		found, err := inDirectory(path)
		if err != nil {
			return nil, err
		}
		files = append(files, found...)
	}

	return files, nil
}

// inDirectory lists a directory's rule files in lexical order. It does not
// descend: what a directory contributes should be answerable by listing it.
func inDirectory(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnreadable, dir, err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ruleFileSuffix) {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}

	slices.Sort(files)
	return files, nil
}

// read decodes one rule file, reporting a syntax error with the position the
// decoder found it at. A denial caused by a broken rule file has to say where,
// because until it is fixed every command is failing.
func read(path string) (File, error) {
	var parsed File

	if _, err := toml.DecodeFile(path, &parsed); err != nil {
		if perr, ok := errors.AsType[toml.ParseError](err); ok {
			return File{}, fmt.Errorf("%w: %s:\n%s",
				ErrSyntax, path, perr.ErrorWithPosition())
		}
		return File{}, fmt.Errorf("%w: %s: %w", ErrUnreadable, path, err)
	}

	return parsed, nil
}

// merge folds one file into the set, refusing any definition another file
// already created.
func (s *Set) merge(path string, file File) error {
	if file.Shell != nil {
		if err := s.claim(path, "shell", "the permitted shells"); err != nil {
			return err
		}
		s.Shell = *file.Shell
	}

	for name, group := range file.Group {
		if err := s.claim(path, "group:"+name, "group "+name); err != nil {
			return err
		}
		s.Groups[name] = group
	}

	for name, declaration := range file.Command {
		if err := s.claim(path, "command:"+name, "command "+name); err != nil {
			return err
		}
		s.Commands[name] = declaration
	}

	s.Rules = append(s.Rules, file.Rule...)
	return nil
}

// claim records that a file created a definition, refusing it if another file
// created it first.
func (s *Set) claim(path, key, described string) error {
	if owner, taken := s.origin[key]; taken {
		return fmt.Errorf("%w: %s is declared in %s and again in %s",
			ErrRedefined, described, owner, path)
	}
	s.origin[key] = path
	return nil
}
