package puritycheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewedModuleBoundary(t *testing.T) {
	root := fixture(t, map[string]string{"record/pure.go": `package record;import "golang.org/x/mod/module";func F(){_ = module.Check("example.com/a","v1.0.0");_ = module.CheckImportPath("example.com/a")}`})
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.invalid/fitness\n\ngo 1.27.1\nrequire golang.org/x/mod v0.41.0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.sum"), []byte("golang.org/x/mod v0.41.0 h1:qJmnOUb4YB+FsEuM3HcWucdZASCPGhsX6uljO6pog0c=\ngolang.org/x/mod v0.41.0/go.mod h1:Ek9pY8RKWXwsWvd3rQiHYtMqkjSUV+s1Rj7j4H5Ur6o=\n"), 0600); err != nil {
		t.Fatal(err)
	}
	p, err := load(root, goCommand(), host, allPure)
	if err != nil {
		t.Fatal(err)
	}
	m := p.metadata["golang.org/x/mod/module"]
	if err := reviewedModule(m, "Check"); err != nil {
		t.Fatal(err)
	}
	findings, err := Check(root, goCommand(), host, allPure)
	if err != nil {
		t.Fatal(err)
	}
	requireCategories(t, findings)
	if err := os.WriteFile(filepath.Join(root, "record/pure.go"), []byte(`package record;import "golang.org/x/mod/module";func F(){_ = module.CheckPath("example.com/a")}`), 0600); err != nil {
		t.Fatal(err)
	}
	findings, err = Check(root, goCommand(), host, allPure)
	if err != nil {
		t.Fatal(err)
	}
	requireCategories(t, findings, "host-effect")
	for _, symbol := range []string{"CheckPath", "SplitPathVersion"} {
		if err := reviewedModule(m, symbol); err == nil {
			t.Fatalf("unchecked symbol %s admitted", symbol)
		}
	}
	original := *m.Module
	t.Run("version", func(t *testing.T) {
		changed := original
		changed.Version = "v0.40.0"
		m.Module = &changed
		if err := reviewedModule(m, "Check"); err == nil || !strings.Contains(err.Error(), "v0.41.0") {
			t.Fatal(err)
		}
	})
	t.Run("replacement", func(t *testing.T) {
		changed := original
		changed.Replace = &struct{ Path, Version string }{Path: "local"}
		m.Module = &changed
		if err := reviewedModule(m, "Check"); err == nil {
			t.Fatal("replacement admitted")
		}
	})
	for _, tampered := range []string{"module/module.go", "module/pseudo.go"} {
		t.Run(tampered, func(t *testing.T) {
			changed := original
			changed.Dir = t.TempDir()
			m.Module = &changed
			for name := range reviewedModuleSources {
				data, err := os.ReadFile(filepath.Join(original.Dir, name))
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(changed.Dir, name)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if name == tampered {
					data = append(data, []byte("\n// tampered initializer\n")...)
				}
				if err := os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := reviewedModule(m, "Check"); err == nil || !strings.Contains(err.Error(), "source drift") {
				t.Fatal(err)
			}
		})
	}
	t.Run("extra-initializer", func(t *testing.T) {
		changed := original
		changed.Dir = t.TempDir()
		m.Module = &changed
		for name := range reviewedModuleSources {
			data, err := os.ReadFile(filepath.Join(original.Dir, name))
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(changed.Dir, name)
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(changed.Dir, "module/evil.go"), []byte("package module\nfunc init(){badWindowsNames=nil}\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := reviewedModule(m, "Check"); err == nil || !strings.Contains(err.Error(), "inventory") {
			t.Fatal(err)
		}
	})
}
