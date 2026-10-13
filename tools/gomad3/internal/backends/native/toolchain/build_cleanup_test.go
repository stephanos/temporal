package toolchain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/backends/native/toolchain/installation"
)

const cleanupLauncher = "#!/bin/sh\n" +
	"toolchain_dir=$(CDPATH= cd \"$(dirname \"$0\")/..\" && pwd) || exit\n" +
	"build_key=$(cat \"$toolchain_dir/build-key\") || exit\n" +
	"unset GOROOT\n" +
	"exec \"$toolchain_dir/builds/$build_key/bin/go\" \"$@\"\n"

type stablePublicationFixture struct {
	label        string
	contents     []byte
	mode         os.FileMode
	events       *[]string
	published    bool
	publishErr   error
	cleanupErr   error
	cleanupCalls int
}

func (stage *stablePublicationFixture) Path() string { return stage.label }

func (stage *stablePublicationFixture) PublishContext(_ context.Context, destination string) (bool, error) {
	*stage.events = append(*stage.events, "publish "+stage.label)
	if stage.published {
		if err := os.WriteFile(destination, stage.contents, stage.mode); err != nil {
			return false, err
		}
	}
	return stage.published, stage.publishErr
}

func (stage *stablePublicationFixture) Cleanup() error {
	stage.cleanupCalls++
	*stage.events = append(*stage.events, "cleanup "+stage.label)
	return stage.cleanupErr
}

type stablePublicationFailureWriter struct{ err error }

func (writer stablePublicationFailureWriter) Write([]byte) (int, error) { return 0, writer.err }

func stablePublicationLabel(pattern string) string {
	if strings.HasPrefix(pattern, ".go.next-") {
		return "launcher"
	}
	return "stamp"
}

func TestBuildCleanupSnapshots(t *testing.T) {
	root := writeBuildFixture(t)
	config := testConfig(root)
	config.Patch = filepath.Join(root, "internal/backends/native/toolchain/runtime/gomad.patch")
	config.Overlay = filepath.Join(root, "internal/backends/native/toolchain/runtime/overlay")
	first, err := snapshotInputs(config)
	if err != nil {
		t.Fatal(err)
	}
	second, err := snapshotInputs(config)
	if err != nil {
		t.Fatal(err)
	}
	if first.patch == second.patch || first.overlay == second.overlay {
		t.Fatalf("snapshots alias: %+v, %+v", first, second)
	}
	if err := os.WriteFile(config.Patch, []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(config.Overlay, "src/runtime/gomad.go"), []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, snapshot := range []inputSnapshot{first, second} {
		for path, want := range map[string]string{
			snapshot.patch: "fixture patch\n",
			filepath.Join(snapshot.overlay, "src/runtime/gomad.go"): "package runtime\n",
		} {
			contents, err := os.ReadFile(path)
			if err != nil || string(contents) != want {
				t.Fatalf("snapshot %s = %q, %v; want %q", path, contents, err, want)
			}
			info, err := os.Stat(path)
			if err != nil || info.Mode().Perm() != 0o600 {
				t.Fatalf("snapshot mode %s = %v, %v", path, info, err)
			}
		}
		for _, path := range []string{snapshot.overlay, filepath.Join(snapshot.overlay, "src"), filepath.Join(snapshot.overlay, "src/runtime")} {
			info, err := os.Stat(path)
			if err != nil || info.Mode().Perm() != 0o700 {
				t.Fatalf("snapshot directory %s = %v, %v", path, info, err)
			}
		}
		if err := snapshot.remove(); err != nil {
			t.Fatal(err)
		}
	}
	assertNoBuildTemporaryState(t, config.ToolchainRoot)
}

func TestBuildCleanupSnapshotPrimaryFailures(t *testing.T) {
	for _, failure := range []string{"patch", "overlay", "unsupported"} {
		t.Run(failure, func(t *testing.T) {
			root := writeBuildFixture(t)
			config := testConfig(root)
			config.Patch = filepath.Join(root, "internal/backends/native/toolchain/runtime/gomad.patch")
			config.Overlay = filepath.Join(root, "internal/backends/native/toolchain/runtime/overlay")
			var want string
			if failure == "unsupported" {
				path := filepath.Join(config.Overlay, "unsupported")
				if err := os.Symlink("missing", path); err != nil {
					t.Fatal(err)
				}
				want = "snapshot gomad3 overlay: overlay entry is not a regular file: " + path
			} else {
				path := filepath.Join(root, "missing-"+failure)
				if failure == "patch" {
					config.Patch = path
					want = "snapshot gomad3 patch: lstat " + path + ": no such file or directory"
				} else {
					config.Overlay = path
					want = "snapshot gomad3 overlay: lstat " + path + ": no such file or directory"
				}
			}
			snapshot, err := snapshotInputs(config)
			if snapshot.patch != "" || snapshot.overlay != "" || snapshot.semantics != nil || err == nil || err.Error() != want {
				t.Fatalf("snapshot = %+v, error = %v; want zero, %q", snapshot, err, want)
			}
			if failure != "unsupported" && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("missing input cause lost: %v", err)
			}
			assertNoBuildTemporaryState(t, config.ToolchainRoot)
		})
	}
}

