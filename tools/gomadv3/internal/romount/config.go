package romount

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Mapping struct {
	Source string
	Target string
}

func ParseMappings(values []string, workingDirectory string) ([]Mapping, error) {
	mappings := make([]Mapping, 0, len(values))
	for _, value := range values {
		source, target, found := strings.Cut(value, "=")
		if !found {
			return nil, fmt.Errorf("read-only mount %q must use HOST_DIRECTORY=TARGET_DIRECTORY", value)
		}
		if source == "" {
			return nil, fmt.Errorf("read-only mount source is required")
		}
		if target == "" || strings.IndexByte(target, 0) >= 0 || path.IsAbs(target) || path.Clean(target) == "." || path.Clean(target) == ".." || strings.HasPrefix(path.Clean(target), "../") {
			return nil, fmt.Errorf("invalid read-only mount target %q", target)
		}
		sourcePath := source
		if !filepath.IsAbs(sourcePath) {
			sourcePath = filepath.Join(workingDirectory, sourcePath)
		}
		sourcePath, err := filepath.Abs(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("resolve read-only mount source %q: %w", source, err)
		}
		info, err := os.Lstat(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("inspect read-only mount source %q: %w", source, err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("read-only mount source %q is not a directory", source)
		}
		mappings = append(mappings, Mapping{Source: filepath.Clean(sourcePath), Target: "/" + path.Clean(target)})
	}
	for left := range mappings {
		for right := left + 1; right < len(mappings); right++ {
			if overlaps(mappings[left].Target, mappings[right].Target) {
				return nil, fmt.Errorf("read-only mount target %q overlaps %q", mappings[left].Target, mappings[right].Target)
			}
		}
	}
	return mappings, nil
}

func overlaps(left, right string) bool {
	return left == right || strings.HasPrefix(left, right+"/") || strings.HasPrefix(right, left+"/")
}
