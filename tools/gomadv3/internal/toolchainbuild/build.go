package toolchainbuild

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/buildkey"
	"go.temporal.io/server/tools/gomadv3/internal/commandrun"
	"go.temporal.io/server/tools/gomadv3/internal/patchset"
	"go.temporal.io/server/tools/gomadv3/internal/safefile"
	"go.temporal.io/server/tools/gomadv3/internal/sourcearchive"
	gomadversion "go.temporal.io/server/tools/gomadv3/internal/version"
)

const buildRecipeVersion = "canonical-v5"
const defaultBuildTimeout = 30 * time.Minute
const commandOutputLimit = 16 << 20
const commandTerminationGrace = 2 * time.Second

var failurePhases = []string{
	"after-lock", "after-extract", "after-patch", "after-overlay", "after-compile",
	"after-build-publish", "after-stamp-publish", "after-launcher-publish",
}

type Config struct {
	Root             string
	ToolchainRoot    string
	Patch            string
	Overlay          string
	BootstrapGo      string
	BuildBash        string
	BuildBashVersion string
	BuildPath        string
	BuildTimeout     time.Duration
	Testing          bool
	FailurePhase     string
}

type Result struct {
	BuildKey string
	BuildDir string
	HostOS   string
	HostArch string
	Reused   bool
	Waited   bool
}

type InjectedFailure struct {
	Phase string
}

func (failure *InjectedFailure) Error() string {
	return "gomadv3 injected builder failure: " + failure.Phase
}

type dependencies struct {
	run           func(context.Context, commandrun.Request) (commandrun.Result, error)
	validate      func(patchset.Config) error
	materialize   func(context.Context, patchset.Config) error
	ensureArchive func(context.Context, sourcearchive.Config) (string, error)
	extract       func(context.Context, string, string) error
}

var productionDependencies = dependencies{
	run:           commandrun.Run,
	validate:      patchset.Validate,
	materialize:   patchset.Materialize,
	ensureArchive: sourcearchive.Ensure,
	extract:       sourcearchive.Extract,
}

func Build(ctx context.Context, config Config) (Result, error) {
	return buildWith(ctx, config, productionDependencies)
}

