package artifact

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/host/fs"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
)

// References is the live state that an owner's final committed state
// requires of one payload root it exclusively owns: a Campaign's success,
// failure or diagnostic directory, a corpus's case directory, or a
// minimizer's output root. It lists every artifact and execution-owned file
// that committed state still references, including the payloads of durable
// pending-result receipts until an outcome transaction retires or adopts
// them, and every borrowed artifact the owner reads but does not own.
type References struct {
	// Artifacts names the live artifact directories in the owned root, one
	// path element each.
	Artifacts []string
	// Files are the live execution-owned files in the owned root, such as
	// diagnostic sidecars and pending-result payloads, with the identity the
	// owner recorded for each.
	Files []FileReference
	// Borrowed are absolute paths of artifacts that another owner holds:
	// shard evidence an aggregate references or a corpus case's source. They
	// are validated like owned references but never swept, and none may lie
	// inside the owned root.
	Borrowed []string
}

// FileReference is an execution-owned file the owner's state records.
type FileReference struct {
	Name   string
	SHA256 record.SHA256
	Bytes  uint64
}

// ErrCorruptReference matches every ReferenceError.
var ErrCorruptReference = errors.New("live artifact reference is missing or corrupt")

// ReferenceError reports a live reference that is missing or fails
// validation. It is corruption of the owner's state: SweepOwnedRoot removes
// nothing once one is found. errors.Is matches ErrCorruptReference and the
// underlying error, such as os.ErrNotExist or ErrUnsupportedFormat.
type ReferenceError struct {
	Reference string
	Err       error
}

func (err *ReferenceError) Error() string {
	return fmt.Sprintf("%s: %s: %v", ErrCorruptReference, err.Reference, err.Err)
}

func (err *ReferenceError) Is(target error) bool {
	return target == ErrCorruptReference
}

func (err *ReferenceError) Unwrap() error {
	return err.Err
}

// ValidateReferences validates every live reference of the owned root: each
// artifact opens, each file matches its recorded identity, and each borrowed
// artifact opens outside the owned root. The first failure is a
// ReferenceError.
func ValidateReferences(root string, live References) error {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve owned payload root: %w", err)
	}
	for _, name := range live.Artifacts {
		if err := validOwnedName(name); err != nil {
			return &ReferenceError{Reference: name, Err: err}
		}
		if err := validateArtifactReference(filepath.Join(absoluteRoot, name)); err != nil {
			return &ReferenceError{Reference: name, Err: err}
		}
	}
	for _, file := range live.Files {
		if err := validOwnedName(file.Name); err != nil {
			return &ReferenceError{Reference: file.Name, Err: err}
		}
		if err := validateFileReference(absoluteRoot, file); err != nil {
			return &ReferenceError{Reference: file.Name, Err: err}
		}
	}
	for _, borrowed := range live.Borrowed {
		if !filepath.IsAbs(borrowed) {
			return &ReferenceError{Reference: borrowed, Err: errors.New("borrowed artifact path is not absolute")}
		}
		if within(absoluteRoot, filepath.Clean(borrowed)) {
			return &ReferenceError{Reference: borrowed, Err: errors.New("borrowed artifact lies inside the owned root")}
		}
		if err := validateArtifactReference(borrowed); err != nil {
			return &ReferenceError{Reference: borrowed, Err: err}
		}
	}
	return nil
}

// removeOwnedEntry removes one unreferenced entry of an owned root. Tests
// replace it to inject removal failures.
var removeOwnedEntry = func(root *os.Root, name string) error {
	return root.RemoveAll(name)
}

