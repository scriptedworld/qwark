// Package reach answers whether a path a command names falls inside the
// directory the agent was started in: its blast radius.
//
// The eventual rule this serves is that any path given to a command that writes
// must be inside that radius. A useful consequence follows for free: anything
// outside it is protected without a rule of its own, which is where qwark's own
// rule files, its state, and the task manifest are meant to live.
//
// **That consequence holds for Bash only.** The Write and Edit tools reach
// those same paths without passing through qwark, so living outside the working
// directory is necessary but not sufficient; it wants filesystem ownership and
// a permissions rule as well.
package reach

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// What containment cannot establish. Each is a refusal rather than a
// malfunction: a path qwark cannot place is one it cannot judge.
var (
	ErrRelativeRoot = errors.New("the blast radius is not an absolute path")
	ErrRelativeBase = errors.New("the directory to resolve against is not absolute")
)

// A Radius is the directory a command's paths must stay inside.
type Radius struct {
	root string
}

// New fixes a blast radius at an absolute path.
//
// A relative root is refused rather than resolved against the current
// directory, because the current directory of the process asking the question
// has nothing to do with where the agent was started.
func New(root string) (Radius, error) {
	if !filepath.IsAbs(root) {
		return Radius{}, fmt.Errorf("%w: %q", ErrRelativeRoot, root)
	}
	return Radius{root: Resolve(filepath.Clean(root))}, nil
}

// Root returns the radius as it will be compared, so a message can quote it.
func (r Radius) Root() string { return r.root }

// Contains reports whether a path names something inside the radius.
//
// A relative path is resolved against base, which is the directory the command
// will run in. Both `..` and symbolic links are resolved before comparing,
// because either can leave the radius while the text says otherwise.
func (r Radius) Contains(base, path string) (bool, error) {
	if !filepath.IsAbs(base) {
		return false, fmt.Errorf("%w: %q", ErrRelativeBase, base)
	}

	full := path
	if !filepath.IsAbs(full) {
		full = filepath.Join(base, full)
	}

	return within(r.root, Resolve(filepath.Clean(full))), nil
}

// within compares two cleaned absolute paths by their components.
//
// **This is the trap the whole file exists to avoid.** Comparing them as
// strings puts `/home/x/project` inside `/home/x/proj`, because one is a prefix
// of the other as text and neither is inside the other as a path. Requiring the
// separator is what makes the comparison about directories.
func within(root, path string) bool {
	if path == root {
		return true
	}
	if root == string(filepath.Separator) {
		return strings.HasPrefix(path, root)
	}
	return strings.HasPrefix(path, root+string(filepath.Separator))
}

// Resolve follows symbolic links as far as the path exists, then reattaches
// whatever does not exist yet.
//
// **Anything that is a symlink is resolved to its full path**, wherever a path
// is compared: containment, the permitted shells, the protected paths. Two
// spellings of one file must reach one answer, or a rule about a file is a rule
// about one way of writing its name.
//
// The reattaching is the point. A rule about writing is asked about files that
// have not been created, so the leaf usually does not exist and cannot be
// resolved, but its parent directory can, and a link is far more likely to be
// a directory in the middle of the path than the file at the end of it. A purely
// lexical check would call `<radius>/link/x` contained while the shell wrote
// through `link` to somewhere else entirely.
//
// A path that cannot be resolved at all is returned as it was: the caller is
// deciding containment, and a lexically-cleaned path is a better answer than an
// error that stops every command because a directory is missing.
func Resolve(path string) string {
	remainder := ""
	current := path

	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			return filepath.Join(resolved, remainder)
		}
		if !errors.Is(err, os.ErrNotExist) {
			return path
		}

		parent := filepath.Dir(current)
		if parent == current {
			return path
		}
		remainder = filepath.Join(filepath.Base(current), remainder)
		current = parent
	}
}
