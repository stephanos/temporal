package workspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Options struct {
	MaxBytes int64
	MaxFiles int
	Exclude  []string
}

type Prepared struct {
	Source    string
	Base      string
	Candidate string
	Digest    string
	Files     int
	Bytes     int64
	options   Options
}

type Change struct {
	Path   string
	Kind   string
	Bytes  int64
	Digest string
}

type entry struct {
	kind   string
	mode   fs.FileMode
	size   int64
	digest string
	target string
}

type inventory struct {
	digest  string
	entries map[string]entry
	files   int
	bytes   int64
}

func Reopen(source, runDirectory, digest string, options Options) (Prepared, error) {
	sourceRoot, err := safeRoot(source, "source")
	if err != nil {
		return Prepared{}, err
	}
	runRoot, err := safeRoot(runDirectory, "run directory")
	if err != nil {
		return Prepared{}, err
	}
	options.Exclude, err = normalizeExcludes(options.Exclude)
	if err != nil {
		return Prepared{}, err
	}
	options.Exclude = append(options.Exclude, ".git")
	if relative, relErr := filepath.Rel(sourceRoot, runRoot); relErr == nil && containedRelative(relative) && relative != "." {
		options.Exclude = append(options.Exclude, filepath.ToSlash(relative))
	}
	options.Exclude = compactStrings(options.Exclude)
	prepared := Prepared{
		Source: sourceRoot, Base: filepath.Join(runRoot, "workspaces", "base"),
		Candidate: filepath.Join(runRoot, "workspaces", "candidate"), Digest: digest, options: options,
	}
	for _, path := range []string{prepared.Base, prepared.Candidate} {
		info, statErr := os.Stat(path)
		if statErr != nil || !info.IsDir() {
			return Prepared{}, errors.Join(fmt.Errorf("workspace %q is not a directory", path), statErr)
		}
	}
	base, err := scan(context.Background(), prepared.Base, options)
	if err != nil {
		return Prepared{}, err
	}
	if base.digest != digest {
		return Prepared{}, errors.New("workspace base digest does not match admitted source")
	}
	prepared.Files = base.files
	prepared.Bytes = base.bytes
	return prepared, nil
}

func Prepare(ctx context.Context, source, runDirectory string, options Options) (Prepared, error) {
	sourceRoot, err := safeRoot(source, "source")
	if err != nil {
		return Prepared{}, err
	}
	runRoot, err := safeRoot(runDirectory, "run directory")
	if err != nil {
		return Prepared{}, err
	}
	if options.MaxBytes <= 0 || options.MaxFiles <= 0 {
		return Prepared{}, errors.New("workspace bounds must be positive")
	}
	options.Exclude, err = normalizeExcludes(options.Exclude)
	if err != nil {
		return Prepared{}, err
	}
	options.Exclude = append(options.Exclude, ".git")
	if relative, relErr := filepath.Rel(sourceRoot, runRoot); relErr == nil && containedRelative(relative) && relative != "." {
		options.Exclude = append(options.Exclude, filepath.ToSlash(relative))
	}
	options.Exclude = compactStrings(options.Exclude)

	base := filepath.Join(runRoot, "workspaces", "base")
	candidate := filepath.Join(runRoot, "workspaces", "candidate")
	if err := os.MkdirAll(filepath.Dir(base), 0o700); err != nil {
		return Prepared{}, fmt.Errorf("create workspace parent: %w", err)
	}
	if err := copyTree(ctx, sourceRoot, base, options); err != nil {
		return Prepared{}, fmt.Errorf("copy source snapshot: %w", err)
	}
	snapshot, err := scan(ctx, base, options)
	if err != nil {
		return Prepared{}, fmt.Errorf("inventory source snapshot: %w", err)
	}
	if err := copyTree(ctx, base, candidate, options); err != nil {
		return Prepared{}, fmt.Errorf("create candidate workspace: %w", err)
	}
	candidateSnapshot, err := scan(ctx, candidate, options)
	if err != nil {
		return Prepared{}, fmt.Errorf("inventory candidate workspace: %w", err)
	}
	if snapshot.digest != candidateSnapshot.digest {
		return Prepared{}, errors.New("candidate workspace differs from source snapshot")
	}
	return Prepared{
		Source: sourceRoot, Base: base, Candidate: candidate, Digest: snapshot.digest,
		Files: snapshot.files, Bytes: snapshot.bytes, options: options,
	}, nil
}

