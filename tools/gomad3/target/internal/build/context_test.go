package build

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveOwnsModuleContextAndTags(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	packagePath := filepath.Join(root, "nested", "pkg")
	if err := os.MkdirAll(packagePath, 0o700); err != nil {
		t.Fatal(err)
	}
	context, err := Resolve(root, packagePath, []string{"test_dep", "gomad", "test_dep"})
	if err != nil {
		t.Fatal(err)
	}
	if context.Directory != root || context.Package != "./nested/pkg" || len(context.Tags) != 2 || context.Tags[0] != "gomad" || context.Tags[1] != "test_dep" {
		t.Fatalf("Resolve() = %#v", context)
	}
}
