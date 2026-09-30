package target

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	targetbuild "go.temporal.io/server/tools/gomad3/target/internal/build"
)

// ModuleDirectory returns the directory of the module that owns a go-run or
// go-test target: the nearest go.mod above a path source, or the working
// directory for an import-path source. Build adapters read that module's
// go.mod, so a target named from a subdirectory selects the same adapters as one
// named from the module root.
func ModuleDirectory(spec Spec) (string, error) {
	if spec.Kind == KindExec {
		return filepath.Abs(spec.WorkingDir)
	}
	context, err := targetbuild.Resolve(spec.WorkingDir, spec.Source, spec.BuildTags)
	if err != nil {
		return "", err
	}
	return filepath.Abs(context.Directory)
}

// ValidateWorkingDirectory checks an explicitly supplied target working
// directory: it must be an absolute, clean path to a directory that holds a
// go.mod, because the go command runs there with workspaces disabled.
func ValidateWorkingDirectory(directory string) error {
	if !filepath.IsAbs(directory) || filepath.Clean(directory) != directory {
		return fmt.Errorf("working directory %q must be an absolute, clean path", directory)
	}
	info, err := os.Stat(directory)
	if err != nil {
		return fmt.Errorf("working directory %q: %w", directory, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("working directory %q is not a directory", directory)
	}
	moduleInfo, err := os.Stat(filepath.Join(directory, "go.mod"))
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("working directory %q is not a module root: it has no go.mod", directory)
	}
	if err != nil {
		return fmt.Errorf("working directory %q: %w", directory, err)
	}
	if !moduleInfo.Mode().IsRegular() {
		return fmt.Errorf("working directory %q has a go.mod that is not a regular file", directory)
	}
	return nil
}