func Snapshot(ctx context.Context, root string, options Options) (string, error) {
	root, err := safeRoot(root, "workspace")
	if err != nil {
		return "", err
	}
	options.Exclude, err = normalizeExcludes(options.Exclude)
	if err != nil {
		return "", err
	}
	options.Exclude = compactStrings(append(options.Exclude, ".git"))
	result, err := scan(ctx, root, options)
	if err != nil {
		return "", err
	}
	return result.digest, nil
}

func SnapshotExact(ctx context.Context, root string, options Options) (string, error) {
	root, err := safeRoot(root, "workspace")
	if err != nil {
		return "", err
	}
	options.Exclude, err = normalizeExcludes(options.Exclude)
	if err != nil {
		return "", err
	}
	result, err := scan(ctx, root, options)
	if err != nil {
		return "", err
	}
	return result.digest, nil
}

func Diff(ctx context.Context, prepared Prepared) ([]Change, string, error) {
	base, err := scan(ctx, prepared.Base, prepared.options)
	if err != nil {
		return nil, "", fmt.Errorf("inventory base workspace: %w", err)
	}
	candidate, err := scan(ctx, prepared.Candidate, prepared.options)
	if err != nil {
		return nil, "", fmt.Errorf("inventory candidate workspace: %w", err)
	}
	changes := diffEntries(base.entries, candidate.entries)
	return changes, candidate.digest, nil
}

func CopyReview(ctx context.Context, prepared Prepared, name string) (string, error) {
	if err := validComponent(name); err != nil {
		return "", err
	}
	destination := filepath.Join(filepath.Dir(prepared.Base), "reviews", name)
	if info, err := os.Stat(destination); err == nil && info.IsDir() {
		candidate, candidateErr := scan(ctx, prepared.Candidate, prepared.options)
		review, reviewErr := scan(ctx, destination, prepared.options)
		if candidateErr != nil || reviewErr != nil {
			return "", errors.Join(candidateErr, reviewErr)
		}
		if candidate.digest != review.digest {
			return "", fmt.Errorf("existing review workspace %q differs from candidate", name)
		}
		return destination, nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect review workspace: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return "", fmt.Errorf("create review workspace parent: %w", err)
	}
	if err := copyTree(ctx, prepared.Candidate, destination, prepared.options); err != nil {
		return "", fmt.Errorf("create review workspace: %w", err)
	}
	return destination, nil
}

func ValidateChanges(changes []Change, forbidden []string) error {
	paths, err := normalizeExcludes(forbidden)
	if err != nil {
		return fmt.Errorf("normalize forbidden paths: %w", err)
	}
	for _, change := range changes {
		for _, denied := range paths {
			if change.Path == denied || strings.HasPrefix(change.Path, denied+"/") {
				return fmt.Errorf("candidate changed forbidden path %q", change.Path)
			}
		}
	}
	return nil
}

