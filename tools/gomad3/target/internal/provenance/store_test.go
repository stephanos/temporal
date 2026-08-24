package provenance

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRoundTripsCanonicalPrivateDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "provenance.json")
	written, err := Store(path, map[string]any{"schema": "test/v1", "value": 7})
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Schema string `json:"schema"`
		Value  int    `json:"value"`
	}
	read, err := Load(path, 1024, &decoded)
	if err != nil {
		t.Fatal(err)
	}
	if string(read) != string(written) || decoded.Schema != "test/v1" || decoded.Value != 7 {
		t.Fatalf("Load() = %q, %#v", read, decoded)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}
