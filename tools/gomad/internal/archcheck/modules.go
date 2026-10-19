package archcheck

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

func nativeOrDefault(defaults, native map[string]bool) map[string]bool {
	merged := make(map[string]bool, len(defaults)+len(native))
	for _, set := range []map[string]bool{defaults, native} {
		for path := range set {
			merged[path] = true
		}
	}
	return merged
}

// nativeModulePresent reports whether the native module is checked out under
// root; it is false without the native selection.
func nativeModulePresent(root string) bool {
	if nativeModulePath == "" {
		return false
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(nativeDirectory), "go.mod"))
	return err == nil && modfile.ModulePath(data) == nativeModulePath
}

// localModule is a nested module the module at root replaces with a directory
// inside root.
type localModule struct{ Path, Dir string }

// localModules reads root's go.mod replace directives and returns the nested
// modules checked out under root, with slash-separated root-relative Dir.
func localModules(root string) ([]localModule, error) {
	path := filepath.Join(root, "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	file, err := modfile.Parse(path, data, nil)
	if err != nil {
		return nil, err
	}
	var result []localModule
	for _, replacement := range file.Replace {
		if replacement.New.Version != "" || filepath.IsAbs(replacement.New.Path) {
			continue
		}
		relative := filepath.ToSlash(filepath.Clean(filepath.FromSlash(replacement.New.Path)))
		if relative == "." || relative == ".." || strings.HasPrefix(relative, "../") {
			continue
		}
		nested, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative), "go.mod"))
		if err != nil || modfile.ModulePath(nested) != replacement.Old.Path {
			continue
		}
		result = append(result, localModule{Path: replacement.Old.Path, Dir: relative})
	}
	return result, nil
}