func Apply(ctx context.Context, prepared Prepared, backupDirectory string) error {
	current, err := scan(ctx, prepared.Source, prepared.options)
	if err != nil {
		return fmt.Errorf("inventory promotion target: %w", err)
	}
	if current.digest != prepared.Digest {
		return fmt.Errorf("%w: admitted %s, current %s", errors.New("source drift"), prepared.Digest, current.digest)
	}
	base, err := scan(ctx, prepared.Base, prepared.options)
	if err != nil {
		return fmt.Errorf("inventory promotion base: %w", err)
	}
	candidate, err := scan(ctx, prepared.Candidate, prepared.options)
	if err != nil {
		return fmt.Errorf("inventory promotion candidate: %w", err)
	}
	changes := diffEntries(base.entries, candidate.entries)
	if len(changes) == 0 {
		return nil
	}
	backupDirectory, err = createBackupDirectory(backupDirectory)
	if err != nil {
		return fmt.Errorf("create promotion backup: %w", err)
	}
	for _, change := range changes {
		if _, found := base.entries[change.Path]; !found {
			continue
		}
		if err := copyPath(filepath.Join(prepared.Source, filepath.FromSlash(change.Path)), filepath.Join(backupDirectory, filepath.FromSlash(change.Path))); err != nil {
			return fmt.Errorf("back up %q: %w", change.Path, err)
		}
	}
	wrote, err := applyChanges(ctx, prepared.Source, prepared.Candidate, promotionChanges(base.entries, candidate.entries, changes), base.entries, prepared.options)
	if err != nil {
		var rollbackErr error
		if wrote {
			rollbackErr = restoreChanges(context.Background(), prepared.Source, backupDirectory, base.entries, changes)
		}
		return errors.Join(fmt.Errorf("apply candidate: %w", err), rollbackErr)
	}
	updated, err := scan(ctx, prepared.Source, prepared.options)
	if err != nil {
		rollbackErr := restoreChanges(context.Background(), prepared.Source, backupDirectory, base.entries, changes)
		return errors.Join(fmt.Errorf("verify promoted source: %w", err), rollbackErr)
	}
	if updated.digest != candidate.digest {
		rollbackErr := restoreChanges(context.Background(), prepared.Source, backupDirectory, base.entries, changes)
		return errors.Join(errors.New("promoted source does not match candidate"), rollbackErr)
	}
	return nil
}

func createBackupDirectory(requested string) (string, error) {
	if err := os.Mkdir(requested, 0o700); err == nil {
		return requested, nil
	} else if !errors.Is(err, os.ErrExist) {
		return "", err
	}
	return os.MkdirTemp(filepath.Dir(requested), filepath.Base(requested)+"-")
}

func applyChanges(ctx context.Context, destination, candidate string, changes []Change, before map[string]entry, options Options) (bool, error) {
	wrote := false
	for _, change := range changes {
		if err := ctx.Err(); err != nil {
			return wrote, err
		}
		if change.Kind == "deleted" {
			continue
		}
		if err := verifyPromotionPath(ctx, destination, change.Path, before, options); err != nil {
			return wrote, err
		}
		if err := copyPath(filepath.Join(candidate, filepath.FromSlash(change.Path)), filepath.Join(destination, filepath.FromSlash(change.Path))); err != nil {
			return wrote, err
		}
		wrote = true
	}
	for index := len(changes) - 1; index >= 0; index-- {
		change := changes[index]
		if change.Kind != "deleted" {
			continue
		}
		if err := ctx.Err(); err != nil {
			return wrote, err
		}
		if err := verifyPromotionPath(ctx, destination, change.Path, before, options); err != nil {
			return wrote, err
		}
		path := filepath.Join(destination, filepath.FromSlash(change.Path))
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			if !blockedByNondirectory(destination, path) {
				return wrote, fmt.Errorf("remove %q: %w", change.Path, err)
			}
		}
		wrote = true
	}
	return wrote, nil
}

func promotionChanges(before, after map[string]entry, changes []Change) []Change {
	result := make([]Change, 0, len(changes))
	for _, change := range changes {
		covered := false
		for parent := filepath.ToSlash(filepath.Dir(filepath.FromSlash(change.Path))); parent != "."; parent = filepath.ToSlash(filepath.Dir(filepath.FromSlash(parent))) {
			beforeParent, existed := before[parent]
			afterParent, remains := after[parent]
			if existed && beforeParent.kind == "directory" && (!remains || afterParent.kind != "directory") {
				covered = true
				break
			}
		}
		if !covered {
			result = append(result, change)
		}
	}
	return result
}