func buildWith(ctx context.Context, config Config, dependency dependencies) (returned Result, returnedErr error) {
	config, descriptor, err := resolveConfig(config)
	if err != nil {
		return Result{}, err
	}
	if config.FailurePhase != "" {
		if !config.Testing {
			return Result{}, errors.New("gomadv3 builder failure injection requires testing mode")
		}
		if !slices.Contains(failurePhases, config.FailurePhase) {
			return Result{}, fmt.Errorf("gomadv3 builder failure phase is invalid: %s", config.FailurePhase)
		}
	}
	bootstrap, err := inspectBootstrap(ctx, config, dependency.run)
	if err != nil {
		return Result{}, err
	}
	platform := bootstrap.hostOS + "/" + bootstrap.hostArch
	if !slices.Contains(descriptor.SupportedPlatforms, platform) {
		return Result{}, fmt.Errorf("gomadv3 complete deterministic mode requires host %s; got %s", strings.Join(descriptor.SupportedPlatforms, ","), platform)
	}
	if err := dependency.validate(patchset.Config{Root: config.Root, Patch: config.Patch, Overlay: config.Overlay}); err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(config.ToolchainRoot, 0o755); err != nil {
		return Result{}, fmt.Errorf("create gomadv3 toolchain root: %w", err)
	}
	if info, err := os.Lstat(config.ToolchainRoot); err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return Result{}, errors.Join(errors.New("gomadv3 toolchain root is not a real directory"), err)
	}
	snapshot, err := snapshotInputs(config)
	if err != nil {
		return Result{}, err
	}
	defer func() {
		returnedErr = errors.Join(returnedErr, snapshot.remove())
	}()
	if err := dependency.validate(patchset.Config{Root: config.Root, Patch: snapshot.patch, Overlay: snapshot.overlay}); err != nil {
		return Result{}, err
	}
	key, err := computeBuildKey(config, descriptor, bootstrap, snapshot)
	if err != nil {
		return Result{}, err
	}
	result := Result{
		BuildKey: key, BuildDir: filepath.Join(config.ToolchainRoot, "builds", key),
		HostOS: bootstrap.hostOS, HostArch: bootstrap.hostArch,
	}
	if err := os.MkdirAll(filepath.Join(config.ToolchainRoot, "locks"), 0o755); err != nil {
		return result, buildFailure(key, err)
	}
	lock, waited, err := acquireBuildLock(ctx, filepath.Join(config.ToolchainRoot, "locks", key+".lock"))
	if err != nil {
		return result, buildFailure(key, err)
	}
	result.Waited = waited
	defer func() {
		returnedErr = errors.Join(returnedErr, lock.release())
	}()
	if err := inject(config, "after-lock"); err != nil {
		return result, buildFailure(key, err)
	}
	if complete, err := buildComplete(ctx, dependency.run, result.BuildDir, descriptor.GoVersion); err != nil {
		return result, buildFailure(key, err)
	} else if complete {
		if err := publishStable(config.ToolchainRoot, key, config); err != nil {
			return result, buildFailure(key, err)
		}
		result.Reused = true
		return result, nil
	}
	archivePath, err := dependency.ensureArchive(ctx, sourcearchive.Config{
		CacheDir: filepath.Join(config.ToolchainRoot, "downloads"), Name: descriptor.Archive.Name,
		URL: descriptor.Archive.URL, SHA256: descriptor.Archive.SHA256,
	})
	if err != nil {
		return result, buildFailure(key, err)
	}
	if err := os.MkdirAll(filepath.Join(config.ToolchainRoot, "builds"), 0o755); err != nil {
		return result, buildFailure(key, err)
	}
	work, err := os.MkdirTemp(config.ToolchainRoot, "build-*")
	if err != nil {
		return result, buildFailure(key, err)
	}
	defer func() {
		returnedErr = errors.Join(returnedErr, os.RemoveAll(work))
	}()
	extractionRoot := filepath.Join(work, "source")
	if err := dependency.extract(ctx, archivePath, extractionRoot); err != nil {
		return result, buildFailure(key, err)
	}
	sourceRoot := filepath.Join(extractionRoot, "go")
	if err := inject(config, "after-extract"); err != nil {
		return result, buildFailure(key, err)
	}
	if err := rejectOverlayCollisions(snapshot.overlay, sourceRoot); err != nil {
		return result, buildFailure(key, err)
	}
	if err := dependency.materialize(ctx, patchset.Config{
		Root: config.Root, Patch: snapshot.patch, Overlay: snapshot.overlay, SourceRoot: sourceRoot,
	}); err != nil {
		return result, buildFailure(key, err)
	}
	if err := inject(config, "after-patch"); err != nil {
		return result, buildFailure(key, err)
	}
	if err := copyOverlay(snapshot.overlay, sourceRoot); err != nil {
		return result, buildFailure(key, err)
	}
	if err := inject(config, "after-overlay"); err != nil {
		return result, buildFailure(key, err)
	}
	cache := filepath.Join(work, "bootstrap-cache")
	temporary := filepath.Join(work, "tmp")
	if err := os.MkdirAll(cache, 0o700); err != nil {
		return result, buildFailure(key, err)
	}
	if err := os.MkdirAll(temporary, 0o700); err != nil {
		return result, buildFailure(key, err)
	}
	request := commandrun.Request{
		Command: []string{config.BuildBash, "./make.bash"}, Dir: filepath.Join(sourceRoot, "src"),
		Env: buildEnvironment(config, bootstrap, cache, temporary), Timeout: config.BuildTimeout,
		TerminateGrace: commandTerminationGrace, OutputLimit: commandOutputLimit,
	}
	if _, err := runCommand(ctx, dependency.run, request); err != nil {
		return result, buildFailure(key, fmt.Errorf("build gomadv3 Go toolchain: %w", err))
	}
	if complete, err := buildComplete(ctx, dependency.run, sourceRoot, descriptor.GoVersion); err != nil {
		return result, buildFailure(key, err)
	} else if !complete {
		return result, buildFailure(key, errors.New("built toolchain reported an unexpected version"))
	}
	if err := inject(config, "after-compile"); err != nil {
		return result, buildFailure(key, err)
	}
	if err := publishBuild(sourceRoot, result.BuildDir); err != nil {
		return result, buildFailure(key, err)
	}
	if err := inject(config, "after-build-publish"); err != nil {
		return result, buildFailure(key, err)
	}
	if err := publishStable(config.ToolchainRoot, key, config); err != nil {
		return result, buildFailure(key, err)
	}
	return result, nil
}

