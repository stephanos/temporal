package build

import (
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
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
	for name, spec := range map[string]target.Spec{
		"relative from root":    {Kind: target.KindGoTest, Source: "./service/cluster", WorkingDir: module},
		"relative from nested":  {Kind: target.KindGoTest, Source: ".", WorkingDir: nested},
		"absolute":              {Kind: target.KindGoRun, Source: nested, WorkingDir: t.TempDir()},
		"import path uses cwd":  {Kind: target.KindGoTest, Source: "example.com/downstream/service/cluster", WorkingDir: module},
		"exec uses working dir": {Kind: target.KindExec, Source: "/bin/true", WorkingDir: module},
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
	if _, err := ModuleDirectory(target.Spec{Kind: target.KindGoTest, Source: "./absent", WorkingDir: module}); err == nil {
		t.Fatal("ModuleDirectory() accepted a missing package directory")
	}
}
