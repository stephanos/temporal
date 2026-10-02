package main

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// signatureLinePrefix starts the one line a failing live assertion prints; the JSON after it is the
// whole interface between the harness and the live tests.
const signatureLinePrefix = "TESTPILOT-SIGNATURE "

// The reserved signatures carry only a test and an assertion: unparsed is a failure with nothing
// readable, process a test that started and never reached a pass or fail event.
const (
	reservedUnparsed = "unparsed"
	reservedProcess  = "process"
)

const timeoutPanicPrefix = "panic: test timed out"

// Signature is one failure's signature, in the canonical field order the spec names. Reserved,
// Location and Detail are the harness's own: Reserved is hashed so a reserved signature never
// merges into a parsed one, while Location and Detail are kept for reading only.
type Signature struct {
	Test            string           `json:"test"`
	Assertion       string           `json:"assertion"`
	RunDisposition  string           `json:"run_disposition,omitempty"`
	VerdictStatus   string           `json:"verdict_status,omitempty"`
	UnresolvedRules []UnresolvedRule `json:"unresolved_rules,omitempty"`
	Diagnostics     []Diagnostic     `json:"diagnostics,omitempty"`
	Leaks           []string         `json:"leaks,omitempty"`
	Reserved        string           `json:"reserved,omitempty"`
	Location        string           `json:"location,omitempty"`
	Detail          string           `json:"detail,omitempty"`
}

type UnresolvedRule struct {
	RuleID          string `json:"rule_id"`
	Status          string `json:"status"`
	TerminalStateID string `json:"terminal_state_id"`
}

type Diagnostic struct {
	Kind string `json:"kind"`
	Code string `json:"code"`
}

// canonicalSignature is what the hash covers. It is a struct, not a map, so the encoding's field
// order is fixed by the declaration.
type canonicalSignature struct {
	Test            string           `json:"test"`
	Assertion       string           `json:"assertion"`
	RunDisposition  string           `json:"run_disposition"`
	VerdictStatus   string           `json:"verdict_status"`
	UnresolvedRules []UnresolvedRule `json:"unresolved_rules"`
	Diagnostics     []Diagnostic     `json:"diagnostics"`
	Leaks           []string         `json:"leaks"`
	Reserved        string           `json:"reserved"`
}

// Hash is the signature's identity: every iteration-specific value normalized out, lists sorted,
// and the assertion's source location dropped, so repeated occurrences of one cause hash equal.
func (s Signature) Hash() (string, error) {
	assertion, _ := stripLocations(s.Assertion)
	canonical := canonicalSignature{
		Test:            s.Test,
		Assertion:       normalize(assertion),
		RunDisposition:  normalize(s.RunDisposition),
		VerdictStatus:   normalize(s.VerdictStatus),
		UnresolvedRules: []UnresolvedRule{},
		Diagnostics:     []Diagnostic{},
		Leaks:           []string{},
		Reserved:        s.Reserved,
	}
	for _, rule := range s.UnresolvedRules {
		canonical.UnresolvedRules = append(canonical.UnresolvedRules, UnresolvedRule{
			RuleID: normalize(rule.RuleID), Status: normalize(rule.Status), TerminalStateID: normalize(rule.TerminalStateID),
		})
	}
	slices.SortFunc(canonical.UnresolvedRules, func(a, b UnresolvedRule) int {
		return cmp.Or(cmp.Compare(a.RuleID, b.RuleID), cmp.Compare(a.Status, b.Status), cmp.Compare(a.TerminalStateID, b.TerminalStateID))
	})
	for _, diagnostic := range s.Diagnostics {
		canonical.Diagnostics = append(canonical.Diagnostics, Diagnostic{Kind: normalize(diagnostic.Kind), Code: normalize(diagnostic.Code)})
	}
	slices.SortFunc(canonical.Diagnostics, func(a, b Diagnostic) int {
		return cmp.Or(cmp.Compare(a.Kind, b.Kind), cmp.Compare(a.Code, b.Code))
	})
	for _, leak := range s.Leaks {
		canonical.Leaks = append(canonical.Leaks, normalize(leak))
	}
	slices.Sort(canonical.Leaks)
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("encode signature of %s: %w", s.Test, err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])[:12], nil
}

