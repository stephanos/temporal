package artifactio

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
)

const immutableSetStagingPrefix = ".temporal-immutable-set-"

// ImmutableDirectory describes one manifest-addressed directory whose files
// become visible together. MemberPaths must validate the manifest before
// returning its exact member paths, and Validate must validate the complete
// byte snapshot without consulting the filesystem.
type ImmutableDirectory struct {
	ManifestPath     string
	MaximumFileBytes int64
	MemberPaths      func(manifest []byte) ([]string, error)
	Validate         func(files map[string][]byte) error
}

type immutablePublishHooks struct {
	beforeInstall func() error
}

// Publish installs one previously absent root/sets/<digest> directory with a
// single sibling rename. An existing byte-identical directory is revalidated
// and returned without replacement.
func (directory ImmutableDirectory) Publish(
	root string,
	digest string,
	files map[string][]byte,
) (string, error) {
	return publishImmutableDirectoryWithHooks(directory, root, digest, files, immutablePublishHooks{})
}

func publishImmutableDirectoryWithHooks(
	directory ImmutableDirectory,
	root string,
	digest string,
	files map[string][]byte,
	hooks immutablePublishHooks,
) (destination string, resultErr error) {
	files = cloneFileMap(files)
	paths, err := directory.validateFiles(digest, files)
	if err != nil {
		return "", err
	}
	resolvedRoot, err := resolveSetRoot(root)
	if err != nil {
		return "", err
	}
	setsRoot, err := ensureImmutableSetsRoot(resolvedRoot)
	if err != nil {
		return "", err
	}
	lock, err := acquireSetLock(resolvedRoot)
	if err != nil {
		return "", err
	}
	defer releaseSetLock(lock)
	if err := cleanImmutableStagingDirectories(setsRoot); err != nil {
		return "", fmt.Errorf("recover immutable publication: %w", err)
	}
	stageRoot, err := prepareImmutableCandidate(directory, setsRoot, digest, paths, files)
	if err != nil {
		return "", err
	}
	defer func() {
		resultErr = errors.Join(resultErr, os.RemoveAll(stageRoot))
	}()
	if hooks.beforeInstall != nil {
		if err := hooks.beforeInstall(); err != nil {
			return "", fmt.Errorf("install immutable directory: %w", err)
		}
	}

	destination = filepath.Join(setsRoot, digest)
	exists, err := revalidateExistingImmutableDirectory(directory, destination, files)
	if err != nil {
		return "", err
	}
	if exists {
		return destination, nil
	}
	if err := installImmutableCandidate(directory, setsRoot, stageRoot, destination, files); err != nil {
		return "", err
	}
	return destination, nil
}

func prepareImmutableCandidate(
	directory ImmutableDirectory,
	setsRoot string,
	digest string,
	paths []string,
	files map[string][]byte,
) (string, error) {
	stageRoot, err := os.MkdirTemp(setsRoot, immutableSetStagingPrefix+digest+"-")
	if err != nil {
		return "", fmt.Errorf("create immutable staging directory: %w", err)
	}
	if err := stageImmutableFiles(stageRoot, paths, files); err != nil {
		return "", errors.Join(err, os.RemoveAll(stageRoot))
	}
	staged, err := directory.read(stageRoot, digest, false)
	if err != nil {
		return "", errors.Join(fmt.Errorf("validate immutable candidate: %w", err), os.RemoveAll(stageRoot))
	}
	if !equalFileMaps(staged, files) {
		return "", errors.Join(
			errors.New("validate immutable candidate: staged bytes differ from publication bytes"),
			os.RemoveAll(stageRoot),
		)
	}
	return stageRoot, nil
}

