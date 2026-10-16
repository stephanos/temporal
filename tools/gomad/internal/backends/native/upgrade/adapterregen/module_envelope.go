package adapterregen

import (
	"fmt"
	"os"
	"path/filepath"
)

// copyModuleEnvelope retains the original local replacements without including
// outer repository sources in the generator's publication set.
func copyModuleEnvelope(root, stage string) (func() error, error) {
	unchanged := func() error { return nil }
	if _, err := os.Stat(filepath.Join(root, "internal/backends/native/go.mod")); os.IsNotExist(err) {
		return unchanged, nil
	} else if err != nil {
		return nil, err
	}
	if filepath.Base(stage) != "gomad" || filepath.Base(filepath.Dir(stage)) != "tools" {
		return nil, fmt.Errorf("module stage must retain tools/gomad depth: %s", stage)
	}
	read := map[string]string{}
	for _, name := range []string{"go.mod", "go.sum"} {
		source := filepath.Join(root, "..", "..", name)
		contents, err := os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("read module envelope: %w", err)
		}
		read[source] = digest(contents)
		if err := os.WriteFile(filepath.Join(stage, "..", "..", name), contents, 0o400); err != nil {
			return nil, fmt.Errorf("stage module envelope: %w", err)
		}
	}
	return func() error {
		for source, previous := range read {
			current, err := fileDigest(source)
			if err != nil {
				return err
			}
			if current != previous {
				return fmt.Errorf("%s changed", source)
			}
		}
		return nil
	}, nil
}
