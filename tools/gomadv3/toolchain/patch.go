package toolchain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/hostexec"
	"go.temporal.io/server/tools/gomadv3/internal/hostfs"
	gomadversion "go.temporal.io/server/tools/gomadv3/toolchain/version"
)

const maximumPatchBytes = 64 << 20
const maximumCommandOutput = 4 << 20
const commandTimeout = 2 * time.Minute
const patchCommandTerminationGrace = 2 * time.Second

type PatchSpec struct {
	Root          string
	Patch         string
	Overlay       string
	SourceRoot    string
	CandidateRoot string
	Archive       string
	Output        string
	Gofmt         string
	ToolchainRoot string
}

func ValidatePatch(config PatchSpec) error {
	descriptor, patchPath, overlayRoot, err := resolvePatchSpec(config)
	if err != nil {
		return err
	}
	if _, err := validatePatch(config.Root, patchPath, descriptor); err != nil {
		return err
	}
	return validateOverlay(overlayRoot, descriptor)
}

func MaterializePatch(ctx context.Context, config PatchSpec) error {
	if config.SourceRoot == "" {
		return errors.New("patch materialization requires a source root")
	}
	descriptor, patchPath, _, err := resolvePatchSpec(config)
	if err != nil {
		return err
	}
	patch, err := validatePatch(config.Root, patchPath, descriptor)
	if err != nil {
		return err
	}
	sourceRoot, err := filepath.Abs(config.SourceRoot)
	if err != nil || sourceRoot == string(filepath.Separator) {
		return errors.Join(errors.New("patch source root must be an absolute non-root directory"), err)
	}
	info, err := os.Stat(sourceRoot)
	if err != nil || !info.IsDir() {
		return errors.Join(errors.New("patch source root is not a directory"), err)
	}
	command := []string{"patch", "--dry-run", "--batch", "-V", "none", "-p1", "-F", "0"}
	if _, err := runPatchCommand(ctx, sourceRoot, command, patch); err != nil {
		return errors.Join(errors.New("gomadv3 patch does not apply with zero fuzz"), err)
	}
	command = []string{"patch", "--batch", "-V", "none", "-p1", "-F", "0"}
	if _, err := runPatchCommand(ctx, sourceRoot, command, patch); err != nil {
		return fmt.Errorf("materialize gomadv3 patch: %w", err)
	}
	return nil
}

func resolvePatchSpec(config PatchSpec) (gomadversion.Descriptor, string, string, error) {
	if config.Root == "" {
		return gomadversion.Descriptor{}, "", "", errors.New("patch set requires a module root")
	}
	root, err := filepath.Abs(config.Root)
	if err != nil || root == string(filepath.Separator) {
		return gomadversion.Descriptor{}, "", "", errors.Join(errors.New("patch set root must be an absolute non-root directory"), err)
	}
	descriptor, err := gomadversion.Load(root)
	if err != nil {
		return gomadversion.Descriptor{}, "", "", err
	}
	patchPath := config.Patch
	if patchPath == "" {
		patchPath = filepath.Join(root, filepath.FromSlash(descriptor.Patch))
	} else if !filepath.IsAbs(patchPath) {
		patchPath = filepath.Join(root, patchPath)
	}
	overlayRoot := config.Overlay
	if overlayRoot == "" {
		overlayRoot = filepath.Join(root, "toolchain", "runtime", "overlay")
	} else if !filepath.IsAbs(overlayRoot) {
		overlayRoot = filepath.Join(root, overlayRoot)
	}
	return descriptor, patchPath, overlayRoot, nil
}

