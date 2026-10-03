package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
)

const maximumManifestBytes = 16 << 20

// Opened is an artifact whose directory is pinned open for validated payload
// access. The caller that opened it owns it and must Close it. Its manifest is
// private: Manifest and Snapshot return copies, so no caller can change what
// payload access validates against. Path, Manifest, StoredBytes and Snapshot
// stay available after Close; payload access fails.
type Opened struct {
	path        string
	manifest    record.ExecutionRecord
	storedBytes uint64
	root        *os.Root
}

// OpenArtifact validates the artifact directory at path against its manifest
// and returns a handle pinned to that directory.
func OpenArtifact(path string) (*Opened, error) {
	rootInfo, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("open artifact directory: %w", err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("artifact path is not a directory")
	}
	if rootInfo.Mode().Perm() != 0o700 {
		return nil, fmt.Errorf("artifact directory mode is %#o, want 0700", rootInfo.Mode().Perm())
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, fmt.Errorf("pin artifact directory: %w", err)
	}
	pinnedInfo, err := root.Stat(".")
	if err != nil || !os.SameFile(rootInfo, pinnedInfo) {
		return nil, errors.Join(fmt.Errorf("artifact directory changed while opening"), err, root.Close())
	}
	manifestBytes, err := readValidatedFile(root, "manifest.json", 0o600, maximumManifestBytes)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("read artifact manifest: %w", err), root.Close())
	}
	manifest, err := record.DecodeExecutionRecord(manifestBytes)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("decode artifact manifest: %w", err), root.Close())
	}
	expected := map[string]record.File{}
	for _, file := range manifest.Files {
		if file.Path == "manifest.json" {
			return nil, errors.Join(fmt.Errorf("manifest cannot list itself"), root.Close())
		}
		expected[filepath.FromSlash(file.Path)] = file
	}
	seen := map[string]bool{}
	err = validateDirectory(root, ".", expected, seen, manifest.Target.File)
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	for file := range expected {
		if !seen[file] {
			return nil, errors.Join(fmt.Errorf("artifact is missing listed file %s", filepath.ToSlash(file)), root.Close())
		}
	}
	storedBytes, err := artifactStoredBytes(manifest, uint64(len(manifestBytes)))
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	return &Opened{path: path, manifest: manifest, storedBytes: storedBytes, root: root}, nil
}

// Close releases the pinned directory. Closing again, or closing a nil handle,
// does nothing.
func (opened *Opened) Close() error {
	if opened == nil || opened.root == nil {
		return nil
	}
	err := opened.root.Close()
	opened.root = nil
	return err
}

// Path is the directory the artifact was opened at.
func (opened *Opened) Path() string {
	return opened.path
}

// Manifest returns a copy of the validated manifest that shares no memory with
// the handle.
func (opened *Opened) Manifest() record.ExecutionRecord {
	return cloneManifest(opened.manifest)
}

// StoredBytes counts the manifest and every listed file (Artifact.StoredBytes).
func (opened *Opened) StoredBytes() uint64 {
	return opened.storedBytes
}

// Snapshot returns a detached reference to the opened artifact. It holds no
// open resource and its manifest is a copy, so changing it cannot change the
// handle. Its TargetSharing is empty, as only publication sets it.
func (opened *Opened) Snapshot() Artifact {
	return Artifact{Path: opened.path, Manifest: cloneManifest(opened.manifest), StoredBytes: opened.storedBytes}
}

