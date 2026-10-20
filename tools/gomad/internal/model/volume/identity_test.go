package volume

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
)

type mutatedImplementationFS struct {
	fs.ReadFileFS
	path string
}

func (sources mutatedImplementationFS) ReadFile(path string) ([]byte, error) {
	content, err := sources.ReadFileFS.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if path == sources.path {
		return append(content, []byte("\n// mutated implementation\n")...), nil
	}
	return content, nil
}

type missingImplementationFS struct{ fs.ReadFileFS }

func (missingImplementationFS) ReadFile(string) ([]byte, error) { return nil, fs.ErrNotExist }

type openImplementationError struct{}

func (openImplementationError) Error() string { panic("identity dispatched open error formatter") }

type erroredImplementationFS struct{ fs.ReadFileFS }

func (erroredImplementationFS) ReadFile(string) ([]byte, error) {
	return nil, openImplementationError{}
}

func TestImplementationIdentityDoesNotDispatchMissingSourceError(t *testing.T) {
	identity, err := implementationSHA256(erroredImplementationFS{})
	if identity != "" || err != (openImplementationError{}) {
		t.Fatal("identity must fail closed and retain the source error without formatting it")
	}
}

func TestImplementationIdentityBindsModelAndHostAdapters(t *testing.T) {
	identity := ImplementationSHA256()
	if len(identity) != len("sha256:")+64 || !strings.HasPrefix(identity, "sha256:") {
		t.Fatalf("invalid implementation identity: %q", identity)
	}
	unchanged, err := implementationSHA256(implementationSources)
	if err != nil || unchanged != identity {
		t.Fatalf("unchanged identity = %q, %v", unchanged, err)
	}
	for _, path := range []string{"runtime.go", "metadata.go", "clock.go", "captured.go", "recorder.go", "types.go", "fs.go", "internal/hashwire/hash.go", "internal/hashwire/wire_generated.go", "identity.go"} {
		t.Run(path, func(t *testing.T) {
			modified, err := implementationSHA256(mutatedImplementationFS{ReadFileFS: implementationSources, path: path})
			if err != nil {
				t.Fatal(err)
			}
			if modified == identity {
				t.Fatalf("identity ignores %s", path)
			}
		})
	}
}

func TestImplementationIdentityRejectsMissingCompiledSources(t *testing.T) {
	identity, err := implementationSHA256(missingImplementationFS{ReadFileFS: implementationSources})
	if !errors.Is(err, fs.ErrNotExist) || identity != "" {
		t.Fatalf("missing source identity = %q, %v", identity, err)
	}
}

func TestImplementationInventoryNamesEveryEmbeddedSource(t *testing.T) {
	seen := make(map[string]bool)
	for _, path := range implementationPaths {
		if seen[path] {
			t.Fatalf("duplicate implementation path: %s", path)
		}
		seen[path] = true
	}
	if err := fs.WalkDir(implementationSources, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && !seen[path] {
			t.Errorf("embedded source missing from identity: %s", path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