type bootstrapIdentity struct {
	root, version, hostOS, hostArch, bashVersion string
}

func resolveConfig(config Config) (Config, gomadversion.Descriptor, error) {
	if config.Root == "" {
		return Config{}, gomadversion.Descriptor{}, errors.New("gomadv3 module root is required")
	}
	root, err := filepath.Abs(config.Root)
	if err != nil || root == string(filepath.Separator) {
		return Config{}, gomadversion.Descriptor{}, errors.Join(errors.New("gomadv3 module root must be an absolute non-root directory"), err)
	}
	config.Root = root
	if config.ToolchainRoot == "" {
		config.ToolchainRoot = filepath.Join(root, ".toolchain")
	}
	config.ToolchainRoot, err = filepath.Abs(config.ToolchainRoot)
	if err != nil || config.ToolchainRoot == string(filepath.Separator) {
		return Config{}, gomadversion.Descriptor{}, errors.Join(errors.New("gomadv3 toolchain directory must be an absolute non-root path"), err)
	}
	if config.BootstrapGo == "" || config.BuildBash == "" || config.BuildPath == "" {
		return Config{}, gomadversion.Descriptor{}, errors.New("gomadv3 bootstrap Go, build Bash, and build PATH are required")
	}
	if config.BuildTimeout == 0 {
		config.BuildTimeout = defaultBuildTimeout
	}
	if config.BuildTimeout < 0 {
		return Config{}, gomadversion.Descriptor{}, errors.New("gomadv3 build timeout must be positive")
	}
	descriptor, err := gomadversion.Load(root)
	if err != nil {
		return Config{}, gomadversion.Descriptor{}, err
	}
	if config.Patch == "" {
		config.Patch = filepath.Join(root, filepath.FromSlash(descriptor.Patch))
	} else if !filepath.IsAbs(config.Patch) {
		config.Patch = filepath.Join(root, config.Patch)
	}
	if config.Overlay == "" {
		config.Overlay = filepath.Join(root, "overlay")
	} else if !filepath.IsAbs(config.Overlay) {
		config.Overlay = filepath.Join(root, config.Overlay)
	}
	return config, descriptor, nil
}

func inspectBootstrap(ctx context.Context, config Config, run func(context.Context, commandrun.Request) (commandrun.Result, error)) (bootstrapIdentity, error) {
	environment := filteredEnvironment(os.Environ(), "GOMADSEED", "GOMADV3_CHILD_SEED", "GOROOT")
	value := func(arguments ...string) (string, error) {
		request := commandrun.Request{
			Command: append([]string{config.BootstrapGo}, arguments...), Dir: config.Root, Env: environment,
			Timeout: 30 * time.Second, TerminateGrace: commandTerminationGrace, OutputLimit: 64 << 10,
		}
		result, err := runCommand(ctx, run, request)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(result.Stdout.RawBytes)), nil
	}
	root, err := value("env", "GOROOT")
	if err != nil {
		return bootstrapIdentity{}, fmt.Errorf("inspect bootstrap Go root: %w", err)
	}
	version, err := value("version")
	if err != nil {
		return bootstrapIdentity{}, fmt.Errorf("inspect bootstrap Go version: %w", err)
	}
	hostOS, err := value("env", "GOHOSTOS")
	if err != nil {
		return bootstrapIdentity{}, fmt.Errorf("inspect bootstrap host OS: %w", err)
	}
	hostArch, err := value("env", "GOHOSTARCH")
	if err != nil {
		return bootstrapIdentity{}, fmt.Errorf("inspect bootstrap host architecture: %w", err)
	}
	bashVersion := config.BuildBashVersion
	if bashVersion == "" {
		request := commandrun.Request{
			Command: []string{config.BuildBash, "--version"}, Dir: config.Root, Env: environment,
			Timeout: 30 * time.Second, TerminateGrace: commandTerminationGrace, OutputLimit: 64 << 10,
		}
		result, err := runCommand(ctx, run, request)
		if err != nil {
			return bootstrapIdentity{}, fmt.Errorf("inspect build Bash version: %w", err)
		}
		bashVersion = strings.SplitN(strings.TrimSpace(string(result.Stdout.RawBytes)), "\n", 2)[0]
	}
	for name, value := range map[string]string{"root": root, "version": version, "host OS": hostOS, "host architecture": hostArch, "Bash version": bashVersion} {
		if value == "" || strings.ContainsRune(value, '\n') {
			return bootstrapIdentity{}, fmt.Errorf("bootstrap %s is empty or multiline", name)
		}
	}
	return bootstrapIdentity{root: root, version: version, hostOS: hostOS, hostArch: hostArch, bashVersion: bashVersion}, nil
}

