package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// digestLength is how much of the hash is kept.
//
// Sixteen hex characters, 64 bits. This identifies a rule set in a log entry so
// that entries made under different policies are not compared with each other;
// it is not defending against anybody constructing a collision, because someone
// who can rewrite the rule files has already won and does not need to forge its
// name.
const digestLength = 16

// digestOf identifies a rule set by what it contains.
//
// **Content only, never paths.** The same rules installed at
// ~/.config/qwark/rules and sitting in the repository's own rules/ must produce
// the same digest, because the question anybody asks is whether the two are the
// same policy, and a digest that included the path could never answer it. That
// makes this the drift check as well as the identifier: equal digests mean equal
// policy, whatever it is called and wherever it lives.
//
// Files are hashed in the order they were loaded, and that order is already
// fixed: a directory contributes its `.toml` files in lexical order. So the
// digest is stable across runs and changes when any file's content changes.
//
// The length of each file goes into the hash before its bytes. Without it, two
// files could be concatenated differently and reach the same digest, which is a
// silly way to lose a property that costs one line to keep.
func digestOf(files []string) (string, error) {
	sum := sha256.New()

	for _, file := range files {
		// The names arrive from the rule loader, which built them by walking
		// the directories the registration named, so none of them is the
		// subject's to choose. Cleaned anyway: this hash is what says which
		// rule set judged a command, and a digest over the wrong file is a
		// record that reads as authoritative and is not.
		content, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			return "", fmt.Errorf("%w: hashing %s: %w", ErrUnreadable, file, err)
		}
		_, _ = fmt.Fprintf(sum, "%d\n", len(content))
		_, _ = sum.Write(content)
	}

	return hex.EncodeToString(sum.Sum(nil))[:digestLength], nil
}