func validatePatch(root, path string, descriptor gomadversion.Descriptor) ([]byte, error) {
	file, info, err := hostfs.OpenPath(path)
	if err != nil {
		return nil, fmt.Errorf("open gomadv3 patch: %w", err)
	}
	if info.Size() <= 0 || info.Size() > maximumPatchBytes {
		return nil, errors.Join(fmt.Errorf("gomadv3 patch must be nonempty and no larger than %d bytes", maximumPatchBytes), file.Close())
	}
	contents := make([]byte, info.Size())
	if _, err := io.ReadFull(file, contents); err != nil {
		return nil, errors.Join(fmt.Errorf("read gomadv3 patch: %w", err), file.Close())
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close gomadv3 patch: %w", err)
	}
	paths, err := patchPaths(contents)
	if err != nil {
		return nil, err
	}
	if unexpected := firstUnexpected(paths, descriptor.PatchAllowlist); unexpected != "" {
		return nil, prohibitedPath(unexpected)
	}
	result, err := runPatchCommand(context.Background(), root, []string{"git", "apply", "--numstat"}, contents)
	if err != nil || result.ExitCode != 0 {
		return nil, errors.Join(errors.New("gomadv3 patch is malformed"), err)
	}
	return contents, nil
}

func patchPaths(contents []byte) ([]string, error) {
	lines := strings.Split(string(contents), "\n")
	var paths []string
	var current string
	oldHeader, newHeader := false, false
	finish := func() error {
		if current != "" && (!oldHeader || !newHeader) {
			return errors.New("gomadv3 patch must contain complete Git file headers")
		}
		return nil
	}
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			if err := finish(); err != nil {
				return nil, err
			}
			fields := strings.Fields(line)
			if len(fields) != 4 || !strings.HasPrefix(fields[2], "a/") || !strings.HasPrefix(fields[3], "b/") || fields[2][2:] != fields[3][2:] {
				return nil, errors.New("gomadv3 patch has a non-canonical Git file header")
			}
			current = fields[2][2:]
			if err := validatePath(current); err != nil {
				return nil, err
			}
			paths = append(paths, current)
			oldHeader, newHeader = false, false
		case strings.HasPrefix(line, "new file mode "), strings.HasPrefix(line, "deleted file mode "), strings.HasPrefix(line, "GIT binary patch"), strings.HasPrefix(line, "Binary files "):
			return nil, errors.New("gomadv3 patch may only modify existing text files")
		case strings.HasPrefix(line, "--- "):
			if current == "" || line != "--- a/"+current || oldHeader {
				return nil, errors.New("gomadv3 patch has an invalid old-file header")
			}
			oldHeader = true
		case strings.HasPrefix(line, "+++ "):
			if current == "" || line != "+++ b/"+current || !oldHeader || newHeader {
				return nil, errors.New("gomadv3 patch has an invalid new-file header")
			}
			newHeader = true
		case strings.HasPrefix(line, "+") && strings.Contains(line, "Code generated by ") && strings.Contains(line, " DO NOT EDIT"):
			return nil, errors.New("gomadv3 patch contains generated output")
		}
	}
	if err := finish(); err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, errors.New("gomadv3 patch contains no file changes")
	}
	slices.Sort(paths)
	if len(slices.Compact(paths)) != len(paths) {
		return nil, errors.New("gomadv3 patch contains duplicate file sections")
	}
	return paths, nil
}

func validateOverlay(root string, descriptor gomadversion.Descriptor) error {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root || entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("gomadv3 overlay contains a non-regular file: %s", path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if err := validatePath(relative); err != nil {
			return err
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.IndexByte(contents, 0) >= 0 {
			return fmt.Errorf("gomadv3 overlay contains binary output: %s", relative)
		}
		if bytes.Contains(contents, []byte("Code generated by ")) && bytes.Contains(contents, []byte(" DO NOT EDIT")) && !generatedOverlay(relative) {
			return fmt.Errorf("gomadv3 overlay contains generated output: %s", relative)
		}
		paths = append(paths, relative)
		return nil
	})
	if err != nil {
		return errors.Join(errors.New("validate gomadv3 overlay"), err)
	}
	slices.Sort(paths)
	if len(paths) == 0 {
		return errors.New("gomadv3 overlay contains no runtime source files")
	}
	if unexpected := firstUnexpected(paths, descriptor.OverlayAllowlist); unexpected != "" {
		return prohibitedPath(unexpected)
	}
	return nil
}

func firstUnexpected(paths, allowed []string) string {
	for _, path := range paths {
		if _, found := slices.BinarySearch(allowed, path); !found {
			return path
		}
	}
	return ""
}