func revalidateExistingImmutableDirectory(
	directory ImmutableDirectory,
	destination string,
	files map[string][]byte,
) (bool, error) {
	destinationInfo, err := os.Lstat(destination)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect immutable destination: %w", err)
	}
	if destinationInfo.Mode()&os.ModeSymlink != 0 || !destinationInfo.IsDir() {
		return false, fmt.Errorf("immutable destination %q is not a regular directory", destination)
	}
	existing, err := directory.Read(destination)
	if err != nil {
		return false, fmt.Errorf("revalidate immutable destination: %w", err)
	}
	if !equalFileMaps(existing, files) {
		return false, errors.New("immutable destination contains conflicting bytes")
	}
	return true, nil
}

func installImmutableCandidate(
	directory ImmutableDirectory,
	setsRoot string,
	stageRoot string,
	destination string,
	files map[string][]byte,
) error {
	if err := os.Rename(stageRoot, destination); err != nil {
		return fmt.Errorf("install immutable directory: %w", err)
	}
	if err := syncDirectory(setsRoot); err != nil {
		return err
	}
	installed, err := directory.Read(destination)
	if err == nil && equalFileMaps(installed, files) {
		return nil
	}
	if err == nil {
		err = errors.New("installed bytes differ from publication bytes")
	}
	removeErr := os.RemoveAll(destination)
	syncErr := syncDirectory(setsRoot)
	return errors.Join(fmt.Errorf("revalidate installed immutable directory: %w", err), removeErr, syncErr)
}

// Read opens an exact sets/<manifest-sha256> directory without following
// symlinks and returns a private byte snapshot only after full validation.
func (directory ImmutableDirectory) Read(destination string) (map[string][]byte, error) {
	digest, err := validateImmutableDestination(destination)
	if err != nil {
		return nil, err
	}
	return directory.read(destination, digest, true)
}

func (directory ImmutableDirectory) read(
	destination string,
	digest string,
	requireExactDestination bool,
) (map[string][]byte, error) {
	if requireExactDestination {
		if _, err := validateImmutableDestination(destination); err != nil {
			return nil, err
		}
		setsRoot, err := openDirectoryNoFollow(filepath.Dir(destination))
		if err != nil {
			return nil, fmt.Errorf("open immutable sets directory: %w", err)
		}
		permissionErr := requirePrivateDirectory(setsRoot, filepath.Dir(destination))
		closeErr := setsRoot.Close()
		if permissionErr != nil || closeErr != nil {
			return nil, errors.Join(permissionErr, closeErr)
		}
	}
	root, err := openDirectoryNoFollow(destination)
	if err != nil {
		return nil, fmt.Errorf("open immutable directory: %w", err)
	}
	defer func() { _ = root.Close() }()
	if err := requirePrivateDirectory(root, destination); err != nil {
		return nil, err
	}
	manifest, err := readRegularFileAt(root, directory.ManifestPath, directory.MaximumFileBytes)
	if err != nil {
		return nil, fmt.Errorf("read immutable manifest: %w", err)
	}
	manifestHash := sha256.Sum256(manifest)
	if hex.EncodeToString(manifestHash[:]) != digest {
		return nil, errors.New("immutable directory name does not match manifest SHA-256")
	}
	paths, err := directory.pathsFromManifest(manifest)
	if err != nil {
		return nil, err
	}
	if err := validateImmutableTree(root, paths); err != nil {
		return nil, err
	}
	files := make(map[string][]byte, len(paths))
	for _, path := range paths {
		encoded, readErr := readRegularFileAt(root, path, directory.MaximumFileBytes)
		if readErr != nil {
			return nil, fmt.Errorf("read immutable file %q: %w", path, readErr)
		}
		files[path] = encoded
	}
	reopenedManifest, err := readRegularFileAt(root, directory.ManifestPath, directory.MaximumFileBytes)
	if err != nil {
		return nil, fmt.Errorf("reopen immutable manifest: %w", err)
	}
	if !bytes.Equal(manifest, reopenedManifest) {
		return nil, errors.New("immutable manifest changed while it was read")
	}
	if directory.Validate != nil {
		if err := directory.Validate(cloneFileMap(files)); err != nil {
			return nil, fmt.Errorf("validate immutable directory: %w", err)
		}
	}
	return cloneFileMap(files), nil
}

