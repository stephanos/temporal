package artifactio

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
)

// Set describes one complete collection of generated files and the roots it
// exclusively manages. Paths are relative to the publication root.
type Set struct {
	// Roots are replaced as complete files or directory trees.
	Roots []string
	// Paths are the exact artifact files required in each publication.
	Paths []string
}

// Publish validates and stages a complete artifact set before replacing any
// managed root. validate may elaborate or otherwise inspect the staged tree.
func (set Set) Publish(
	root string,
	artifacts map[string][]byte,
	validate func(candidateRoot string) error,
) error {
	return publishSetWithHooks(set, root, artifacts, validate, publishHooks{})
}

type (
	publishHooks struct {
		beforeInstall func(index int, root string) error
	}
	publicationManifest struct {
		Roots []publicationRoot `json:"roots"`
	}
	publicationOwner struct {
		Identity string   `json:"identity"`
		Roots    []string `json:"roots"`
	}
	publicationRoot struct {
		Path    string `json:"path"`
		Existed bool   `json:"existed"`
	}
)

var errSimulatedInterruption = errors.New("simulated publication interruption")

func publishSetWithHooks(
	set Set,
	root string,
	artifacts map[string][]byte,
	validate func(candidateRoot string) error,
	hooks publishHooks,
) error {
	roots, paths, err := validateSet(set, artifacts)
	if err != nil {
		return err
	}
	resolvedRoot, err := resolveSetRoot(root)
	if err != nil {
		return err
	}
	checkedPaths := append(slices.Clone(roots), paths...)
	if err := rejectSymlinkedPaths(resolvedRoot, checkedPaths); err != nil {
		return err
	}

	identity := setIdentity(resolvedRoot, roots, paths)
	lock, err := acquireSetLock(resolvedRoot)
	if err != nil {
		return err
	}
	defer releaseSetLock(lock)

	if err := recoverInterruptedTransactions(resolvedRoot, identity, roots); err != nil {
		return fmt.Errorf("recover interrupted publication: %w", err)
	}
	if err := rejectSymlinkedPaths(resolvedRoot, checkedPaths); err != nil {
		return err
	}
	transactionRoot, stageRoot, manifest, err := prepareCandidate(
		resolvedRoot,
		identity,
		roots,
		paths,
		checkedPaths,
		artifacts,
		validate,
	)
	if err != nil {
		return err
	}
	return installCandidate(resolvedRoot, transactionRoot, stageRoot, identity, roots, manifest, hooks)
}

func prepareCandidate(
	root string,
	identity string,
	roots []string,
	paths []string,
	checkedPaths []string,
	artifacts map[string][]byte,
	validate func(candidateRoot string) error,
) (transactionRoot string, stageRoot string, manifest publicationManifest, resultErr error) {
	transactionRoot, err := createTransaction(root, identity, roots)
	if err != nil {
		return "", "", publicationManifest{}, err
	}
	stageRoot = filepath.Join(transactionRoot, "stage")
	if err := stageCandidate(root, transactionRoot, stageRoot, paths, artifacts, validate); err != nil {
		return "", "", publicationManifest{}, err
	}
	if err := rejectSymlinkedPaths(root, checkedPaths); err != nil {
		_ = os.RemoveAll(transactionRoot)
		return "", "", publicationManifest{}, err
	}
	manifest, err = inspectPublicationRoots(root, roots)
	if err != nil {
		_ = os.RemoveAll(transactionRoot)
		return "", "", publicationManifest{}, err
	}
	if err := writePublicationManifest(transactionRoot, manifest); err != nil {
		_ = os.RemoveAll(transactionRoot)
		return "", "", publicationManifest{}, err
	}
	return transactionRoot, stageRoot, manifest, nil
}

func stageCandidate(
	root string,
	transactionRoot string,
	stageRoot string,
	paths []string,
	artifacts map[string][]byte,
	validate func(candidateRoot string) error,
) error {
	if err := os.MkdirAll(stageRoot, 0o700); err != nil {
		return fmt.Errorf("create candidate staging directory: %w", err)
	}
	if err := syncDirectory(root); err != nil {
		_ = os.RemoveAll(transactionRoot)
		return err
	}
	for _, path := range paths {
		if err := Publish(filepath.Join(stageRoot, filepath.FromSlash(path)), artifacts[path]); err != nil {
			_ = os.RemoveAll(transactionRoot)
			return fmt.Errorf("stage artifact %q: %w", path, err)
		}
	}
	if validate != nil {
		if err := validate(stageRoot); err != nil {
			_ = os.RemoveAll(transactionRoot)
			return fmt.Errorf("validate candidate: %w", err)
		}
	}
	return nil
}

