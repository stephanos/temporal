// Package casefile decides the byte forms a Testpilot JSON artifact is stored and exchanged in.
//
// The Lean renderer emits compact canonical ProtoJSON, which is the canonical form: the exploration
// bridge hands a Case out in it and a recorded Case is written in it. A checked-in fixture is that
// form re-indented with two spaces and one trailing newline, so a Case change reads as a line
// diff. Both passes preserve key order and string escapes exactly, so compaction inverts the
// persisted form and the canonical bytes are recoverable from either.
package casefile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// Indent is the persisted form's indentation.
const Indent = "  "

// Compact is the canonical form of encoded: its JSON with every insignificant byte removed.
func Compact(encoded []byte) ([]byte, error) {
	var compact bytes.Buffer
	if err := json.Compact(&compact, encoded); err != nil {
		return nil, fmt.Errorf("compact JSON artifact: %w", err)
	}
	return compact.Bytes(), nil
}

// Persisted is how a generated artifact is stored: compacted, then indented with Indent and ended
// with exactly one newline. It is idempotent whatever whitespace encoded carried.
func Persisted(encoded []byte) ([]byte, error) {
	compact, err := Compact(encoded)
	if err != nil {
		return nil, err
	}
	var indented bytes.Buffer
	if err := json.Indent(&indented, compact, "", Indent); err != nil {
		return nil, fmt.Errorf("indent JSON artifact: %w", err)
	}
	return append(indented.Bytes(), '\n'), nil
}

// ErrNoncanonical says the input is neither the canonical form (one trailing newline allowed) nor
// its persisted re-indentation: whitespace or layout that no writer of the form produces.
var ErrNoncanonical = errors.New("not the canonical compact form nor its persisted form")

// Canonical recovers the canonical compact bytes of input, which must be the compact form itself,
// optionally ended with one newline, or the persisted form; anything else is ErrNoncanonical.
func Canonical(input []byte) ([]byte, error) {
	compact, err := Compact(input)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(input, compact) || bytes.Equal(input, append(bytes.Clone(compact), '\n')) {
		return compact, nil
	}
	persisted, err := Persisted(compact)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(input, persisted) {
		return compact, nil
	}
	return nil, ErrNoncanonical
}
