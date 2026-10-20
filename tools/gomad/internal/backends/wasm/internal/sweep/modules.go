package sweep

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

type moduleSelection struct {
	Directory string
	Patterns  []string
}

func discoverModules(checkout string, patterns []string) ([]moduleSelection, error) {
	modules := []string{checkout}
	err := filepath.WalkDir(filepath.Join(checkout, "tests"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == "vendor" || strings.HasPrefix(entry.Name(), ".")) {
			return filepath.SkipDir
		}
		if !entry.IsDir() && entry.Name() == "go.mod" {
			modules = append(modules, filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover test modules: %w", err)
	}
	sort.Strings(modules)
	selected := map[string][]string{}
	for _, pattern := range patterns {
		recursive := strings.HasSuffix(pattern, "/...")
		base := filepath.Join(checkout, filepath.FromSlash(strings.TrimPrefix(strings.TrimSuffix(pattern, "/..."), "./")))
		owner := checkout
		for _, module := range modules {
			if within(module, base) && len(module) > len(owner) {
				owner = module
			}
		}
		target, err := relativePattern(owner, base)
		if err != nil {
			return nil, err
		}
		if recursive {
			if target == "./." {
				target = "./..."
			} else {
				target += "/..."
			}
		}
		selected[owner] = append(selected[owner], target)
		if recursive {
			for _, module := range modules {
				if module != owner && within(base, module) {
					selected[module] = append(selected[module], "./...")
				}
			}
		}
	}
	var selections []moduleSelection
	for directory, targets := range selected {
		sort.Strings(targets)
		selections = append(selections, moduleSelection{Directory: directory, Patterns: slices.Compact(targets)})
	}
	sort.Slice(selections, func(i, j int) bool { return selections[i].Directory < selections[j].Directory })
	return selections, nil
}
