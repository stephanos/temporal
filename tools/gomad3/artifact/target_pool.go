package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
)

// TargetSharing says how a published artifact holds its target.
type TargetSharing string

const (
	// TargetPrivate is an artifact of a store without a target pool: it owns
	// its copy of the target.
	TargetPrivate TargetSharing = "private"
	// TargetShared is an artifact whose target is a hard link to its store's
	// pool entry, the one copy every artifact of that target in the pool shares.
	TargetShared TargetSharing = "shared"
	// TargetUnshared is an artifact of a store with a target pool that could
	// not be linked to it, so sharing is off and it owns a private copy.
	TargetUnshared TargetSharing = "unshared"
)

// TargetPool is the pool of the directory that owns one: the artifacts root
// that holds every campaign, a corpus, or a minimizer output root. It is never
// below a store root, because a staged round's stores are renamed on commit.
func TargetPool(owner string) string {
	return filepath.Join(owner, "targets")
}

var linkFile = os.Link

// placeSharedPayload makes destination a hard link to the pool's copy of the
// payload, creating that copy when the pool does not hold it yet. Concurrent
// publishers of one payload end with one entry: the entry is published with a
// no-replace rename, and the loser links to the winner's.
func placeSharedPayload(ctx context.Context, pool string, payload Payload, destination string) (record.File, TargetSharing, error) {
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return record.File{}, "", fmt.Errorf("create payload parent %s: %w", payload.Path, err)
	}
	if err := os.Chmod(filepath.Dir(destination), 0o700); err != nil {
		return record.File{}, "", fmt.Errorf("make payload parent private %s: %w", payload.Path, err)
	}
	entry := filepath.Join(pool, poolEntryName(payload.SHA256))
	for created := false; ; created = true {
		if err := ctx.Err(); err != nil {
			return record.File{}, "", err
		}
		err := linkFile(entry, destination)
		switch {
		case err == nil:
			file, err := verifySharedPayload(ctx, destination, payload)
			if err != nil {
				return record.File{}, "", fmt.Errorf("shared target pool entry %s: %w", entry, err)
			}
			return file, TargetShared, nil
		case errors.Is(err, os.ErrNotExist) && !created:
			if err := createPoolEntry(ctx, pool, entry, payload); err != nil {
				return record.File{}, "", err
			}
		case hardLinksUnavailable(err):
			file, err := placePayload(ctx, payload, destination)
			return file, TargetUnshared, err
		default:
			return record.File{}, "", fmt.Errorf("link payload %s to the shared target pool: %w", payload.Path, err)
		}
	}
}

func poolEntryName(digest record.SHA256) string {
	return "sha256-" + strings.TrimPrefix(string(digest), "sha256:")
}

func hardLinksUnavailable(err error) bool {
	return errors.Is(err, errors.ErrUnsupported) || errors.Is(err, syscall.EXDEV) || errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EMLINK)
}

func createPoolEntry(ctx context.Context, pool, entry string, payload Payload) (retErr error) {
	if err := os.MkdirAll(pool, 0o700); err != nil {
		return fmt.Errorf("create shared target pool: %w", err)
	}
	if err := os.Chmod(pool, 0o700); err != nil {
		return fmt.Errorf("make shared target pool private: %w", err)
	}
	staging, err := os.MkdirTemp(pool, ".publish-")
	if err != nil {
		return fmt.Errorf("create shared target staging directory: %w", err)
	}
	defer func() {
		retErr = errors.Join(retErr, os.RemoveAll(staging))
	}()
	if err := os.Chmod(staging, 0o700); err != nil {
		return fmt.Errorf("make shared target staging directory private: %w", err)
	}
	staged := filepath.Join(staging, "target")
	file, err := placePayload(ctx, payload, staged)
	if err != nil {
		return err
	}
	if file.SHA256 != payload.SHA256 || file.Size != payload.Size {
		return fmt.Errorf("artifact payload %s identity changed during publication", payload.Path)
	}
	if err := renameNoReplace(staged, entry); err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		return fmt.Errorf("publish shared target: %w", err)
	}
	if err := syncDirectoryContext(ctx, pool); err != nil {
		return fmt.Errorf("sync shared target pool: %w", err)
	}
	return nil
}

func verifySharedPayload(ctx context.Context, path string, payload Payload) (record.File, error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return record.File{}, err
	}
	defer root.Close()
	file, info, err := openSharedFile(root, filepath.Base(path))
	if err != nil {
		return record.File{}, err
	}
	defer file.Close()
	if info.Mode().Perm() != payload.Mode || info.Size() < 0 || uint64(info.Size()) != uint64(payload.Size) {
		return record.File{}, errors.New("metadata does not match its identity")
	}
	hasher := sha256.New()
	size, err := copyWithContext(ctx, hasher, file)
	if err != nil {
		return record.File{}, err
	}
	digest := record.SHA256("sha256:" + hex.EncodeToString(hasher.Sum(nil)))
	if size != uint64(payload.Size) || digest != payload.SHA256 {
		return record.File{}, errors.New("content does not match its identity")
	}
	return record.File{Path: payload.Path, Mode: formatMode(payload.Mode), Size: record.Uint64String(size), SHA256: digest}, nil
}

// sharesPoolEntry reports whether the artifact's target is the pool's entry
// for it.
func sharesPoolEntry(pool, artifactPath string, manifest record.ExecutionRecord) bool {
	entry, err := os.Lstat(filepath.Join(pool, poolEntryName(manifest.Target.SHA256)))
	if err != nil {
		return false
	}
	target, err := os.Lstat(filepath.Join(artifactPath, filepath.FromSlash(manifest.Target.File)))
	return err == nil && os.SameFile(entry, target)
}

// openSharedFile opens a regular file that may have other hard links. It is
// hostfs.OpenRoot without the single-link check, for the one payload a store
// shares; the caller verifies the content against the manifest.
func openSharedFile(root *os.Root, path string) (*os.File, os.FileInfo, error) {
	info, err := root.Lstat(path)
	if err != nil {
		return nil, nil, err
	}
	name := filepath.Base(path)
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, nil, fmt.Errorf("%s is a %w", name, hostfs.ErrSymbolicLink)
	}
	if !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%s is not a regular file", name)
	}
	file, err := root.Open(path)
	if err != nil {
		return nil, nil, err
	}
	openedInfo, err := file.Stat()
	if err != nil || !os.SameFile(info, openedInfo) || openedInfo.Mode() != info.Mode() || openedInfo.Size() != info.Size() {
		return nil, nil, errors.Join(fmt.Errorf("%s changed while opening", name), err, file.Close())
	}
	return file, openedInfo, nil
}
