package backend

import (
	"encoding/json"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestProvenanceRejectsChangedIdentityBeforeExecution(t *testing.T) {
	p := integrationProvider(t)
	pr := provenance{BuildCachePolicy: "private-build-key/v1", Schema: provenanceSchema, Profile: p.options.Config.Profile, ReplayMode: record.ReplayObserved, CompilerSHA256: p.options.CompilerSHA256, GoVersion: p.options.GoVersion, HelperSHA256: p.options.HelperSHA256, ModelSHA256: p.options.ModelSHA256, Engine: p.options.Engine, MemoryBytes: p.options.MemoryBytes, Fuel: p.options.Fuel, Config: p.options.Config, ModuleSHA256: string(record.HashBytes([]byte("module"))), ModuleBytes: 6, Kind: target.KindGoRun, Source: "fixture", Args: []string{}, BuildTags: []string{}}
	var err error
	pr.BuildKey, err = buildKey(pr)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(pr)
	if err != nil {
		t.Fatal(err)
	}
	prepared := preparedFrom(pr, encoded)
	if err := p.validateProvenance(pr, prepared); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		change func(*provenance)
	}{
		{"compiler", func(pr *provenance) { pr.CompilerSHA256 = string(record.HashBytes([]byte("different"))) }},
		{"helper", func(pr *provenance) { pr.HelperSHA256 = string(record.HashBytes([]byte("different"))) }},
		{"model", func(pr *provenance) { pr.ModelSHA256 = string(record.HashBytes([]byte("different"))) }},
		{"engine", func(pr *provenance) { pr.Engine.Version = "different" }},
		{"configuration", func(pr *provenance) { pr.Config.Clock.ReadStepNanos++ }},
		{"source", func(pr *provenance) { pr.Source = "different" }},
		{"module", func(pr *provenance) { pr.ModuleSHA256 = string(record.HashBytes([]byte("different"))) }},
		{"fuel", func(pr *provenance) { pr.Fuel++ }},
		{"memory", func(pr *provenance) { pr.MemoryBytes += 65536 }},
		{"exact", func(pr *provenance) { pr.ReplayMode = record.ReplayExact }},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := pr
			test.change(&changed)
			if err := p.validateProvenance(changed, prepared); err == nil {
				t.Fatal("changed provenance identity accepted")
			}
		})
	}
	options := p.options
	options.ModelSHA256 = string(record.HashBytes([]byte("caller invented")))
	if _, err := New(options); err == nil {
		t.Fatal("declared model identity was not compared to compiled implementation")
	}
}

func TestProviderRejectsEmptyModelIdentity(t *testing.T) {
	provider := integrationProvider(t)
	options := provider.options
	options.ModelSHA256 = ""
	if _, err := New(options); err == nil {
		t.Fatal("empty compiled-model identity accepted")
	}
}