func TestBuildCleanupRepeatedPublication(t *testing.T) {
	layout, err := installation.At(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{strings.Repeat("a", 64), strings.Repeat("b", 64)} {
		if err := publishStable(layout, key, BuildSpec{}); err != nil {
			t.Fatal(err)
		}
		for path, want := range map[string]string{layout.GoCommand(): cleanupLauncher, layout.BuildKeyFile(): key + "\n"} {
			contents, err := os.ReadFile(path)
			if err != nil || string(contents) != want {
				t.Fatalf("publication %s = %q, %v; want %q", path, contents, err, want)
			}
			info, err := os.Stat(path)
			mode := os.FileMode(0o644)
			if path == layout.GoCommand() {
				mode = 0o755
			}
			if err != nil || info.Mode().Perm() != mode {
				t.Fatalf("publication mode %s = %v, %v; want %o", path, info, err, mode)
			}
		}
		assertNoBuildTemporaryState(t, layout.Root())
	}
}

func TestBuildCleanupStagesStableTransactionBeforePublication(t *testing.T) {
	layout, err := installation.At(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key := strings.Repeat("a", 64)
	var events []string
	var stages []*stablePublicationFixture
	stager := func(ctx context.Context, directory, pattern string, mode os.FileMode, fill func(context.Context, io.Writer) error) (stagedPublication, error) {
		var contents bytes.Buffer
		if err := fill(ctx, &contents); err != nil {
			return nil, err
		}
		label := stablePublicationLabel(pattern)
		events = append(events, "stage "+label)
		stage := &stablePublicationFixture{
			label: label, contents: contents.Bytes(), mode: mode,
			events: &events, published: true,
		}
		stages = append(stages, stage)
		wantDirectory := layout.Root()
		wantPattern := ".build-key.next-*"
		wantMode := os.FileMode(0o644)
		wantContents := key + "\n"
		if label == "launcher" {
			wantDirectory = layout.Bin()
			wantPattern = ".go.next-*"
			wantMode = 0o755
			wantContents = cleanupLauncher
		}
		if directory != wantDirectory || pattern != wantPattern || mode != wantMode || contents.String() != wantContents {
			t.Fatalf("stage %s = directory %q, pattern %q, mode %o, contents %q", label, directory, pattern, mode, contents.String())
		}
		return stage, nil
	}
	if err := publishStableWithStager(layout, key, BuildSpec{}, stager); err != nil {
		t.Fatal(err)
	}
	wantEvents := []string{
		"stage launcher", "stage stamp", "publish stamp", "publish launcher",
		"cleanup stamp", "cleanup launcher",
	}
	if fmt.Sprint(events) != fmt.Sprint(wantEvents) {
		t.Fatalf("transaction events = %v, want %v", events, wantEvents)
	}
	for _, stage := range stages {
		if stage.cleanupCalls != 1 {
			t.Fatalf("%s cleanup calls = %d, want 1", stage.label, stage.cleanupCalls)
		}
	}
	for path, want := range map[string]struct {
		contents string
		mode     os.FileMode
	}{
		layout.BuildKeyFile(): {contents: key + "\n", mode: 0o644},
		layout.GoCommand():    {contents: cleanupLauncher, mode: 0o755},
	} {
		contents, err := os.ReadFile(path)
		if err != nil || string(contents) != want.contents {
			t.Fatalf("published %s = %q, %v; want %q", path, contents, err, want.contents)
		}
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != want.mode {
			t.Fatalf("published mode %s = %v, %v; want %o", path, info, err, want.mode)
		}
	}
}

func TestBuildCleanupStageFailuresPreserveStableState(t *testing.T) {
	for _, test := range []struct {
		name      string
		failStage string
		operation string
	}{
		{name: "launcher-write", failStage: "launcher", operation: "write"},
		{name: "launcher-close", failStage: "launcher", operation: "close"},
		{name: "stamp-write", failStage: "stamp", operation: "write"},
		{name: "stamp-close", failStage: "stamp", operation: "close"},
	} {
		t.Run(test.name, func(t *testing.T) {
			layout, err := installation.At(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(layout.Bin(), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(layout.BuildKeyFile(), []byte("old-key\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(layout.GoCommand(), []byte("old launcher\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			failure := errors.New(test.name + " failed")
			cleanupFailure := errors.New("launcher cleanup failed")
			var events []string
			stager := func(ctx context.Context, _ string, pattern string, mode os.FileMode, fill func(context.Context, io.Writer) error) (stagedPublication, error) {
				label := stablePublicationLabel(pattern)
				events = append(events, "stage "+label)
				if label == test.failStage {
					if test.operation == "write" {
						if err := fill(ctx, stablePublicationFailureWriter{err: failure}); !errors.Is(err, failure) {
							t.Fatalf("fill error = %v, want %v", err, failure)
						}
						return nil, fmt.Errorf("fill staged file: %w", failure)
					}
					var contents bytes.Buffer
					if err := fill(ctx, &contents); err != nil {
						t.Fatal(err)
					}
					return nil, fmt.Errorf("close staged file: %w", failure)
				}
				stage := &stablePublicationFixture{label: label, mode: mode, events: &events, published: true}
				if label == "launcher" && test.failStage == "stamp" {
					stage.cleanupErr = cleanupFailure
				}
				return stage, nil
			}
			err = publishStableWithStager(layout, strings.Repeat("b", 64), BuildSpec{}, stager)
			if !errors.Is(err, failure) {
				t.Fatalf("publishStableWithStager() error = %v, want %v", err, failure)
			}
			if test.failStage == "stamp" && !errors.Is(err, cleanupFailure) {
				t.Fatalf("publishStableWithStager() error = %v, want cleanup %v", err, cleanupFailure)
			}
			for path, want := range map[string]string{
				layout.BuildKeyFile(): "old-key\n",
				layout.GoCommand():    "old launcher\n",
			} {
				contents, readErr := os.ReadFile(path)
				if readErr != nil || string(contents) != want {
					t.Fatalf("stable preimage %s = %q, %v; want %q", path, contents, readErr, want)
				}
			}
			if slicesContainPrefix(events, "publish ") {
				t.Fatalf("stage failure reached publication: %v", events)
			}
		})
	}
}

func TestBuildCleanupDirectorySyncFailuresRetainPublicationDisposition(t *testing.T) {
	for _, failStage := range []string{"stamp", "launcher"} {
		t.Run(failStage, func(t *testing.T) {
			layout, err := installation.At(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(layout.Bin(), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(layout.BuildKeyFile(), []byte("old-key\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(layout.GoCommand(), []byte("old launcher\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			key := strings.Repeat("c", 64)
			failure := errors.New(failStage + " directory sync failed")
			var events []string
			stager := func(ctx context.Context, _ string, pattern string, mode os.FileMode, fill func(context.Context, io.Writer) error) (stagedPublication, error) {
				var contents bytes.Buffer
				if err := fill(ctx, &contents); err != nil {
					return nil, err
				}
				label := stablePublicationLabel(pattern)
				events = append(events, "stage "+label)
				stage := &stablePublicationFixture{
					label: label, contents: contents.Bytes(), mode: mode,
					events: &events, published: true,
				}
				if label == failStage {
					stage.publishErr = fmt.Errorf("sync publication directory: %w", failure)
				}
				return stage, nil
			}
			err = publishStableWithStager(layout, key, BuildSpec{}, stager)
			if err != failure {
				t.Fatalf("publishStableWithStager() error = %v, want exact cause %v", err, failure)
			}
			stamp, readErr := os.ReadFile(layout.BuildKeyFile())
			if readErr != nil || string(stamp) != key+"\n" {
				t.Fatalf("published stamp = %q, %v", stamp, readErr)
			}
			launcher, readErr := os.ReadFile(layout.GoCommand())
			wantLauncher := "old launcher\n"
			if failStage == "launcher" {
				wantLauncher = cleanupLauncher
			}
			if readErr != nil || string(launcher) != wantLauncher {
				t.Fatalf("launcher after %s failure = %q, %v; want %q", failStage, launcher, readErr, wantLauncher)
			}
			wantPublishEvents := []string{"publish stamp"}
			if failStage == "launcher" {
				wantPublishEvents = append(wantPublishEvents, "publish launcher")
			}
			var publishEvents []string
			for _, event := range events {
				if strings.HasPrefix(event, "publish ") {
					publishEvents = append(publishEvents, event)
				}
			}
			if fmt.Sprint(publishEvents) != fmt.Sprint(wantPublishEvents) {
				t.Fatalf("publish events = %v, want %v", publishEvents, wantPublishEvents)
			}
		})
	}
}

func TestBuildCleanupRejectsUnsafeStableBinBeforeStaging(t *testing.T) {
	for _, entry := range []string{"file", "symlink"} {
		t.Run(entry, func(t *testing.T) {
			layout, err := installation.At(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if entry == "file" {
				if err := os.WriteFile(layout.Bin(), []byte("blocked\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			} else {
				target := t.TempDir()
				if err := os.Symlink(target, layout.Bin()); err != nil {
					t.Fatal(err)
				}
			}
			staged := false
			err = publishStableWithStager(layout, strings.Repeat("d", 64), BuildSpec{}, func(context.Context, string, string, os.FileMode, func(context.Context, io.Writer) error) (stagedPublication, error) {
				staged = true
				return nil, errors.New("unexpected staging")
			})
			if err == nil || err.Error() != "gomad3 stable bin path is not a real directory" {
				t.Fatalf("unsafe stable bin error = %v", err)
			}
			if staged {
				t.Fatal("unsafe stable bin reached staging")
			}
		})
	}
}

func slicesContainPrefix(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func TestBuildCleanupPublicationFailures(t *testing.T) {
	for _, failure := range []string{"after-stamp-publish", "after-launcher-publish", "stamp-rename", "launcher-rename"} {
		t.Run(failure, func(t *testing.T) {
			root := writeBuildFixture(t)
			config := testConfig(root)
			layout, err := installation.At(config.ToolchainRoot)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(layout.Bin(), 0o755); err != nil {
				t.Fatal(err)
			}
			blocked := ""
			switch failure {
			case "stamp-rename":
				blocked = layout.BuildKeyFile()
			case "launcher-rename":
				blocked = layout.GoCommand()
			default:
				config.Testing = true
				config.FailurePhase = failure
			}
			if blocked != "" {
				if err := os.Mkdir(blocked, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(blocked, "retained"), []byte("blocked\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if blocked != layout.GoCommand() {
				if err := os.WriteFile(layout.GoCommand(), []byte("old launcher\n"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			result, err := buildWith(context.Background(), config, fakeDependencies(t, &atomic.Int64{}))
			if err == nil || !strings.HasPrefix(err.Error(), "gomad3 toolchain build failed (key "+result.BuildKey+"): ") {
				t.Fatalf("build failure = %+v, %v", result, err)
			}
			if blocked == "" {
				var injected *InjectedFailure
				if !errors.As(err, &injected) || injected.Phase != failure {
					t.Fatalf("injected cause = %v", err)
				}
			} else {
				var cause *os.LinkError
				if !errors.As(err, &cause) || cause.Op != "rename" || cause.New != blocked {
					t.Fatalf("rename cause = %v", err)
				}
				phase := "publish gomad3 build key: "
				if failure == "launcher-rename" {
					phase = "publish gomad3 launcher: "
				}
				want := "gomad3 toolchain build failed (key " + result.BuildKey + "): " + phase + cause.Error()
				if err.Error() != want {
					t.Fatalf("rename error = %q; want %q", err, want)
				}
				contents, readErr := os.ReadFile(filepath.Join(blocked, "retained"))
				if readErr != nil || string(contents) != "blocked\n" {
					t.Fatalf("blocked destination = %q, %v", contents, readErr)
				}
			}
			contents, readErr := os.ReadFile(layout.Build(result.BuildKey).GoCommand())
			if readErr != nil || string(contents) != "#!/bin/sh\nexit 0\n" {
				t.Fatalf("immutable build = %q, %v", contents, readErr)
			}
			if failure != "stamp-rename" {
				contents, readErr = os.ReadFile(layout.BuildKeyFile())
				if readErr != nil || string(contents) != result.BuildKey+"\n" {
					t.Fatalf("published stamp = %q, %v", contents, readErr)
				}
			}
			if failure != "launcher-rename" {
				want := "old launcher\n"
				if failure == "after-launcher-publish" {
					want = cleanupLauncher
				}
				contents, readErr = os.ReadFile(layout.GoCommand())
				if readErr != nil || string(contents) != want {
					t.Fatalf("surviving launcher = %q, %v; want %q", contents, readErr, want)
				}
			}
			if err := filepath.WalkDir(layout.Root(), func(path string, entry os.DirEntry, visitErr error) error {
				if visitErr != nil {
					return visitErr
				}
				if path == layout.Root() || path == blocked {
					return nil
				}
				name := entry.Name()
				if strings.Contains(name, ".next-") || strings.HasPrefix(name, "patch-") || strings.HasPrefix(name, "overlay-") || entry.IsDir() && strings.HasPrefix(name, "build-") {
					return fmt.Errorf("temporary build state remains: %s", path)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
