package deterministicio

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

func TestVerifyRegisteredAdapterScratchCleanup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("listing wrapper requires a POSIX shell")
	}
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("resolve release Go: %v", err)
	}
	command := exec.CommandContext(t.Context(), goCommand, "env", "GOVERSION", "GOMODCACHE")
	command.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	output, err := command.Output()
	if err != nil {
		t.Fatalf("resolve release Go and module cache: %v", err)
	}
	settings := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(settings) != 2 || settings[0] != gomadversion.GoVersion {
		t.Fatalf("release Go settings = %q, want %s and a module cache", output, gomadversion.GoVersion)
	}
	for _, adapter := range []struct {
		name, module, cached, scratchPrefix string
	}{
		{name: "sentry", module: sentryModulePath, cached: filepath.Join(append([]string{settings[1]}, sentryAdapter.cacheElements...)...), scratchPrefix: "gomad3-adapter-verify-"},
		{name: "libc", module: libcModulePath, cached: filepath.Join(settings[1], "modernc.org", "libc@"+libcPinnedVersion()), scratchPrefix: "gomad3-libc-verify-"},
	} {
		t.Run(adapter.name, func(t *testing.T) {
			if _, err := os.Stat(adapter.cached); errors.Is(err, fs.ErrNotExist) {
				t.Skipf("pinned module cache is unavailable: %v", err)
			} else if err != nil {
				t.Fatal(err)
			}
			moduleDirectory := filepath.Join(t.TempDir(), "module")
			copyFixtureTree(t, adapter.cached, moduleDirectory)
			for _, test := range []struct {
				name                   string
				malformed, denyCleanup bool
			}{
				{name: "success"},
				{name: "primary", malformed: true},
				{name: "primary-and-cleanup", malformed: true, denyCleanup: true},
				{name: "cleanup", denyCleanup: true},
			} {
				t.Run(test.name, func(t *testing.T) {
					if test.denyCleanup {
						requireVerifierCleanupDenial(t)
					}
					scratchParent := t.TempDir()
					t.Setenv("TMPDIR", scratchParent)
					fixture := t.TempDir()
					marker := filepath.Join(fixture, "blocked-path")
					t.Cleanup(func() {
						contents, err := os.ReadFile(marker)
						if errors.Is(err, fs.ErrNotExist) {
							return
						}
						if err != nil {
							t.Error(err)
							return
						}
						blocked := string(contents)
						scratch := filepath.Dir(filepath.Dir(blocked))
						relative, err := filepath.Rel(scratchParent, blocked)
						if err != nil || !filepath.IsLocal(relative) || filepath.Base(blocked) != ".cleanup-denied" || filepath.Dir(scratch) != scratchParent || !strings.HasPrefix(filepath.Base(scratch), adapter.scratchPrefix) {
							t.Errorf("invalid cleanup fixture path %q: %v", blocked, err)
							return
						}
						if err := os.Chmod(blocked, 0o700); err != nil {
							t.Error(err)
							return
						}
						if err := os.RemoveAll(scratch); err != nil {
							t.Error(err)
						}
					})
					script := "#!/bin/sh\nset -eu\n"
					if test.malformed {
						script += "printf '{'\n"
					} else {
						script += shellQuoteVerifierFixture(goCommand) + " \"$@\"\n"
					}
					if test.denyCleanup {
						script += "blocked=\"$PWD/.cleanup-denied\"\nif [ ! -d \"$blocked\" ]; then\nmkdir \"$blocked\"\nprintf 'leaf' > \"$blocked/leaf\"\nprintf '%s' \"$blocked\" > " + shellQuoteVerifierFixture(marker) + "\nchmod 000 \"$blocked\"\nfi\n"
					}
					wrapper := filepath.Join(fixture, "go-list")
					if err := os.WriteFile(wrapper, []byte(script), 0o700); err != nil {
						t.Fatal(err)
					}
					err := VerifyRegisteredAdapter(t.Context(), adapter.module, moduleDirectory, wrapper)
					var syntaxErr *json.SyntaxError
					primaryMessage := "decode prepared package " + adapter.module + " listing: unexpected end of JSON input"
					if test.malformed && !errors.As(err, &syntaxErr) {
						t.Fatalf("verification error = %v, want listing syntax error", err)
					}
					if test.denyCleanup {
						contents, markerErr := os.ReadFile(marker)
						if markerErr != nil {
							t.Fatalf("listing fixture did not create the fault: %v; verification: %v", markerErr, err)
						}
						blocked := string(contents)
						if _, readErr := os.ReadDir(blocked); !errors.Is(readErr, fs.ErrPermission) {
							t.Fatalf("residual scratch permission fault = %v", readErr)
						}
						var cleanupErr *os.PathError
						if !errors.As(err, &cleanupErr) || !errors.Is(err, fs.ErrPermission) {
							t.Fatalf("real cleanup denial at %s was discarded: %v", blocked, err)
						}
						if cleanupErr.Path != blocked && !strings.HasPrefix(cleanupErr.Path, blocked+string(filepath.Separator)) {
							t.Fatalf("cleanup error path = %q, want inside %q", cleanupErr.Path, blocked)
						}
						if test.malformed {
							joined, ok := err.(interface{ Unwrap() []error })
							if !ok {
								t.Fatalf("simultaneous failure has no joined causes: %v", err)
							}
							causes := joined.Unwrap()
							if len(causes) != 2 || causes[0].Error() != primaryMessage || errors.Unwrap(causes[0]) != syntaxErr || causes[1] != cleanupErr {
								t.Fatalf("failure causes = %v, want original primary then raw cleanup", causes)
							}
						} else if err != cleanupErr {
							t.Fatalf("sole cleanup error = %T, want raw *os.PathError", err)
						}
						t.Logf("observed real cleanup permission failure: %v", cleanupErr)
					} else {
						if test.malformed {
							if err.Error() != primaryMessage || errors.Unwrap(err) != syntaxErr {
								t.Fatalf("primary error shape changed: %v", err)
							}
						} else if err != nil {
							t.Fatal(err)
						}
						entries, readErr := os.ReadDir(scratchParent)
						if readErr != nil || len(entries) != 0 {
							t.Fatalf("scratch remains after verification: %v, %v", entries, readErr)
						}
					}
				})
			}
		})
	}
}

func shellQuoteVerifierFixture(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func requireVerifierCleanupDenial(t *testing.T) {
	t.Helper()
	probe := filepath.Join(t.TempDir(), "denied")
	if err := os.Mkdir(probe, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(probe, 0o700); errors.Is(err, fs.ErrNotExist) {
			return
		} else if err != nil {
			t.Error(err)
			return
		}
		if err := os.RemoveAll(probe); err != nil {
			t.Error(err)
		}
	})
	if err := os.WriteFile(filepath.Join(probe, "leaf"), []byte("leaf"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(probe, 0); err != nil {
		t.Fatal(err)
	}
	err := os.RemoveAll(probe)
	if err == nil {
		t.Skip("host bypasses mode-000 cleanup permissions")
	}
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("cleanup permission probe: %v", err)
	}
	t.Logf("permission probe confirmed: %v", err)
}