// OpenPayload opens a listed payload of at most maximum bytes after checking
// its mode, size and SHA-256 against the manifest.
func (opened *Opened) OpenPayload(relativePath string, maximum uint64) (*os.File, error) {
	if opened == nil || opened.root == nil {
		return nil, fmt.Errorf("artifact is not open")
	}
	expected := listedFile(opened, relativePath)
	if expected == nil {
		return nil, fmt.Errorf("artifact payload %q is not listed", relativePath)
	}
	if uint64(expected.Size) > maximum {
		return nil, fmt.Errorf("artifact payload %q exceeds its bound", relativePath)
	}
	mode := os.FileMode(0o600)
	if expected.Mode == "0700" {
		mode = 0o700
	}
	file, info, err := openValidatedFile(opened.root, relativePath, mode, uint64(expected.Size), relativePath == opened.manifest.Target.File)
	if err != nil {
		return nil, err
	}
	hasher := sha256.New()
	reader := &io.LimitedReader{R: file, N: info.Size()}
	size, err := io.Copy(hasher, reader)
	if err != nil {
		return nil, errors.Join(err, file.Close())
	}
	var extra [1]byte
	if count, readErr := file.Read(extra[:]); count != 0 || readErr != io.EOF {
		return nil, errors.Join(fmt.Errorf("artifact payload %q changed size while hashing", relativePath), file.Close())
	}
	digest := record.SHA256("sha256:" + hex.EncodeToString(hasher.Sum(nil)))
	if uint64(size) != uint64(expected.Size) || digest != expected.SHA256 {
		return nil, errors.Join(fmt.Errorf("artifact payload %q identity mismatch", relativePath), file.Close())
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return file, nil
}

func validateDirectory(root *os.Root, directory string, expected map[string]record.File, seen map[string]bool, target string) error {
	opened, err := root.Open(directory)
	if err != nil {
		return err
	}
	entries, readErr := opened.ReadDir(-1)
	closeErr := opened.Close()
	if readErr != nil || closeErr != nil {
		return errors.Join(readErr, closeErr)
	}
	for _, entry := range entries {
		relative := entry.Name()
		if directory != "." {
			relative = filepath.ToSlash(filepath.Join(directory, relative))
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if relative != "manifest.json" {
				if _, listed := expected[filepath.FromSlash(relative)]; !listed {
					return fmt.Errorf("artifact contains unlisted file %s", relative)
				}
			}
			return fmt.Errorf("artifact entry %s is a symbolic link", relative)
		}
		if entry.IsDir() {
			if !listedDirectory(relative, expected) {
				return fmt.Errorf("artifact contains unlisted directory %s", relative)
			}
			info, infoErr := root.Lstat(relative)
			if infoErr != nil || info.Mode().Perm() != 0o700 {
				return errors.Join(fmt.Errorf("artifact directory %s is not private", relative), infoErr)
			}
			if err := validateDirectory(root, relative, expected, seen, target); err != nil {
				return err
			}
			continue
		}
		if relative == "manifest.json" {
			continue
		}
		file, listed := expected[filepath.FromSlash(relative)]
		if !listed {
			return fmt.Errorf("artifact contains unlisted file %s", relative)
		}
		mode := os.FileMode(0o600)
		if file.Mode == "0700" {
			mode = 0o700
		}
		digest, size, readErr := hashValidatedFile(root, relative, mode, uint64(file.Size), file.Path == target)
		if readErr != nil {
			return fmt.Errorf("validate artifact file %s: %w", file.Path, readErr)
		}
		if size != uint64(file.Size) || digest != file.SHA256 {
			return fmt.Errorf("artifact file %s identity mismatch", file.Path)
		}
		seen[filepath.FromSlash(relative)] = true
	}
	return nil
}

func listedDirectory(directory string, expected map[string]record.File) bool {
	prefix := filepath.FromSlash(directory) + string(filepath.Separator)
	for path := range expected {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// ReadPayload reads a listed payload of at most maximum bytes and checks it
// against the manifest.
func (opened *Opened) ReadPayload(relativePath string, maximum uint64) ([]byte, error) {
	if opened == nil {
		return nil, fmt.Errorf("artifact is not open")
	}
	expected := listedFile(opened, relativePath)
	if expected == nil {
		return nil, fmt.Errorf("artifact payload %q is not listed", relativePath)
	}
	if uint64(expected.Size) > maximum {
		return nil, fmt.Errorf("artifact payload %q exceeds its bound", relativePath)
	}
	file, err := opened.OpenPayload(relativePath, maximum)
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return nil, errors.Join(readErr, closeErr)
	}
	if uint64(len(data)) != uint64(expected.Size) || record.HashBytes(data) != expected.SHA256 {
		return nil, fmt.Errorf("artifact payload %q identity mismatch", relativePath)
	}
	return data, nil
}

// CopyPayload copies a listed payload to a new file at destination and checks
// the copy against the manifest.
func (opened *Opened) CopyPayload(relativePath, destination string, destinationMode os.FileMode) error {
	if opened == nil || opened.root == nil {
		return fmt.Errorf("artifact is not open")
	}
	expected := listedFile(opened, relativePath)
	if expected == nil {
		return fmt.Errorf("artifact payload %q is not listed", relativePath)
	}
	source, err := opened.OpenPayload(relativePath, uint64(expected.Size))
	if err != nil {
		return err
	}
	defer source.Close()
	destinationFile, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, destinationMode)
	if err != nil {
		return err
	}
	if err := destinationFile.Chmod(destinationMode); err != nil {
		destinationFile.Close()
		return err
	}
	hasher := sha256.New()
	reader := &io.LimitedReader{R: source, N: int64(expected.Size)}
	written, copyErr := io.Copy(io.MultiWriter(destinationFile, hasher), reader)
	if copyErr != nil {
		destinationFile.Close()
		return copyErr
	}
	var extra [1]byte
	if count, readErr := source.Read(extra[:]); count != 0 || readErr != io.EOF {
		destinationFile.Close()
		return fmt.Errorf("artifact payload %q changed size while copying", relativePath)
	}
	digest := record.SHA256("sha256:" + hex.EncodeToString(hasher.Sum(nil)))
	if uint64(written) != uint64(expected.Size) || digest != expected.SHA256 {
		destinationFile.Close()
		return fmt.Errorf("artifact payload %q identity mismatch", relativePath)
	}
	if err := destinationFile.Sync(); err != nil {
		destinationFile.Close()
		return err
	}
	return destinationFile.Close()
}

func listedFile(opened *Opened, relativePath string) *record.File {
	if opened == nil {
		return nil
	}
	for index := range opened.manifest.Files {
		if opened.manifest.Files[index].Path == relativePath {
			return &opened.manifest.Files[index]
		}
	}
	return nil
}

func readValidatedFile(root *os.Root, path string, mode os.FileMode, maximum uint64) ([]byte, error) {
	file, info, err := hostfs.OpenRoot(root, path)
	if err != nil {
		return nil, err
	}
	if info.Mode().Perm() != mode {
		return nil, errors.Join(fmt.Errorf("%s mode is %#o, want %#o", filepath.Base(path), info.Mode().Perm(), mode), file.Close())
	}
	if info.Size() < 0 || uint64(info.Size()) > maximum {
		return nil, errors.Join(fmt.Errorf("%s size exceeds its bound", filepath.Base(path)), file.Close())
	}
	reader := &io.LimitedReader{R: file, N: int64(maximum)}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.Join(err, file.Close())
	}
	if reader.N == 0 {
		var extra [1]byte
		if count, readErr := file.Read(extra[:]); count != 0 || readErr != io.EOF {
			return nil, errors.Join(fmt.Errorf("%s exceeds its bound", filepath.Base(path)), file.Close())
		}
	}
	return data, file.Close()
}

