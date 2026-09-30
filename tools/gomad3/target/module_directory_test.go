package target

import (
	"os"
	"path/filepath"
	"testing"
)

func TestModuleDirectoryResolvesTheOwningModule(t *testing.T) {
	module := t.TempDir()
	if err := os.WriteFile(filepath.Join(module, "go.mod"), []byte("module example.com/downstream\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(module, "service", "cluster")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, spec := range map[string]Spec{
		"relative from root":    {Kind: KindGoTest, Source: "./service/cluster", WorkingDir: module},
		"relative from nested":  {Kind: KindGoTest, Source: ".", WorkingDir: nested},
		"absolute":              {Kind: KindGoRun, Source: nested, WorkingDir: t.TempDir()},
		"import path uses cwd":  {Kind: KindGoTest, Source: "example.com/downstream/service/cluster", WorkingDir: module},
		"exec uses working dir": {Kind: KindExec, Source: "/bin/true", WorkingDir: module},
	} {
		t.Run(name, func(t *testing.T) {
			directory, err := ModuleDirectory(spec)
			if err != nil {
				t.Fatal(err)
			}
			if directory != module {
				t.Fatalf("ModuleDirectory() = %q, want %q", directory, module)
			}
		})
	}
	if _, err := ModuleDirectory(Spec{Kind: KindGoTest, Source: "./absent", WorkingDir: module}); err == nil {
		t.Fatal("ModuleDirectory() accepted a missing package directory")
	}
}
