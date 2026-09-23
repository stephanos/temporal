package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Proposal is one review-only regression source to write: the candidate it came from, the path
// the bridge named relative to the root, and its bytes.
type Proposal struct {
	Candidate string
	Path      string
	Source    string
}

// Resolve makes path absolute and resolves every symlink along its existing part, keeping the part
// that does not exist yet as written: a root is checked where it really is, not where its name
// points before anything is created under it.
func Resolve(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	existing, rest := absolute, ""
	for {
		resolved, err := filepath.EvalSymlinks(existing)
		if err == nil {
			return filepath.Join(resolved, rest), nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return absolute, nil
		}
		rest = filepath.Join(filepath.Base(existing), rest)
		existing = parent
	}
}

// Within reports whether path is root or under it; both are absolute and clean.
func Within(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

// OutsideModel resolves a root a command writes under and refuses one under the model root, both
// resolved through their symlinks first: a proposal is for review and a record is a replay's
// input, and the model never receives either.
func OutsideModel(flag, root, modelRoot string) (string, error) {
	resolvedRoot, err := Resolve(root)
	if err != nil {
		return "", fmt.Errorf("%s: %w", flag, err)
	}
	resolvedModel, err := Resolve(modelRoot)
	if err != nil {
		return "", fmt.Errorf("model root: %w", err)
	}
	if Within(resolvedModel, resolvedRoot) {
		return "", fmt.Errorf("%s must not be under the model root %s", flag, resolvedModel)
	}
	return resolvedRoot, nil
}

// WriteProposals writes each proposal under root, at the path the bridge named, and returns the
// written path by candidate. No root, no writing. Every path is checked before any file is
// written, so a path that would leave the root writes nothing at all. Each file is created
// exclusively: an existing destination is never replaced, and a directory that resolves outside
// the root through a symlink is refused. A write that fails returns what was written before it.
func WriteProposals(root string, proposals []Proposal) (map[string]string, error) {
	if root == "" {
		return nil, nil
	}
	resolvedRoot, err := Resolve(root)
	if err != nil {
		return nil, fmt.Errorf("promotion root: %w", err)
	}
	paths := make(map[string]string, len(proposals))
	for _, proposal := range proposals {
		relative := filepath.Clean(filepath.FromSlash(proposal.Path))
		if proposal.Path == "" || filepath.IsAbs(relative) || relative == "." || relative == ".." ||
			strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("proposal for %s names a path outside the promotion root: %q", proposal.Candidate, proposal.Path)
		}
		paths[proposal.Candidate] = filepath.Join(resolvedRoot, relative)
	}
	written := map[string]string{}
	for _, proposal := range proposals {
		path := paths[proposal.Candidate]
		if err := writeExclusive(resolvedRoot, path, proposal.Source); err != nil {
			return written, fmt.Errorf("write proposal for %s: %w", proposal.Candidate, err)
		}
		written[proposal.Candidate] = path
	}
	return written, nil
}

func writeExclusive(root, path, source string) error {
	// The directory is resolved through its existing part and checked before anything is created,
	// so a symlink under the root never gets a directory made on its far side.
	directory := filepath.Dir(path)
	resolved, err := Resolve(directory)
	if err != nil {
		return err
	}
	if !Within(root, resolved) {
		return fmt.Errorf("%s resolves outside the promotion root", directory)
	}
	if err := os.MkdirAll(resolved, 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(resolved, filepath.Base(path)), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, writeErr := file.WriteString(source)
	return errors.Join(writeErr, file.Close())
}
