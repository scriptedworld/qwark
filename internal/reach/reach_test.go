package reach_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/scriptedworld/qwark/internal/reach"
)

// radiusAt fixes a blast radius, failing the test if it will not.
func radiusAt(t *testing.T, root string) reach.Radius {
	t.Helper()

	radius, err := reach.New(root)
	if err != nil {
		t.Fatalf("New(%q) = %v", root, err)
	}
	return radius
}

// contains answers the question, failing the test if it cannot be asked.
func contains(t *testing.T, radius reach.Radius, base, path string) bool {
	t.Helper()

	inside, err := radius.Contains(base, path)
	if err != nil {
		t.Fatalf("Contains(%q, %q) = %v", base, path, err)
	}
	return inside
}

// COVERS: FR-9.1 | positive
func TestAPathInsideTheRadiusIsContained(t *testing.T) {
	t.Parallel()

	radius := radiusAt(t, "/home/user/proj")

	for _, path := range []string{
		"/home/user/proj",
		"/home/user/proj/a",
		"/home/user/proj/a/b/c.go",
		"/home/user/proj/./a",
		"/home/user/proj/a/../b",
	} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			if !contains(t, radius, "/home/user/proj", path) {
				t.Errorf("%q reported as outside", path)
			}
		})
	}
}

// COVERS: FR-9.2 | negative
func TestASiblingWhoseNameSharesAPrefixIsNotContained(t *testing.T) {
	t.Parallel()

	// The trap. `/home/user/project` begins with `/home/user/proj` as
	// text, and is not inside it as a path. A gate comparing strings would let
	// every write to the neighbouring directory through.
	radius := radiusAt(t, "/home/user/proj")

	for _, path := range []string{
		"/home/user/project",
		"/home/user/project/secrets",
		"/home/user/proj-backup",
		"/home/user/projX",
	} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			if contains(t, radius, "/home/user/proj", path) {
				t.Errorf("%q reported as inside; it only shares a prefix as text", path)
			}
		})
	}
}

// COVERS: FR-9.2 | negative
func TestClimbingOutIsNotContained(t *testing.T) {
	t.Parallel()

	radius := radiusAt(t, "/home/user/proj")

	for _, path := range []string{
		"/home/user/proj/../../../etc/passwd",
		"/home/user/proj/a/../../b",
		"/etc/passwd",
		"/",
		"/home/user",
	} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			if contains(t, radius, "/home/user/proj", path) {
				t.Errorf("%q reported as inside", path)
			}
		})
	}
}

// COVERS: FR-9.3 | positive
func TestARelativePathIsResolvedAgainstWhereTheCommandRuns(t *testing.T) {
	t.Parallel()

	radius := radiusAt(t, "/home/user/proj")

	cases := []struct {
		base string
		path string
		want bool
	}{
		{base: "/home/user/proj", path: "a.go", want: true},
		{base: "/home/user/proj/sub", path: "a.go", want: true},
		{base: "/home/user/proj/sub", path: "../a.go", want: true},
		{base: "/home/user/proj", path: "../other", want: false},
		{base: "/home/user/proj/sub", path: "../../../etc/passwd", want: false},
	}

	for _, c := range cases {
		t.Run(c.base+" :: "+c.path, func(t *testing.T) {
			t.Parallel()

			if got := contains(t, radius, c.base, c.path); got != c.want {
				t.Errorf("Contains(%q, %q) = %v, want %v", c.base, c.path, got, c.want)
			}
		})
	}
}

// COVERS: FR-9.4 | negative
func TestASymlinkLeavingTheRadiusIsNotContained(t *testing.T) {
	t.Parallel()

	// A purely lexical check calls this contained: every component of the text
	// is inside the radius. The shell writes through the link to somewhere
	// else entirely, which is the answer that matters.
	root := t.TempDir()
	inside := filepath.Join(root, "work")
	outside := filepath.Join(root, "elsewhere")
	for _, dir := range []string{inside, outside} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("creating %s: %v", dir, err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(inside, "escape")); err != nil {
		t.Fatalf("linking: %v", err)
	}

	radius := radiusAt(t, inside)

	if contains(t, radius, inside, filepath.Join(inside, "escape", "x.txt")) {
		t.Error("a path through a symlink out of the radius reported as inside")
	}
	if !contains(t, radius, inside, filepath.Join(inside, "real.txt")) {
		t.Error("an ordinary path inside the radius reported as outside")
	}
}

// COVERS: FR-9.4 | edge
func TestAFileThatDoesNotExistYetIsStillPlaced(t *testing.T) {
	t.Parallel()

	// A rule about writing is asked about files that have not been created, so
	// the leaf usually cannot be resolved. Its directories can, and that is
	// where a link would be.
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(deep, 0o750); err != nil {
		t.Fatalf("creating %s: %v", deep, err)
	}

	radius := radiusAt(t, root)

	for _, path := range []string{
		filepath.Join(deep, "not-created-yet.txt"),
		filepath.Join(root, "nor", "these", "directories", "x.txt"),
	} {
		if !contains(t, radius, root, path) {
			t.Errorf("%q reported as outside though it is within the radius", path)
		}
	}
}

// COVERS: FR-9.4 | property
func TestTheRadiusReportsItselfAsItWillBeCompared(t *testing.T) {
	t.Parallel()

	// A message about a refusal quotes the radius, and quoting the spelling
	// the author typed rather than the one being compared would send them
	// looking for a mismatch that is not there.
	root := t.TempDir()
	actual := filepath.Join(root, "actual")
	link := filepath.Join(root, "link")
	if err := os.MkdirAll(actual, 0o750); err != nil {
		t.Fatalf("creating %s: %v", actual, err)
	}
	if err := os.Symlink(actual, link); err != nil {
		t.Fatalf("linking: %v", err)
	}

	radius := radiusAt(t, link)
	if got := radius.Root(); got != reachResolved(t, actual) {
		t.Errorf("Root() = %q, want the resolved %q", got, actual)
	}
}

// reachResolved is what EvalSymlinks makes of a path that exists, which on
// macOS and some Linux setups differs from the path handed in.
func reachResolved(t *testing.T, path string) string {
	t.Helper()

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) = %v", path, err)
	}
	return resolved
}

// COVERS: FR-9.5 | negative
func TestARelativeRadiusOrBaseIsRefused(t *testing.T) {
	t.Parallel()

	// Resolving either against this process's own working directory would
	// answer a question about the wrong directory entirely.
	if _, err := reach.New("proj"); !errors.Is(err, reach.ErrRelativeRoot) {
		t.Errorf("New(relative) = %v, want %v", err, reach.ErrRelativeRoot)
	}

	radius := radiusAt(t, "/home/user/proj")
	if _, err := radius.Contains("relative/base", "a.go"); !errors.Is(err, reach.ErrRelativeBase) {
		t.Errorf("Contains(relative base) = %v, want %v", err, reach.ErrRelativeBase)
	}
}

// COVERS: FR-9.1 | edge
func TestARadiusOfTheWholeFilesystemContainsEverything(t *testing.T) {
	t.Parallel()

	// Degenerate, and worth pinning: the separator handling that stops
	// `/home/x/project` matching `/home/x/proj` must not stop `/` containing
	// anything at all.
	radius := radiusAt(t, "/")

	for _, path := range []string{"/", "/etc/passwd", "/home/user/proj"} {
		if !contains(t, radius, "/", path) {
			t.Errorf("%q reported as outside the root of the filesystem", path)
		}
	}
}
