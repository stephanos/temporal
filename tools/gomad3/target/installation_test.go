package target

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	targetbuild "go.temporal.io/server/tools/gomad3/target/internal/build"
	"go.temporal.io/server/tools/gomad3/target/internal/gocommand"
)

const installationRepair = "; set --toolchain-root or GOMAD3_TOOLCHAIN_DIR to a complete Gomad installation"

func writeToolchainInstallation(t *testing.T, root, key string) {
	t.Helper()
	for _, path := range []string{filepath.Join(root, "bin", "go"), filepath.Join(root, "builds", key, "bin", "go")} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "build-key"), []byte(key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func identityRunner(stdout string, requests *[]gocommand.Request) gocommand.Runner {
	return gocommand.New(func(_ context.Context, request hostexec.Request) (hostexec.Result, error) {
		if requests != nil {
			*requests = append(*requests, gocommand.Request{Command: request.Command, Dir: request.Dir})
		}
		return hostexec.Result{Termination: hostexec.TerminationExit, Stdout: hostexec.Output{RawBytes: []byte(stdout)}}, nil
	})
}

func TestReadToolchainIdentityQueriesTheValidatedInstallation(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "toolchain")
	key := strings.Repeat("a", 64)
	writeToolchainInstallation(t, root, key)
	t.Chdir(parent)
	var requests []gocommand.Request
	identity, err := readPinnedToolchainWith(context.Background(), "toolchain", identityRunner("go1.27.1\n"+runtime.GOOS+"\n"+runtime.GOARCH+"\n0\n", &requests))
	if err != nil {
		t.Fatal(err)
	}
	want := ToolchainIdentity{GoVersion: "go1.27.1", BuildKey: key, TargetGOOS: runtime.GOOS, TargetGOARCH: runtime.GOARCH}
	if identity.ToolchainIdentity != want {
		t.Fatalf("identity = %#v, want %#v", identity.ToolchainIdentity, want)
	}
	wantRequests := []gocommand.Request{{Command: []string{filepath.Join(root, "bin", "go"), "env", "GOVERSION", "GOOS", "GOARCH", "CGO_ENABLED"}, Dir: root}}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("requests = %#v, want %#v", requests, wantRequests)
	}
}