// The order matters: a timestamp holds what would otherwise read as ports and durations, and a UUID
// holds what would otherwise read as hex.
var normalizations = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:?\d{2})?( [A-Z]{3,4})?`), "<time>"},
	{regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`), "<uuid>"},
	{regexp.MustCompile(`-deleted-[0-9A-Za-z]+`), "-deleted-<suffix>"},
	{regexp.MustCompile(`\[[0-9a-fA-F:]*:[0-9a-fA-F:]*\]:\d+`), "<address>"},
	{regexp.MustCompile(`\b\d{1,3}(\.\d{1,3}){3}(:\d+)?\b`), "<address>"},
	{regexp.MustCompile(`\blocalhost:\d+\b`), "<address>"},
	{regexp.MustCompile(`(?i)\b0x[0-9a-f]+\b`), "<hex>"},
	{regexp.MustCompile(`(?i)\b[0-9a-f]{16,}\b`), "<hex>"},
	{regexp.MustCompile(`\b(\d+(\.\d+)?(ns|us|µs|ms|s|m|h))+\b`), "<duration>"},
	{regexp.MustCompile(`:\d{4,5}\b`), ":<port>"},
}

func normalize(text string) string {
	for _, normalization := range normalizations {
		text = normalization.pattern.ReplaceAllString(text, normalization.replacement)
	}
	return text
}

var (
	locationPattern        = regexp.MustCompile(`(?:[\w.@+-]*/)*[\w.@+-]+\.go:\d+(?::\d+)?:?`)
	leadingLocationPattern = regexp.MustCompile(`^(?:[\w.@+-]*/)*[\w.@+-]+\.go:\d+(?::\d+)?:\s*`)
)

// stripLocations removes every `file.go:line` from text and returns the first one it removed.
func stripLocations(text string) (stripped, location string) {
	location = strings.TrimSuffix(locationPattern.FindString(text), ":")
	stripped = strings.TrimSpace(locationPattern.ReplaceAllString(text, ""))
	return stripped, location
}

// outputLine is one line a test printed, with the kind test2json gave the event it came in.
type outputLine struct {
	text string
	kind string
}

// signatureOf builds the signature of one failing leaf test from its output. A signature line wins;
// the first one counts. Without one, the timeout panic, then the first failing assertion, stand in
// with every other field empty; with neither, the failure is unparsed.
func signatureOf(test string, lines []outputLine) Signature {
	for _, line := range lines {
		// A line printed with t.Log arrives decorated with the `file.go:line: ` it was logged from.
		text := leadingLocationPattern.ReplaceAllString(strings.TrimSpace(line.text), "")
		payload, ok := strings.CutPrefix(text, signatureLinePrefix)
		if !ok {
			continue
		}
		var signature Signature
		if err := json.Unmarshal([]byte(payload), &signature); err != nil {
			// A signature line that does not decode is not a signature; the fallback still reads the
			// assertion, and the raw line stays in the record.
			fallback := fallbackSignature(test, lines)
			fallback.Detail = strings.TrimSpace(line.text)
			return fallback
		}
		if signature.Test == "" {
			signature.Test = test
		}
		signature.Reserved = ""
		if signature.Location == "" {
			_, signature.Location = stripLocations(signature.Assertion)
		}
		return signature
	}
	return fallbackSignature(test, lines)
}

func fallbackSignature(test string, lines []outputLine) Signature {
	if timeout, ok := timeoutLine(lines); ok {
		return Signature{Test: test, Assertion: timeout}
	}
	if block := firstErrorBlock(lines); len(block) > 0 {
		assertion, location := readAssertion(block)
		if assertion != "" {
			return Signature{Test: test, Assertion: assertion, Location: location}
		}
	}
	if block := lastTestifyBlock(lines); len(block) > 0 {
		assertion, location := readAssertion(block)
		if assertion != "" {
			return Signature{Test: test, Assertion: assertion, Location: location}
		}
	}
	return Signature{Test: test, Reserved: reservedUnparsed, Detail: tail(lines, 5)}
}