func hashValidatedFile(root *os.Root, path string, mode os.FileMode, expectedSize uint64, shared bool) (record.SHA256, uint64, error) {
	file, info, err := openValidatedFile(root, path, mode, expectedSize, shared)
	if err != nil {
		return "", 0, err
	}
	hasher := sha256.New()
	reader := &io.LimitedReader{R: file, N: int64(info.Size())}
	size, err := io.Copy(hasher, reader)
	if err != nil {
		return "", 0, errors.Join(err, file.Close())
	}
	var extra [1]byte
	if count, readErr := file.Read(extra[:]); count != 0 || readErr != io.EOF {
		return "", 0, errors.Join(fmt.Errorf("%s changed size while hashing", filepath.Base(path)), file.Close())
	}
	return record.SHA256("sha256:" + hex.EncodeToString(hasher.Sum(nil))), uint64(size), file.Close()
}

// openValidatedFile opens a listed payload. Only the target may have other
// hard links, because a store shares it with its pool; every other payload
// keeps the single-link check.
func openValidatedFile(root *os.Root, path string, mode os.FileMode, expectedSize uint64, shared bool) (*os.File, os.FileInfo, error) {
	open := hostfs.OpenRoot
	if shared {
		open = openSharedFile
	}
	file, info, err := open(root, path)
	if err != nil {
		return nil, nil, err
	}
	if info.Mode().Perm() != mode || info.Size() < 0 || uint64(info.Size()) != expectedSize {
		return nil, nil, errors.Join(fmt.Errorf("%s metadata does not match its manifest", filepath.Base(path)), file.Close())
	}
	return file, info, nil
}
