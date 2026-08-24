package toolchain

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

func RegeneratePatch(ctx context.Context, config PatchSpec) error {
	root, err := filepath.Abs(config.Root)
	if err != nil || root == string(filepath.Separator) {
		return errors.Join(errors.New("patch set root must be an absolute non-root directory"), err)
	}
	config.Root = root
	descriptor, _, overlayRoot, err := resolvePatchSpec(config)
	if err != nil {
		return err
	}
	if config.CandidateRoot == "" || config.Gofmt == "" {
		return errors.New("patch regeneration requires a candidate root and gofmt")
	}
	candidateRoot, err := filepath.Abs(config.CandidateRoot)
	if err != nil || candidateRoot == string(filepath.Separator) {
		return errors.Join(errors.New("patch candidate must be an absolute non-root directory"), err)
	}
	if err := validateCandidateVersion(candidateRoot, descriptor.GoVersion); err != nil {
		return err
	}
	archivePath, err := resolveArchive(ctx, config, descriptor.Archive.Name, descriptor.Archive.URL, descriptor.Archive.SHA256)
	if err != nil {
		return err
	}
	workRoot := config.ToolchainRoot
	if workRoot == "" {
		workRoot = filepath.Join(config.Root, ".toolchain")
	}
	workRoot, err = filepath.Abs(workRoot)
	if err != nil || workRoot == string(filepath.Separator) {
		return errors.Join(errors.New("patch regeneration work root must be an absolute non-root directory"), err)
	}
	if err := os.MkdirAll(workRoot, 0o755); err != nil {
		return fmt.Errorf("create patch regeneration work root: %w", err)
	}
	work, err := os.MkdirTemp(workRoot, "regenerate-patch-*")
	if err != nil {
		return fmt.Errorf("create patch regeneration work directory: %w", err)
	}
	defer os.RemoveAll(work)
	extracted := filepath.Join(work, "source")
	if err := ExtractSource(ctx, archivePath, extracted); err != nil {
		return err
	}
	pristineRoot := filepath.Join(extracted, "go")
	changed, err := changedFiles(ctx, pristineRoot, candidateRoot)
	if err != nil {
		return err
	}
	if len(changed) == 0 {
		return errors.New("gomad3 patch candidate contains no changes")
	}
	if unexpected := firstUnexpected(changed, descriptor.PatchAllowlist); unexpected != "" {
		return prohibitedPath(unexpected)
	}
	if err := prepareDiffTree(ctx, pristineRoot, candidateRoot, config.Gofmt, changed); err != nil {
		return err
	}
	result, err := runLimit(ctx, pristineRoot, []string{
		"git", "diff", "--no-ext-diff", "--binary", "--src-prefix=a/", "--dst-prefix=b/", "--",
	}, nil, maximumPatchBytes)
	if err != nil {
		return fmt.Errorf("generate gomad3 patch: %w", err)
	}
	if result.Stdout.Truncated {
		return fmt.Errorf("regenerated gomad3 patch exceeds %d bytes", maximumPatchBytes)
	}
	patch := result.Stdout.RawBytes
	if len(patch) == 0 {
		return errors.New("gomad3 patch regeneration produced no changes")
	}
	if err := validateOverlay(overlayRoot, descriptor); err != nil {
		return err
	}
	output := config.Output
	if output == "" {
		output = filepath.Join(config.Root, filepath.FromSlash(descriptor.Patch))
	} else if !filepath.IsAbs(output) {
		output = filepath.Join(config.Root, output)
	}
	return publishRegeneratedPatch(ctx, config.Root, pristineRoot, output, patch, descriptor)
}

func validateCandidateVersion(root, want string) error {
	info, err := os.Lstat(root)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.Join(errors.New("gomad3 patch candidate is not a real directory"), err)
	}
	file, info, err := hostfs.OpenPath(filepath.Join(root, "VERSION"))
	if err != nil {
		return fmt.Errorf("gomad3 patch candidate must be %s: %w", want, err)
	}
	defer file.Close()
	if info.Size() <= 0 || info.Size() > 128 {
		return fmt.Errorf("gomad3 patch candidate must be %s", want)
	}
	contents, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read patch candidate version: %w", err)
	}
	if strings.SplitN(string(contents), "\n", 2)[0] != want {
		return fmt.Errorf("gomad3 patch candidate must be %s", want)
	}
	return nil
}

func resolveArchive(ctx context.Context, config PatchSpec, name, url, digest string) (string, error) {
	if config.Archive != "" {
		actual, err := FileSHA256(config.Archive)
		if err != nil {
			return "", fmt.Errorf("hash source archive: %w", err)
		}
		if actual != digest {
			return "", fmt.Errorf("source archive checksum mismatch: got %s, want %s", actual, digest)
		}
		return config.Archive, nil
	}
	toolchainRoot := config.ToolchainRoot
	if toolchainRoot == "" {
		toolchainRoot = filepath.Join(config.Root, ".toolchain")
	}
	return EnsureSource(ctx, SourceSpec{
		CacheDir: filepath.Join(toolchainRoot, "downloads"), Name: name, URL: url, SHA256: digest,
	})
}

