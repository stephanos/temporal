package toolchain

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"go.temporal.io/server/tools/gomad3/toolchain/installation"
)

const cleanupLauncher = "#!/bin/sh\n" +
	"toolchain_dir=$(CDPATH= cd \"$(dirname \"$0\")/..\" && pwd) || exit\n" +
	"build_key=$(cat \"$toolchain_dir/build-key\") || exit\n" +
	"unset GOROOT\n" +
	"exec \"$toolchain_dir/builds/$build_key/bin/go\" \"$@\"\n"

func TestBuildCleanupSnapshots(t *testing.T) {
	root := writeBuildFixture(t)
	config := testConfig(root)
	config.Patch = filepath.Join(root, "toolchain/runtime/gomad.patch")
	config.Overlay = filepath.Join(root, "toolchain/runtime/overlay")
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
			config.Patch = filepath.Join(root, "toolchain/runtime/gomad.patch")
			config.Overlay = filepath.Join(root, "toolchain/runtime/overlay")
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
			if snapshot != (inputSnapshot{}) || err == nil || err.Error() != want {
				t.Fatalf("snapshot = %+v, error = %v; want zero, %q", snapshot, err, want)
			}
			if failure != "unsupported" && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("missing input cause lost: %v", err)
			}
			assertNoBuildTemporaryState(t, config.ToolchainRoot)
		})
	}
}

func TestBuildCleanupTemporaryFile(t *testing.T) {
	for _, mode := range []os.FileMode{0o644, 0o755} {
		t.Run(fmt.Sprintf("%o", mode), func(t *testing.T) {
			root := t.TempDir()
			path, err := temporaryFile(root, ".next-*", []byte("literal\x00bytes\n"), mode)
			if err != nil {
				t.Fatal(err)
			}
			contents, err := os.ReadFile(path)
			if err != nil || string(contents) != "literal\x00bytes\n" {
				t.Fatalf("temporary bytes = %q, %v", contents, err)
			}
			info, err := os.Stat(path)
			if err != nil || info.Mode().Perm() != mode {
				t.Fatalf("temporary mode = %v, %v; want %o", info, err, mode)
			}
			if err := os.Rename(path, filepath.Join(root, "usable")); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filepath.Join(root, "usable")); err != nil {
				t.Fatal(err)
			}
		})
	}
	root := t.TempDir()
	path, err := temporaryFile(filepath.Join(root, "missing"), ".next-*", nil, 0o644)
	var cause *os.PathError
	if path != "" || !errors.As(err, &cause) || err != cause || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid destination = %q, %v", path, err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("invalid destination debris = %v, %v", entries, err)
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
