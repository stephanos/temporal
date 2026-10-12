package puritycheck

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const reviewedModuleVersion = "v0.41.0"

// The callable boundary is Check and CheckImportPath, not package startup.
// lazyregexp's os.Args read changes compilation timing, not these validators.
var reviewedModuleSources = map[string]string{
	"module/module.go":              "aa2b672b6d0bbc701d46556d7d46399b2ef6ce1fb918687bd067bf8d5efd7741",
	"semver/semver.go":              "58528919e1a4a35f2bf95c7dbe2124c8aa8d151ca8f5dbe87f70058f7bb35a0d",
	"module/pseudo.go":              "55f36593e394a3595a52f636028442f6be9ed010a6a7e0403dac980b545a374d",
	"internal/lazyregexp/lazyre.go": "2c98e15329ce3fb10d4348f791446d1a7f557b92c954a26a7188dad26f78974b",
}

func reviewedModule(pkg metadata, symbol string) error {
	if symbol != "Check" && symbol != "CheckImportPath" {
		return fmt.Errorf("golang.org/x/mod/module.%s has not been reviewed", symbol)
	}
	if pkg.Module == nil || pkg.Module.Path != "golang.org/x/mod" || pkg.Module.Version != reviewedModuleVersion || pkg.Module.Replace != nil {
		return fmt.Errorf("golang.org/x/mod requires unreplaced reviewed version %s", reviewedModuleVersion)
	}
	root, err := filepath.EvalSymlinks(pkg.Module.Dir)
	if err != nil {
		return fmt.Errorf("reviewed module directory: %w", err)
	}
	for _, dir := range []string{"module", "semver", "internal/lazyregexp"} {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			return fmt.Errorf("reviewed module source inventory: %w", err)
		}
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
				if _, reviewed := reviewedModuleSources[dir+"/"+entry.Name()]; !reviewed {
					return fmt.Errorf("reviewed module source inventory drift: %s/%s", dir, entry.Name())
				}
			}
		}
	}
	for name, want := range reviewedModuleSources {
		path, err := filepath.EvalSymlinks(filepath.Join(root, name))
		if err != nil || path != filepath.Join(root, name) {
			return fmt.Errorf("reviewed module source %s is missing or redirected", name)
		}
		stat, err := os.Stat(path)
		if err != nil || !stat.Mode().IsRegular() || stat.Size() > 1<<20 {
			return fmt.Errorf("reviewed module source %s is not bounded regular source", name)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reviewed module source %s: %w", name, err)
		}
		if fmt.Sprintf("%x", sha256.Sum256(data)) != want {
			return fmt.Errorf("golang.org/x/mod %s source drift in %s requires review", reviewedModuleVersion, name)
		}
	}
	return nil
}
