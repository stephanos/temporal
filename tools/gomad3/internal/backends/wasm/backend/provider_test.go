package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/toolchain"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	stagedfs "go.temporal.io/server/tools/gomad3/internal/host/fs"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
)

func TestCachePublicationStagesAndValidatesBeforeReplacing(t *testing.T) {
	for _, fault := range []string{"module-stage", "receipt-stage", "module-validation", "receipt-validation", "cancel", "cleanup", "receipt-rename", "success"} {
		t.Run(fault, func(t *testing.T) {
			p := integrationProvider(t)
			source := t.TempDir()
			for name, data := range map[string]string{"go.mod": "module cache.stages.fixture\n\ngo 1.27.1\n", "main.go": "package main\nfunc main(){}\n"} {
				if err := os.WriteFile(filepath.Join(source, name), []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			failure := errors.New("cache fill failure")
			var paths []string
			var cache string
			var moduleMode os.FileMode
			stage := func(ctx context.Context, directory, pattern string, mode os.FileMode, fill func(context.Context, io.Writer) error) (*stagedfs.StagedFile, error) {
				cache = directory
				index := len(paths)
				if index == 0 {
					moduleMode = mode
					for name, data := range map[string]string{"module.wasm": "prior module", "receipt.json": "prior receipt"} {
						if err := os.WriteFile(filepath.Join(cache, name), []byte(data), 0600); err != nil {
							t.Fatal(err)
						}
					}
				}
				if fault == "module-stage" && index == 0 || (fault == "receipt-stage" || fault == "cleanup") && index == 1 {
					fill = func(context.Context, io.Writer) error { return failure }
				}
				staged, err := stagedfs.StageContext(ctx, directory, pattern, mode, fill)
				if err != nil {
					return nil, err
				}
				paths = append(paths, staged.Path())
				if filepath.Dir(staged.Path()) != directory {
					t.Fatal("cache stage is outside publication directory")
				}
				if index == 0 && fault == "module-validation" || index == 1 && fault == "receipt-validation" {
					if err := os.WriteFile(staged.Path(), []byte("changed exact stage"), mode); err != nil {
						t.Fatal(err)
					}
				}
				if index == 0 && fault == "cleanup" {
					if err := os.Remove(staged.Path()); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(staged.Path(), 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(staged.Path(), "child"), []byte("blocks removal"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if index == 1 && fault == "cancel" {
					cancel()
				}
				if index == 1 && fault == "receipt-rename" {
					if err := os.Remove(filepath.Join(cache, "receipt.json")); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(filepath.Join(cache, "receipt.json"), 0700); err != nil {
						t.Fatal(err)
					}
				}
				return staged, nil
			}
			prepared, err := p.prepareWithStager(ctx, target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: source, PreparationRoot: t.TempDir()}, stage)
			if fault == "success" {
				if err != nil {
					t.Fatal(err)
				}
				for name, mode := range map[string]os.FileMode{"module.wasm": moduleMode, "receipt.json": 0600} {
					info, err := os.Stat(filepath.Join(cache, name))
					if err != nil || info.Mode().Perm() != mode {
						t.Fatalf("cache mode changed: %s %v", name, err)
					}
				}
				if prepared.BuildKey != filepath.Base(cache) {
					t.Fatal("publication changed build identity")
				}
			} else if err == nil {
				t.Fatal("cache publication fault was ignored")
			}
			if fault == "module-stage" || fault == "receipt-stage" || fault == "cleanup" {
				if !errors.Is(err, failure) {
					t.Fatalf("primary fill error lost: %v", err)
				}
			}
			if fault == "cleanup" && !strings.Contains(err.Error(), "directory not empty") {
				t.Fatalf("real cleanup failure not joined: %v", err)
			}
			if fault == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation identity lost: %v", err)
			}
			if fault == "receipt-rename" {
				if !strings.Contains(err.Error(), "receipt (published=false)") || stringMustRead(t, filepath.Join(cache, "module.wasm")) == "prior module" {
					t.Fatalf("receipt failure did not follow module publication: %v", err)
				}
			} else if fault != "success" {
				if stringMustRead(t, filepath.Join(cache, "module.wasm")) != "prior module" || stringMustRead(t, filepath.Join(cache, "receipt.json")) != "prior receipt" {
					t.Fatal("prepublication failure replaced prior destinations")
				}
			}
			for _, path := range paths {
				if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) && fault != "cleanup" {
					t.Fatalf("unpublished stage remains: %s %v", path, err)
				}
			}
		})
	}
}

func TestCachePublicationCancellationLeavesNoCommitMarker(t *testing.T) {
	p := integrationProvider(t)
	source := t.TempDir()
	for name, data := range map[string]string{"go.mod": "module cache.cancel.fixture\n\ngo 1.27.1\n", "main.go": "package main\nfunc main(){}\n"} {
		if err := os.WriteFile(filepath.Join(source, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	environments := 0
	p.options.ObserveCompilerCommand = func(metrics CompilerCommandMetrics) {
		if metrics.Arguments[0] == "env" {
			environments++
			if environments == 2 {
				cancel()
			}
		}
	}
	_, err := p.Prepare(ctx, target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: source, PreparationRoot: t.TempDir()})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled cache publication returned %v", err)
	}
	markers, err := filepath.Glob(filepath.Join(p.options.CacheRoot, "*", "receipt.json"))
	if err != nil || len(markers) != 0 {
		t.Fatalf("cancelled publication committed receipt: %v %v", markers, err)
	}
}

func TestProvenanceAcceptsOrdinaryJSONSpellings(t *testing.T) {
	pr := provenance{Schema: provenanceSchema, Source: "<&", Args: []string{}, BuildTags: nil}
	encoded, err := json.Marshal(pr)
	if err != nil {
		t.Fatal(err)
	}
	var indented bytes.Buffer
	if err := json.Indent(&indented, encoded, "", "  "); err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{indented.Bytes(), append(bytes.Clone(encoded), '\n'), bytes.Replace(encoded, []byte(`"Schema":`), []byte(`"schema":`), 1), bytes.Replace(encoded, []byte(`\u003c`), []byte("<"), 1)} {
		decoded, err := decodeProvenance(data)
		if err != nil || !reflect.DeepEqual(decoded, pr) {
			t.Fatalf("ordinary provenance spelling rejected or changed values: %v", err)
		}
	}
}

func TestProvenanceRejectsStrictSyntaxAndUnknownFields(t *testing.T) {
	for _, data := range []string{`{"Schema":"a","Schema":"b"}`, `{"Config":{"Args":[],"Args":[]}}`, `{"Source":"` + string([]byte{0xff}) + `"}`, `{"unknown":1}`, `{} {}`, `{`, `{"ModuleBytes":18446744073709551616}`} {
		if _, err := decodeProvenance([]byte(data)); err == nil {
			t.Fatalf("invalid provenance admitted: %q", data)
		}
	}
}

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

func TestStockProvenanceRoundTripKeepsNilAndEmpty(t *testing.T) {
	for _, pr := range []provenance{{}, {Args: []string{}, BuildTags: []string{}, Sources: []sourceIdentity{}, Imports: []wasi.Import{}}} {
		data, err := json.Marshal(pr)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := decodeProvenance(data)
		if err != nil || !reflect.DeepEqual(decoded, pr) {
			t.Fatalf("stock provenance round trip changed typed values: %v", err)
		}
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

func TestValidatePreparedAcceptsCanonicalEmptySequences(t *testing.T) {
	options := portableProviderOptions(wasi.StockProfile)
	provider, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	module := []byte("module")
	pr := provenance{BuildCachePolicy: "private-build-key/v1", Schema: provenanceSchema, Profile: wasi.StockProfile, ReplayMode: record.ReplayObserved, CompilerSHA256: options.CompilerSHA256, GoVersion: options.GoVersion, HelperSHA256: options.HelperSHA256, ModelSHA256: options.ModelSHA256, Engine: options.Engine, MemoryBytes: options.MemoryBytes, Fuel: options.Fuel, Config: options.Config, ModuleSHA256: string(record.HashBytes(module)), ModuleBytes: uint64(len(module)), Kind: target.KindGoRun, Source: "fixture", Args: []string{}, BuildTags: []string{}}
	pr.BuildKey, err = buildKey(pr)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(pr)
	if err != nil {
		t.Fatal(err)
	}
	prepared := preparedFrom(pr, data)
	prepared.BuildTags = nil // Resume reconstructs an empty recorded array by cloning it from nil.
	prepared.Path = filepath.Join(t.TempDir(), "module.wasm")
	if err := os.WriteFile(prepared.Path, module, 0o500); err != nil {
		t.Fatal(err)
	}
	if err := provider.ValidatePrepared(target.Spec{Backend: Name, Kind: pr.Kind, Source: pr.Source}, prepared, nil); err != nil {
		t.Fatalf("semantically empty build tags changed prepared identity: %v", err)
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