func changedFiles(ctx context.Context, pristineRoot, candidateRoot string) ([]string, error) {
	pristine, err := sourceFiles(ctx, pristineRoot)
	if err != nil {
		return nil, fmt.Errorf("inspect pristine Go source: %w", err)
	}
	candidate, err := sourceFiles(ctx, candidateRoot)
	if err != nil {
		return nil, fmt.Errorf("inspect gomad3 patch candidate: %w", err)
	}
	var paths []string
	for name := range pristine {
		paths = append(paths, name)
	}
	for name := range candidate {
		if _, found := pristine[name]; !found {
			return nil, fmt.Errorf("gomad3 patch candidate adds a source path: %s", name)
		}
	}
	slices.Sort(paths)
	var changed []string
	for _, name := range paths {
		candidatePath, found := candidate[name]
		if !found {
			return nil, fmt.Errorf("gomad3 patch candidate deletes a source path: %s", name)
		}
		pristineDigest, err := FileSHA256(pristine[name])
		if err != nil {
			return nil, err
		}
		candidateDigest, err := FileSHA256(candidatePath)
		if err != nil {
			return nil, err
		}
		if pristineDigest != candidateDigest {
			changed = append(changed, name)
		}
	}
	return changed, nil
}

func sourceFiles(ctx context.Context, root string) (map[string]string, error) {
	files := make(map[string]string)
	err := filepath.WalkDir(root, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if filePath == root || entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("gomad3 patch candidate contains a non-regular entry: %s", filePath)
		}
		relative, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if err := validatePath(relative); err != nil {
			return err
		}
		files[relative] = filePath
		return nil
	})
	return files, err
}

func prepareDiffTree(ctx context.Context, pristineRoot, candidateRoot, gofmt string, changed []string) error {
	if _, err := runPatchCommand(ctx, pristineRoot, []string{"git", "init", "-q"}, nil); err != nil {
		return fmt.Errorf("initialize patch regeneration repository: %w", err)
	}
	for _, setting := range [][]string{{"git", "config", "core.autocrlf", "false"}, {"git", "config", "core.filemode", "true"}} {
		if _, err := runPatchCommand(ctx, pristineRoot, setting, nil); err != nil {
			return fmt.Errorf("configure patch regeneration repository: %w", err)
		}
	}
	add := append([]string{"git", "add", "--"}, changed...)
	if _, err := runPatchCommand(ctx, pristineRoot, add, nil); err != nil {
		return fmt.Errorf("stage pristine patch sources: %w", err)
	}
	for _, relative := range changed {
		if err := copyFile(filepath.Join(candidateRoot, filepath.FromSlash(relative)), filepath.Join(pristineRoot, filepath.FromSlash(relative))); err != nil {
			return err
		}
		if strings.HasSuffix(relative, ".go") {
			if _, err := runPatchCommand(ctx, pristineRoot, []string{gofmt, "-w", relative}, nil); err != nil {
				return fmt.Errorf("format patch candidate %s: %w", relative, err)
			}
		}
	}
	return nil
}

func copyFile(source, destination string) error {
	input, _, err := hostfs.OpenPath(source)
	if err != nil {
		return fmt.Errorf("open patch candidate file: %w", err)
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return fmt.Errorf("open pristine patch file: %w", err)
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil || closeErr != nil {
		return errors.Join(fmt.Errorf("copy patch candidate file: %w", copyErr), closeErr)
	}
	return nil
}

func publishRegeneratedPatch(ctx context.Context, root, pristineRoot, output string, patch []byte, descriptor gomadversion.Descriptor) error {
	output, err := filepath.Abs(output)
	if err != nil || output == string(filepath.Separator) {
		return errors.Join(errors.New("regenerated patch output must be an absolute file path"), err)
	}
	directory := filepath.Dir(output)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create regenerated patch directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".gomad3-patch-*")
	if err != nil {
		return fmt.Errorf("create regenerated patch: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return fmt.Errorf("set regenerated patch mode: %w", err)
	}
	if _, err := temporary.Write(patch); err != nil {
		temporary.Close()
		return fmt.Errorf("write regenerated patch: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync regenerated patch: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close regenerated patch: %w", err)
	}
	if _, err := validatePatch(root, temporaryPath, descriptor); err != nil {
		return fmt.Errorf("validate regenerated patch: %w", err)
	}
	if _, err := runPatchCommand(ctx, pristineRoot, []string{"git", "apply", "--cached", "--check", temporaryPath}, nil); err != nil {
		return fmt.Errorf("verify regenerated patch against pristine source: %w", err)
	}
	if err := os.Rename(temporaryPath, output); err != nil {
		return fmt.Errorf("publish regenerated patch: %w", err)
	}
	directoryFile, err := os.Open(directory)
	if err != nil {
		return err
	}
	return errors.Join(directoryFile.Sync(), directoryFile.Close())
}