type inputSnapshot struct {
	patch, overlay string
}

func snapshotInputs(config Config) (inputSnapshot, error) {
	patch, err := os.CreateTemp(config.ToolchainRoot, "patch-*")
	if err != nil {
		return inputSnapshot{}, fmt.Errorf("create patch snapshot: %w", err)
	}
	patchPath := patch.Name()
	if err := copyInto(config.Patch, patch); err != nil {
		patch.Close()
		os.Remove(patchPath)
		return inputSnapshot{}, fmt.Errorf("snapshot gomadv3 patch: %w", err)
	}
	if err := patch.Close(); err != nil {
		os.Remove(patchPath)
		return inputSnapshot{}, fmt.Errorf("close patch snapshot: %w", err)
	}
	overlayPath, err := os.MkdirTemp(config.ToolchainRoot, "overlay-*")
	if err != nil {
		os.Remove(patchPath)
		return inputSnapshot{}, fmt.Errorf("create overlay snapshot: %w", err)
	}
	if err := copyTree(config.Overlay, overlayPath); err != nil {
		os.Remove(patchPath)
		os.RemoveAll(overlayPath)
		return inputSnapshot{}, fmt.Errorf("snapshot gomadv3 overlay: %w", err)
	}
	return inputSnapshot{patch: patchPath, overlay: overlayPath}, nil
}

func (snapshot inputSnapshot) remove() error {
	patchErr := os.Remove(snapshot.patch)
	if errors.Is(patchErr, os.ErrNotExist) {
		patchErr = nil
	}
	return errors.Join(patchErr, os.RemoveAll(snapshot.overlay))
}

func computeBuildKey(config Config, descriptor gomadversion.Descriptor, bootstrap bootstrapIdentity, snapshot inputSnapshot) (string, error) {
	patchDigest, err := buildkey.FileDigest(snapshot.patch)
	if err != nil {
		return "", err
	}
	overlayDigest, err := buildkey.TreeDigest(snapshot.overlay)
	if err != nil {
		return "", err
	}
	return buildkey.Compute(buildkey.Input{
		GoVersion: descriptor.GoVersion, ArchiveSHA256: descriptor.Archive.SHA256,
		PatchSHA256: patchDigest, OverlaySHA256: overlayDigest, HostOS: bootstrap.hostOS,
		HostArch: bootstrap.hostArch, BootstrapVersion: bootstrap.version, RecipeVersion: buildRecipeVersion,
		BuildPath: config.BuildPath, BashPath: config.BuildBash, BashVersion: bootstrap.bashVersion,
	})
}

