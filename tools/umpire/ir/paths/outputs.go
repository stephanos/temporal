package paths

import (
	"os"
	"path/filepath"
	"strings"
)

const FeatureDirectory = "model/temporal/features"

// ModelPath resolves a logical IR name without changing the identity carried by its payload.
func ModelPath(root, name string) string {
	stem := strings.TrimSuffix(name, ".json")
	features := FeatureRoot(root)
	primary := filepath.Join(features, strings.ReplaceAll(stem, "-", "_"), "generated", "ir.json")
	var found string
	err := filepath.WalkDir(features, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if entry.Name() == "generated" {
			if ModelName(filepath.Join(path, "ir.json")) == name {
				found = filepath.Join(path, "ir.json")
				return filepath.SkipAll
			}
			legacy := filepath.Join(path, "legacy", name)
			if _, err := os.Stat(legacy); err == nil {
				found = legacy
				return filepath.SkipAll
			} else if !os.IsNotExist(err) {
				return err
			}
			return filepath.SkipDir
		}
		if ModelName(filepath.Join(path, "generated", "ir.json")) == name {
			primary = filepath.Join(path, "generated", "ir.json")
		}
		return nil
	})
	if err == nil && found != "" {
		return found
	}
	return primary
}

func FeatureRoot(root string) string {
	for _, path := range []string{filepath.Join(root, filepath.FromSlash(FeatureDirectory)), filepath.Join(root, "temporal", "features")} {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}
	absolute, err := filepath.Abs(root)
	if err == nil {
		return absolute
	}
	return root
}

// ModelName is the stable logical name used by lowering and historical manifests.
func ModelName(path string) string {
	if filepath.Base(path) != "ir.json" || filepath.Base(filepath.Dir(path)) != "generated" {
		return filepath.Base(path)
	}
	owner := filepath.Dir(filepath.Dir(path))
	parts := []string{filepath.Base(owner)}
	anchored := false
	for parent := filepath.Dir(owner); parent != filepath.Dir(parent); parent = filepath.Dir(parent) {
		if filepath.Base(parent) == "features" {
			anchored = true
			break
		}
		parts = append([]string{filepath.Base(parent)}, parts...)
	}
	// Direct temporary generated trees have no features ancestor.
	if !anchored {
		parts = []string{filepath.Base(owner)}
	}
	return strings.ReplaceAll(strings.Join(parts, "-"), "_", "-") + ".json"
}

func CasePath(root, model, query string) string {
	path := ModelPath(root, model)
	if filepath.Base(filepath.Dir(path)) == "legacy" {
		return filepath.Join(filepath.Dir(path), strings.TrimSuffix(model, ".json")+"-"+query+"-case.json")
	}
	return filepath.Join(filepath.Dir(path), query+".case.json")
}
