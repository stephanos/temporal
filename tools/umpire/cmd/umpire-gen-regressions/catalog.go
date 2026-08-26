package main

import (
	"fmt"
	"path"
	"strings"
)

const callerClosureIdentity = "workflow-nexus.query.exact-action-caller-closure"

type manifestEntry struct {
	Identity           string
	FixturePath        string
	GoOutputPath       string
	MarkdownOutputPath string
}

func productionManifest() []manifestEntry {
	return []manifestEntry{{
		Identity:           callerClosureIdentity,
		FixturePath:        "model/Temporal/Feature/Nexus/testdata/nexus-caller-closure-experiment-spec.json",
		GoOutputPath:       "tools/umpire/regression/catalog_generated_test.go",
		MarkdownOutputPath: "model/Temporal/Tool/Generated/Regressions.md",
	}}
}

func validateManifest(entries []manifestEntry) error {
	if len(entries) == 0 {
		return fmt.Errorf("projection manifest must contain at least one entry")
	}
	identities := make(map[string]struct{}, len(entries))
	ownedPaths := make(map[string]string, len(entries)*2)
	for _, entry := range entries {
		if strings.TrimSpace(entry.Identity) == "" {
			return fmt.Errorf("projection manifest identity is required")
		}
		if _, exists := identities[entry.Identity]; exists {
			return fmt.Errorf("projection manifest contains duplicate identity %q", entry.Identity)
		}
		identities[entry.Identity] = struct{}{}

		for _, candidate := range []struct {
			label string
			value string
		}{
			{label: "fixture", value: entry.FixturePath},
			{label: "Go output", value: entry.GoOutputPath},
			{label: "Markdown output", value: entry.MarkdownOutputPath},
		} {
			if err := validateRepositoryPath(candidate.value); err != nil {
				return fmt.Errorf("projection manifest %s for %q: %w", candidate.label, entry.Identity, err)
			}
		}
		if !strings.HasSuffix(entry.FixturePath, ".json") {
			return fmt.Errorf("projection manifest fixture for %q must be JSON", entry.Identity)
		}
		if !strings.HasSuffix(entry.GoOutputPath, "_test.go") {
			return fmt.Errorf("projection manifest Go output for %q must end in _test.go", entry.Identity)
		}
		if !strings.HasSuffix(entry.MarkdownOutputPath, ".md") {
			return fmt.Errorf("projection manifest Markdown output for %q must end in .md", entry.Identity)
		}
		for label, value := range map[string]string{
			"Go output":       entry.GoOutputPath,
			"Markdown output": entry.MarkdownOutputPath,
		} {
			if previous, exists := ownedPaths[value]; exists {
				return fmt.Errorf("projection manifest %s %q collides with %q", label, value, previous)
			}
			ownedPaths[value] = entry.Identity
		}
	}
	return nil
}

func validateRepositoryPath(value string) error {
	if value == "" || strings.ContainsRune(value, '\x00') || strings.Contains(value, "\\") ||
		strings.HasPrefix(value, "/") || hasWindowsVolume(value) || path.Clean(value) != value ||
		value == "." || value == ".." || strings.HasPrefix(value, "../") {
		return fmt.Errorf("repository-relative path %q is unsafe", value)
	}
	return nil
}

func hasWindowsVolume(value string) bool {
	return len(value) >= 2 && ((value[0] >= 'a' && value[0] <= 'z') ||
		(value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':'
}
