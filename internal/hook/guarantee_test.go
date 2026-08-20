package hook_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// COVERS: FR-10.8 | property
func TestQwarkRunsInOneGoroutine(t *testing.T) {
	t.Parallel()

	// Run recovers from a panic so that a gate dying mid-judgement refuses
	// rather than letting the command through. **That guarantee is only
	// complete while there is one goroutine**: recover catches a panic in the
	// goroutine that deferred it and in no other, so a panic anywhere else
	// would take the process down past every safeguard in this package.
	//
	// This is therefore an architectural invariant rather than a style
	// preference, and it is enforced here rather than remembered.
	found := goStatementsIn(t, filepath.Join("..", ".."))

	if len(found) != 0 {
		t.Errorf("qwark starts %d goroutine(s), which Run cannot recover from:\n  %s",
			len(found), strings.Join(found, "\n  "))
	}
}

// COVERS: FR-4.19 | property
func TestQwarkNeverExecutesAnything(t *testing.T) {
	t.Parallel()

	// qwark reads rule files, shell snapshots and `.git/HEAD` -- all of them
	// files its own subject can write. Running any of them would be running the
	// subject's code inside the judge, which is the mistake this project keeps
	// finding in other places: sourcing the snapshot to ask what an alias is,
	// or asking git what branch it is on when an alias in `.git/config`
	// executes on a plain invocation.
	//
	// The ban is on the CALLS that spawn a process rather than on whole
	// packages. An earlier version banned `syscall` outright, which was too
	// blunt in both directions: it forbade asking who owns a file -- something
	// qwark must do to check its own rule files are not writable -- while a
	// package ban says nothing about `os.StartProcess`, which is in a package
	// nothing could ban.
	root := filepath.Join("..", "..")

	if found := importsOf(t, root, []string{`"os/exec"`, `"plugin"`}); len(found) != 0 {
		t.Errorf("qwark imports a package whose purpose is to run things:\n  %s",
			strings.Join(found, "\n  "))
	}

	spawns := []string{
		"exec.Command", "exec.CommandContext", "exec.LookPath",
		"syscall.Exec", "syscall.ForkExec", "syscall.StartProcess",
		"os.StartProcess",
	}
	if found := callsTo(t, root, spawns); len(found) != 0 {
		t.Errorf("qwark calls something that starts a process:\n  %s",
			strings.Join(found, "\n  "))
	}
}

// qualified reports a node written as `package.Name`, which is how a call into
// another package appears whatever it is called through.
func qualified(node ast.Node) (string, bool) {
	selector, ok := node.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	pkg, ok := selector.X.(*ast.Ident)
	if !ok {
		return "", false
	}
	return pkg.Name + "." + selector.Sel.Name, true
}

// callsTo reports every production use of the named package functions.
func callsTo(t *testing.T, root string, forbidden []string) []string {
	t.Helper()

	var found []string
	fileSet := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip(entry) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		parsed, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			if name, ok := qualified(node); ok && slices.Contains(forbidden, name) {
				found = append(found, fileSet.Position(node.Pos()).String()+": "+name)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}

	return found
}

// importsOf reports every production file importing one of the given paths.
func importsOf(t *testing.T, root string, forbidden []string) []string {
	t.Helper()

	var found []string
	fileSet := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip(entry) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		parsed, err := parser.ParseFile(fileSet, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		for _, imported := range parsed.Imports {
			for _, banned := range forbidden {
				if imported.Path.Value == banned {
					found = append(found, path+": "+banned)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}

	return found
}

// goStatementsIn reports every `go` statement in the production source of a
// module, by position. Test files are excluded: they are not what runs inside
// the hook.
func goStatementsIn(t *testing.T, root string) []string {
	t.Helper()

	var found []string
	fileSet := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip(entry) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		parsed, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			if _, isGo := node.(*ast.GoStmt); isGo {
				found = append(found, fileSet.Position(node.Pos()).String())
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}

	return found
}

// skip reports whether an entry is not production Go source.
func skip(entry fs.DirEntry) bool {
	name := entry.Name()
	if entry.IsDir() {
		return strings.HasPrefix(name, ".") || name == "rules"
	}
	return !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go")
}