func verifyPromotionPath(ctx context.Context, root, relative string, before map[string]entry, options Options) error {
	current, err := scan(ctx, root, options)
	if err != nil {
		return err
	}
	expected, expectedFound := before[relative]
	actual, actualFound := current.entries[relative]
	if expectedFound != actualFound || expected != actual {
		return fmt.Errorf("source drift at %q", relative)
	}
	return nil
}

func restoreChanges(ctx context.Context, destination, backup string, before map[string]entry, changes []Change) error {
	var result error
	for index := len(changes) - 1; index >= 0; index-- {
		if err := ctx.Err(); err != nil {
			return errors.Join(result, err)
		}
		change := changes[index]
		path := filepath.Join(destination, filepath.FromSlash(change.Path))
		if _, existed := before[change.Path]; !existed {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				result = errors.Join(result, fmt.Errorf("remove added path %q during rollback: %w", change.Path, err))
			}
		}
	}
	for _, change := range changes {
		if err := ctx.Err(); err != nil {
			return errors.Join(result, err)
		}
		if _, existed := before[change.Path]; !existed {
			continue
		}
		path := filepath.Join(destination, filepath.FromSlash(change.Path))
		if err := copyPath(filepath.Join(backup, filepath.FromSlash(change.Path)), path); err != nil {
			result = errors.Join(result, fmt.Errorf("restore %q: %w", change.Path, err))
		}
	}
	return result
}

func blockedByNondirectory(root, path string) bool {
	for parent := filepath.Dir(path); parent != root && parent != filepath.Dir(parent); parent = filepath.Dir(parent) {
		info, err := os.Lstat(parent)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			return true
		}
	}
	return false
}

func copyTree(ctx context.Context, source, destination string, options Options) error {
	if _, err := os.Lstat(destination); err == nil {
		return fmt.Errorf("destination %q already exists", destination)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Mkdir(destination, 0o700); err != nil {
		return err
	}
	copier := treeCopier{ctx: ctx, source: source, destination: destination, options: options}
	return filepath.WalkDir(source, copier.visit)
}

type treeCopier struct {
	ctx         context.Context
	source      string
	destination string
	options     Options
	files       int
	bytes       int64
}

func (copier *treeCopier) visit(path string, item fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if err := copier.ctx.Err(); err != nil {
		return err
	}
	relative, err := filepath.Rel(copier.source, path)
	if err != nil || !containedRelative(relative) {
		return errors.Join(errors.New("workspace path escaped source"), err)
	}
	if relative == "." {
		return nil
	}
	relative = filepath.ToSlash(relative)
	if excluded(relative, copier.options.Exclude) {
		if item.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}
	info, err := item.Info()
	if err != nil {
		return err
	}
	if err := copier.admit(info); err != nil {
		return err
	}
	target := filepath.Join(copier.destination, filepath.FromSlash(relative))
	return copyPathValidated(copier.source, path, target, info)
}

func (copier *treeCopier) admit(info fs.FileInfo) error {
	copier.files++
	if copier.files > copier.options.MaxFiles {
		return errors.New("workspace file limit exceeded")
	}
	if info.Mode().IsRegular() {
		copier.bytes += info.Size()
	}
	if copier.bytes > copier.options.MaxBytes {
		return errors.New("workspace byte limit exceeded")
	}
	return nil
}

func copyPathValidated(root, source, destination string, info fs.FileInfo) error {
	switch {
	case info.IsDir():
		return os.Mkdir(destination, info.Mode().Perm()|0o700)
	case info.Mode().IsRegular():
		return atomicCopyFile(source, destination, info.Mode().Perm())
	case info.Mode()&os.ModeSymlink != 0:
		resolved, err := filepath.EvalSymlinks(source)
		if err != nil {
			return fmt.Errorf("resolve symlink %q: %w", source, err)
		}
		if !within(root, resolved) {
			return fmt.Errorf("symlink %q escapes source root", source)
		}
		target, err := os.Readlink(source)
		if err != nil {
			return err
		}
		return os.Symlink(target, destination)
	default:
		return fmt.Errorf("unsupported special file %q", source)
	}
}

func copyPath(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	if err := reconcileDestination(destination, info); err != nil {
		return err
	}
	switch {
	case info.IsDir():
		if err := os.MkdirAll(destination, info.Mode().Perm()|0o700); err != nil {
			return err
		}
		return os.Chmod(destination, info.Mode().Perm()|0o700)
	case info.Mode().IsRegular():
		return atomicCopyFile(source, destination, info.Mode().Perm())
	case info.Mode()&os.ModeSymlink != 0:
		return atomicCopySymlink(source, destination)
	default:
		return fmt.Errorf("unsupported special file %q", source)
	}
}

func reconcileDestination(destination string, sourceInfo fs.FileInfo) error {
	existing, err := os.Lstat(destination)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.IsDir() && !sourceInfo.IsDir() {
		return os.RemoveAll(destination)
	}
	if !existing.IsDir() && sourceInfo.IsDir() {
		return os.Remove(destination)
	}
	return nil
}

func atomicCopySymlink(source, destination string) (returnedErr error) {
	target, err := os.Readlink(source)
	if err != nil {
		return err
	}
	placeholder, err := os.CreateTemp(filepath.Dir(destination), ".agentworkflow-link-*")
	if err != nil {
		return err
	}
	temporary := placeholder.Name()
	defer func() {
		if err := os.Remove(temporary); err != nil && !errors.Is(err, os.ErrNotExist) {
			returnedErr = errors.Join(returnedErr, err)
		}
	}()
	if err := placeholder.Close(); err != nil {
		return err
	}
	if err := os.Remove(temporary); err != nil {
		return err
	}
	if err := os.Symlink(target, temporary); err != nil {
		return err
	}
	return os.Rename(temporary, destination)
}

func atomicCopyFile(source, destination string, mode fs.FileMode) (returnedErr error) {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { returnedErr = errors.Join(returnedErr, input.Close()) }()
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".agentworkflow-copy-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() {
		if err := os.Remove(temporaryPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			returnedErr = errors.Join(returnedErr, err)
		}
	}()
	if err := temporary.Chmod(mode); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if _, err := io.Copy(temporary, input); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Sync(); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, destination)
}