func TestReadToolchainIdentityFailsClosedWithRepairGuidance(t *testing.T) {
	key := strings.Repeat("b", 64)
	valid := "go1.27.1\n" + runtime.GOOS + "\n" + runtime.GOARCH + "\n0\n"
	for _, test := range []struct {
		name     string
		mutate   func(t *testing.T, root string)
		stdout   string
		want     func(root string) string
		notExist bool
	}{
		{
			name:   "missing pinned command",
			mutate: func(t *testing.T, root string) { removePath(t, filepath.Join(root, "bin", "go")) },
			want: func(root string) string {
				return "stat pinned Go command in " + root + ": lstat " + filepath.Join(root, "bin", "go") + ": no such file or directory" + installationRepair
			},
			notExist: true,
		},
		{
			name:   "non-executable pinned command",
			mutate: func(t *testing.T, root string) { chmodPath(t, filepath.Join(root, "bin", "go"), 0o600) },
			want:   func(string) string { return "pinned Go command is not a regular executable" },
		},
		{
			name: "directory pinned command",
			mutate: func(t *testing.T, root string) {
				removePath(t, filepath.Join(root, "bin", "go"))
				if err := os.Mkdir(filepath.Join(root, "bin", "go"), 0o700); err != nil {
					t.Fatal(err)
				}
			},
			want: func(string) string { return "pinned Go command is not a regular executable" },
		},
		{
			name:   "missing build key",
			mutate: func(t *testing.T, root string) { removePath(t, filepath.Join(root, "build-key")) },
			want: func(root string) string {
				return "read toolchain build key in " + root + ": open " + filepath.Join(root, "build-key") + ": no such file or directory" + installationRepair
			},
			notExist: true,
		},
		{
			name:   "build key without newline",
			mutate: func(t *testing.T, root string) { writePath(t, filepath.Join(root, "build-key"), key) },
			want:   func(string) string { return "toolchain build key is malformed" },
		},
		{
			name: "uppercase build key",
			mutate: func(t *testing.T, root string) {
				writePath(t, filepath.Join(root, "build-key"), strings.ToUpper(key)+"\n")
			},
			want: func(string) string { return "toolchain build key is malformed" },
		},
		{
			name:   "short build key",
			mutate: func(t *testing.T, root string) { writePath(t, filepath.Join(root, "build-key"), key[:63]+"\n") },
			want:   func(string) string { return "toolchain build key is malformed" },
		},
		{
			name:   "trailing build key data",
			mutate: func(t *testing.T, root string) { writePath(t, filepath.Join(root, "build-key"), key+"\n\n") },
			want:   func(string) string { return "toolchain build key is malformed" },
		},
		{
			name:   "missing build",
			mutate: func(t *testing.T, root string) { removePath(t, filepath.Join(root, "builds", key)) },
			want: func(root string) string {
				return "toolchain build " + key + " is missing or stale in " + root + installationRepair
			},
		},
		{
			name: "stale build key",
			mutate: func(t *testing.T, root string) {
				writePath(t, filepath.Join(root, "build-key"), strings.Repeat("c", 64)+"\n")
			},
			want: func(root string) string {
				return "toolchain build " + strings.Repeat("c", 64) + " is missing or stale in " + root + installationRepair
			},
		},
		{
			name:   "non-executable build command",
			mutate: func(t *testing.T, root string) { chmodPath(t, filepath.Join(root, "builds", key, "bin", "go"), 0o600) },
			want: func(root string) string {
				return "toolchain build " + key + " is missing or stale in " + root + installationRepair
			},
		},
		{
			name:   "platform mismatch",
			stdout: "go1.27.1\nplan9\namd64\n0\n",
			want: func(string) string {
				return "pinned Go target plan9/amd64 does not match host " + runtime.GOOS + "/" + runtime.GOARCH
			},
		},
		{
			name:   "cgo identity",
			stdout: "go1.27.1\n" + runtime.GOOS + "\n" + runtime.GOARCH + "\n1\n",
			want: func(string) string {
				return `pinned Go command returned invalid identity "go1.27.1\n` + runtime.GOOS + `\n` + runtime.GOARCH + `\n1\n"`
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "toolchain")
			writeToolchainInstallation(t, root, key)
			if test.mutate != nil {
				test.mutate(t, root)
			}
			stdout := test.stdout
			if stdout == "" {
				stdout = valid
			}
			_, err := readPinnedToolchainWith(context.Background(), root, identityRunner(stdout, nil))
			if err == nil || err.Error() != test.want(root) {
				t.Fatalf("readPinnedToolchainWith() error = %v, want %q", err, test.want(root))
			}
			if errors.Is(err, os.ErrNotExist) != test.notExist {
				t.Fatalf("readPinnedToolchainWith() error %v matches os.ErrNotExist = %t, want %t", err, !test.notExist, test.notExist)
			}
		})
	}
	if _, err := readPinnedToolchainWith(context.Background(), "", identityRunner(valid, nil)); err == nil || err.Error() != "toolchain root is required" {
		t.Fatalf("readPinnedToolchainWith(\"\") error = %v", err)
	}
}

func TestPinnedToolchainSuppliesBuildCacheLocations(t *testing.T) {
	root := filepath.Join(t.TempDir(), "toolchain")
	key := strings.Repeat("d", 64)
	writeToolchainInstallation(t, root, key)
	toolchain, err := readPinnedToolchainWith(context.Background(), root, identityRunner("go1.27.1\n"+runtime.GOOS+"\n"+runtime.GOARCH+"\n0\n", nil))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := toolchain.installation.PinnedBuild().PreparedTargets(), filepath.Join(root, "builds", key, "prepared-targets"); got != want {
		t.Fatalf("prepared target cache = %q, want %q", got, want)
	}
	cache, err := targetbuild.PrepareCache(toolchain.installation.PinnedBuild().TargetCache())
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "builds", key, "target-cache"); cache != want {
		t.Fatalf("target build cache = %q, want %q", cache, want)
	}
	if info, err := os.Lstat(cache); err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
		t.Fatalf("target build cache = %v, %v", info, err)
	}
}

func removePath(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
}

func chmodPath(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func writePath(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
