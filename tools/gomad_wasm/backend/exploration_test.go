package backend

import (
	"encoding/json"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad_wasm/toolchain"
	"go.temporal.io/server/tools/gomad_wasm/wasi"
)

func TestOptionsObservationIsNotSerialized(t *testing.T) {
	for _, options := range []Options{{}, {ObserveExecution: func(ExecutionMetrics) {}, ObserveCompilerCommand: func(CompilerCommandMetrics) {}}} {
		encoded, err := json.Marshal(options)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), "ObserveExecution") {
			t.Fatal("host observation callback entered retained configuration")
		}
	}
}

func TestStockPreparationRejectsChoiceProfilesBeforeExecution(t *testing.T) {
	options := portableProviderOptions(wasi.StockProfile)
	provider, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	pr := provenance{BuildCachePolicy: "private-build-key/v1", Schema: provenanceSchema, Profile: wasi.StockProfile, ReplayMode: record.ReplayObserved, CompilerSHA256: options.CompilerSHA256, GoVersion: options.GoVersion, HelperSHA256: options.HelperSHA256, ModelSHA256: options.ModelSHA256, Engine: options.Engine, MemoryBytes: options.MemoryBytes, Fuel: options.Fuel, Config: options.Config, ModuleSHA256: string(record.HashBytes([]byte("module"))), ModuleBytes: 6, Kind: target.KindGoRun, Source: "fixture", Args: []string{}, BuildTags: []string{}}
	pr.BuildKey, err = buildKey(pr)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(pr)
	if err != nil {
		t.Fatal(err)
	}
	prepared := preparedFrom(pr, data)
	for _, profile := range []string{"GOMAD3_CHOICE_PROFILE=" + choice.Profile, choice.DiagnosticProfileEnvironment + "=" + choice.DiagnosticProfile} {
		err := provider.ValidatePrepared(target.Spec{Backend: Name, Kind: pr.Kind, Source: pr.Source, Args: []string{}}, prepared, []string{profile})
		if err == nil || !strings.Contains(err.Error(), "does not support runtime choices or diagnostics") {
			t.Fatalf("stock profile reached prepared module verification: %v", err)
		}
	}
}

func TestCooperativePreparationRequiresChoiceProfileBeforeExecution(t *testing.T) {
	options := portableProviderOptions(wasi.CooperativeProfile)
	provider, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	pr := provenance{BuildCachePolicy: "private-build-key/v1", Schema: provenanceSchema, Profile: wasi.CooperativeProfile, ReplayMode: record.ReplayExact, CompilerSHA256: options.CompilerSHA256, GoVersion: options.GoVersion, HelperSHA256: options.HelperSHA256, ModelSHA256: options.ModelSHA256, Engine: options.Engine, MemoryBytes: options.MemoryBytes, Fuel: options.Fuel, Config: options.Config, ModuleSHA256: string(record.HashBytes([]byte("module"))), ModuleBytes: 6, Kind: target.KindGoRun, Source: "fixture", Args: []string{}, BuildTags: []string{}, RuntimeSHA256: string(record.HashBytes([]byte("runtime"))), RuntimeImplementationSHA256: toolchain.ImplementationSHA256()}
	pr.BuildKey, err = buildKey(pr)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(pr)
	if err != nil {
		t.Fatal(err)
	}
	prepared := preparedFrom(pr, data)
	for _, environment := range [][]string{{"GOMAD3_IO_PROFILE=wasm"}, {"GOMAD3_CHOICE_PROFILE=invalid"}} {
		err := provider.ValidatePrepared(target.Spec{Backend: Name, Kind: pr.Kind, Source: pr.Source, Args: []string{}}, prepared, environment)
		if err == nil || !strings.Contains(err.Error(), "cooperative WASM preparation requires") {
			t.Fatalf("missing or invalid choices reached prepared module verification: %v", err)
		}
	}
}

func TestProviderRequiresAbsoluteCompiledModuleCache(t *testing.T) {
	options := portableProviderOptions(wasi.StockProfile)
	options.CompileCacheRoot = "relative-cache"
	if _, err := New(options); err == nil {
		t.Fatal("relative compiled module cache accepted")
	}
	options.CompileCacheRoot = t.TempDir()
	if _, err := New(options); err != nil {
		t.Fatal(err)
	}
}
