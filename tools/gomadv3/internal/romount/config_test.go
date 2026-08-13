package romount

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMappingsNormalizesSourcesAndTargets(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "schema")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatal(err)
	}

	for _, target := range []string{"go.temporal.io/server/schema", "/go.temporal.io/server/schema"} {
		mappings, err := ParseMappings([]string{"schema=" + target}, root)
		if err != nil {
			t.Fatal(err)
		}
		if len(mappings) != 1 || mappings[0].Source != source || mappings[0].Target != "/go.temporal.io/server/schema" {
			t.Fatalf("ParseMappings(%q) = %#v", target, mappings)
		}
	}
}

func TestParseMappingsRejectsUnsafeOrOverlappingMappings(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"one", "two"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	tests := []struct {
		name     string
		mappings []string
		want     string
	}{
		{name: "missing equals", mappings: []string{"one"}, want: "HOST_DIRECTORY=TARGET_DIRECTORY"},
		{name: "empty source", mappings: []string{"=/target"}, want: "source"},
		{name: "target traversal", mappings: []string{"one=../target"}, want: "target"},
		{name: "target root", mappings: []string{"one=/"}, want: "target"},
		{name: "duplicate", mappings: []string{"one=target", "two=target"}, want: "overlaps"},
		{name: "nested", mappings: []string{"one=target", "two=target/child"}, want: "overlaps"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseMappings(test.mappings, root)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ParseMappings() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestParseMappingsRejectsNonDirectoryAndSymlinkRoots(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(root, "link")
	if err := os.Symlink(root, symlink); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{file, symlink, filepath.Join(root, "missing")} {
		if _, err := ParseMappings([]string{source + "=target"}, root); err == nil {
			t.Fatalf("ParseMappings(%q) succeeded", source)
		}
	}
}