func writePublicationManifest(transactionRoot string, manifest publicationManifest) error {
	encodedManifest, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("encode publication recovery marker: %w", err)
	}
	if err := Publish(filepath.Join(transactionRoot, "manifest.json"), encodedManifest); err != nil {
		return fmt.Errorf("write publication recovery marker: %w", err)
	}
	return nil
}

func installCandidate(
	root string,
	transactionRoot string,
	stageRoot string,
	identity string,
	roots []string,
	manifest publicationManifest,
	hooks publishHooks,
) error {
	backupRoot := filepath.Join(transactionRoot, "backup")
	if err := backupManagedRoots(root, backupRoot, manifest); err != nil {
		return rollbackHandledFailure(root, transactionRoot, identity, roots, fmt.Errorf("backup managed roots: %w", err))
	}
	for index, managedRoot := range roots {
		if err := installManagedRoot(root, stageRoot, index, managedRoot, hooks); err != nil {
			if errors.Is(err, errSimulatedInterruption) {
				return err
			}
			return rollbackHandledFailure(root, transactionRoot, identity, roots, err)
		}
	}
	if err := os.RemoveAll(transactionRoot); err != nil {
		return fmt.Errorf("clean publication transaction: %w", err)
	}
	if err := syncDirectory(root); err != nil {
		return fmt.Errorf("sync publication root: %w", err)
	}
	return nil
}

func installManagedRoot(
	root string,
	stageRoot string,
	index int,
	managedRoot string,
	hooks publishHooks,
) error {
	if hooks.beforeInstall != nil {
		if err := hooks.beforeInstall(index, managedRoot); err != nil {
			return fmt.Errorf("install managed root %q: %w", managedRoot, err)
		}
	}
	staged := filepath.Join(stageRoot, filepath.FromSlash(managedRoot))
	destination := filepath.Join(root, filepath.FromSlash(managedRoot))
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return fmt.Errorf("create managed parent %q: %w", managedRoot, err)
	}
	if err := os.Rename(staged, destination); err != nil {
		return fmt.Errorf("install managed root %q: %w", managedRoot, err)
	}
	if err := syncDirectory(filepath.Dir(destination)); err != nil {
		return fmt.Errorf("sync managed root %q: %w", managedRoot, err)
	}
	return nil
}

func validateSet(
	set Set,
	artifacts map[string][]byte,
) (roots []string, paths []string, resultErr error) {
	if len(set.Roots) == 0 || len(set.Paths) == 0 {
		return nil, nil, errors.New("artifact set roots and paths are required")
	}
	roots = slices.Clone(set.Roots)
	paths = slices.Clone(set.Paths)
	slices.Sort(roots)
	slices.Sort(paths)
	if slices.ContainsFunc(roots, func(path string) bool { return !safeRelativePath(path) }) ||
		slices.ContainsFunc(paths, func(path string) bool { return !safeRelativePath(path) }) {
		return nil, nil, errors.New("artifact set contains an unsafe managed path")
	}
	if hasDuplicate(roots) || hasDuplicate(paths) {
		return nil, nil, errors.New("artifact set contains duplicate managed paths")
	}
	if err := validateManagedPaths(roots, paths); err != nil {
		return nil, nil, err
	}
	if err := validateArtifactMap(paths, artifacts); err != nil {
		return nil, nil, err
	}
	return roots, paths, nil
}

func validateManagedPaths(roots []string, paths []string) error {
	for index, left := range roots {
		for _, right := range roots[index+1:] {
			if pathContains(left, right) || pathContains(right, left) {
				return fmt.Errorf("managed roots %q and %q overlap", left, right)
			}
		}
	}
	rootUse := make(map[string]bool, len(roots))
	for _, path := range paths {
		matched := ""
		for _, root := range roots {
			if path == root || pathContains(root, path) {
				matched = root
				break
			}
		}
		if matched == "" {
			return fmt.Errorf("managed artifact %q is outside every managed root", path)
		}
		rootUse[matched] = true
	}
	for _, root := range roots {
		if !rootUse[root] {
			return fmt.Errorf("managed root %q contains no artifact", root)
		}
	}
	return nil
}