func scan(ctx context.Context, root string, options Options) (inventory, error) {
	scanner := inventoryScanner{
		ctx: ctx, root: root, options: options,
		result: inventory{entries: make(map[string]entry)}, hasher: sha256.New(),
	}
	_, _ = io.WriteString(scanner.hasher, "agentworkflow.workspace/v1\x00")
	err := filepath.WalkDir(root, scanner.visit)
	if err != nil {
		return inventory{}, err
	}
	scanner.result.digest = "sha256:" + hex.EncodeToString(scanner.hasher.Sum(nil))
	return scanner.result, nil
}

type inventoryScanner struct {
	ctx     context.Context
	root    string
	options Options
	result  inventory
	hasher  hash.Hash
}

func (scanner *inventoryScanner) visit(path string, item fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if err := scanner.ctx.Err(); err != nil {
		return err
	}
	relative, err := filepath.Rel(scanner.root, path)
	if err != nil || !containedRelative(relative) {
		return errors.Join(errors.New("workspace inventory escaped root"), err)
	}
	if relative == "." {
		return nil
	}
	relative = filepath.ToSlash(relative)
	if excluded(relative, scanner.options.Exclude) {
		if item.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}
	info, err := item.Info()
	if err != nil {
		return err
	}
	value, err := scanner.describe(path, relative, info)
	if err != nil {
		return err
	}
	return scanner.retain(relative, value)
}