func prohibitedPath(path string) error {
	const runtimePrefix = "src/runtime/"
	if strings.HasPrefix(path, runtimePrefix) {
		name := strings.TrimPrefix(path, runtimePrefix)
		if !strings.Contains(name, "/") && strings.HasSuffix(name, ".go") {
			if prohibitedRuntimeArea(name) {
				return fmt.Errorf("gomadv3 input touches prohibited runtime area: %s", path)
			}
			if platformRuntimeFile(name) {
				return fmt.Errorf("gomadv3 input touches prohibited platform file: %s", path)
			}
		}
	}
	return fmt.Errorf("gomadv3 input touches prohibited path: %s", path)
}

func prohibitedRuntimeArea(name string) bool {
	if strings.HasSuffix(name, "_test.go") {
		return true
	}
	for _, exact := range []string{
		"arena.go", "chan.go", "heapdump.go", "mbarrier.go", "mbitmap.go", "mcache.go", "mcentral.go",
		"mcheckmark.go", "mcleanup.go", "mem.go", "mfinal.go", "mfixalloc.go", "mpallocbits.go", "mprof.go",
		"mranges.go", "scavenger.go", "select.go", "sema.go", "sizeclasses.go",
	} {
		if name == exact {
			return true
		}
	}
	for _, prefix := range []string{
		"asan", "asm_", "cgo_", "defs_", "malloc", "mem_", "mgc", "mheap", "mpagealloc", "mpagecache",
		"msan", "msize", "mspan", "mstats", "mwbbuf", "netpoll_", "os_", "race", "rt0_", "signal_",
		"sigqueue_", "stubs_", "sys_", "vdso_",
	} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func platformRuntimeFile(name string) bool {
	for _, suffix := range []string{
		"_32bit.go", "_64bit.go", "_386.go", "_amd64.go", "_arm.go", "_arm64.go", "_loong64.go", "_mips.go",
		"_mips64.go", "_mips64le.go", "_mipsle.go", "_ppc64.go", "_ppc64le.go", "_riscv64.go", "_s390x.go",
		"_wasm.go", "_aix.go", "_android.go", "_darwin.go", "_dragonfly.go", "_freebsd.go", "_illumos.go",
		"_ios.go", "_linux.go", "_netbsd.go", "_openbsd.go", "_plan9.go", "_solaris.go", "_unix.go",
		"_wasip1.go", "_windows.go",
	} {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

func generatedOverlay(path string) bool {
	return path == "src/cmd/compile/internal/gomadintercept/spec_go126.go" ||
		strings.HasSuffix(path, "_generated.go") || strings.HasSuffix(path, "_generated_test.go")
}

func validatePath(path string) error {
	if strings.ContainsRune(path, '\n') {
		return fmt.Errorf("gomadv3 input path contains a newline: %q", path)
	}
	if path == "" || strings.ContainsRune(path, '\x00') || filepath.IsAbs(path) || filepath.Clean(path) != path || strings.HasPrefix(path, "../") || strings.Contains(path, "\\") {
		return fmt.Errorf("gomadv3 input path is invalid: %q", path)
	}
	return nil
}

func runPatchCommand(ctx context.Context, dir string, command []string, stdin []byte) (hostexec.Result, error) {
	return runLimit(ctx, dir, command, stdin, maximumCommandOutput)
}

func runLimit(ctx context.Context, dir string, command []string, stdin []byte, outputLimit uint64) (hostexec.Result, error) {
	result, err := hostexec.Run(ctx, hostexec.Request{
		Command: command, Dir: dir, Env: os.Environ(), Stdin: bytes.NewReader(stdin), Timeout: commandTimeout,
		TerminateGrace: patchCommandTerminationGrace, OutputLimit: outputLimit,
	})
	if err != nil {
		return result, err
	}
	if result.WatchdogTimeout {
		return result, context.DeadlineExceeded
	}
	if result.Cancelled {
		return result, context.Canceled
	}
	if result.Termination != hostexec.TerminationExit || result.ExitCode != 0 {
		return result, fmt.Errorf("%s failed with status %d: %s%s", command[0], result.ExitCode, result.Stdout.Bytes, result.Stderr.Bytes)
	}
	return result, nil
}
