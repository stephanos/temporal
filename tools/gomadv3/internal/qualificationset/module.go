package qualificationset

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/safefile"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

const maximumGoModBytes = 4 << 20

type ModuleIdentity struct {
	Path        string        `json:"path"`
	GoModSHA256 record.SHA256 `json:"go_mod_sha256"`
}

func identifyModule(workingDirectory, expected string) (ModuleIdentity, error) {
	absolute, err := filepath.Abs(workingDirectory)
	if err != nil {
		return ModuleIdentity{}, fmt.Errorf("resolve target module directory: %w", err)
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return ModuleIdentity{}, fmt.Errorf("inspect target module directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return ModuleIdentity{}, errors.New("target module directory must not be a symbolic link")
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return ModuleIdentity{}, fmt.Errorf("resolve target module directory links: %w", err)
	}
	file, info, err := safefile.OpenPath(filepath.Join(resolved, "go.mod"))
	if err != nil {
		return ModuleIdentity{}, fmt.Errorf("open target go.mod: %w", err)
	}
	if info.Size() <= 0 || info.Size() > maximumGoModBytes {
		return ModuleIdentity{}, errors.Join(fmt.Errorf("target go.mod must be between 1 and %d bytes", maximumGoModBytes), file.Close())
	}
	contents, readErr := io.ReadAll(io.LimitReader(file, maximumGoModBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return ModuleIdentity{}, errors.Join(fmt.Errorf("read target go.mod: %w", readErr), closeErr)
	}
	parsed, err := modfile.Parse("go.mod", contents, nil)
	if err != nil {
		return ModuleIdentity{}, fmt.Errorf("parse target go.mod: %w", err)
	}
	if parsed.Module == nil {
		return ModuleIdentity{}, errors.New("target go.mod has no module directive")
	}
	path := parsed.Module.Mod.Path
	if err := module.CheckPath(path); err != nil {
		return ModuleIdentity{}, fmt.Errorf("target module path is invalid: %w", err)
	}
	if expected != "" && path != expected {
		return ModuleIdentity{}, fmt.Errorf("target module is %q, want %q", path, expected)
	}
	return ModuleIdentity{Path: path, GoModSHA256: record.HashBytes(contents)}, nil
}