// processSignature is the signature of a test that started and got no pass or fail event. The
// assertion is the timeout panic when the binary timed out, else the last line the test printed.
func processSignature(test string, lines, packageLines []outputLine) Signature {
	if timeout, ok := timeoutLine(lines); ok {
		return Signature{Test: test, Reserved: reservedProcess, Assertion: timeout}
	}
	if timeout, ok := timeoutLine(packageLines); ok {
		return Signature{Test: test, Reserved: reservedProcess, Assertion: timeout}
	}
	last := lastLine(lines)
	if last == "" {
		last = lastLine(packageLines)
	}
	return Signature{Test: test, Reserved: reservedProcess, Assertion: last}
}

func timeoutLine(lines []outputLine) (string, bool) {
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line.text); strings.HasPrefix(trimmed, timeoutPanicPrefix) {
			return trimmed, true
		}
	}
	return "", false
}

// firstErrorBlock is the first line test2json marked as an Error or Fatal, with its continuation.
func firstErrorBlock(lines []outputLine) []string {
	for index, line := range lines {
		if line.kind != "error" {
			continue
		}
		block := []string{line.text}
		for _, next := range lines[index+1:] {
			if next.kind != "error-continue" {
				break
			}
			block = append(block, next.text)
		}
		return block
	}
	return nil
}

// lastTestifyBlock finds the last testify failure in unmarked output, the way the CI re-runner's
// log reader does: the line before the last `Error Trace:` through the block's end.
func lastTestifyBlock(lines []outputLine) []string {
	for index := len(lines) - 1; index >= 0; index-- {
		if !strings.Contains(lines[index].text, "Error Trace:") {
			continue
		}
		start := index
		if index > 0 && strings.TrimSpace(lines[index-1].text) != "" {
			start = index - 1
		}
		var block []string
		for _, line := range lines[start:] {
			trimmed := strings.TrimSpace(line.text)
			if len(block) > 1 && (trimmed == "" || strings.HasPrefix(trimmed, "--- ") || strings.HasPrefix(trimmed, "=== ")) {
				break
			}
			block = append(block, line.text)
		}
		return block
	}
	return nil
}

var testifyKey = regexp.MustCompile(`^(Error Trace|Error|Test|Messages):\s*(.*)$`)

// readAssertion reads a failure block: a testify block yields its Error and Messages with the
// Error Trace as its location; any other block yields its message with the `file.go:line: ` prefix
// as the location.
func readAssertion(block []string) (assertion, location string) {
	fields := map[string][]string{}
	key := ""
	var plain []string
	for _, line := range block {
		trimmed := strings.TrimSpace(line)
		if match := testifyKey.FindStringSubmatch(trimmed); match != nil {
			key = match[1]
			if match[2] != "" {
				fields[key] = append(fields[key], strings.TrimSpace(match[2]))
			}
			continue
		}
		if key != "" {
			if trimmed != "" {
				fields[key] = append(fields[key], trimmed)
			}
			continue
		}
		if trimmed != "" {
			plain = append(plain, trimmed)
		}
	}
	if errorLines, ok := fields["Error"]; ok {
		assertion = strings.Join(errorLines, "\n")
		if messages := fields["Messages"]; len(messages) > 0 {
			assertion += "\nMessages: " + strings.Join(messages, "\n")
		}
		if trace := fields["Error Trace"]; len(trace) > 0 {
			location = strings.Fields(trace[0])[0]
		}
		return assertion, location
	}
	message := strings.Join(plain, "\n")
	stripped, location := stripLocations(message)
	if location == "" {
		return message, ""
	}
	return stripped, location
}

func lastLine(lines []outputLine) string {
	for index := len(lines) - 1; index >= 0; index-- {
		if trimmed := strings.TrimSpace(lines[index].text); trimmed != "" && !isFrame(trimmed) {
			return trimmed
		}
	}
	return ""
}

func isFrame(line string) bool {
	return strings.HasPrefix(line, "=== ") || strings.HasPrefix(line, "--- ")
}

func tail(lines []outputLine, count int) string {
	var kept []string
	for index := len(lines) - 1; index >= 0 && len(kept) < count; index-- {
		if trimmed := strings.TrimSpace(lines[index].text); trimmed != "" && !isFrame(trimmed) {
			kept = append(kept, trimmed)
		}
	}
	slices.Reverse(kept)
	return strings.Join(kept, "\n")
}
