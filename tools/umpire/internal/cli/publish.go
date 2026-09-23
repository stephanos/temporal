package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// The publication statuses.
const (
	// StatusPublished: the name did not exist and now holds the bytes.
	StatusPublished = "published"
	// StatusAlreadyPublished: the name already held exactly these bytes.
	StatusAlreadyPublished = "already-published"
)

// Publication is where published bytes stand and whether this call put them there.
type Publication struct {
	Status string
	Path   string
}

// ConflictError says the final name holds something other than the bytes: other bytes, or not a
// regular file at all. Nothing was overwritten.
type ConflictError struct {
	Path   string
	Detail string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("%s: %s; it is never overwritten", e.Path, e.Detail)
}

// Publish writes contents under root at name, exclusively and atomically: to a temporary file in
// root, synced and made 0644, then hard-linked to the final name, which fails when the name exists
// and never exposes a partial file under it. The name is the content's own identity, so two
// publishers of the same bytes need no lock: the second finds them already published.
//
// The root must exist; it is resolved through its symlinks. The name must be a bare base name. The
// context is checked immediately before the link: cancelled before it, nothing is published;
// after it, the bytes stand. The directory is not synced after the link, so a crash may lose the
// name, never expose a partial file; publishing again restores it. A leftover temporary file is
// dot-prefixed and never ends in the final name's extension.
func Publish(ctx context.Context, root, name string, contents []byte) (Publication, error) {
	if name == "" || name == "." || name == ".." || name != filepath.Base(name) {
		return Publication{}, fmt.Errorf("publication name %q is not a bare file name", name)
	}
	resolvedRoot, err := Resolve(root)
	if err != nil {
		return Publication{}, fmt.Errorf("publication root: %w", err)
	}
	info, err := os.Stat(resolvedRoot)
	if err != nil {
		return Publication{}, fmt.Errorf("publication root: %w", err)
	}
	if !info.IsDir() {
		return Publication{}, fmt.Errorf("publication root %s is not a directory", resolvedRoot)
	}
	final := filepath.Join(resolvedRoot, name)

	temporary, err := os.CreateTemp(resolvedRoot, "."+name+".tmp-*")
	if err != nil {
		return Publication{}, fmt.Errorf("publish %s: %w", final, err)
	}
	defer func() { _ = os.Remove(temporary.Name()) }()
	if _, err := temporary.Write(contents); err != nil {
		return Publication{}, errors.Join(fmt.Errorf("publish %s: %w", final, err), temporary.Close())
	}
	if err := temporary.Chmod(0o644); err != nil {
		return Publication{}, errors.Join(fmt.Errorf("publish %s: %w", final, err), temporary.Close())
	}
	if err := temporary.Sync(); err != nil {
		return Publication{}, errors.Join(fmt.Errorf("publish %s: %w", final, err), temporary.Close())
	}
	if err := temporary.Close(); err != nil {
		return Publication{}, fmt.Errorf("publish %s: %w", final, err)
	}
	if err := ctx.Err(); err != nil {
		return Publication{}, fmt.Errorf("publish %s: interrupted before publishing: %w", final, err)
	}
	err = os.Link(temporary.Name(), final)
	if err == nil {
		return Publication{Status: StatusPublished, Path: final}, nil
	}
	if !errors.Is(err, fs.ErrExist) {
		return Publication{}, fmt.Errorf("publish %s: %w", final, err)
	}
	if err := sameContents(final, contents); err != nil {
		return Publication{}, err
	}
	return Publication{Status: StatusAlreadyPublished, Path: final}, nil
}

// sameContents requires the existing final name to be a regular file holding exactly contents. It
// inspects the name without following it, opens it without following it or blocking on it, checks
// again that what it opened is a regular file, and reads at most one byte past contents, so no
// existing name can make it read more than it wrote.
func sameContents(path string, contents []byte) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("publish %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return &ConflictError{Path: path, Detail: fmt.Sprintf("the name exists and is not a regular file (%s)", info.Mode().Type())}
	}
	file, err := openExisting(path)
	if err != nil {
		return &ConflictError{Path: path, Detail: fmt.Sprintf("the name exists and cannot be read as a file: %v", err)}
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil {
		return fmt.Errorf("publish %s: %w", path, err)
	}
	if !opened.Mode().IsRegular() {
		return &ConflictError{Path: path, Detail: "the name changed to something other than a regular file"}
	}
	existing, err := io.ReadAll(io.LimitReader(file, int64(len(contents))+1))
	if err != nil {
		return fmt.Errorf("publish %s: %w", path, err)
	}
	if !bytes.Equal(existing, contents) {
		return &ConflictError{Path: path, Detail: "the name holds other bytes"}
	}
	return nil
}