func validateArtifactMap(paths []string, artifacts map[string][]byte) error {
	if len(artifacts) != len(paths) {
		return errors.New("artifact map must contain exactly the managed paths")
	}
	for _, path := range paths {
		if _, exists := artifacts[path]; !exists {
			return fmt.Errorf("artifact map must contain exactly the managed paths: missing %q", path)
		}
	}
	return nil
}

func resolveSetRoot(root string) (string, error) {
	if root == "" {
		return "", errors.New("artifact set root is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve artifact set root: %w", err)
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", fmt.Errorf("inspect artifact set root: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("artifact set root %q is a symlink", root)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("artifact set root %q is not a directory", root)
	}
	for component := absolute; ; component = filepath.Dir(component) {
		info, err := os.Lstat(component)
		if err != nil {
			return "", fmt.Errorf("inspect artifact set root component %q: %w", component, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("artifact set root %q contains symlink %q", root, component)
		}
		if parent := filepath.Dir(component); parent == component {
			break
		}
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve artifact set root %q: %w", root, err)
	}
	return resolved, nil
}

func rejectSymlinkedPaths(root string, roots []string) error {
	for _, managedRoot := range roots {
		current := root
		for _, component := range strings.Split(filepath.FromSlash(managedRoot), string(filepath.Separator)) {
			current = filepath.Join(current, component)
			info, err := os.Lstat(current)
			if errors.Is(err, os.ErrNotExist) {
				break
			}
			if err != nil {
				return fmt.Errorf("inspect managed path %q: %w", managedRoot, err)
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("managed path %q contains symlink %q", managedRoot, current)
			}
		}
	}
	return nil
}

func acquireSetLock(root string) (*os.File, error) {
	lock, err := os.Open(root)
	if err != nil {
		return nil, fmt.Errorf("open artifact set lock: %w", err)
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = lock.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, errors.New("artifact set has a concurrent writer")
		}
		return nil, fmt.Errorf("lock artifact set: %w", err)
	}
	return lock, nil
}

func releaseSetLock(lock *os.File) {
	_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	_ = lock.Close()
}

func inspectPublicationRoots(root string, roots []string) (publicationManifest, error) {
	manifest := publicationManifest{Roots: make([]publicationRoot, 0, len(roots))}
	for _, managedRoot := range roots {
		_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(managedRoot)))
		switch {
		case err == nil:
			manifest.Roots = append(manifest.Roots, publicationRoot{Path: managedRoot, Existed: true})
		case errors.Is(err, os.ErrNotExist):
			manifest.Roots = append(manifest.Roots, publicationRoot{Path: managedRoot})
		default:
			return publicationManifest{}, fmt.Errorf("inspect managed root %q: %w", managedRoot, err)
		}
	}
	return manifest, nil
}

func backupManagedRoots(root string, backupRoot string, manifest publicationManifest) error {
	for _, managedRoot := range manifest.Roots {
		if !managedRoot.Existed {
			continue
		}
		source := filepath.Join(root, filepath.FromSlash(managedRoot.Path))
		backup := filepath.Join(backupRoot, filepath.FromSlash(managedRoot.Path))
		if err := os.MkdirAll(filepath.Dir(backup), 0o700); err != nil {
			return err
		}
		if err := os.Rename(source, backup); err != nil {
			return err
		}
		if err := syncDirectory(filepath.Dir(source)); err != nil {
			return err
		}
		if err := syncDirectory(filepath.Dir(backup)); err != nil {
			return err
		}
	}
	return nil
}

func rollbackHandledFailure(
	root string,
	transactionRoot string,
	identity string,
	roots []string,
	operationErr error,
) error {
	if recoveryErr := recoverTransaction(root, transactionRoot, identity, roots); recoveryErr != nil {
		return errors.Join(operationErr, fmt.Errorf("rollback publication: %w", recoveryErr))
	}
	return operationErr
}

func recoverInterruptedTransactions(root string, identity string, roots []string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	prefix := ".temporal-artifact-set-" + identity + "-"
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		transactionRoot := filepath.Join(root, entry.Name())
		owned, err := transactionIsOwned(transactionRoot, identity, roots)
		if err != nil {
			return err
		}
		if !owned {
			continue
		}
		if err := recoverTransaction(root, transactionRoot, identity, roots); err != nil {
			return err
		}
	}
	return nil
}

