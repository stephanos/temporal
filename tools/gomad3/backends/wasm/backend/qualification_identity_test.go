package backend

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestQualificationIdentityRejectsStalePreparedSource(t *testing.T) {
	options := portableProviderOptions(wasi.StockProfile)
	provider, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "source.go")
	if err := os.WriteFile(path, []byte("original source"), 0600); err != nil {
		t.Fatal(err)
	}
	pr := provenance{BuildCachePolicy: "private-build-key/v1", Schema: provenanceSchema, Profile: wasi.StockProfile, ReplayMode: record.ReplayObserved, CompilerSHA256: options.CompilerSHA256, GoVersion: options.GoVersion, HelperSHA256: options.HelperSHA256, ModelSHA256: options.ModelSHA256, Engine: options.Engine, MemoryBytes: options.MemoryBytes, Fuel: options.Fuel, Config: options.Config, ModuleSHA256: string(record.HashBytes([]byte("module"))), ModuleBytes: 6, Kind: target.KindGoRun, Source: "fixture", Args: []string{}, BuildTags: []string{}, Sources: []sourceIdentity{{Path: path, SHA256: record.HashBytes([]byte("original source"))}}}
	pr.BuildKey, err = buildKey(pr)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(pr)
	if err != nil {
		t.Fatal(err)
	}
	prepared := preparedFrom(pr, data)
	if _, err := provider.QualificationIdentity(prepared); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed source"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.QualificationIdentity(prepared); err == nil {
		t.Fatal("retained preparation reused stale current source")
	}
}
