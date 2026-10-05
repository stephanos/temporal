package target

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
)

func TestPreparedCacheDigestOverlayEmpty(t *testing.T) {
	directory := t.TempDir()
	writeFile(t, filepath.Join(directory, "overlay.json"), `{"Replace":{}}`)
	got, err := overlayDigest("overlay.json", directory)
	requireTestNoError(t, err)
	if want := record.SHA256("sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"); got != want {
		t.Fatalf("empty overlay digest = %q, want %q", got, want)
	}
}

func TestPreparedCacheDigestOverlayCanonicalInputs(t *testing.T) {
	for _, test := range []struct {
		name string
		wire string
	}{
		{name: "reverse map order and unclean original", wire: `{"Replace":{"/fn109/digest/z.go":%q,"/fn109/digest/zz/../a.go":%q}}`},
		{name: "sorted map order and clean original", wire: `{"Replace":{"/fn109/digest/a.go":%[2]q,"/fn109/digest/z.go":%[1]q}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			a := filepath.Join(t.TempDir(), "replacement-a.go")
			z := filepath.Join(t.TempDir(), "replacement-z.go")
			writeFile(t, a, "package a\n")
			writeFile(t, z, "package z\n")
			path := filepath.Join(directory, "overlay.json")
			wire := fmt.Sprintf(test.wire, z, a)
			writeFile(t, path, wire)
			got, err := overlayDigest(path, directory)
			requireTestNoError(t, err)
			if want := record.SHA256("sha256:1db3af338bd4366b249e4f3ae4bf2af122071bad27645a84d6f601516d1d25f6"); got != want {
				t.Fatalf("overlay digest = %q, want %q", got, want)
			}
			for name, want := range map[string]string{path: wire, a: "package a\n", z: "package z\n"} {
				contents, err := os.ReadFile(name)
				requireTestNoError(t, err)
				if string(contents) != want {
					t.Fatalf("overlay input %s changed: %q", name, contents)
				}
			}
		})
	}
}

func TestPreparedCacheDigestOverlayReadFailure(t *testing.T) {
	directory := t.TempDir()
	missing := filepath.Join(directory, "missing.go")
	path := filepath.Join(directory, "overlay.json")
	writeFile(t, path, fmt.Sprintf(`{"Replace":{"/fn109/digest/a.go":%q}}`, missing))
	got, err := overlayDigest(path, directory)
	var pathError *os.PathError
	if got != "" || !errors.Is(err, os.ErrNotExist) || !errors.As(err, &pathError) {
		t.Fatalf("unreadable replacement = %q, %T %v", got, err, err)
	}
	if pathError.Op != "lstat" || pathError.Path != missing || errors.Unwrap(err) != pathError {
		t.Fatalf("replacement error identity changed: %T %v", err, err)
	}
	if want := "read target build overlay replacement: lstat " + missing + ": no such file or directory"; err.Error() != want {
		t.Fatalf("replacement error = %q, want %q", err, want)
	}
}

func TestPreparedCacheDigestModuleInputs(t *testing.T) {
	modContents := "module example.com/digest\n\ngo 1.27.1\n"
	sumContents := "example.com/dependency v1.0.0 h1:fixture\n"
	empty := ""
	for _, test := range []struct {
		name    string
		modName string
		sum     *string
		reverse bool
		want    record.SHA256
	}{
		{name: "present", modName: "go.mod", sum: &sumContents, want: "sha256:cace9d14953fa84d0beac296d7937707921775c4c283615f329c0c4f23fb5674"},
		{name: "absent go.sum", modName: "go.mod", want: "sha256:b9aa172d3a27d4116f21ae146eaf64403e30c4b9e19abb775fb48464dfee4d6d"},
		{name: "empty go.sum", modName: "go.mod", sum: &empty, want: "sha256:c1c810346073420fefb509e1463b85b7d724297bbc13fca78fa1e71b72434c23"},
		{name: "argument order", modName: "go.mod", sum: &sumContents, reverse: true, want: "sha256:195265fcebde348a5b2bc830009fbab082ee679517c2ed47cd439af3e42c77d1"},
		{name: "basename framing", modName: "module.txt", sum: &sumContents, want: "sha256:512c10c73c8891cdbc87e147a83fad61c078da38cff9653a8d337404c598b9be"},
	} {
		t.Run(test.name, func(t *testing.T) {
			modPath := filepath.Join(t.TempDir(), test.modName)
			sumPath := filepath.Join(t.TempDir(), "go.sum")
			writeFile(t, modPath, modContents)
			if test.sum != nil {
				writeFile(t, sumPath, *test.sum)
			}
			paths := []string{modPath, sumPath}
			if test.reverse {
				paths = []string{sumPath, modPath}
			}
			got, err := filesDigest(paths...)
			requireTestNoError(t, err)
			if got != test.want {
				t.Fatalf("module digest = %q, want %q", got, test.want)
			}
			contents, err := os.ReadFile(modPath)
			requireTestNoError(t, err)
			if string(contents) != modContents {
				t.Fatalf("module file changed: %q", contents)
			}
			contents, err = os.ReadFile(sumPath)
			if test.sum == nil {
				if !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("absent go.sum changed: %q, %v", contents, err)
				}
			} else {
				requireTestNoError(t, err)
				if string(contents) != *test.sum {
					t.Fatalf("go.sum changed: %q", contents)
				}
			}
		})
	}
}

func TestPreparedCacheDigestModuleEmptyInputs(t *testing.T) {
	got, err := filesDigest()
	requireTestNoError(t, err)
	if want := record.SHA256("sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"); got != want {
		t.Fatalf("empty module digest = %q, want %q", got, want)
	}
}

func TestPreparedCacheDigestModuleReadFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "go.mod")
	requireTestNoError(t, os.Mkdir(path, 0o700))
	got, err := filesDigest(path)
	if got != "" || err == nil {
		t.Fatalf("nonregular module file = %q, %v", got, err)
	}
	if want := "read target module file: go.mod is not a regular file"; err.Error() != want {
		t.Fatalf("module error = %q, want %q", err, want)
	}
	if cause := errors.Unwrap(err); cause == nil || cause.Error() != "go.mod is not a regular file" || errors.Unwrap(cause) != nil {
		t.Fatalf("module error wrapping changed: %T %v", err, err)
	}
}