func (directory ImmutableDirectory) validateFiles(digest string, files map[string][]byte) ([]string, error) {
	if !validLowerHexDigest(digest) {
		return nil, errors.New("immutable directory digest must be 64 lowercase hexadecimal digits")
	}
	manifest, exists := files[directory.ManifestPath]
	if !exists {
		return nil, fmt.Errorf("immutable file map is missing manifest %q", directory.ManifestPath)
	}
	hash := sha256.Sum256(manifest)
	if hex.EncodeToString(hash[:]) != digest {
		return nil, errors.New("immutable directory digest does not match manifest SHA-256")
	}
	paths, err := directory.pathsFromManifest(manifest)
	if err != nil {
		return nil, err
	}
	if len(files) != len(paths) {
		return nil, errors.New("immutable file map must contain exactly the manifest and its members")
	}
	for _, path := range paths {
		encoded, exists := files[path]
		if !exists {
			return nil, fmt.Errorf("immutable file map is missing %q", path)
		}
		if int64(len(encoded)) > directory.MaximumFileBytes {
			return nil, fmt.Errorf("immutable file %q exceeds the file byte limit", path)
		}
	}
	return paths, nil
}

func (directory ImmutableDirectory) pathsFromManifest(manifest []byte) ([]string, error) {
	if !safeRelativePath(directory.ManifestPath) || directory.MaximumFileBytes <= 0 || directory.MemberPaths == nil {
		return nil, errors.New("immutable directory requires a safe manifest path, positive file limit, and member resolver")
	}
	if int64(len(manifest)) > directory.MaximumFileBytes {
		return nil, errors.New("immutable manifest exceeds the file byte limit")
	}
	members, err := directory.MemberPaths(bytes.Clone(manifest))
	if err != nil {
		return nil, fmt.Errorf("resolve immutable members: %w", err)
	}
	paths := append([]string{directory.ManifestPath}, members...)
	checked := slices.Clone(paths)
	slices.Sort(checked)
	if slices.ContainsFunc(checked, func(path string) bool { return !safeRelativePath(path) }) {
		return nil, errors.New("immutable directory contains an unsafe path")
	}
	if hasDuplicate(checked) {
		return nil, errors.New("immutable directory contains duplicate paths")
	}
	for index, left := range checked {
		for _, right := range checked[index+1:] {
			if pathContains(left, right) {
				return nil, fmt.Errorf("immutable paths %q and %q overlap", left, right)
			}
		}
	}
	return paths, nil
}

func ensureImmutableSetsRoot(root string) (string, error) {
	setsRoot := filepath.Join(root, "sets")
	info, err := os.Lstat(setsRoot)
	switch {
	case errors.Is(err, os.ErrNotExist):
		if err := os.Mkdir(setsRoot, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("create immutable sets directory: %w", err)
		}
		if err := syncDirectory(root); err != nil {
			return "", err
		}
		info, err = os.Lstat(setsRoot)
	case err != nil:
		return "", fmt.Errorf("inspect immutable sets directory: %w", err)
	default:
	}
	if err != nil {
		return "", fmt.Errorf("inspect immutable sets directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", errors.New("immutable sets path is not a regular directory")
	}
	if info.Mode().Perm() != 0o700 {
		return "", fmt.Errorf("immutable sets directory permissions are %04o; expected 0700", info.Mode().Perm())
	}
	return setsRoot, nil
}

func stageImmutableFiles(root string, paths []string, files map[string][]byte) error {
	for _, path := range paths {
		absolute := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil {
			return fmt.Errorf("create immutable member directory: %w", err)
		}
		if err := Publish(absolute, files[path]); err != nil {
			return fmt.Errorf("stage immutable file %q: %w", path, err)
		}
	}
	directories := []string{root}
	seen := map[string]struct{}{root: {}}
	for _, path := range paths {
		for current := filepath.Dir(filepath.Join(root, filepath.FromSlash(path))); current != root; current = filepath.Dir(current) {
			if _, exists := seen[current]; exists {
				continue
			}
			seen[current] = struct{}{}
			directories = append(directories, current)
		}
	}
	slices.SortFunc(directories, func(left string, right string) int {
		return strings.Count(right, string(filepath.Separator)) - strings.Count(left, string(filepath.Separator))
	})
	for _, directory := range directories {
		if err := syncDirectory(directory); err != nil {
			return err
		}
	}
	return nil
}

func cleanImmutableStagingDirectories(setsRoot string) error {
	entries, err := os.ReadDir(setsRoot)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), immutableSetStagingPrefix) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			continue
		}
		if err := os.RemoveAll(filepath.Join(setsRoot, entry.Name())); err != nil {
			return err
		}
	}
	return syncDirectory(setsRoot)
}

