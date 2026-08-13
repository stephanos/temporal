package toolchainbuild

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/commandrun"
	"go.temporal.io/server/tools/gomadv3/internal/outputcapture"
	"go.temporal.io/server/tools/gomadv3/internal/patchset"
	"go.temporal.io/server/tools/gomadv3/internal/sourcearchive"
)

func TestBuildRejectsUnsupportedHostBeforePreparingInputs(t *testing.T) {
	root := writeBuildFixture(t)
	var validated atomic.Bool
	dependencies := fakeDependencies(t, &atomic.Int64{})
	dependencies.validate = func(patchset.Config) error {
		validated.Store(true)
		return nil
	}
	dependencies.run = fakeRunner(t, "linux", "amd64", &atomic.Int64{}, nil)
	_, err := buildWith(context.Background(), testConfig(root), dependencies)
	if err == nil || !strings.Contains(err.Error(), "requires host darwin/arm64; got linux/amd64") {
		t.Fatalf("Build() error = %v", err)
	}
	if validated.Load() {
		t.Fatal("Build() prepared inputs before rejecting the host")
	}
}

func TestBuildPublishesAndReusesImmutableToolchain(t *testing.T) {
	root := writeBuildFixture(t)
	var builds atomic.Int64
	dependencies := fakeDependencies(t, &builds)
	config := testConfig(root)
	first, err := buildWith(context.Background(), config, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	if first.Reused || len(first.BuildKey) != 64 || builds.Load() != 1 {
		t.Fatalf("first result = %+v, builds = %d", first, builds.Load())
	}
	stamp, err := os.ReadFile(filepath.Join(config.ToolchainRoot, "build-key"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(stamp)) != first.BuildKey {
		t.Fatalf("stable build key = %q, want %q", stamp, first.BuildKey)
	}
	launcher, err := os.ReadFile(filepath.Join(config.ToolchainRoot, "bin", "go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(launcher), `builds/$build_key/bin/go`) {
		t.Fatalf("stable launcher = %q", launcher)
	}

	second, err := buildWith(context.Background(), config, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Reused || second.BuildKey != first.BuildKey || builds.Load() != 1 {
		t.Fatalf("second result = %+v, builds = %d", second, builds.Load())
	}
}

func TestBuildSerializesConcurrentSameKey(t *testing.T) {
	root := writeBuildFixture(t)
	var builds atomic.Int64
	started := make(chan struct{})
	release := make(chan struct{})
	dependencies := fakeDependencies(t, &builds)
	dependencies.run = fakeRunner(t, "darwin", "arm64", &builds, func() {
		close(started)
		<-release
	})
	config := testConfig(root)
	type outcome struct {
		result Result
		err    error
	}
	results := make(chan outcome, 2)
	go func() {
		result, err := buildWith(context.Background(), config, dependencies)
		results <- outcome{result: result, err: err}
	}()
	<-started
	go func() {
		result, err := buildWith(context.Background(), config, dependencies)
		results <- outcome{result: result, err: err}
	}()
	time.Sleep(25 * time.Millisecond)
	close(release)
	first := <-results
	second := <-results
	if first.err != nil || second.err != nil {
		t.Fatalf("concurrent Build() errors = %v, %v", first.err, second.err)
	}
	if first.result.BuildKey != second.result.BuildKey || builds.Load() != 1 {
		t.Fatalf("concurrent results = %+v, %+v; builds = %d", first.result, second.result, builds.Load())
	}
	if !first.result.Waited && !second.result.Waited {
		t.Fatalf("neither concurrent result reported lock waiting: %+v, %+v", first.result, second.result)
	}
}

func TestBuildInjectedFailuresLeaveNoTemporaryState(t *testing.T) {
	for _, phase := range failurePhases {
		t.Run(phase, func(t *testing.T) {
			root := writeBuildFixture(t)
			var builds atomic.Int64
			config := testConfig(root)
			config.Testing = true
			config.FailurePhase = phase
			_, err := buildWith(context.Background(), config, fakeDependencies(t, &builds))
			var injected *InjectedFailure
			if err == nil || !errorsAs(err, &injected) || injected.Phase != phase {
				t.Fatalf("Build() error = %v", err)
			}
			assertNoBuildTemporaryState(t, config.ToolchainRoot)
		})
	}
}

func TestBuildRejectsOverlayCollisionBeforePatching(t *testing.T) {
	root := writeBuildFixture(t)
	var materialized atomic.Bool
	dependencies := fakeDependencies(t, &atomic.Int64{})
	dependencies.extract = func(ctx context.Context, archive, destination string) error {
		if err := os.MkdirAll(filepath.Join(destination, "go", "src", "runtime"), 0o700); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(destination, "go", "src", "runtime", "gomad.go"), []byte("upstream"), 0o600)
	}
	dependencies.materialize = func(context.Context, patchset.Config) error {
		materialized.Store(true)
		return nil
	}
	_, err := buildWith(context.Background(), testConfig(root), dependencies)
	if err == nil || !strings.Contains(err.Error(), "overlay collides with upstream Go source") {
		t.Fatalf("Build() error = %v", err)
	}
	if materialized.Load() {
		t.Fatal("Build() patched source before rejecting overlay collision")
	}
}

func TestBuildRejectsUnguardedFailureInjection(t *testing.T) {
	root := writeBuildFixture(t)
	config := testConfig(root)
	config.FailurePhase = "after-lock"
	_, err := buildWith(context.Background(), config, fakeDependencies(t, &atomic.Int64{}))
	if err == nil || !strings.Contains(err.Error(), "failure injection requires") {
		t.Fatalf("Build() error = %v", err)
	}
}

func writeBuildFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, directory := range []string{"overlay/src/runtime", "boundary"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(directory)), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	descriptor := `{
  "schema_version": 1,
  "go_version": "go1.26.4",
  "archive": {"name":"go1.26.4.src.tar.gz","url":"https://go.dev/dl/go1.26.4.src.tar.gz","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
  "supported_platforms": ["darwin/arm64"],
  "boundary_manifest_version": "go1.26.4-darwin-arm64-v1",
  "patch": "gomad.patch",
  "adapters": [{"module":"modernc.org/libc","version":"v1.72.3","sum":"h1:test"}],
  "patch_allowlist": ["src/runtime/proc.go"],
  "overlay_allowlist": ["src/runtime/gomad.go"]
}
`
	for name, contents := range map[string]string{
		"version.json":                 descriptor,
		"gomad.patch":                  "fixture patch\n",
		"overlay/src/runtime/gomad.go": "package runtime\n",
	} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func testConfig(root string) Config {
	return Config{
		Root: root, ToolchainRoot: filepath.Join(root, "toolchain"), BootstrapGo: "/bootstrap/go",
		BuildBash: "/bin/bash", BuildBashVersion: "fixture-bash", BuildPath: "/usr/bin:/bin", BuildTimeout: time.Minute,
	}
}

func fakeDependencies(t *testing.T, builds *atomic.Int64) dependencies {
	t.Helper()
	return dependencies{
		run:      fakeRunner(t, "darwin", "arm64", builds, nil),
		validate: func(patchset.Config) error { return nil },
		materialize: func(context.Context, patchset.Config) error {
			return nil
		},
		ensureArchive: func(context.Context, sourcearchive.Config) (string, error) {
			return "/verified/archive", nil
		},
		extract: func(ctx context.Context, archive, destination string) error {
			if err := os.MkdirAll(filepath.Join(destination, "go", "src"), 0o700); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(destination, "go", "src", "make.bash"), []byte("fixture"), 0o700)
		},
	}
}

func fakeRunner(t *testing.T, hostOS, hostArch string, builds *atomic.Int64, onBuild func()) func(context.Context, commandrun.Request) (commandrun.Result, error) {
	t.Helper()
	var buildOnce sync.Once
	return func(ctx context.Context, request commandrun.Request) (commandrun.Result, error) {
		joined := strings.Join(request.Command, " ")
		switch {
		case joined == "/bootstrap/go env GOROOT":
			return commandSuccess("/bootstrap\n"), nil
		case joined == "/bootstrap/go env GOHOSTOS":
			return commandSuccess(hostOS + "\n"), nil
		case joined == "/bootstrap/go env GOHOSTARCH":
			return commandSuccess(hostArch + "\n"), nil
		case joined == "/bootstrap/go version":
			return commandSuccess(fmt.Sprintf("go version go1.26.4 %s/%s\n", hostOS, hostArch)), nil
		case len(request.Command) == 2 && request.Command[1] == "./make.bash":
			buildOnce.Do(func() {
				builds.Add(1)
				if onBuild != nil {
					onBuild()
				}
			})
			goRoot := filepath.Dir(request.Dir)
			if err := os.MkdirAll(filepath.Join(goRoot, "bin"), 0o700); err != nil {
				return commandrun.Result{}, err
			}
			if err := os.WriteFile(filepath.Join(goRoot, "bin", "go"), []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
				return commandrun.Result{}, err
			}
			return commandSuccess("built\n"), nil
		case len(request.Command) == 2 && request.Command[1] == "version" && strings.HasSuffix(request.Command[0], "/bin/go"):
			return commandSuccess("go version go1.26.4 darwin/arm64\n"), nil
		default:
			return commandrun.Result{}, fmt.Errorf("unexpected command %q", request.Command)
		}
	}
}

func commandSuccess(stdout string) commandrun.Result {
	return commandrun.Result{
		Termination: commandrun.TerminationExit, ExitCode: 0, GroupGone: true,
		Stdout: outputcapture.Output{Bytes: []byte(stdout), RawBytes: []byte(stdout)},
	}
}

func assertNoBuildTemporaryState(t *testing.T, root string) {
	t.Helper()
	var debris []string
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || path == root {
			return err
		}
		name := entry.Name()
		if strings.Contains(name, ".next-") || entry.IsDir() && strings.HasPrefix(name, "build-") || strings.HasPrefix(name, "patch-") || strings.HasPrefix(name, "overlay-") {
			debris = append(debris, path)
		}
		return nil
	})
	if len(debris) != 0 {
		t.Fatalf("temporary build state remains: %v", debris)
	}
}

func errorsAs(err error, target any) bool {
	return errors.As(err, target)
}
