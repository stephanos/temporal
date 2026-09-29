// Package authoring keeps model/AUTHORING.md and the Model files it walks through in step: every
// Lean block the walkthrough quotes is a marked region of one of those files, byte for byte, so the
// walkthrough cannot describe a Model that no longer compiles.
package authoring

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// A region of a Model file starts at a marker line and runs to the next marker or the end of the
// file; the walkthrough quotes it under an HTML comment carrying the same name, followed by one
// fenced Lean block.
var (
	regionMarker = regexp.MustCompile(`^-- authoring: ([a-z]+)$`)
	blockMarker  = regexp.MustCompile(`^<!-- authoring: ([a-z]+) -->$`)
)

// terminator is the marker that closes a Model file's last quoted region. What follows it, at least
// the file's namespace end, is not a region the walkthrough quotes; every file has its own.
const terminator = "end"

// Regions returns the marked regions of a Model file by name, each trimmed of the blank lines that
// separate it from its markers. A marker that appears twice is an error, because two regions of one
// name would leave the walkthrough quoting either.
func Regions(model string) (map[string]string, error) {
	lines := strings.Split(model, "\n")
	regions := map[string]string{}
	name := ""
	start := 0
	flush := func(end int) {
		if name != "" {
			regions[name] = trimBlank(lines[start:end])
		}
	}
	for index, line := range lines {
		match := regionMarker.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		flush(index)
		if _, seen := regions[match[1]]; seen {
			return nil, fmt.Errorf("marker %q appears twice in the Model file", match[1])
		}
		name, start = match[1], index+1
	}
	flush(len(lines))
	return regions, nil
}

// Blocks returns the Lean blocks a walkthrough quotes by marker name. A marker without a fenced
// Lean block on the next line, or quoted twice, is an error.
func Blocks(markdown string) (map[string]string, error) {
	lines := strings.Split(markdown, "\n")
	blocks := map[string]string{}
	for index := 0; index < len(lines); index++ {
		match := blockMarker.FindStringSubmatch(lines[index])
		if match == nil {
			continue
		}
		name := match[1]
		if _, seen := blocks[name]; seen {
			return nil, fmt.Errorf("block %q is quoted twice in the walkthrough", name)
		}
		if index+1 >= len(lines) || lines[index+1] != "```lean" {
			return nil, fmt.Errorf("block %q is not followed by a fenced Lean block", name)
		}
		end := index + 2
		for end < len(lines) && lines[end] != "```" {
			end++
		}
		if end == len(lines) {
			return nil, fmt.Errorf("block %q is not closed", name)
		}
		blocks[name] = strings.Join(lines[index+2:end], "\n")
		index = end
	}
	return blocks, nil
}

// Check reports the first way the walkthrough and the Model files disagree: a block naming a marker
// no Model file has, a region the walkthrough does not quote, or a block whose bytes differ from its
// region. Models maps each file's path to its contents. A block is named by its marker alone, so a
// region name marked in two files is an error; each file's terminator region is never quoted.
func Check(markdown string, models map[string]string) error {
	regions := map[string]string{}
	files := map[string]string{}
	for _, path := range slices.Sorted(maps.Keys(models)) {
		fileRegions, err := Regions(models[path])
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		for name, region := range fileRegions {
			if name == terminator {
				continue
			}
			if other, seen := files[name]; seen {
				return fmt.Errorf("marker %q appears in both %s and %s", name, other, path)
			}
			regions[name], files[name] = region, path
		}
	}
	blocks, err := Blocks(markdown)
	if err != nil {
		return err
	}
	if len(blocks) == 0 {
		return errors.New("the walkthrough quotes no block")
	}
	for _, name := range sortedKeys(blocks) {
		region, present := regions[name]
		if !present {
			return fmt.Errorf("block %q names a marker no Model file has", name)
		}
		if blocks[name] != region {
			return fmt.Errorf("block %q differs from its region in %s", name, files[name])
		}
	}
	for _, name := range sortedKeys(regions) {
		if _, quoted := blocks[name]; !quoted {
			return fmt.Errorf("region %q of %s is not quoted by the walkthrough", name, files[name])
		}
	}
	return nil
}

func trimBlank(lines []string) string {
	start, end := 0, len(lines)
	for start < end && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return strings.Join(lines[start:end], "\n")
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