func validateImmutableDestination(destination string) (string, error) {
	if destination == "" || filepath.Clean(destination) != destination {
		return "", errors.New("immutable destination must be an exact clean path")
	}
	if filepath.Base(filepath.Dir(destination)) != "sets" {
		return "", errors.New("immutable destination must be an exact sets/<digest> directory")
	}
	digest := filepath.Base(destination)
	if !validLowerHexDigest(digest) {
		return "", errors.New("immutable destination digest must be 64 lowercase hexadecimal digits")
	}
	return digest, nil
}

func validLowerHexDigest(digest string) bool {
	if len(digest) != sha256.Size*2 {
		return false
	}
	for _, character := range digest {
		if character < '0' || (character > '9' && character < 'a') || character > 'f' {
			return false
		}
	}
	return true
}

type immutableTree struct {
	directories map[string]*immutableTree
	files       map[string]struct{}
}

func validateImmutableTree(root *os.File, paths []string) error {
	tree := &immutableTree{directories: make(map[string]*immutableTree), files: make(map[string]struct{})}
	for _, path := range paths {
		components := strings.Split(filepath.ToSlash(path), "/")
		current := tree
		for _, component := range components[:len(components)-1] {
			next := current.directories[component]
			if next == nil {
				next = &immutableTree{directories: make(map[string]*immutableTree), files: make(map[string]struct{})}
				current.directories[component] = next
			}
			current = next
		}
		current.files[components[len(components)-1]] = struct{}{}
	}
	return validateImmutableTreeAt(root, tree, ".")
}

func validateImmutableTreeAt(directory *os.File, expected *immutableTree, relative string) error {
	duplicate, err := syscall.Dup(int(directory.Fd()))
	if err != nil {
		return fmt.Errorf("duplicate immutable directory descriptor: %w", err)
	}
	reader := os.NewFile(uintptr(duplicate), relative)
	entries, readErr := reader.ReadDir(-1)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		return fmt.Errorf("read immutable directory %q: %w", relative, errors.Join(readErr, closeErr))
	}
	if len(entries) != len(expected.directories)+len(expected.files) {
		return fmt.Errorf("immutable directory %q contains unexpected or missing entries", relative)
	}
	for _, entry := range entries {
		if err := validateImmutableEntry(directory, expected, relative, entry); err != nil {
			return err
		}
	}
	return nil
}

