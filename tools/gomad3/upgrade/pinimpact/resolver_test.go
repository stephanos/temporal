package pinimpact_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/upgrade/pinimpact"
	"golang.org/x/mod/module"
)

// proxyModule is one module version a file-based module proxy serves.
type proxyModule struct {
	path, version string
	requires      []string
}

// fileProxy serves modules' go.mod and info files from a directory, so module
// resolution needs no network.
func fileProxy(t *testing.T, modules ...proxyModule) string {
	t.Helper()
	root := t.TempDir()
	for _, served := range modules {
		escaped, err := module.EscapePath(served.path)
		if err != nil {
			t.Fatal(err)
		}
		directory := filepath.Join(root, filepath.FromSlash(escaped), "@v")
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		goMod := "module " + served.path + "\n\ngo 1.21\n"
		for _, required := range served.requires {
			goMod += "\nrequire " + required + "\n"
		}
		files := map[string]string{
			served.version + ".mod":  goMod,
			served.version + ".info": fmt.Sprintf(`{"Version":%q,"Time":"2026-01-01T00:00:00Z"}`, served.version),
		}
		for name, contents := range files {
			if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		list, err := os.OpenFile(filepath.Join(directory, "list"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := list.WriteString(served.version + "\n"); err != nil {
			t.Fatal(err)
		}
		if err := list.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return "file://" + filepath.ToSlash(root)
}

func goCommand(t *testing.T) string {
	t.Helper()
	command, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go command is unavailable")
	}
	command, err = filepath.Abs(command)
	if err != nil {
		t.Fatal(err)
	}
	return command
}

func TestGoResolverSelectsTheModuleGraphOutsideTheModule(t *testing.T) {
	t.Setenv("GOPROXY", fileProxy(t,
		proxyModule{path: "example.test/a", version: "v1.1.0", requires: []string{"example.test/b v1.2.0"}},
		proxyModule{path: "example.test/b", version: "v1.0.0"},
		proxyModule{path: "example.test/b", version: "v1.2.0"},
	))
	t.Setenv("GOSUMDB", "off")
	t.Setenv("GOFLAGS", "-mod=readonly")
	directory := t.TempDir()
	// go.mod is untidy: example.test/a v1.1.0 raises example.test/b to v1.2.0.
	files := pinimpact.ModuleFiles{
		GoMod:     []byte("module example.test/main\n\ngo 1.27.0\n\nrequire (\n\texample.test/a v1.1.0\n\texample.test/b v1.0.0 // indirect\n)\n\nreplace example.test/local => ./local\n"),
		GoSum:     []byte("example.test/a v1.1.0 h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\n"),
		Directory: directory,
	}
	if err := os.MkdirAll(filepath.Join(directory, "local"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, contents := range map[string][]byte{"go.mod": files.GoMod, "go.sum": files.GoSum, "local/go.mod": []byte("module example.test/local\n")} {
		if err := os.WriteFile(filepath.Join(directory, filepath.FromSlash(name)), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	resolver, err := pinimpact.NewGoResolver(goCommand(t), os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	selected, err := resolver.Resolve(context.Background(), files)
	closeErr := resolver.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("Resolve() error = %v, Close() error = %v", err, closeErr)
	}
	if selected["example.test/a"] != "v1.1.0" || selected["example.test/b"] != "v1.2.0" {
		t.Fatalf("selected = %v, want the module graph's versions", selected)
	}
	for name, want := range map[string][]byte{"go.mod": files.GoMod, "go.sum": files.GoSum} {
		got, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil || string(got) != string(want) {
			t.Fatalf("%s = %q, %v; want unchanged", name, got, err)
		}
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 3 {
		t.Fatalf("module directory entries = %v, %v; want go.mod, go.sum, and local only", entries, err)
	}
}

func TestGoResolverReportsUnavailableModulesAsInfrastructureFailures(t *testing.T) {
	t.Setenv("GOPROXY", "off")
	resolver, err := pinimpact.NewGoResolver(goCommand(t), os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resolver.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	_, err = resolver.Resolve(context.Background(), pinimpact.ModuleFiles{
		GoMod: []byte("module example.test/main\n\ngo 1.27.0\n\nrequire example.test/missing v1.0.0\n"),
	})
	if err == nil || pinimpact.IsInputError(err) || !strings.Contains(err.Error(), "example.test/missing") {
		t.Fatalf("Resolve() error = %v, want an infrastructure failure naming the module", err)
	}
	_, err = resolver.Resolve(context.Background(), pinimpact.ModuleFiles{
		GoMod: []byte("module example.test/main\n\nreplace example.test/local => ./local\n"),
	})
	if !pinimpact.IsInputError(err) {
		t.Fatalf("Resolve() error = %v, want invalid input for a local replacement without a directory", err)
	}
}
