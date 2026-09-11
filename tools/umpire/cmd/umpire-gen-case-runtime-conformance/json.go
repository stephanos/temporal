package main

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const persistedIndent = "  "

func marshalExpected(expected expectedResult) ([]byte, error) {
	encoded, err := json.MarshalIndent(expected, "", persistedIndent)
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

// persistedForm is how every generated Testpilot JSON artifact is stored: two-space indentation and
// exactly one trailing newline, so a Case or a conformance corpus change reads as a line diff.
//
// The renderer's own encoding policy is untouched -- it still emits compact canonical ProtoJSON, and
// indentation is presentation of the stored file. Compacting before indenting makes the form
// idempotent whatever whitespace the input carried; both passes preserve key order and string
// escapes exactly.
func persistedForm(encoded []byte) ([]byte, error) {
	var compact bytes.Buffer
	if err := json.Compact(&compact, encoded); err != nil {
		return nil, fmt.Errorf("compact JSON artifact: %w", err)
	}
	var indented bytes.Buffer
	if err := json.Indent(&indented, compact.Bytes(), "", persistedIndent); err != nil {
		return nil, fmt.Errorf("indent JSON artifact: %w", err)
	}
	return append(indented.Bytes(), '\n'), nil
}

// requirePersistedForm rejects a staged artifact that is valid JSON but not stored the way the
// generator writes it, naming the file. Without it a hand-edited or compact fixture would surface
// only as an unexplained `diff -ru` hunk.
func requirePersistedForm(path string, encoded []byte) error {
	expected, err := persistedForm(encoded)
	if err != nil {
		return fmt.Errorf("artifact %q is not valid JSON: %w", path, err)
	}
	if !bytes.Equal(expected, encoded) {
		return fmt.Errorf("artifact %q is not in persisted form: indent with %d spaces and end in exactly one newline",
			path, len(persistedIndent))
	}
	return nil
}
