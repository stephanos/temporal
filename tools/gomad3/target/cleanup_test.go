package target

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestTargetFileCleanupExecutable(t *testing.T) {
	directory := t.TempDir()
	source, destination := filepath.Join(directory, "source"), filepath.Join(directory, "destination")
	const contents = "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(source, []byte(contents), 0o751); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(source, 0o751); err != nil {
		t.Fatal(err)
	}
	digest, size, err := hashRegularFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if digest != "sha256:306c6ca7407560340797866e077e053627ad409277d1b9da58106fce4cf717cb" || size != 17 {
		t.Fatalf("hash = (%q, %d)", digest, size)
	}
	if err := copyRegularFile(source, destination); err != nil {
		t.Fatal(err)
	}
	checkTargetCleanupFile(t, source, contents, 0o751)
	checkTargetCleanupFile(t, destination, contents, 0o700)
}

func TestTargetFileCleanupSourceRejection(t *testing.T) {
	for _, kind := range []string{"missing", "symlink", "nonexecutable", "directory"} {
		t.Run(kind, func(t *testing.T) {
			directory := t.TempDir()
			source, destination := filepath.Join(directory, "source"), filepath.Join(directory, "destination")
			switch kind {
			case "symlink":
				referent := filepath.Join(directory, "referent")
				if err := os.WriteFile(referent, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(referent, source); err != nil {
					t.Fatal(err)
				}
			case "nonexecutable":
				if err := os.WriteFile(source, []byte("not executable\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(source, 0o700); err != nil {
					t.Fatal(err)
				}
			case "missing":
			}
			digest, size, hashErr := hashRegularFile(source)
			if digest != "" || size != 0 {
				t.Fatalf("rejected hash = (%q, %d)", digest, size)
			}
			if err := os.WriteFile(destination, []byte("existing destination\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			copyErr := copyRegularFile(source, destination)
			switch kind {
			case "missing":
				checkTargetCleanupPathError(t, hashErr, "", "lstat", source, syscall.ENOENT)
				checkTargetCleanupPathError(t, copyErr, "stat exec target: ", "lstat", source, syscall.ENOENT)
			case "directory":
				checkTargetCleanupPlainError(t, hashErr, "source is not a regular file")
				if copyErr == nil || copyErr.Error() != "stat exec target: source is not a regular file" || fmt.Sprintf("%T", copyErr) != "*fmt.wrapError" {
					t.Fatalf("copy error = (%T, %v)", copyErr, copyErr)
				}
				checkTargetCleanupPlainError(t, errors.Unwrap(copyErr), "source is not a regular file")
			case "symlink", "nonexecutable":
				checkTargetCleanupPlainError(t, hashErr, source+" is not a regular executable")
				checkTargetCleanupPlainError(t, copyErr, "exec target is not a regular executable")
			}
			checkTargetCleanupFile(t, destination, "existing destination\n", 0o600)
			if kind == "missing" {
				if _, err := os.Lstat(source); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("missing source changed: %v", err)
				}
			}
		})
	}
}

func TestTargetFileCleanupDestinationRejection(t *testing.T) {
	for _, operation := range []string{"copy", "write"} {
		for _, kind := range []string{"collision", "missing-parent"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				directory := t.TempDir()
				source, destination := filepath.Join(directory, "source"), filepath.Join(directory, "destination")
				if err := os.WriteFile(source, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
					t.Fatal(err)
				}
				cause := syscall.EEXIST
				if kind == "collision" {
					if err := os.WriteFile(destination, []byte("existing destination\n"), 0o600); err != nil {
						t.Fatal(err)
					}
				} else {
					destination = filepath.Join(directory, "missing", "destination")
					cause = syscall.ENOENT
				}
				var err error
				prefix := ""
				if operation == "copy" {
					err = copyRegularFile(source, destination)
					prefix = "create prepared exec target: "
				} else {
					err = writePreparedFile(destination, []byte("new destination\n"), 0o400)
				}
				checkTargetCleanupPathError(t, err, prefix, "open", destination, cause)
				checkTargetCleanupFile(t, source, "#!/bin/sh\nexit 0\n", 0o700)
				if kind == "collision" {
					checkTargetCleanupFile(t, destination, "existing destination\n", 0o600)
				} else if _, err := os.Lstat(filepath.Dir(destination)); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("missing parent changed: %v", err)
				}
			})
		}
	}
}

func TestTargetFileCleanupPrivateWrite(t *testing.T) {
	for _, mode := range []os.FileMode{0o400, 0o700} {
		for _, contents := range []string{"", "{\"prepared\":true}\n"} {
			t.Run(fmt.Sprintf("%04o/%d-bytes", mode, len(contents)), func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "prepared")
				if err := writePreparedFile(path, []byte(contents), mode); err != nil {
					t.Fatal(err)
				}
				checkTargetCleanupFile(t, path, contents, mode)
			})
		}
	}
}

func checkTargetCleanupFile(t *testing.T, path, contents string, mode os.FileMode) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != contents || !info.Mode().IsRegular() || info.Mode().Perm() != mode || info.Size() != int64(len(contents)) {
		t.Fatalf("file %s = (%q, %v, %d), want (%q, %v, %d)", path, data, info.Mode(), info.Size(), contents, mode, len(contents))
	}
}

func checkTargetCleanupPlainError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || err.Error() != want || fmt.Sprintf("%T", err) != "*errors.errorString" || errors.Unwrap(err) != nil {
		t.Fatalf("error = (%T, %v), want direct error %q", err, err, want)
	}
}

func checkTargetCleanupPathError(t *testing.T, err error, prefix, operation, path string, cause syscall.Errno) {
	t.Helper()
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) || pathErr.Op != operation || pathErr.Path != path || pathErr.Err != cause || !errors.Is(err, cause) {
		t.Fatalf("error = (%T, %v), want %s PathError for %s with %v", err, err, operation, path, cause)
	}
	want := prefix + operation + " " + path + ": " + cause.Error()
	if err.Error() != want || prefix == "" && err != pathErr || prefix != "" && errors.Unwrap(err) != pathErr {
		t.Fatalf("error shape = (%T, %v), want %q with one original PathError", err, err, want)
	}
}