// SweepOwnedRoot removes what the owner's final committed state no longer
// references from a payload root it exclusively owns, and returns the names
// it removed.
//
// The caller must be the root's only writer, holding its owner's lock, and
// must pass every live reference. SweepOwnedRoot first validates them all
// (ValidateReferences); a missing or corrupt one fails with a ReferenceError
// and removes nothing. It then classifies every entry before removing any:
//
//   - an unreferenced artifact directory (sha256-…) or publication staging
//     directory (.publish-…) is reclaimed, as is an unreferenced regular file
//     with a single link (an orphaned sidecar or a publication temporary) or
//     either name of a closed, root-local hard-link pair left by an older
//     interrupted owned-file publication;
//   - a shared target pool entry (a regular sha256-… file), a file with other
//     links, a symbolic link, the target pool directory and any other entry
//     are not the caller's to sweep and fail the sweep before removal.
//
// Shared target pools are pruned only by PruneTargetPool, and borrowed
// artifacts only by their own owners. A removal failure is returned wrapped,
// so errors.Is still matches its cause. A missing root with no live reference
// has nothing to sweep.
func SweepOwnedRoot(root string, live References) (_ []string, retErr error) {
	if err := ValidateReferences(root, live); err != nil {
		return nil, err
	}
	info, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open owned payload root: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.New("owned payload root is not a directory")
	}
	pinned, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("pin owned payload root: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, pinned.Close())
	}()
	directory, err := pinned.Open(".")
	if err != nil {
		return nil, fmt.Errorf("open owned payload root: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, directory.Close())
	}()
	if pinnedInfo, err := directory.Stat(); err != nil || !os.SameFile(info, pinnedInfo) {
		return nil, errors.Join(errors.New("owned payload root changed while opening"), err)
	}
	entries, err := directory.ReadDir(-1)
	if err != nil {
		return nil, fmt.Errorf("list owned payload root: %w", err)
	}
	referenced := make(map[string]struct{}, len(live.Artifacts)+len(live.Files))
	for _, name := range live.Artifacts {
		referenced[name] = struct{}{}
	}
	for _, file := range live.Files {
		referenced[file.Name] = struct{}{}
	}
	infos := make(map[string]os.FileInfo, len(entries))
	for _, entry := range entries {
		info, err := pinned.Lstat(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("inspect owned payload %s: %w", entry.Name(), err)
		}
		infos[entry.Name()] = info
	}
	interruptedPublications := interruptedOwnedFilePublications(entries, infos, referenced)
	var reclaim []string
	for _, entry := range entries {
		name := entry.Name()
		if _, found := referenced[name]; found {
			continue
		}
		if err := reclaimable(name, infos[name], interruptedPublications[name]); err != nil {
			return nil, err
		}
		reclaim = append(reclaim, name)
	}
	slices.Sort(reclaim)
	var removed []string
	for _, name := range reclaim {
		if err := removeOwnedEntry(pinned, name); err != nil && !errors.Is(err, os.ErrNotExist) {
			return removed, errors.Join(fmt.Errorf("remove unreferenced owned payload %s: %w", name, err), syncRemoval(directory, removed))
		}
		removed = append(removed, name)
	}
	return removed, syncRemoval(directory, removed)
}

func syncRemoval(directory *os.File, removed []string) error {
	if len(removed) == 0 {
		return nil
	}
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync owned payload root: %w", err)
	}
	return nil
}

func interruptedOwnedFilePublications(entries []os.DirEntry, infos map[string]os.FileInfo, referenced map[string]struct{}) map[string]bool {
	interrupted := make(map[string]bool)
	for _, entry := range entries {
		temporary := entry.Name()
		if !strings.HasPrefix(temporary, ".publish-") {
			continue
		}
		if _, live := referenced[temporary]; live {
			continue
		}
		info := infos[temporary]
		if !info.Mode().IsRegular() {
			continue
		}
		links, known := hostfs.LinkCount(info)
		if !known || links != 2 {
			continue
		}
		peer := ""
		for _, candidate := range entries {
			name := candidate.Name()
			candidateInfo := infos[name]
			if name == temporary || !candidateInfo.Mode().IsRegular() || !os.SameFile(info, candidateInfo) {
				continue
			}
			if peer != "" {
				peer = ""
				break
			}
			peer = name
		}
		if peer == "" || strings.HasPrefix(peer, ".publish-") || isPoolEntryName(peer) || isArtifactDirectoryName(peer) {
			continue
		}
		interrupted[temporary] = true
		interrupted[peer] = true
	}
	return interrupted
}

func reclaimable(name string, info os.FileInfo, interruptedPublication bool) error {
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		return fmt.Errorf("owned payload root entry %s is a %w", name, hostfs.ErrSymbolicLink)
	case info.IsDir() && (isArtifactDirectoryName(name) || strings.HasPrefix(name, ".publish-")):
		return nil
	case info.Mode().IsRegular() && isPoolEntryName(name):
		return fmt.Errorf("owned payload root entry %s is a shared target pool entry; only PruneTargetPool removes it", name)
	case info.Mode().IsRegular() && interruptedPublication:
		return nil
	case info.Mode().IsRegular():
		if links, known := hostfs.LinkCount(info); !known || links != 1 {
			return fmt.Errorf("owned payload root entry %s has other links and is not owned by this root", name)
		}
		return nil
	default:
		return fmt.Errorf("owned payload root contains unexpected entry %s", name)
	}
}

