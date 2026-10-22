package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestActualStorageInputRejectsWrongBoundsAndChangedPayload(t *testing.T) {
	root := t.TempDir()
	payload := filepath.Join(root, "storage_state")
	if err := os.WriteFile(payload, []byte("actual captured payload"), 0600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("actual captured payload"))
	input := request{StatePath: payload, StateSHA256: hex.EncodeToString(digest[:]), LogPath: payload, LogSHA256: hex.EncodeToString(digest[:]), Limits: bounds{32, 8, 8, 4 << 20, 1 << 16}}
	path := filepath.Join(root, "input.json")
	for _, test := range []struct {
		name   string
		mutate func(*request)
	}{
		{"changed-original-bound", func(value *request) { value.Limits.States++ }},
		{"changed-actual-state", func(value *request) { value.StateSHA256 = hex.EncodeToString(make([]byte, 32)) }},
		{"changed-actual-log", func(value *request) { value.LogSHA256 = hex.EncodeToString(make([]byte, 32)) }},
		{"relative-foreign-path", func(value *request) { value.StatePath = "relative/storage_state" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			value := input
			test.mutate(&value)
			encoded, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, encoded, 0600); err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			if err := project(path, &output); err == nil || output.Len() != 0 {
				t.Fatal("unbound evidence emitted a native catalogue")
			}
		})
	}
}
