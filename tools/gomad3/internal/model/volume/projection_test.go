package volume_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/model/volume/internal/projection"
)

func TestGeneratedProjectionMatchesNativeSources(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := projection.Generate(root)
	if err != nil {
		t.Fatal(err)
	}
	for path, expected := range outputs {
		actual, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actual, expected) {
			t.Errorf("%s is stale; run go generate ./volume", path)
		}
	}
}