// isArtifactDirectoryName reports a directory name PublishArtifact gives an
// artifact: identityDirectory of a SHA-256, shortened or complete.
func isArtifactDirectoryName(name string) bool {
	digest, found := strings.CutPrefix(name, "sha256-")
	if !found || (len(digest) != 32 && len(digest) != 64) {
		return false
	}
	_, err := hex.DecodeString(digest)
	return err == nil
}

// PublishOwnedFile durably publishes data as name in a payload root its caller
// owns, such as a diagnostic sidecar, before the caller references it. The
// file is written and synced under a temporary name, atomically renamed to
// name without replacement, and the root synced (with any directory created for it), so
// the returned reference is durable. An existing file with the same content
// is reused; one with other content fails. A crash leaves at most an
// unreferenced file that SweepOwnedRoot reclaims.
func PublishOwnedFile(ctx context.Context, root, name string, data []byte) (_ FileReference, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return FileReference{}, err
	}
	if err := validOwnedName(name); err != nil {
		return FileReference{}, err
	}
	if isPoolEntryName(name) || isArtifactDirectoryName(name) || strings.HasPrefix(name, ".publish-") {
		return FileReference{}, fmt.Errorf("owned file name %q is reserved", name)
	}
	if err := makeDurableDirectory(ctx, root); err != nil {
		return FileReference{}, fmt.Errorf("create owned payload root: %w", err)
	}
	reference := FileReference{Name: name, SHA256: record.HashBytes(data), Bytes: uint64(len(data))}
	temporary, err := os.CreateTemp(root, ".publish-*")
	if err != nil {
		return FileReference{}, fmt.Errorf("create owned file %s: %w", name, err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		if temporaryPath == "" {
			return
		}
		if err := os.Remove(temporaryPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			retErr = errors.Join(retErr, err)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return FileReference{}, errors.Join(err, temporary.Close())
	}
	if _, err := copyWithContext(ctx, temporary, bytes.NewReader(data)); err != nil {
		return FileReference{}, errors.Join(fmt.Errorf("write owned file %s: %w", name, err), temporary.Close())
	}
	if err := syncFileContext(ctx, temporary); err != nil {
		return FileReference{}, errors.Join(fmt.Errorf("sync owned file %s: %w", name, err), temporary.Close())
	}
	if err := temporary.Close(); err != nil {
		return FileReference{}, fmt.Errorf("close owned file %s: %w", name, err)
	}
	final := filepath.Join(root, name)
	if err := hostfs.RenameNoReplace(temporaryPath, final); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return FileReference{}, fmt.Errorf("publish owned file %s: %w", name, err)
		}
		absoluteRoot, absErr := filepath.Abs(root)
		if absErr != nil {
			return FileReference{}, absErr
		}
		if err := validateFileReference(absoluteRoot, reference); err != nil {
			return FileReference{}, fmt.Errorf("existing owned file %s differs: %w", name, err)
		}
	} else {
		temporaryPath = ""
	}
	if temporaryPath != "" {
		if err := os.Remove(temporaryPath); err != nil {
			return FileReference{}, fmt.Errorf("remove owned file temporary: %w", err)
		}
		temporaryPath = ""
	}
	if err := syncDirectoryContext(ctx, root); err != nil {
		return FileReference{}, fmt.Errorf("sync owned payload root: %w", err)
	}
	return reference, nil
}

func validOwnedName(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) || filepath.Base(name) != name {
		return fmt.Errorf("owned payload name %q is not one path element", name)
	}
	return nil
}

func validateArtifactReference(path string) error {
	opened, err := OpenArtifact(path)
	if err != nil {
		return err
	}
	return opened.Close()
}

func validateFileReference(root string, reference FileReference) (retErr error) {
	pinned, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer func() {
		retErr = errors.Join(retErr, pinned.Close())
	}()
	file, info, err := hostfs.OpenRoot(pinned, reference.Name)
	if err != nil {
		return err
	}
	defer func() {
		retErr = errors.Join(retErr, file.Close())
	}()
	if info.Size() < 0 || uint64(info.Size()) != reference.Bytes {
		return errors.New("size does not match its reference")
	}
	hasher := sha256.New()
	size, err := io.Copy(hasher, &io.LimitedReader{R: file, N: info.Size()})
	if err != nil {
		return err
	}
	digest := record.SHA256("sha256:" + hex.EncodeToString(hasher.Sum(nil)))
	if uint64(size) != reference.Bytes || digest != reference.SHA256 {
		return errors.New("content does not match its reference")
	}
	return nil
}

func within(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
