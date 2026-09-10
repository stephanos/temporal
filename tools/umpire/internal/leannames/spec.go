package leannames

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// SpecName is one backticked Lean name a document cites, with the Flow spec that
// owns it when the document marks it as planned rather than present.
type SpecName struct {
	Name string
	Line int
	// PlannedSpec is the Flow spec ID from a `(planned: fn-...)` marker in the same
	// block, or "" when the document claims the name exists today.
	PlannedSpec string
}

// modelName matches a dotted name rooted at one of the model tree's four libraries.
// A single segment such as `Umpire` is not a citation worth checking, so at least one
// dot is required.
var modelName = regexp.MustCompile(`^(?:Umpire|Testpilot|Temporal|Shared)(?:\.[A-Za-z0-9_'!?]+)+$`)

var backticked = regexp.MustCompile("`([^`\n]+)`")

var plannedMarker = regexp.MustCompile(`\(planned: (fn-[0-9]+-[a-z0-9-]+)\)`)

// blockStart matches the first line of a list item. A list has no blank line between
// its items, so the marker on one item must not leak into the next.
var blockStart = regexp.MustCompile(`^\s*[-*]\s`)

// ExtractSpecNames returns every backticked model name in document, in order, each
// carrying the planned marker of the block it appears in. A block is one list item or
// one blank-line-separated paragraph, so a rule marked planned covers the names in its
// own text and no others.
func ExtractSpecNames(document string) []SpecName {
	lines := strings.Split(document, "\n")
	blockOf := make([]int, len(lines))
	block := 0
	previousBlank := true
	for index, line := range lines {
		blank := strings.TrimSpace(line) == ""
		if blockStart.MatchString(line) || (previousBlank && !blank) {
			block++
		}
		blockOf[index] = block
		previousBlank = blank
	}

	plannedOf := map[int]string{}
	for index, line := range lines {
		if match := plannedMarker.FindStringSubmatch(line); match != nil {
			plannedOf[blockOf[index]] = match[1]
		}
	}

	var names []SpecName
	for index, line := range lines {
		for _, match := range backticked.FindAllStringSubmatch(line, -1) {
			candidate := match[1]
			if !modelName.MatchString(candidate) {
				continue
			}
			names = append(names, SpecName{
				Name:        candidate,
				Line:        index + 1,
				PlannedSpec: plannedOf[blockOf[index]],
			})
		}
	}
	return names
}

// Unresolved returns one message per citation a document makes that the tree does not
// back: a name that is not in the index, or a name marked planned whose owning Flow spec
// is not open. Messages are sorted, and an empty result means the document's mechanical
// half holds. `label` prefixes each message, so a caller can name the document.
func Unresolved(index *Index, names []SpecName, label, specsDirectory string) ([]string, error) {
	openSpec := map[string]bool{}
	var messages []string
	for _, name := range names {
		// A name that resolves is accepted whatever its block says. The planned marker
		// exempts a name the tree does not have yet; it must not stop checking the ones
		// it does, and it must not start failing real names when its owner closes.
		if index.Resolve(name.Name) {
			continue
		}
		if name.PlannedSpec == "" {
			messages = append(messages, fmt.Sprintf(
				"%s:%d: %s names no module, namespace, or declaration",
				label, name.Line, name.Name))
			continue
		}
		open, known := openSpec[name.PlannedSpec]
		if !known {
			var err error
			open, err = SpecIsOpen(specsDirectory, name.PlannedSpec)
			if err != nil {
				return nil, err
			}
			openSpec[name.PlannedSpec] = open
		}
		if !open {
			messages = append(messages, fmt.Sprintf(
				"%s:%d: %s is marked planned under %s, which is not open",
				label, name.Line, name.Name, name.PlannedSpec))
		}
	}
	slices.Sort(messages)
	return messages, nil
}

// SpecIsOpen reports whether the Flow spec record in specsDirectory is open. A planned
// term is only allowed while somebody owns delivering it, so a missing record and a
// closed one are both refusals rather than errors.
func SpecIsOpen(specsDirectory, specID string) (bool, error) {
	path := filepath.Join(specsDirectory, specID+".json")
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}
	var record struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(content, &record); err != nil {
		return false, fmt.Errorf("decode %s: %w", path, err)
	}
	return record.Status == "open", nil
}
