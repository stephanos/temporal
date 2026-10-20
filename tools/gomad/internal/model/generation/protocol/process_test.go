package protocol

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadProcessSchemaRejectsInvalidDefinition(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "internal", "model", "schema", "processwire.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ name, contents string }{
		{"trailing", string(contents) + "{}"},
		{"unknown", strings.Replace(string(contents), `"version": 3`, `"version": 3, "unknown": true`, 1)},
		{"unsupported", strings.Replace(string(contents), `"version": 3`, `"version": 2`, 1)},
		{"prefix-order", strings.Replace(string(contents), `"name": "Profile"`, `"name": "Kind"`, 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "schema.json")
			if err := os.WriteFile(path, []byte(test.contents), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := readProcessSchema(path); err == nil {
				t.Fatal("invalid process definition was accepted")
			}
		})
	}
}
