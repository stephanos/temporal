package sqlitebusy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
)

func TestPrepare(t *testing.T) {
	source, err := os.ReadFile("testdata/conn.go.orig")
	if err != nil {
		t.Fatal(err)
	}
	dependency := Dependency{Path: "github.com/ncruces/go-sqlite3", Version: "v0.35.6"}
	for _, test := range []struct {
		name       string
		dependency Dependency
		source     []byte
		valid      bool
	}{
		{"pinned", dependency, source, true},
		{"module", Dependency{Path: "other", Version: dependency.Version}, source, false},
		{"version", Dependency{Path: dependency.Path, Version: "v0.35.5"}, source, false},
		{"replacement", Dependency{Path: dependency.Path, Version: dependency.Version, Replaced: true}, source, false},
		{"source-drift", dependency, append(bytes.Clone(source), '\n'), false},
		{"missing-source", dependency, nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			baseline := bytes.Clone(test.source)
			adapter, err := Prepare(test.dependency, test.source)
			if (err == nil) != test.valid {
				t.Fatalf("admission expected %v: %v", test.valid, err)
			}
			if !test.valid {
				return
			}
			replacement := sha256.Sum256(adapter.Source)
			if adapter.Policy != Policy || adapter.OriginalSHA256 != "85d9934324e945f1209c69eab06a81faa13f68ae157aee7a25f426736e1e3dd6" || adapter.ReplacementSHA256 != "ebd9d8f39c7b349a5d5b4586be4cb889daaa78ed146692c76ac5a308d109a50a" || adapter.ReplacementSHA256 != hex.EncodeToString(replacement[:]) {
				t.Fatal("adapter changed bytes beyond the reviewed cooperative branch or lost provenance")
			}
			adapter.Source[0] = 0
			if !bytes.Equal(test.source, baseline) {
				t.Fatal("adapter aliases source input")
			}
		})
	}
}
