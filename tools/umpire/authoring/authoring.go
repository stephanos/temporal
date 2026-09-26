// Package authoring keeps model/AUTHORING.md and the Model file it walks through in step: every
// Lean block the walkthrough quotes is a marked region of the Model file, byte for byte, so the
// walkthrough cannot describe a Model that no longer compiles.
package authoring

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// A region of the Model file starts at a marker line and runs to the next marker or the end of the
// file; the walkthrough quotes it under an HTML comment carrying the same name, followed by one
// fenced Lean block.
var (
	regionMarker = regexp.MustCompile(`^-- authoring: ([a-z]+)$`)
	blockMarker  = regexp.MustCompile(`^<!-- authoring: ([a-z]+) -->$`)
)

// terminator is the marker that closes the last quoted region. It holds the Model file's namespace
// end and is not a region the walkthrough quotes.
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

// Check reports the first way the walkthrough and the Model file disagree: a block naming a marker
// the Model file lacks, a region the walkthrough does not quote, or a block whose bytes differ from
// its region. The terminator region is never quoted.
func Check(markdown, model string) error {
	regions, err := Regions(model)
	if err != nil {
		return err
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
			return fmt.Errorf("block %q names a marker the Model file lacks", name)
		}
		if blocks[name] != region {
			return fmt.Errorf("block %q differs from the Model file's region", name)
		}
	}
	for _, name := range sortedKeys(regions) {
		if name == terminator {
			continue
		}
		if _, quoted := blocks[name]; !quoted {
			return fmt.Errorf("region %q is not quoted by the walkthrough", name)
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