func (scanner *inventoryScanner) describe(path, relative string, info fs.FileInfo) (entry, error) {
	value := entry{mode: info.Mode().Perm()}
	switch {
	case info.IsDir():
		value.kind = "directory"
	case info.Mode().IsRegular():
		value.kind = "file"
		value.size = info.Size()
		digest, err := fileDigest(path)
		if err != nil {
			return entry{}, err
		}
		value.digest = digest
	case info.Mode()&os.ModeSymlink != 0:
		value.kind = "symlink"
		target, err := os.Readlink(path)
		if err != nil {
			return entry{}, err
		}
		value.target = target
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil || !within(scanner.root, resolved) {
			return entry{}, errors.Join(fmt.Errorf("symlink %q escapes or cannot be resolved", relative), err)
		}
	default:
		return entry{}, fmt.Errorf("unsupported special file %q", relative)
	}
	return value, nil
}

func (scanner *inventoryScanner) retain(relative string, value entry) error {
	scanner.result.files++
	if scanner.result.files > scanner.options.MaxFiles {
		return errors.New("workspace file limit exceeded")
	}
	scanner.result.bytes += value.size
	if scanner.result.bytes > scanner.options.MaxBytes {
		return errors.New("workspace byte limit exceeded")
	}
	scanner.result.entries[relative] = value
	_, _ = io.WriteString(scanner.hasher, relative)
	_, _ = scanner.hasher.Write([]byte{0})
	_, _ = io.WriteString(scanner.hasher, value.kind)
	_, _ = scanner.hasher.Write([]byte{0})
	_, _ = io.WriteString(scanner.hasher, fmt.Sprintf("%o\x00%d\x00%s\x00%s\x00", value.mode, value.size, value.digest, value.target))
	return nil
}

func diffEntries(base, candidate map[string]entry) []Change {
	paths := make([]string, 0, len(base)+len(candidate))
	seen := make(map[string]struct{}, len(base)+len(candidate))
	for path := range base {
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	for path := range candidate {
		if _, found := seen[path]; !found {
			paths = append(paths, path)
		}
	}
	slices.Sort(paths)
	changes := make([]Change, 0)
	for _, path := range paths {
		before, beforeFound := base[path]
		after, afterFound := candidate[path]
		switch {
		case !beforeFound:
			changes = append(changes, Change{Path: path, Kind: "added", Bytes: after.size, Digest: after.digest})
		case !afterFound:
			changes = append(changes, Change{Path: path, Kind: "deleted"})
		case before != after:
			changes = append(changes, Change{Path: path, Kind: "modified", Bytes: after.size, Digest: after.digest})
		default:
			continue
		}
	}
	return changes
}

func fileDigest(path string) (_ string, returnedErr error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { returnedErr = errors.Join(returnedErr, file.Close()) }()
	hasher := sha256.New()
	_, _ = io.WriteString(hasher, "agentworkflow.file/v1\x00")
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), nil
}

func safeRoot(path, kind string) (string, error) {
	root, err := filepath.Abs(path)
	if err != nil || root == string(filepath.Separator) {
		return "", errors.Join(fmt.Errorf("workspace %s must be a non-root directory", kind), err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", errors.Join(fmt.Errorf("workspace %s is not a directory", kind), err)
	}
	return root, nil
}

func normalizeExcludes(paths []string) ([]string, error) {
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
		if path == "" || path == "." || filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, "../") {
			return nil, fmt.Errorf("workspace relative path %q is invalid", path)
		}
		result = append(result, path)
	}
	return compactStrings(result), nil
}

func compactStrings(values []string) []string {
	slices.Sort(values)
	return slices.Compact(values)
}

func excluded(path string, excludes []string) bool {
	for _, exclude := range excludes {
		if path == exclude || strings.HasPrefix(path, exclude+"/") {
			return true
		}
	}
	return false
}

func containedRelative(path string) bool {
	return path != ".." && !filepath.IsAbs(path) && !strings.HasPrefix(path, ".."+string(filepath.Separator))
}

func within(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && containedRelative(relative)
}

func validComponent(value string) error {
	if value == "" || value == "." || value == ".." {
		return errors.New("workspace component is invalid")
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || character == '-' || character == '_' || character == '.' {
			continue
		}
		return fmt.Errorf("workspace component %q contains an invalid character", value)
	}
	return nil
}
