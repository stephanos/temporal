package backend

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad_wasm/toolchain"
	"go.temporal.io/server/tools/gomad_wasm/wasi"
)

func TestProviderAdmitsPinnedCooperativeProfile(t *testing.T) {
	options := portableProviderOptions(wasi.CooperativeProfile)
	if _, err := New(options); err != nil {
		t.Fatalf("pinned cooperative profile rejected: %v", err)
	}
	options.GoVersion = "go1.27.0"
	if _, err := New(options); err == nil {
		t.Fatal("cooperative runtime accepted a compiler outside its pinned version")
	}
}

func TestSelectedRuntimeOverlayInputsInvalidateBuildIdentity(t *testing.T) {
	directory := t.TempDir()
	original := filepath.Join(directory, "runtime.go")
	effective := filepath.Join(directory, "effective.go")
	virtual := filepath.Join(directory, "gomad_wasm.go")
	added := filepath.Join(directory, "added.go")
	for path, data := range map[string]string{original: "original runtime", effective: "effective runtime", added: "added runtime hook"} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	listed, err := json.Marshal(struct {
		Dir     string
		GoFiles []string
	}{directory, []string{"runtime.go", "gomad_wasm.go"}})
	if err != nil {
		t.Fatal(err)
	}
	replacements := map[string]string{original: effective, virtual: added}
	inputs, err := sourceInventory(listed, directory, replacements)
	if err != nil {
		t.Fatalf("selected virtual runtime file was not captured through its overlay: %v", err)
	}
	if len(inputs) != 3 {
		t.Fatalf("runtime inventory must bind original and effective inputs: %v", inputs)
	}
	initial, err := buildKey(provenance{Sources: inputs})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{original, effective, added} {
		t.Run(filepath.Base(path), func(t *testing.T) {
			prior, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := os.WriteFile(path, prior, 0600); err != nil {
					t.Error(err)
				}
			})
			if err := os.WriteFile(path, []byte("changed runtime input"), 0600); err != nil {
				t.Fatal(err)
			}
			inputs, err := sourceInventory(listed, directory, replacements)
			if err != nil {
				t.Fatal(err)
			}
			changed, err := buildKey(provenance{Sources: inputs})
			if err != nil || changed == initial {
				t.Fatalf("changed runtime input reused build identity: %v", err)
			}
		})
	}
}

func TestPreparedCooperativeTargetRequiresExactReplay(t *testing.T) {
	pr := provenance{Schema: provenanceSchema, Profile: wasi.CooperativeProfile, ReplayMode: record.ReplayExact, Kind: target.KindGoRun, Source: "fixture", Args: []string{}}
	data, err := json.Marshal(pr)
	if err != nil {
		t.Fatal(err)
	}
	prepared := preparedFrom(pr, data)
	if prepared.Backend.ReplayMode != record.ReplayExact {
		t.Fatal("cooperative target does not require exact replay")
	}
}

func TestCooperativeProvenanceRejectsChangedRuntimeImplementation(t *testing.T) {
	options := portableProviderOptions(wasi.CooperativeProfile)
	provider, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	pr := provenance{BuildCachePolicy: "private-build-key/v1", Schema: provenanceSchema, Profile: wasi.CooperativeProfile, ReplayMode: record.ReplayExact, CompilerSHA256: options.CompilerSHA256, GoVersion: options.GoVersion, HelperSHA256: options.HelperSHA256, ModelSHA256: options.ModelSHA256, Engine: options.Engine, MemoryBytes: options.MemoryBytes, Fuel: options.Fuel, Config: options.Config, ModuleSHA256: string(record.HashBytes([]byte("module"))), ModuleBytes: 6, Kind: target.KindGoRun, Source: "fixture", Args: []string{}, BuildTags: []string{}, RuntimeSHA256: string(record.HashBytes([]byte("runtime-source"))), RuntimeImplementationSHA256: toolchain.ImplementationSHA256()}
	encode := func() target.Prepared {
		t.Helper()
		pr.BuildKey, err = buildKey(pr)
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(pr)
		if err != nil {
			t.Fatal(err)
		}
		return preparedFrom(pr, data)
	}
	prepared := encode()
	if err := provider.validateProvenance(pr, prepared); err != nil {
		t.Fatal(err)
	}
	pr.RuntimeImplementationSHA256 = string(record.HashBytes([]byte("changed compiled runtime builder")))
	prepared = encode()
	if err := provider.validateProvenance(pr, prepared); err == nil {
		t.Fatal("changed runtime builder was accepted after recomputing provenance identity")
	}
}

func TestStockProvenancePreservesCanonicalBytes(t *testing.T) {
	const legacy = `{"BuildCachePolicy":"","BuildFlags":null,"BuildEnvironment":null,"Schema":"","Profile":"","ReplayMode":"","CompilerSHA256":"","GoVersion":"","HelperSHA256":"","ModelSHA256":"","ModuleSHA256":"","BuildKey":"","Engine":{"name":"","version":"","configuration":"","host_os":"","host_arch":""},"Sources":null,"Kind":"","Source":"","Args":null,"BuildTags":null,"Imports":null,"InitialMemoryPages":0,"MemoryBytes":0,"Fuel":0,"ModuleBytes":0,"Config":{"Profile":"","Args":null,"Environment":null,"WorkingDirectory":"","WritableDirectories":null,"CapturedInputs":{"Manifest":{"Bytes":0,"Entries":0,"File":"","Limits":{"DirectoryEntries":0,"Files":0,"PathBytes":0,"Requests":0,"SingleFileBytes":0,"TotalBytes":0},"Mappings":null,"NotExist":0,"Schema":"","SHA256":"","TotalBytes":0},"Descriptor":null,"Payloads":null},"Stdin":null,"EntropyKey":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"Clock":{"EpochNanos":0,"ReadStepNanos":0},"Limits":{"OutputBytes":0,"TranscriptBytes":0,"Calls":0,"Descriptors":0,"Files":0,"FilesystemBytes":0,"PendingEvents":0}}}`
	data, err := json.Marshal(provenance{})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != legacy {
		t.Fatalf("stock provenance canonical bytes changed: %s", data)
	}
}

func portableProviderOptions(profile string) Options {
	digest := string(record.HashBytes([]byte("portable-provider-input")))
	return Options{
		CompilerSHA256: digest, HelperSHA256: digest, ModelSHA256: wasi.ImplementationSHA256(),
		GoVersion: "go1.27.1", Engine: wasi.PinnedEngine(), MemoryBytes: 256 << 20, Fuel: 4000000000,
		Config: wasi.Config{
			Profile: profile, Environment: []string{"PWD=/workspace"}, WorkingDirectory: "/workspace",
			WritableDirectories: []string{"/workspace", "/tmp"},
			Clock:               wasi.ClockPolicy{EpochNanos: 946684800000000000, ReadStepNanos: 1000},
			Limits:              wasi.Limits{OutputBytes: 1 << 20, TranscriptBytes: 8 << 20, Calls: 10000, Descriptors: 64, Files: 1000, FilesystemBytes: 8 << 20, PendingEvents: 4096},
		},
	}
}
