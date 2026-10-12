package toolchain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

const cleanupRegeneratedPatch = "diff --git a/src/runtime/proc.go b/src/runtime/proc.go\nindex f85fc2f..5287f8a 100644\n--- a/src/runtime/proc.go\n+++ b/src/runtime/proc.go\n@@ -2,2 +2,2 @@ package runtime\n \n-func target() {}\n+func replacement() {}\n"

func TestPatchCleanupRegenerate(t *testing.T) {
	for _, test := range []struct {
		name      string
		cancel    bool
		temporary string
		work      bool
		cause     error
	}{
		{name: "healthy"},
		{name: "cancel", cancel: true},
		{name: "missing-temporary", cancel: true, temporary: "missing", cause: os.ErrNotExist},
		{name: "nonempty-temporary", cancel: true, temporary: "nonempty", cause: syscall.ENOTEMPTY},
		{name: "work-after-cancel", cancel: true, work: true, cause: syscall.EACCES},
		{name: "work-after-publication", work: true, cause: syscall.EACCES},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.work && os.Geteuid() == 0 {
				t.Skip("work removal permission fault requires an unprivileged process")
			}
			root, archive, candidate := writeRegenerateFixture(t)
			if err := os.WriteFile(filepath.Join(candidate, "src", "runtime", "proc.go"), []byte("package runtime\n\nfunc replacement() {}\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(t.TempDir(), "result.patch")
			if err := os.WriteFile(output, []byte("previous\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			workRoot := filepath.Join(t.TempDir(), "work")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var obstruction string
			boundary := &patchCleanupContext{Context: ctx, directory: filepath.Dir(output)}
			boundary.atTemporary = func(path string) {
				if test.temporary != "" {
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
					if test.temporary == "nonempty" {
						if err := os.Mkdir(path, 0o700); err != nil {
							t.Fatal(err)
						}
						obstruction = filepath.Join(path, "child")
						if err := os.WriteFile(obstruction, []byte("owned\n"), 0o600); err != nil {
							t.Fatal(err)
						}
					}
				}
				if test.work {
					works, err := filepath.Glob(filepath.Join(workRoot, "regenerate-patch-*"))
					if err != nil || len(works) != 1 {
						t.Fatalf("regeneration work paths = %v, error = %v", works, err)
					}
					blocked := filepath.Join(works[0], "blocked")
					if err := os.Mkdir(blocked, 0o700); err != nil {
						t.Fatal(err)
					}
					obstruction = filepath.Join(blocked, "child")
					if err := os.WriteFile(obstruction, []byte("owned\n"), 0o600); err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() {
						if err := os.Chmod(blocked, 0o700); err != nil && !errors.Is(err, os.ErrNotExist) {
							t.Error(err)
						}
					})
					if err := os.Chmod(blocked, 0); err != nil {
						t.Fatal(err)
					}
				}
				if test.cancel {
					cancel()
				}
			}
			config := PatchSpec{Root: root, CandidateRoot: candidate, Archive: archive, Output: output, Gofmt: "gofmt", ToolchainRoot: workRoot}
			err := RegeneratePatch(boundary, config)
			if !boundary.fired {
				t.Fatal("regeneration did not reach the temporary publication boundary")
			}
			if test.cause != nil {
				var probe error
				if test.work {
					probe = os.RemoveAll(workRoot)
				} else {
					probe = os.Remove(boundary.temporary)
				}
				if !errors.Is(probe, test.cause) {
					t.Fatalf("cleanup obstruction probe = %v, want %v", probe, test.cause)
				}
				t.Logf("confirmed filesystem obstruction: %v", probe)
			}
			const primary = "verify regenerated patch against pristine source: context canceled"
			if test.cancel {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("RegeneratePatch() error = %v, want cancellation", err)
				}
				if test.cause == nil {
					if err.Error() != primary || errors.Unwrap(err) != context.Canceled {
						t.Fatalf("RegeneratePatch() changed primary error identity: %v", err)
					}
				} else {
					joined, ok := err.(interface{ Unwrap() []error })
					if !ok || len(joined.Unwrap()) != 2 || joined.Unwrap()[0].Error() != primary || errors.Unwrap(joined.Unwrap()[0]) != context.Canceled {
						t.Fatalf("RegeneratePatch() error = %v, want primary-first cleanup join", err)
					}
				}
			}
			if test.cause != nil {
				if !errors.Is(err, test.cause) {
					t.Fatalf("RegeneratePatch() error = %v, want real cleanup cause %v", err, test.cause)
				}
				if !test.cancel {
					if _, ok := err.(*os.PathError); !ok {
						t.Fatalf("sole cleanup error = %T, want direct *os.PathError", err)
					}
				}
				var pathErr *os.PathError
				cleanupErr := err
				if test.cancel {
					cleanupErr = err.(interface{ Unwrap() []error }).Unwrap()[1]
				}
				if !errors.As(cleanupErr, &pathErr) || pathErr.Path == "" || !errors.Is(pathErr.Err, test.cause) {
					t.Fatalf("cleanup error = %v, want filesystem removal error", cleanupErr)
				}
				t.Logf("observed cleanup syscall failure: %v", pathErr)
			} else if !test.cancel && err != nil {
				t.Fatal(err)
			}
			contents, readErr := os.ReadFile(output)
			if readErr != nil {
				t.Fatal(readErr)
			}
			want := cleanupRegeneratedPatch
			if test.cancel {
				want = "previous\n"
			}
			if string(contents) != want {
				t.Fatalf("output = %q, want %q", contents, want)
			}
			info, statErr := os.Stat(output)
			if statErr != nil {
				t.Fatal(statErr)
			}
			wantMode := os.FileMode(0o644)
			if test.cancel {
				wantMode = 0o600
			}
			if info.Mode().Perm() != wantMode {
				t.Fatalf("output mode = %o, want %o", info.Mode().Perm(), wantMode)
			}
			if !test.work {
				entries, readErr := os.ReadDir(workRoot)
				if readErr != nil || len(entries) != 0 {
					t.Fatalf("work cleanup left entries = %v, error = %v", entries, readErr)
				}
			}
			if test.temporary != "nonempty" {
				if _, statErr := os.Stat(boundary.temporary); !errors.Is(statErr, os.ErrNotExist) {
					t.Fatalf("temporary remains: %v", statErr)
				}
			} else if child, readErr := os.ReadFile(obstruction); readErr != nil || string(child) != "owned\n" {
				t.Fatalf("temporary obstruction = %q, error = %v", child, readErr)
			}
			if test.work {
				if err := os.Chmod(filepath.Dir(obstruction), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.RemoveAll(workRoot); err != nil {
					t.Fatal(err)
				}
			}
			if test.cancel || test.work {
				if err := RegeneratePatch(context.Background(), config); err != nil {
					t.Fatalf("healthy retry: %v", err)
				}
				contents, readErr := os.ReadFile(output)
				if readErr != nil || string(contents) != cleanupRegeneratedPatch {
					t.Fatalf("healthy retry output = %q, error = %v", contents, readErr)
				}
			}
		})
	}
}

type patchCleanupContext struct {
	context.Context
	directory   string
	temporary   string
	fired       bool
	atTemporary func(string)
}

func (ctx *patchCleanupContext) Err() error {
	if !ctx.fired {
		paths, err := filepath.Glob(filepath.Join(ctx.directory, ".gomad3-patch-*"))
		if err != nil {
			return err
		}
		if len(paths) == 1 {
			ctx.fired = true
			ctx.temporary = paths[0]
			ctx.atTemporary(paths[0])
		}
	}
	return ctx.Context.Err()
}