func createTransaction(root string, identity string, roots []string) (string, error) {
	transactionRoot, err := os.MkdirTemp(root, ".temporal-artifact-set-"+identity+"-")
	if err != nil {
		return "", fmt.Errorf("create publication transaction: %w", err)
	}
	owner := publicationOwner{Identity: identity, Roots: roots}
	encoded, err := json.Marshal(owner)
	if err != nil {
		_ = os.RemoveAll(transactionRoot)
		return "", fmt.Errorf("encode publication owner marker: %w", err)
	}
	if err := Publish(filepath.Join(transactionRoot, "owner.json"), encoded); err != nil {
		_ = os.RemoveAll(transactionRoot)
		return "", fmt.Errorf("write publication owner marker: %w", err)
	}
	if err := syncDirectory(root); err != nil {
		_ = os.RemoveAll(transactionRoot)
		return "", err
	}
	return transactionRoot, nil
}

func transactionIsOwned(transactionRoot string, identity string, roots []string) (bool, error) {
	info, err := os.Lstat(transactionRoot)
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return false, nil
	}
	encoded, err := os.ReadFile(filepath.Join(transactionRoot, "owner.json"))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var owner publicationOwner
	if err := json.Unmarshal(encoded, &owner); err != nil {
		return false, fmt.Errorf("decode publication owner marker: %w", err)
	}
	return owner.Identity == identity && slices.Equal(owner.Roots, roots), nil
}

func recoverTransaction(root string, transactionRoot string, identity string, roots []string) error {
	owned, err := transactionIsOwned(transactionRoot, identity, roots)
	if err != nil {
		return err
	}
	if !owned {
		return fmt.Errorf("publication transaction %q has no matching owner marker", transactionRoot)
	}
	manifest, exists, err := readPublicationManifest(transactionRoot, roots)
	if err != nil {
		return err
	}
	if !exists {
		return os.RemoveAll(transactionRoot)
	}
	if err := restoreManagedRoots(root, filepath.Join(transactionRoot, "backup"), manifest); err != nil {
		return err
	}
	if err := os.RemoveAll(transactionRoot); err != nil {
		return err
	}
	return syncDirectory(root)
}

func readPublicationManifest(
	transactionRoot string,
	roots []string,
) (publicationManifest, bool, error) {
	encoded, err := os.ReadFile(filepath.Join(transactionRoot, "manifest.json"))
	if errors.Is(err, os.ErrNotExist) {
		return publicationManifest{}, false, nil
	}
	if err != nil {
		return publicationManifest{}, false, err
	}
	var manifest publicationManifest
	if err := json.Unmarshal(encoded, &manifest); err != nil {
		return publicationManifest{}, false, fmt.Errorf("decode recovery marker: %w", err)
	}
	manifestRoots := make([]string, len(manifest.Roots))
	for index, managedRoot := range manifest.Roots {
		manifestRoots[index] = managedRoot.Path
	}
	if !slices.Equal(manifestRoots, roots) {
		return publicationManifest{}, false, errors.New("recovery marker does not match managed roots")
	}
	return manifest, true, nil
}

func restoreManagedRoots(root string, backupRoot string, manifest publicationManifest) error {
	for _, managedRoot := range manifest.Roots {
		destination := filepath.Join(root, filepath.FromSlash(managedRoot.Path))
		backup := filepath.Join(backupRoot, filepath.FromSlash(managedRoot.Path))
		_, backupErr := os.Lstat(backup)
		switch {
		case backupErr == nil:
			if err := os.RemoveAll(destination); err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
				return err
			}
			if err := os.Rename(backup, destination); err != nil {
				return err
			}
		case errors.Is(backupErr, os.ErrNotExist):
			if !managedRoot.Existed {
				if err := os.RemoveAll(destination); err != nil {
					return err
				}
			}
		default:
			return backupErr
		}
	}
	return nil
}

func setIdentity(root string, roots []string, paths []string) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(root))
	for _, collection := range [][]string{roots, paths} {
		for _, path := range collection {
			_, _ = hash.Write([]byte{0})
			_, _ = hash.Write([]byte(path))
		}
	}
	return hex.EncodeToString(hash.Sum(nil))[:16]
}

func safeRelativePath(path string) bool {
	if path == "" || filepath.IsAbs(filepath.FromSlash(path)) {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	return clean == path && path != ".." && !strings.HasPrefix(path, "../")
}

func pathContains(root string, path string) bool {
	return strings.HasPrefix(path, root+"/")
}

func hasDuplicate(paths []string) bool {
	for index := 1; index < len(paths); index++ {
		if paths[index-1] == paths[index] {
			return true
		}
	}
	return false
}
