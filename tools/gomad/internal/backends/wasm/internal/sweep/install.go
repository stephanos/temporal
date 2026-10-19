package sweep

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/installation"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
)

// Throwaway installation capacities: the measure-fixture values under which
// TestFrontendSystemInfo and the fn-151 Temporal suites ran.
const (
	throwawayMemoryBytes = 4 << 30
	throwawayFuel        = 1 << 50
)

// GuestDirectory is the guest working directory of a ./tests package.
func GuestDirectory(pattern string) string {
	return path.Join("/workspace", strings.TrimPrefix(pattern, "./"))
}

// ThrowawayConfig derives the cooperative guest configuration from the
// measure-fixture parameters (internal/measure prototypeTemporalConfig):
// schema/sqlite captured at /workspace/schema/sqlite, writable /workspace,
// /tmp and the package directory, the fixture entropy, clock and limits, and
// no network model. Args and the environment are replaced by explore at
// execution; the working directory is not, so each package has its own
// installation. This configuration is a sweep-local throwaway, not fn-220's
// one-command setup.
func ThrowawayConfig(pattern string, inputs captured.CapturedInputs) wasi.Config {
	cwd := GuestDirectory(pattern)
	writable := []string{"/workspace", "/tmp"}
	if cwd != "/workspace" {
		writable = append(writable, cwd)
	}
	return wasi.Config{
		Profile:             wasi.CooperativeProfile,
		WorkingDirectory:    cwd,
		Args:                []string{"temporal.test"},
		Environment:         []string{"PWD=" + cwd, "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC", "TEMPORAL_TEST_LOG_LEVEL=warn"},
		WritableDirectories: writable,
		CapturedInputs:      inputs,
		EntropyKey:          sha256.Sum256([]byte("gomad-wasm-prototype/v1")),
		Clock:               wasi.ClockPolicy{EpochNanos: 946684800000000000},
		Limits:              wasi.Limits{OutputBytes: 256 << 20, TranscriptBytes: 1 << 30, Calls: wasi.MaxCooperativeCalls, Descriptors: 4096, Files: 100000, FilesystemBytes: 512 << 20, PendingEvents: 4096},
	}
}

// packageDirs are one package's installation and cache locations under --work.
type packageDirs struct {
	root, install, manifest, config, preparation, engine, runs string
}

func dirsFor(work, pattern string) packageDirs {
	sum := sha256.Sum256([]byte(pattern))
	slug := strings.NewReplacer("./", "", "/", "-").Replace(pattern) + "-" + fmt.Sprintf("%x", sum[:4])
	root := filepath.Join(work, "packages", slug)
	return packageDirs{
		root: root, install: filepath.Join(root, "install"), manifest: filepath.Join(root, "install", "gomad-wasm.json"),
		config: filepath.Join(root, "install", "guest-config.json"), preparation: filepath.Join(root, "preparation"),
		engine: filepath.Join(root, "engine"), runs: filepath.Join(root, "runs"),
	}
}

// installArgs is the `gomad install` command line for one package.
func installArgs(options Options, pattern string, dirs packageDirs) []string {
	args := []string{"install", "--profile", wasi.CooperativeProfile, "--compiler", options.Compiler, "--helper", options.Helper,
		"--runtime-root", options.RuntimeRoot, "--cache", dirs.preparation, "--compile-cache", dirs.engine, "--output", dirs.manifest}
	if options.ConfigSource == ConfigTemporalInstall {
		// fn-220 hook: the one-command setup derives the same guest
		// configuration inside gomad. Its defaults own memory and fuel.
		return append(args, "--temporal-checkout", options.Checkout, "--temporal-package", pattern)
	}
	return append(args, "--config", dirs.config, "--memory-bytes", strconv.FormatUint(throwawayMemoryBytes, 10), "--fuel", strconv.FormatUint(throwawayFuel, 10))
}

// installPackage creates a fresh installation for one package and returns
// its manifest.
func installPackage(ctx context.Context, executor Executor, options Options, pattern string, dirs packageDirs) (installation.Manifest, error) {
	if err := os.RemoveAll(dirs.root); err != nil {
		return installation.Manifest{}, err
	}
	for _, dir := range []string{dirs.install, dirs.preparation, dirs.engine, dirs.runs} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return installation.Manifest{}, err
		}
	}
	if options.ConfigSource == ConfigThrowaway {
		inputs, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: filepath.Join(options.Checkout, "schema", "sqlite"), Target: "/workspace/schema/sqlite"}}, captured.DefaultLimits())
		if err != nil {
			return installation.Manifest{}, fmt.Errorf("capture schema/sqlite: %w", err)
		}
		encoded, err := json.Marshal(ThrowawayConfig(pattern, inputs))
		if err != nil {
			return installation.Manifest{}, err
		}
		if err := os.WriteFile(dirs.config, encoded, 0o600); err != nil {
			return installation.Manifest{}, err
		}
	}
	completed := executor.Exec(ctx, Command{Path: options.Gomad, Args: installArgs(options, pattern, dirs), Dir: dirs.root, Env: sanitizedEnviron(options.Environ), Timeout: 10 * time.Minute})
	if completed.Err != nil || completed.Exit != 0 {
		return installation.Manifest{}, fmt.Errorf("gomad install (%s) for %s: exit %d %v\n%s", options.ConfigSource, pattern, completed.Exit, completed.Err, completed.Stderr)
	}
	data, err := os.ReadFile(dirs.manifest)
	if err != nil {
		return installation.Manifest{}, err
	}
	var manifest installation.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return installation.Manifest{}, fmt.Errorf("decode installation manifest: %w", err)
	}
	return manifest, nil
}
