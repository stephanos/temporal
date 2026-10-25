// Package snapshot reads reviewed insta text snapshots for native Go tests.
package snapshot

import (
	"fmt"
	"os"
	"strings"
)

// Payload returns the text body without changing whitespace or body delimiters.
func Payload(snapshot string) (string, error) {
	header, ok := strings.CutPrefix(snapshot, "---\n")
	if !ok {
		return "", fmt.Errorf("missing snapshot header")
	}
	metadata, payload, ok := strings.Cut(header, "\n---\n")
	if !ok {
		return "", fmt.Errorf("unterminated snapshot header")
	}
	source := false
	for _, line := range strings.Split(metadata, "\n") {
		if strings.HasPrefix(line, "source: ") {
			source = true
		}
		if line == "snapshot_kind: binary" {
			return "", fmt.Errorf("invalid text snapshot header")
		}
	}
	if !source {
		return "", fmt.Errorf("invalid text snapshot header")
	}
	return payload, nil
}

// Read refuses pending candidates and returns a reviewed text snapshot's payload.
func Read(path string) (string, error) {
	if _, err := os.Stat(path + ".new"); err == nil {
		return "", fmt.Errorf("pending snapshot: %s.new", path)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return Payload(string(data))
}