func validateImmutableEntry(
	directory *os.File,
	expected *immutableTree,
	relative string,
	entry os.DirEntry,
) error {
	path := filepath.Join(relative, entry.Name())
	if child, exists := expected.directories[entry.Name()]; exists {
		opened, err := openDirectoryAt(directory, entry.Name())
		if err != nil {
			return fmt.Errorf("open immutable directory %q: %w", path, err)
		}
		if err := requirePrivateDirectory(opened, path); err != nil {
			_ = opened.Close()
			return err
		}
		err = validateImmutableTreeAt(opened, child, path)
		return errors.Join(err, opened.Close())
	}
	if _, exists := expected.files[entry.Name()]; !exists {
		return fmt.Errorf("immutable directory %q contains unexpected entry %q", relative, entry.Name())
	}
	file, err := openRegularFileAt(directory, entry.Name())
	if err != nil {
		return fmt.Errorf("open immutable file %q: %w", path, err)
	}
	if err := requirePrivateRegularFile(file, path); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func openDirectoryNoFollow(path string) (*os.File, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	rootFD, err := syscall.Open(string(filepath.Separator), syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	current := os.NewFile(uintptr(rootFD), string(filepath.Separator))
	for _, component := range strings.Split(strings.TrimPrefix(absolute, string(filepath.Separator)), string(filepath.Separator)) {
		if component == "" {
			continue
		}
		next, openErr := openDirectoryAt(current, component)
		closeErr := current.Close()
		if openErr != nil || closeErr != nil {
			if next != nil {
				_ = next.Close()
			}
			return nil, errors.Join(openErr, closeErr)
		}
		current = next
	}
	return current, nil
}

func openDirectoryAt(parent *os.File, name string) (*os.File, error) {
	fd, err := syscall.Openat(int(parent.Fd()), name, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}

func openRegularFileAt(parent *os.File, name string) (*os.File, error) {
	fd, err := syscall.Openat(int(parent.Fd()), name, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}

func readRegularFileAt(root *os.File, path string, maximum int64) ([]byte, error) {
	components := strings.Split(filepath.ToSlash(path), "/")
	currentFD, err := syscall.Dup(int(root.Fd()))
	if err != nil {
		return nil, err
	}
	current := os.NewFile(uintptr(currentFD), ".")
	defer func() { _ = current.Close() }()
	for _, component := range components[:len(components)-1] {
		next, openErr := openDirectoryAt(current, component)
		if openErr != nil {
			return nil, openErr
		}
		if err := requirePrivateDirectory(next, component); err != nil {
			_ = next.Close()
			return nil, err
		}
		_ = current.Close()
		current = next
	}
	file, err := openRegularFileAt(current, components[len(components)-1])
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	if err := requirePrivateRegularFile(file, path); err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() < 0 || info.Size() > maximum {
		return nil, fmt.Errorf("immutable file %q exceeds the file byte limit", path)
	}
	encoded, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(encoded)) > maximum {
		return nil, fmt.Errorf("immutable file %q exceeds the file byte limit", path)
	}
	if int64(len(encoded)) != info.Size() {
		return nil, fmt.Errorf("immutable file %q changed while it was read", path)
	}
	return encoded, nil
}

func requirePrivateDirectory(directory *os.File, path string) error {
	info, err := directory.Stat()
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("immutable path %q is not a directory", path)
	}
	if info.Mode().Perm() != 0o700 {
		return fmt.Errorf("immutable directory %q permissions are %04o; expected 0700", path, info.Mode().Perm())
	}
	return nil
}

func requirePrivateRegularFile(file *os.File, path string) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("immutable path %q is not a regular file", path)
	}
	if info.Mode().Perm() != 0o600 {
		return fmt.Errorf("immutable file %q permissions are %04o; expected 0600", path, info.Mode().Perm())
	}
	return nil
}

func equalFileMaps(left map[string][]byte, right map[string][]byte) bool {
	if len(left) != len(right) {
		return false
	}
	for path, encoded := range left {
		rightEncoded, exists := right[path]
		if !exists || !bytes.Equal(encoded, rightEncoded) {
			return false
		}
	}
	return true
}

func cloneFileMap(files map[string][]byte) map[string][]byte {
	cloned := make(map[string][]byte, len(files))
	for path, encoded := range files {
		cloned[path] = bytes.Clone(encoded)
	}
	return cloned
}