func rejectOverlayCollisions(overlayRoot, sourceRoot string) error {
	files, err := treeFiles(overlayRoot)
	if err != nil {
		return err
	}
	for _, relative := range files {
		destination := filepath.Join(sourceRoot, filepath.FromSlash(relative))
		if _, err := os.Lstat(destination); err == nil {
			return fmt.Errorf("gomadv3 overlay collides with upstream Go source: %s", relative)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func copyOverlay(overlayRoot, sourceRoot string) error {
	files, err := treeFiles(overlayRoot)
	if err != nil {
		return err
	}
	for _, relative := range files {
		destination := filepath.Join(sourceRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		if err := copyPath(filepath.Join(overlayRoot, filepath.FromSlash(relative)), destination, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func copyTree(sourceRoot, destinationRoot string) error {
	files, err := treeFiles(sourceRoot)
	if err != nil {
		return err
	}
	for _, relative := range files {
		destination := filepath.Join(destinationRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return err
		}
		if err := copyPath(filepath.Join(sourceRoot, filepath.FromSlash(relative)), destination, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func treeFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root || entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("overlay entry is not a regular file: %s", path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(relative))
		return nil
	})
	slices.Sort(files)
	return files, err
}

func copyPath(source, destination string, mode os.FileMode) error {
	input, _, err := safefile.OpenPath(source)
	if err != nil {
		return err
	}
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return errors.Join(err, input.Close())
	}
	_, copyErr := io.Copy(output, input)
	return errors.Join(copyErr, input.Close(), output.Close())
}

func copyInto(source string, destination *os.File) error {
	input, _, err := safefile.OpenPath(source)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(destination, input)
	return errors.Join(copyErr, input.Close())
}

func buildEnvironment(config Config, bootstrap bootstrapIdentity, cache, temporary string) []string {
	values := map[string]string{
		"BOOT_GO_GCFLAGS": "", "BOOT_GO_LDFLAGS": "", "CC": "", "CC_FOR_TARGET": "", "CGO_ENABLED": "0",
		"CXX": "", "CXX_FOR_TARGET": "", "FC": "", "GOBUILDTIMELOGFILE": "", "GODEBUG": "", "GOCACHE": cache,
		"GO386": "", "GOAMD64": "", "GOARCH": bootstrap.hostArch, "GOARM": "", "GOARM64": "", "GOBOOTSTRAP_TOOLEXEC": "",
		"GO_BUILDER_NAME": "", "GO_DISTFLAGS": "", "GOENV": "off", "GOEXPERIMENT": "", "GO_EXTLINK_ENABLED": "",
		"GO_GCFLAGS": "", "GO_LDFLAGS": "", "GO_LDSO": "", "GOFIPS140": "", "GOFLAGS": "", "GOHOSTARCH": bootstrap.hostArch,
		"GOHOSTOS": bootstrap.hostOS, "GOMIPS": "", "GOMIPS64": "", "GOOS": bootstrap.hostOS, "GOPPC64": "",
		"GORISCV64": "", "GOROOT_BOOTSTRAP": bootstrap.root, "GOTOOLCHAIN": "local", "GOWASM": "", "GOWORK": "off",
		"LC_ALL": "C", "PATH": config.BuildPath, "PKG_CONFIG": "", "TMPDIR": temporary, "TZ": "UTC",
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	slices.Sort(names)
	environment := make([]string, 0, len(names))
	for _, name := range names {
		environment = append(environment, name+"="+values[name])
	}
	return environment
}

func buildComplete(ctx context.Context, run func(context.Context, commandrun.Request) (commandrun.Result, error), root, version string) (bool, error) {
	goCommand := filepath.Join(root, "bin", "go")
	info, err := os.Lstat(goCommand)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return false, err
	}
	result, err := runCommand(ctx, run, commandrun.Request{
		Command: []string{goCommand, "version"}, Dir: root,
		Env:     filteredEnvironment(os.Environ(), "GOMADSEED", "GOMADV3_CHILD_SEED", "GOROOT"),
		Timeout: 30 * time.Second, TerminateGrace: commandTerminationGrace, OutputLimit: 64 << 10,
	})
	if err != nil {
		return false, nil
	}
	return strings.Contains(string(result.Stdout.RawBytes), " "+version+" "), nil
}

func publishBuild(sourceRoot, buildDir string) error {
	if _, err := os.Lstat(buildDir); err == nil {
		incomplete := buildDir + fmt.Sprintf(".incomplete-%d", os.Getpid())
		if err := os.Rename(buildDir, incomplete); err != nil {
			return fmt.Errorf("isolate incomplete gomadv3 build: %w", err)
		}
		if err := os.RemoveAll(incomplete); err != nil {
			return fmt.Errorf("remove incomplete gomadv3 build: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(sourceRoot, buildDir); err != nil {
		return fmt.Errorf("publish immutable gomadv3 build: %w", err)
	}
	return syncDirectory(filepath.Dir(buildDir))
}

func publishStable(toolchainRoot, key string, config Config) error {
	binRoot := filepath.Join(toolchainRoot, "bin")
	if info, err := os.Lstat(binRoot); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return errors.New("gomadv3 stable bin path is not a real directory")
		}
	} else if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(binRoot, 0o755); err != nil {
			return err
		}
	} else {
		return err
	}
	launcher := []byte("#!/bin/sh\n" +
		"toolchain_dir=$(CDPATH= cd \"$(dirname \"$0\")/..\" && pwd) || exit\n" +
		"build_key=$(cat \"$toolchain_dir/build-key\") || exit\n" +
		"unset GOROOT\n" +
		"exec \"$toolchain_dir/builds/$build_key/bin/go\" \"$@\"\n")
	launcherTemporary, err := temporaryFile(binRoot, ".go.next-*", launcher, 0o755)
	if err != nil {
		return err
	}
	defer os.Remove(launcherTemporary)
	stampTemporary, err := temporaryFile(toolchainRoot, ".build-key.next-*", []byte(key+"\n"), 0o644)
	if err != nil {
		return err
	}
	defer os.Remove(stampTemporary)
	if err := os.Rename(stampTemporary, filepath.Join(toolchainRoot, "build-key")); err != nil {
		return fmt.Errorf("publish gomadv3 build key: %w", err)
	}
	if err := syncDirectory(toolchainRoot); err != nil {
		return err
	}
	if err := inject(config, "after-stamp-publish"); err != nil {
		return err
	}
	if err := os.Rename(launcherTemporary, filepath.Join(binRoot, "go")); err != nil {
		return fmt.Errorf("publish gomadv3 launcher: %w", err)
	}
	if err := syncDirectory(binRoot); err != nil {
		return err
	}
	return inject(config, "after-launcher-publish")
}

func temporaryFile(root, pattern string, contents []byte, mode os.FileMode) (string, error) {
	file, err := os.CreateTemp(root, pattern)
	if err != nil {
		return "", err
	}
	name := file.Name()
	if err := file.Chmod(mode); err != nil {
		file.Close()
		os.Remove(name)
		return "", err
	}
	if _, err := file.Write(contents); err != nil {
		file.Close()
		os.Remove(name)
		return "", err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(name)
		return "", err
	}
	if err := file.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

func runCommand(ctx context.Context, run func(context.Context, commandrun.Request) (commandrun.Result, error), request commandrun.Request) (commandrun.Result, error) {
	result, err := run(ctx, request)
	if err != nil {
		return result, err
	}
	if result.WatchdogTimeout {
		return result, context.DeadlineExceeded
	}
	if result.Cancelled {
		return result, context.Canceled
	}
	if result.Termination != commandrun.TerminationExit || result.ExitCode != 0 {
		return result, fmt.Errorf("%s failed with status %d: %s%s", request.Command[0], result.ExitCode, result.Stdout.Bytes, result.Stderr.Bytes)
	}
	return result, nil
}

func inject(config Config, phase string) error {
	if config.Testing && config.FailurePhase == phase {
		return &InjectedFailure{Phase: phase}
	}
	return nil
}

func buildFailure(key string, err error) error {
	return fmt.Errorf("gomadv3 toolchain build failed (key %s): %w", key, err)
}

func filteredEnvironment(environment []string, names ...string) []string {
	filtered := make([]string, 0, len(environment))
	for _, item := range environment {
		keep := true
		for _, name := range names {
			if strings.HasPrefix(item, name+"=") {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}
