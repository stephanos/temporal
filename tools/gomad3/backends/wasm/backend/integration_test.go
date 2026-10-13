package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	runnerbackend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/captured"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
	"go.temporal.io/server/tools/gomad3/target"
)

func integrationProvider(t *testing.T) *Provider {
	t.Helper()
	compiler := os.Getenv("GOMAD3_STOCK_GO")
	if compiler == "" {
		t.Fatal("explicit stock compiler is required for integration gate")
	}
	helper := os.Getenv("GOMAD_WASM_TEST_HELPER")
	if helper == "" {
		helper = "../wasmhost/target/release/gomad3-wasmhost"
	} else if !filepath.IsAbs(helper) {
		t.Fatal("GOMAD_WASM_TEST_HELPER must be absolute")
	}
	helper, err := filepath.Abs(helper)
	if err != nil {
		t.Fatal(err)
	}
	hash := func(path string) string {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(record.HashBytes(data))
	}
	config := wasi.Config{Profile: wasi.StockProfile, Environment: []string{"PWD=/workspace"}, WorkingDirectory: "/workspace", WritableDirectories: []string{"/workspace", "/tmp"}, Clock: wasi.ClockPolicy{EpochNanos: 946684800000000000, ReadStepNanos: 1000}, Limits: wasi.Limits{OutputBytes: 1 << 20, TranscriptBytes: 8 << 20, Calls: 10000, Descriptors: 64, Files: 1000, FilesystemBytes: 8 << 20, PendingEvents: 4096}}
	provider, err := New(Options{CompilerPath: compiler, CompilerSHA256: hash(compiler), GoVersion: "go1.27.1", HelperPath: helper, HelperSHA256: hash(helper), ModelSHA256: wasi.ImplementationSHA256(), CacheRoot: t.TempDir(), Engine: wasi.PinnedEngine(), MemoryBytes: 256 << 20, Fuel: 4000000000, Config: config})
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func TestIntegratedIntentionalFailureObservedReplay(t *testing.T) {
	provider := integrationProvider(t)
	fixture, err := filepath.Abs("../../native/conformance/testdata")
	if err != nil {
		t.Fatal(err)
	}
	result, err := runner.Explore(t.Context(), runner.CampaignSpec{Backend: provider, Seeds: "7", Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 3 * time.Minute, TerminateGrace: time.Millisecond * 100, OnFailure: runner.PolicyFirst, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, Artifacts: t.TempDir(), RunnerBuild: "task4-integration", Coverage: runner.CoverageNone, Target: target.Spec{Backend: Name, Kind: target.KindGoTest, Source: "./io_failure", WorkingDir: fixture, BuildTags: []string{"gomad_fixture", "test_dep"}, Args: []string{"-test.run=^TestDeterministicIOFailure$", "-test.v"}}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Failures != 1 || len(result.Artifacts) != 1 {
		t.Fatalf("intentional failure campaign: %#v", result)
	}
	replay, err := runner.Replay(t.Context(), runner.ReplaySpec{ArtifactPath: result.Artifacts[0], Backend: provider})
	if err != nil || !replay.Match || !replay.ObservedRepetition || replay.Artifact.Manifest.ReplayMode != record.ReplayObserved || replay.Artifact.Manifest.IOProfile.Transcript != nil {
		t.Fatalf("observed replay: %#v %v", replay, err)
	}
	if !strings.Contains(stringMustRead(t, filepath.Join(result.Artifacts[0], "stdout")), `deterministic failure after reading "pending"`) {
		t.Fatal("did not run original intentional failure fixture")
	}
	t.Logf("campaign=%s artifact=%s record=%s failure=%s", result.CampaignPath, result.Artifacts[0], replay.Artifact.Manifest.RecordHash, replay.Artifact.Manifest.Outcome.FailureSignature)
}
func stringMustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

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

func TestTestSourceChangesInvalidateCompilationCache(t *testing.T) {
	provider := integrationProvider(t)
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "go.mod"), []byte("module cache.fixture\n\ngo 1.27.1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(source, "fixture_test.go")
	firstSource := []byte("package fixture\nimport \"testing\"\nfunc TestCache(t *testing.T){t.Log(\"first\")}\n")
	if err := os.WriteFile(file, firstSource, 0600); err != nil {
		t.Fatal(err)
	}
	spec := target.Spec{Backend: Name, Kind: target.KindGoTest, Source: ".", WorkingDir: source, PreparationRoot: filepath.Join(t.TempDir(), "first"), BuildTags: []string{"test_dep"}}
	first, err := provider.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("package fixture\nimport \"testing\"\nfunc TestCache(t *testing.T){t.Log(\"second\")}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	spec.PreparationRoot = filepath.Join(t.TempDir(), "second")
	second, err := provider.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if first.BuildKey == second.BuildKey || first.SHA256 == second.SHA256 {
		t.Fatal("changed selected test source reused stale compilation cache")
	}
	spec.PreparationRoot = filepath.Join(t.TempDir(), "third")
	third, err := provider.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if third.BuildKey != second.BuildKey || third.SHA256 != second.SHA256 {
		t.Fatal("unchanged source did not reuse verified cache identity")
	}
	module := filepath.Join(provider.options.CacheRoot, third.BuildKey, "module.wasm")
	if err := os.Chmod(module, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(module, []byte("corrupt module"), 0600); err != nil {
		t.Fatal(err)
	}
	spec.PreparationRoot = filepath.Join(t.TempDir(), "corrupt")
	if _, err := provider.Prepare(t.Context(), spec); err == nil {
		t.Fatal("corrupt module compilation cache accepted")
	}
}

func TestAssemblyHeaderChangesInvalidateCompilationCache(t *testing.T) {
	for _, test := range []struct {
		name, directive string
		nested, shadow  bool
	}{
		{"selected-header", "#include \"value.h\"", false, false},
		{"nested-include", "#include \"value.h\"", true, false},
		{"adjacent-include", "#include\"value.h\"", true, false},
		{"comment-include", "#/**/include \"value.h\"", true, false},
		{"local-generated-header-shadow", "#include \"go_asm.h\"", true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider := integrationProvider(t)
			source := t.TempDir()
			files := map[string]string{
				"go.mod":  "module cache.header.fixture\n\ngo 1.27.1\n",
				"main.go": "package main\nimport \"fmt\"\nvar Value uint64\nfunc main(){fmt.Println(Value)}\n",
				"value.s": "#include \"textflag.h\"\n" + test.directive + "\nDATA ·Value+0(SB)/8, $FIXTURE_VALUE\nGLOBL ·Value(SB), NOPTR, $8\n",
				"value.h": "#define FIXTURE_VALUE 1\n",
			}
			header := "value.h"
			if test.nested {
				files["value.h"] = "#include \"nested/value.h\"\n"
				files["nested/value.h"] = "#define FIXTURE_VALUE 1\n"
				header = "nested/value.h"
			}
			if test.shadow {
				files["go_asm.h"] = files["value.h"]
				delete(files, "value.h")
			}
			for path, data := range files {
				path = filepath.Join(source, path)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			spec := target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: source, PreparationRoot: filepath.Join(t.TempDir(), "first"), BuildTags: []string{"test_dep"}}
			first, err := provider.Prepare(t.Context(), spec)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(source, header), []byte("#define FIXTURE_VALUE 2\n"), 0600); err != nil {
				t.Fatal(err)
			}
			spec.PreparationRoot = filepath.Join(t.TempDir(), "second")
			second, err := provider.Prepare(t.Context(), spec)
			if err != nil {
				t.Fatal(err)
			}
			if first.BuildKey == second.BuildKey || first.SHA256 == second.SHA256 {
				t.Fatal("changed assembly header reused stale compilation cache")
			}
		})
	}
}

func TestStockPreparationDisablesAmbientPGO(t *testing.T) {
	provider := integrationProvider(t)
	source := t.TempDir()
	for path, data := range map[string]string{"go.mod": "module cache.pgo.fixture\n\ngo 1.27.1\n", "main.go": "package main\nfunc main(){}\n"} {
		if err := os.WriteFile(filepath.Join(source, path), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	spec := target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: source, PreparationRoot: filepath.Join(t.TempDir(), "first"), BuildTags: []string{"test_dep"}}
	first, err := provider.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(source, "default.pgo")
	for _, data := range []string{"malformed ambient profile", "changed malformed ambient profile", ""} {
		if data == "" {
			if err := os.Remove(profile); err != nil {
				t.Fatal(err)
			}
		} else if err := os.WriteFile(profile, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		spec.PreparationRoot = filepath.Join(t.TempDir(), "next")
		options := provider.options
		options.CacheRoot = filepath.Join(t.TempDir(), "uncached")
		uncached, err := New(options)
		if err != nil {
			t.Fatal(err)
		}
		prepared, err := uncached.Prepare(t.Context(), spec)
		if err != nil {
			t.Fatalf("disabled ambient PGO affected preparation: %v", err)
		}
		if prepared.BuildKey != first.BuildKey || prepared.SHA256 != first.SHA256 {
			t.Fatal("disabled ambient PGO changed build identity")
		}
		pr, err := decodeProvenance(prepared.BackendPayloads[0].Data)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(pr.BuildFlags, "-pgo=off") {
			t.Fatal("disabled PGO not bound by provenance")
		}
	}
}

func TestStockPreparationClearsAmbientCompilerSettings(t *testing.T) {
	t.Setenv("GOWASM", "ambient-unknown-feature")
	t.Setenv("GODEBUG", "gocachehash=1")
	environment := buildEnvironment()
	for _, name := range []string{"GOWASM", "GODEBUG"} {
		var values []string
		for _, value := range environment {
			if key, setting, _ := strings.Cut(value, "="); key == name {
				values = append(values, setting)
			}
		}
		if !slices.Equal(values, []string{""}) {
			t.Fatalf("%s reaches the compiler: %q", name, values)
		}
	}
	provider := integrationProvider(t)
	source := t.TempDir()
	for path, data := range map[string]string{"go.mod": "module cache.environment.fixture\n\ngo 1.27.1\n", "main.go": "package main\nfunc main(){}\n"} {
		if err := os.WriteFile(filepath.Join(source, path), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	options := provider.options
	options.CacheRoot = filepath.Join(t.TempDir(), "uncached")
	uncached, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uncached.Prepare(t.Context(), target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: source, PreparationRoot: t.TempDir(), BuildTags: []string{"test_dep"}}); err != nil {
		t.Fatalf("ambient compiler settings affected preparation: %v", err)
	}
}

func TestInterruptedCachePublicationIsRebuilt(t *testing.T) {
	provider := integrationProvider(t)
	source := t.TempDir()
	for path, data := range map[string]string{"go.mod": "module cache.publication.fixture\n\ngo 1.27.1\n", "main.go": "package main\nfunc main(){}\n"} {
		if err := os.WriteFile(filepath.Join(source, path), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	options := provider.options
	options.CacheRoot = filepath.Join(t.TempDir(), "cache")
	isolated, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	spec := target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: source, PreparationRoot: filepath.Join(t.TempDir(), "first"), BuildTags: []string{"test_dep"}}
	first, err := isolated.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	receipt := filepath.Join(options.CacheRoot, first.BuildKey, "receipt.json")
	if err := os.Remove(receipt); err != nil {
		t.Fatal(err)
	}
	spec.PreparationRoot = filepath.Join(t.TempDir(), "second")
	second, err := isolated.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatalf("module without receipt broke its build key: %v", err)
	}
	if second.BuildKey != first.BuildKey || second.SHA256 != first.SHA256 {
		t.Fatal("rebuilt interrupted publication changed identity")
	}
	if _, err := os.Stat(receipt); err != nil {
		t.Fatalf("rebuild did not publish its receipt: %v", err)
	}
}

func TestReplayRejectsChangedArtifactIdentityBeforeGuest(t *testing.T) {
	provider := integrationProvider(t)
	fixture, err := filepath.Abs("../../native/conformance/testdata")
	if err != nil {
		t.Fatal(err)
	}
	campaign, err := runner.Explore(t.Context(), runner.CampaignSpec{Backend: provider, Seeds: "7", Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 3 * time.Minute, TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyFirst, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, Artifacts: t.TempDir(), RunnerBuild: "task4-integration", Coverage: runner.CoverageNone, Target: target.Spec{Backend: Name, Kind: target.KindGoTest, Source: "./io_failure", WorkingDir: fixture, BuildTags: []string{"gomad_fixture", "test_dep"}, Args: []string{"-test.run=^TestDeterministicIOFailure$", "-test.v"}}})
	if err != nil || len(campaign.Artifacts) != 1 {
		t.Fatalf("discovery: %#v %v", campaign, err)
	}
	opened, err := artifact.OpenArtifact(campaign.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	}()
	original := opened.Manifest()
	input := artifact.ArtifactInput{Manifest: original, TargetPath: filepath.Join(campaign.Artifacts[0], "target")}
	input.Stdout, err = opened.ReadPayload("stdout", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	input.Stderr, err = opened.ReadPayload("stderr", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	for _, reference := range record.BackendReferences(*original.Target.Backend) {
		data, err := opened.ReadPayload(reference.File, uint64(reference.Bytes))
		if err != nil {
			t.Fatal(err)
		}
		input.BackendPayloads = append(input.BackendPayloads, target.BackendPayload{Reference: reference, Data: data})
	}
	input.World.Initial, err = opened.ReadPayload(original.World.Initial.File, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	input.World.Transitions, err = opened.ReadPayload(original.World.Transitions.File, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	input.World.Final, err = opened.ReadPayload(original.World.Final.File, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	pr, err := decodeProvenance(input.BackendPayloads[0].Data)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		change func(*provenance)
	}{
		{"compiler", func(p *provenance) { p.CompilerSHA256 = string(record.HashBytes([]byte("changed"))) }},
		{"helper", func(p *provenance) { p.HelperSHA256 = string(record.HashBytes([]byte("changed"))) }},
		{"engine", func(p *provenance) { p.Engine.Configuration = "changed" }},
		{"model", func(p *provenance) { p.ModelSHA256 = string(record.HashBytes([]byte("changed"))) }},
		{"profile", func(p *provenance) { p.Profile = "changed" }},
		{"imports", func(p *provenance) { p.Imports = nil }},
		{"memory", func(p *provenance) { p.InitialMemoryPages++ }},
		{"limits", func(p *provenance) { p.Fuel++ }},
		{"inputs", func(p *provenance) { p.Config.Stdin = []byte("changed") }},
		{"argv", func(p *provenance) { p.Args = []string{"changed"} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := pr
			test.change(&changed)
			data, err := json.Marshal(changed)
			if err != nil {
				t.Fatal(err)
			}
			modified := input
			modified.Manifest = opened.Manifest()
			modified.Manifest.Target.Backend.Provenance.SHA256 = record.HashBytes(data)
			modified.Manifest.Target.Backend.Provenance.Bytes = record.Uint64String(len(data))
			modified.Manifest.IOProfile = record.BackendIOProfile(*modified.Manifest.Target.Backend)
			modified.BackendPayloads = append([]target.BackendPayload(nil), input.BackendPayloads...)
			modified.BackendPayloads[0] = target.BackendPayload{Reference: modified.Manifest.Target.Backend.Provenance, Data: data}
			published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, modified)
			if err != nil {
				t.Fatal(err)
			}
			counted := &countedProvider{Provider: provider}
			if _, err := runner.Replay(t.Context(), runner.ReplaySpec{Backend: counted, ArtifactPath: published.Path}); err == nil {
				t.Fatal("changed identity was accepted")
			}
			if counted.calls != 0 {
				t.Fatal("identity rejection occurred after guest execution")
			}
		})
	}
	if _, err := runner.Replay(t.Context(), runner.ReplaySpec{ArtifactPath: campaign.Artifacts[0]}); err == nil {
		t.Fatal("provider absence selected native fallback")
	}
	for _, file := range []string{"target", original.Target.Backend.Provenance.File, original.Target.Backend.Evidence.File, "stdout"} {
		t.Run("corrupt-"+file, func(t *testing.T) {
			copyRoot := t.TempDir()
			for _, entry := range original.Files {
				source := filepath.Join(campaign.Artifacts[0], filepath.FromSlash(entry.Path))
				data, err := os.ReadFile(source)
				if err != nil {
					t.Fatal(err)
				}
				mode := os.FileMode(0600)
				if entry.Path == "target" {
					mode = 0700
				}
				destination := filepath.Join(copyRoot, filepath.FromSlash(entry.Path))
				if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(destination, data, mode); err != nil {
					t.Fatal(err)
				}
			}
			data, err := os.ReadFile(filepath.Join(campaign.Artifacts[0], "manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(copyRoot, "manifest.json"), data, 0600); err != nil {
				t.Fatal(err)
			}
			corrupted := filepath.Join(copyRoot, filepath.FromSlash(file))
			data, err = os.ReadFile(corrupted)
			if err != nil {
				t.Fatal(err)
			}
			data[0] ^= 1
			if err := os.WriteFile(corrupted, data, 0600); err != nil {
				t.Fatal(err)
			}
			counted := &countedProvider{Provider: provider}
			if _, err := runner.Replay(t.Context(), runner.ReplaySpec{Backend: counted, ArtifactPath: copyRoot}); err == nil {
				t.Fatal("corrupt payload accepted")
			}
			if counted.calls != 0 {
				t.Fatal("corruption reached guest")
			}
		})
	}
}

type countedProvider struct {
	*Provider
	calls        int
	preparedRoot string
}

// Owner gate budget (2026-10-12): fresh helper replays of the retained artifact.
const artifactFreshReplays = 20

func (p *countedProvider) Run(ctx context.Context, request runnerbackend.Request) (runnerbackend.Result, error) {
	p.calls++
	return p.Provider.Run(ctx, request)
}
func (p *countedProvider) Prepare(ctx context.Context, spec target.Spec) (target.Prepared, error) {
	prepared, err := p.Provider.Prepare(ctx, spec)
	if err != nil {
		return prepared, err
	}
	if p.preparedRoot != "" {
		data, err := os.ReadFile(prepared.Path)
		if err != nil {
			return target.Prepared{}, err
		}
		if err := os.MkdirAll(p.preparedRoot, 0700); err != nil {
			return target.Prepared{}, err
		}
		if err := os.WriteFile(filepath.Join(p.preparedRoot, "module.wasm"), data, 0500); err != nil {
			return target.Prepared{}, err
		}
		for _, payload := range prepared.BackendPayloads {
			if err := os.WriteFile(filepath.Join(p.preparedRoot, filepath.Base(payload.Reference.File)), payload.Data, 0600); err != nil {
				return target.Prepared{}, err
			}
		}
	}
	return prepared, nil
}

func TestIntegratedFailureArtifactFreshObservedReplays(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_TASK4_EVIDENCE_ROOT")
	if root == "" {
		t.Skip("explicit retained evidence root selects the fresh-replay artifact gate")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("evidence root must be absolute")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	base := integrationProvider(t)
	options := base.options
	options.CacheRoot = filepath.Join(root, "cache")
	originalInputs := filepath.Join(t.TempDir(), "captured")
	if err := os.Mkdir(originalInputs, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(originalInputs, "input.txt"), []byte("immutable captured control"), 0600); err != nil {
		t.Fatal(err)
	}
	captured, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: originalInputs, Target: "/inputs"}}, captured.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	options.Config.CapturedInputs = captured
	provider, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	counted := &countedProvider{Provider: provider, preparedRoot: filepath.Join(root, "prepared")}
	fixture, err := filepath.Abs("../../native/conformance/testdata")
	if err != nil {
		t.Fatal(err)
	}
	campaign, err := runner.Explore(t.Context(), runner.CampaignSpec{Backend: counted, Seeds: "7", Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 3 * time.Minute, TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyFirst, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, Artifacts: filepath.Join(root, "artifacts"), RunnerBuild: "task4-integration", Coverage: runner.CoverageNone, Target: target.Spec{Backend: Name, Kind: target.KindGoTest, Source: "./io_failure", WorkingDir: fixture, BuildTags: []string{"gomad_fixture", "test_dep"}, Args: []string{"-test.run=^TestDeterministicIOFailure$", "-test.v"}}})
	if err != nil || campaign.Failures != 1 || len(campaign.Artifacts) != 1 {
		t.Fatalf("intentional failure: %#v %v", campaign, err)
	}
	if err := os.RemoveAll(originalInputs); err != nil {
		t.Fatal(err)
	}
	options.CompilerPath = filepath.Join(root, "compiler-deliberately-unavailable")
	replayProvider, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	fresh := &countedProvider{Provider: replayProvider}
	opened, err := artifact.OpenArtifact(campaign.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	manifest := opened.Manifest()
	expectedStdout, err := opened.ReadPayload("stdout", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	expectedStderr, err := opened.ReadPayload("stderr", 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	reference := manifest.Target.Backend.Evidence
	expectedEvidence, err := opened.ReadPayload(reference.File, uint64(reference.Bytes))
	if err != nil {
		t.Fatal(err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	if manifest.ReplayMode != record.ReplayObserved || manifest.IOProfile.Transcript != nil {
		t.Fatal("stock observed artifact claimed native exact transcript")
	}
	receipts := []map[string]any{}
	for i := 0; i < artifactFreshReplays; i++ {
		observed := filepath.Join(root, "observed", fmt.Sprintf("%03d", i+1))
		result, err := runner.Replay(t.Context(), runner.ReplaySpec{Backend: fresh, ArtifactPath: campaign.Artifacts[0], ObservedDir: observed})
		if err != nil || !result.Match || !result.ObservedRepetition || result.ChoiceReplayStatus != runner.ChoiceReplayNone {
			t.Fatalf("fresh replay %d: %#v %v", i+1, result, err)
		}
		for _, payload := range []struct {
			name     string
			expected []byte
		}{{"stdout", expectedStdout}, {"stderr", expectedStderr}, {"backend-evidence.bin", expectedEvidence}} {
			actual, err := os.ReadFile(filepath.Join(observed, payload.name))
			if err != nil || !bytes.Equal(actual, payload.expected) {
				t.Fatalf("fresh replay %d %s bytes differ: %v", i+1, payload.name, err)
			}
		}
		receipts = append(receipts, map[string]any{"fresh": i + 1, "stdout_sha256": record.HashBytes(expectedStdout), "stderr_sha256": record.HashBytes(expectedStderr), "evidence_sha256": record.HashBytes(expectedEvidence), "match": true, "capability": record.ReplayObserved})
		t.Logf("fresh replay %d/%d exact raw output/evidence equality; capability=%s", i+1, artifactFreshReplays, record.ReplayObserved)
	}
	if counted.calls != 1 || fresh.calls != artifactFreshReplays {
		t.Fatalf("actual helper calls discovery=%d fresh=%d", counted.calls, fresh.calls)
	}
	receipt := map[string]any{"campaign": campaign.CampaignPath, "artifact": campaign.Artifacts[0], "discovery_executions": counted.calls, "fresh_replay_executions": fresh.calls, "record_hash": manifest.RecordHash, "failure_signature": manifest.Outcome.FailureSignature, "module_sha256": manifest.Target.SHA256, "provenance_sha256": manifest.Target.Backend.Provenance.SHA256, "helper_sha256": options.HelperSHA256, "compiler_sha256": options.CompilerSHA256, "model_sha256": options.ModelSHA256, "captured_input_sha256": captured.Manifest.SHA256, "original_inputs_removed": true, "compiler_unavailable": true, "replays": receipts}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "receipt.json"), append(encoded, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestObservedCampaignResumeRetainsBackendAndOrdinals(t *testing.T) {
	provider := integrationProvider(t)
	fixture, err := filepath.Abs("../../native/conformance/testdata")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	config := runner.CampaignSpec{Backend: &resumeBoundaryProvider{Provider: provider}, Seeds: "7-8", Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 3 * time.Minute, TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyAll, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, Artifacts: t.TempDir(), RunnerBuild: "task4-integration", Coverage: runner.CoverageNone, ProgressInterval: time.Millisecond, Target: target.Spec{Backend: Name, Kind: target.KindGoTest, Source: "./io_failure", WorkingDir: fixture, BuildTags: []string{"gomad_fixture", "test_dep"}, Args: []string{"-test.run=^TestDeterministicIOFailure$", "-test.v"}}, Progress: func(event runner.CampaignEvent) error {
		if event.Attempted == 1 {
			cancel()
			return nil
		}
		return nil
	}}
	partial, err := runner.Explore(ctx, config)
	if !errors.Is(err, context.Canceled) || partial.Failures != 1 {
		t.Fatalf("partial campaign: %#v %v", partial, err)
	}
	resumed, err := runner.Resume(t.Context(), runner.ResumeSpec{Backend: provider, CampaignPath: partial.CampaignPath, RunnerBuild: config.RunnerBuild})
	if err != nil || resumed.Attempted != 2 || resumed.Failures != 2 || resumed.StopReason != runner.StopSeedsExhausted {
		t.Fatalf("resumed observed campaign: %#v %v", resumed, err)
	}
}

type resumeBoundaryProvider struct{ *Provider }

func (p *resumeBoundaryProvider) Run(ctx context.Context, request runnerbackend.Request) (runnerbackend.Result, error) {
	if request.Seed == 8 {
		<-ctx.Done()
		return runnerbackend.Result{Termination: runnerbackend.Infrastructure, Cancelled: true, Reaped: true}, nil
	}
	return p.Provider.Run(ctx, request)
}

func TestIntegratedCooperativeFailureArtifactDetachedReplay(t *testing.T) {
	stock := integrationProvider(t)
	options := stock.options
	root := t.TempDir()
	options.CacheRoot = filepath.Join(root, "cache")
	// Renamed away below to prove detached replay, so this test owns a private clone.
	compilerRoot := stockgo.Clone(t, filepath.Join(root, "go"))
	options.CompilerPath = filepath.Join(compilerRoot, "bin/go")
	options.Config.Profile = wasi.CooperativeProfile
	options.RuntimeRoot = filepath.Join(root, "runtime-owner")
	if err := os.MkdirAll(options.RuntimeRoot, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"gomad_semantics.go.txt", "gomad_choicewire_generated.go.txt"} {
		data, err := os.ReadFile(filepath.Join("../../../runtimesemantics", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(options.RuntimeRoot, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	provider, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile("../testdata/runtime_probes/main.go")
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"go.mod": []byte("module gomad.wasm.cooperative.fixture\n\ngo 1.27.1\n"), "main.go": fixture} {
		if err := os.WriteFile(filepath.Join(source, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	spec := target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: source, BuildTags: []string{"test_dep"}, PreparationRoot: filepath.Join(root, "prepared-first")}
	first, err := provider.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	spec.PreparationRoot = filepath.Join(root, "prepared-second")
	second, err := provider.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if first.BuildKey != second.BuildKey || first.SHA256 != second.SHA256 {
		t.Fatal("unchanged cooperative preparation did not reuse its complete build identity")
	}
	spec.PreparationRoot = ""
	const seeds = "0,1000003,2000006,3000009,4000012,5000015,6000018,7000021,8000024,9000027,10000030,11000033,12000036,13000039,14000042,15000045,16000048,17000051,18000054,19000057,20000060,21000063,22000066,23000069,24000072,25000075,26000078,27000081,28000084,29000087,30000090,31000093"
	campaign, err := runner.Explore(t.Context(), runner.CampaignSpec{Backend: provider, Seeds: seeds, Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 10 * time.Minute, TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyFirst, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, ChoiceTraceLimit: 1 << 20, Diagnostics: true, Artifacts: filepath.Join(root, "artifacts"), RunnerBuild: "task5-cooperative", Coverage: runner.CoverageNone, Target: spec})
	if err != nil || campaign.Failures != 1 || len(campaign.Artifacts) != 1 {
		t.Fatalf("cooperative canary discovery: %+v err=%v", campaign, err)
	}
	for _, directory := range []string{source, compilerRoot, options.RuntimeRoot, options.CacheRoot} {
		if err := os.Rename(directory, directory+"-unavailable"); err != nil {
			t.Fatal(err)
		}
	}
	replay, err := runner.Replay(t.Context(), runner.ReplaySpec{ArtifactPath: campaign.Artifacts[0], Backend: provider})
	if err != nil || !replay.Match || replay.ObservedRepetition || replay.ChoiceReplayStatus != runner.ChoiceReplayExact || replay.Artifact.Manifest.ReplayMode != record.ReplayExact || replay.Artifact.Manifest.Outcome.ExitCode == nil || *replay.Artifact.Manifest.Outcome.ExitCode != 17 {
		t.Fatalf("detached cooperative artifact replay: %+v err=%v", replay, err)
	}
	if replay.Artifact.Manifest.ChoiceProfile == nil || replay.Artifact.Manifest.ChoiceProfile.Trace.TapeSHA256 == "" || replay.Artifact.Manifest.IOProfile.Transcript != nil {
		t.Fatal("cooperative artifact lacks its exact choice tape or borrowed native I/O identity")
	}
	t.Logf("campaign=%s artifact=%s record=%s module=%s build=%s choice=%s replay=%s", campaign.CampaignPath, campaign.Artifacts[0], replay.Artifact.Manifest.RecordHash, first.SHA256, first.BuildKey, replay.Artifact.Manifest.ChoiceProfile.Trace.TapeSHA256, replay.ChoiceReplayStatus)
}

const explorationGuest = `package main
import ("fmt"; "os")
func choose() int {
	a, b := make(chan int, 1), make(chan int, 1)
	a <- 0; b <- 1
	select { case n := <-a: return n; case n := <-b: return n }
}
func main() {
	a, b := choose(), choose()
	fmt.Printf("choices=%d,%d\n", a, b)
	if a == 1 && b == 1 { os.Exit(17) }
}
`

func explorationProvider(t *testing.T) *Provider {
	t.Helper()
	stock := integrationProvider(t)
	options := stock.options
	helperBytes, err := os.ReadFile(options.HelperPath)
	if err != nil || string(record.HashBytes(helperBytes)) != options.HelperSHA256 {
		t.Fatalf("helper changed before campaign snapshot: %v", err)
	}
	options.HelperPath = filepath.Join(t.TempDir(), "gomad3-wasmhost")
	if err := os.WriteFile(options.HelperPath, helperBytes, 0700); err != nil {
		t.Fatal(err)
	}
	options.CompileCacheRoot = t.TempDir()
	if err := os.Chmod(options.CompileCacheRoot, 0700); err != nil {
		t.Fatal(err)
	}
	options.Config.Profile = wasi.CooperativeProfile
	options.Config.Clock.ReadStepNanos = 0
	options.RuntimeRoot, err = filepath.Abs("../../../runtimesemantics")
	if err != nil {
		t.Fatal(err)
	}
	// Cooperative preparation needs the distribution outside GOMODCACHE.
	options.CompilerPath = stockgo.Compiler(t)
	provider, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func explorationSpec(t *testing.T, provider *Provider, source string) runner.CampaignSpec {
	t.Helper()
	root := t.TempDir()
	for name, data := range map[string]string{"go.mod": "module gomad.wasm.exploration.fixture\n\ngo 1.27.1\n", "main.go": source} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return runner.CampaignSpec{
		Backend: provider, Seeds: "7", Strategy: runner.StrategyChoiceExploration, Parallel: 2,
		ExecutionTimeout: time.Minute, OverallTimeout: 5 * time.Minute, TerminateGrace: 100 * time.Millisecond,
		OnFailure: runner.PolicyAll, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20,
		ChoiceTraceLimit: 1 << 20, MaxExecutions: 16, MaxChoiceDepth: 4, MaxExplorationBytes: 8 << 20,
		Artifacts: t.TempDir(), RunnerBuild: "fn156-exploration", Coverage: runner.CoverageChoice,
		KeepSuccesses: runner.KeepSuccessesAll, SuccessArtifactLimit: 16, SuccessBytesLimit: 256 << 20,
		Target: target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: root, BuildTags: []string{"test_dep"}},
	}
}

func canonicalRounds(t *testing.T, campaign string) map[string][]byte {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(campaign, "choice-exploration", "rounds", "*", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("missing committed rounds: %v", err)
	}
	rounds := make(map[string][]byte, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		rounds[filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file))] = data
	}
	return rounds
}

func verifyLogicalExecutions(t *testing.T, campaign string, want uint64) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(campaign, "choice-exploration", "rounds", "*", "executions", "*.json"))
	if err != nil || uint64(len(files)) != want {
		t.Fatalf("logical execution count=%d want=%d err=%v", len(files), want, err)
	}
	seen := map[uint64]bool{}
	for _, file := range files {
		var execution struct {
			Ordinal record.Uint64String `json:"selection_ordinal"`
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &execution); err != nil {
			t.Fatal(err)
		}
		ordinal := uint64(execution.Ordinal)
		if ordinal >= want || seen[ordinal] {
			t.Fatalf("lost or duplicate logical ordinal %d", ordinal)
		}
		seen[ordinal] = true
	}
}

func TestWASMChoiceCampaignCanonicalRoundsAndDetachedReplay(t *testing.T) {
	provider := explorationProvider(t)
	spec := explorationSpec(t, provider, explorationGuest)
	first, err := runner.Explore(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if first.Failures == 0 || first.Succeeded == 0 || first.ReplayDivergences != 0 || first.RetainedSuccesses != first.Succeeded || first.ChoiceExploration == nil || first.ChoiceExploration.DeepestPrefix < 2 {
		t.Fatalf("bounded branching canary campaign: %+v", first)
	}
	spec.Artifacts = t.TempDir()
	second, err := runner.Explore(t.Context(), spec)
	if err != nil || !reflect.DeepEqual(first.ChoiceExploration, second.ChoiceExploration) || !reflect.DeepEqual(canonicalRounds(t, first.CampaignPath), canonicalRounds(t, second.CampaignPath)) {
		t.Fatalf("canonical campaigns differ: first=%+v second=%+v err=%v", first.ChoiceExploration, second.ChoiceExploration, err)
	}
	verifyLogicalExecutions(t, first.CampaignPath, first.Attempted)
	bounded := spec
	bounded.Artifacts = t.TempDir()
	bounded.MaxExecutions = 1
	limited, err := runner.Explore(t.Context(), bounded)
	if err != nil || limited.Attempted != 1 || limited.StopReason != runner.StopMaxExecutions || limited.ChoiceExploration == nil || limited.ChoiceExploration.OmittedByExecutionBound == 0 {
		t.Fatalf("execution envelope was not reported: %+v err=%v", limited, err)
	}
	bounded.Artifacts = t.TempDir()
	bounded.MaxExecutions, bounded.MaxChoiceDepth = 16, 1
	limited, err = runner.Explore(t.Context(), bounded)
	if err != nil || limited.StopReason != runner.StopChoiceDepthComplete || limited.ChoiceExploration == nil || limited.ChoiceExploration.OmittedByDepth == 0 || limited.ChoiceExploration.DeepestPrefix != 1 {
		t.Fatalf("depth envelope was not reported: %+v err=%v", limited, err)
	}
	rootRoundKey := filepath.Join("00000000000000000000", "round.json")
	rootRound := canonicalRounds(t, first.CampaignPath)[rootRoundKey]
	if len(rootRound) == 0 {
		t.Fatal("missing canonical root round for byte-capacity envelope")
	}
	bounded.Artifacts = t.TempDir()
	bounded.MaxChoiceDepth = spec.MaxChoiceDepth
	// Admit the initial queue and its journal envelope, but no larger forced prefix.
	bounded.MaxExplorationBytes = uint64(len(rootRound))
	limited, err = runner.Explore(t.Context(), bounded)
	if err != nil || limited.StopReason != runner.StopExplorationCapacity || limited.Attempted != 1 || limited.ChoiceExploration == nil {
		t.Fatalf("byte envelope was not reported: %+v err=%v", limited, err)
	}
	capacity := limited.ChoiceExploration
	if capacity.MaxExplorationBytes != bounded.MaxExplorationBytes || capacity.OmittedByCapacity == 0 || capacity.Pending != 0 || capacity.PendingBytes != 0 || capacity.SeenPrefixes != 1 || capacity.LogicalExecutions != 1 || capacity.CommittedRounds != 1 || capacity.DeepestPrefix != 0 || capacity.BoundedComplete {
		t.Fatalf("byte-capacity queue/state escaped its envelope: %+v", capacity)
	}
	verifyLogicalExecutions(t, limited.CampaignPath, 1)
	capacityRounds := canonicalRounds(t, limited.CampaignPath)
	if len(capacityRounds) != 2 || !reflect.DeepEqual(rootRound, capacityRounds[rootRoundKey]) {
		t.Fatalf("byte-capacity campaign changed canonical root order: %v", capacityRounds)
	}
	for _, directory := range []string{spec.Target.WorkingDir, provider.options.CacheRoot} {
		if err := os.Rename(directory, directory+"-unavailable"); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := os.Rename(directory+"-unavailable", directory); err != nil {
				t.Error(err)
			}
		})
	}
	provider.options.CompilerPath = filepath.Join(t.TempDir(), "unavailable-go")
	provider.options.RuntimeRoot = filepath.Join(t.TempDir(), "unavailable-runtime")
	for _, path := range append(first.Artifacts, first.SuccessArtifacts...) {
		replay, err := runner.Replay(t.Context(), runner.ReplaySpec{Backend: provider, ArtifactPath: path})
		if err != nil || !replay.Match || replay.ChoiceReplayStatus != runner.ChoiceReplayExact || replay.ObservedRepetition {
			t.Fatalf("detached exact replay: %+v err=%v", replay, err)
		}
	}
	incompatible := *provider
	incompatible.options.Config.Profile = wasi.StockProfile
	tracked := &minimizationProvider{Provider: &incompatible}
	_, err = runner.Minimize(t.Context(), runner.MinimizeSpec{Backend: tracked, ArtifactPath: first.Artifacts[0], OutputRoot: t.TempDir(), AttemptBudget: 8})
	if err == nil || !strings.Contains(err.Error(), "replay profile mismatch") || len(tracked.controls) != 0 {
		t.Fatalf("incompatible replay profile must reject before minimization execution: %v controls=%v", err, tracked.controls)
	}
	t.Logf("logical=%d rounds=%d failures=%d retained-successes=%d", first.Attempted, first.ChoiceExploration.CommittedRounds, first.Failures, first.RetainedSuccesses)
}

type interruptRoundProvider struct {
	*Provider
	calls  atomic.Uint64
	cancel context.CancelFunc
}

func (p *interruptRoundProvider) Run(ctx context.Context, request runnerbackend.Request) (runnerbackend.Result, error) {
	result, err := p.Provider.Run(ctx, request)
	if p.calls.Add(1) == 2 {
		p.cancel()
	}
	return result, err
}

func TestWASMChoiceCampaignInterruptedRoundRecoveryNoDuplicates(t *testing.T) {
	provider := explorationProvider(t)
	spec := explorationSpec(t, provider, explorationGuest)
	baseline, err := runner.Explore(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	spec.Artifacts = t.TempDir()
	spec.Backend = &interruptRoundProvider{Provider: provider, cancel: cancel}
	partial, err := runner.Explore(ctx, spec)
	if !errors.Is(err, context.Canceled) || partial.Attempted != 1 {
		t.Fatalf("interrupt inside second round: %+v err=%v", partial, err)
	}
	if _, err := runner.Recover(t.Context(), partial.CampaignPath); err != nil {
		t.Fatal(err)
	}
	resumed, err := runner.Resume(t.Context(), runner.ResumeSpec{Backend: provider, CampaignPath: partial.CampaignPath, RunnerBuild: spec.RunnerBuild})
	if err != nil || resumed.Attempted != baseline.Attempted || resumed.RecoveryExecutions == 0 || !reflect.DeepEqual(baseline.ChoiceExploration, resumed.ChoiceExploration) || !reflect.DeepEqual(canonicalRounds(t, baseline.CampaignPath), canonicalRounds(t, resumed.CampaignPath)) {
		t.Fatalf("resumed campaign differs: baseline=%+v resumed=%+v err=%v", baseline, resumed, err)
	}
	verifyLogicalExecutions(t, resumed.CampaignPath, resumed.Attempted)
	if _, err := runner.Resume(t.Context(), runner.ResumeSpec{Backend: provider, CampaignPath: resumed.CampaignPath, RunnerBuild: spec.RunnerBuild}); err == nil {
		t.Fatal("published campaign was executed twice")
	}
	t.Logf("logical=%d recovery=%d canonical-rounds=%d", resumed.Attempted, resumed.RecoveryExecutions, resumed.ChoiceExploration.CommittedRounds)
}

func TestWASMChoiceCampaignFuelCapacityIsNotApplicationFailure(t *testing.T) {
	provider := explorationProvider(t)
	provider.options.Fuel = 100000000
	source := strings.Replace(explorationGuest, `if a == 1 && b == 1 { os.Exit(17) }`, `if os.Getenv("CPU_LOOP_EXIT") == "yes" { os.Exit(0) }; for {}`, 1)
	spec := explorationSpec(t, provider, source)
	result, err := runner.Explore(t.Context(), spec)
	var failure *runnerbackend.FailureError
	if !errors.As(err, &failure) || failure.Termination != runnerbackend.Capacity || result.Failures != 0 || result.Attempted != 0 || len(result.Artifacts) != 0 {
		t.Fatalf("CPU loop must stop with fuel capacity, not an application invariant: %+v err=%v", result, err)
	}
	if !strings.Contains(strings.ToLower(failure.Message), "fuel") {
		t.Fatalf("CPU loop exhausted a different capacity: %v", err)
	}
	files, globErr := filepath.Glob(filepath.Join(result.CampaignPath, "choice-exploration", "rounds", "*", "segment.json"))
	if globErr != nil || len(files) != 0 {
		t.Fatalf("capacity round was committed: %v err=%v", files, globErr)
	}
	if strings.Contains(err.Error(), "nonzero_exit") {
		t.Fatalf("fuel capacity fabricated a guest exit: %v", err)
	}
}

func TestWASMCooperativeCampaignWithExplicitEnvironment(t *testing.T) {
	provider := explorationProvider(t)
	spec := explorationSpec(t, provider, strings.Replace(explorationGuest, `if a == 1 && b == 1 { os.Exit(17) }`, `if os.Getenv("GOMAD_ENV_PROBE") != "set" { os.Exit(3) }`, 1))
	spec.Strategy, spec.Parallel, spec.MaxExecutions, spec.MaxChoiceDepth, spec.MaxExplorationBytes = runner.StrategySeed, 1, 0, 0, 0
	spec.Environment = []string{"GOMAD_ENV_PROBE=set"}
	result, err := runner.Explore(t.Context(), spec)
	if err != nil || result.Attempted != 1 || result.Succeeded != 1 {
		t.Fatalf("cooperative campaign with an explicit environment: %+v err=%v", result, err)
	}
}

const minimizationGuest = `package main
import ("fmt"; "os")
func choose() int {
	a, b := make(chan int, 1), make(chan int, 1)
	a <- 0; b <- 1
	select { case n := <-a: return n; case n := <-b: return n }
}
func main() {
	failure := choose() == 1
	choose(); choose()
	if failure { fmt.Println("essential early choice failed"); os.Exit(17) }
	fmt.Println("success")
}
`

var errMinimizeInterrupted = errors.New("minimization integration interruption")

type minimizationProvider struct {
	*Provider
	controls []int
}

func (p *minimizationProvider) Run(ctx context.Context, request runnerbackend.Request) (runnerbackend.Result, error) {
	if request.Choice == nil {
		return runnerbackend.Result{}, errors.New("minimizer omitted choice request")
	}
	if request.Choice.Mode != choice.ModeReplay {
		length := 0
		if request.Choice.Tape != nil {
			length = len(request.Choice.Tape.Decisions)
		}
		p.controls = append(p.controls, length)
	}
	return p.Provider.Run(ctx, request)
}

type minimizationPublicationReplayer struct {
	provider  *Provider
	interrupt bool
}

func (r *minimizationPublicationReplayer) Replay(ctx context.Context, spec runner.ReplaySpec) (runner.ReplayResult, error) {
	if spec.Backend != r.provider {
		return runner.ReplayResult{}, errors.New("injected replay lost backend provider")
	}
	opened, err := artifact.OpenArtifact(spec.ArtifactPath)
	if err != nil {
		return runner.ReplayResult{}, err
	}
	final := opened.Manifest().Minimization != nil
	if err := opened.Close(); err != nil {
		return runner.ReplayResult{}, err
	}
	if final && r.interrupt {
		return runner.ReplayResult{}, errMinimizeInterrupted
	}
	return runner.Replay(ctx, spec)
}

func TestWASMMinimizeEssentialPrefixDetachedReplayRepeatAndResume(t *testing.T) {
	provider := explorationProvider(t)
	provider.options.Config.Network = &wasi.NetworkModel{Seed: 11, MaxDelayNanos: 1000, Faults: []wasi.NetworkFault{{AtNanos: 5_000_000_000, Action: "reset", Port: 9000}}}
	spec := explorationSpec(t, provider, minimizationGuest)
	spec.Parallel = 1
	seeded := spec
	seeded.Strategy, seeded.Seeds, seeded.Artifacts = runner.StrategySeed, "0-7", t.TempDir()
	seeded.MaxExecutions, seeded.MaxChoiceDepth, seeded.MaxExplorationBytes = 0, 0, 0
	seededCampaign, err := runner.Explore(t.Context(), seeded)
	if err != nil || len(seededCampaign.SuccessArtifacts) == 0 {
		t.Fatalf("canary needs a seed where unforced execution succeeds: %+v err=%v", seededCampaign, err)
	}
	success, err := artifact.OpenArtifact(seededCampaign.SuccessArtifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	spec.Seeds = strconv.FormatUint(uint64(success.Manifest().Seed), 10)
	if err := success.Close(); err != nil {
		t.Fatal(err)
	}
	campaign, err := runner.Explore(t.Context(), spec)
	if err != nil || campaign.Succeeded == 0 || len(campaign.Artifacts) == 0 {
		t.Fatalf("essential canary discovery: %+v err=%v", campaign, err)
	}
	parent, err := artifact.OpenArtifact(campaign.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	manifest := parent.Manifest()
	if manifest.ChoiceProfile == nil || manifest.ChoiceProfile.Trace.Decisions < 3 {
		t.Fatalf("canary needs redundant trailing choices: %+v", manifest.ChoiceProfile)
	}
	if err := parent.Close(); err != nil {
		t.Fatal(err)
	}
	for _, directory := range []string{spec.Target.WorkingDir, provider.options.CacheRoot} {
		if err := os.Rename(directory, directory+"-unavailable"); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := os.Rename(directory+"-unavailable", directory); err != nil {
				t.Error(err)
			}
		})
	}
	provider.options.CompilerPath = filepath.Join(t.TempDir(), "unavailable-go")
	provider.options.RuntimeRoot = filepath.Join(t.TempDir(), "unavailable-runtime")
	observed := &minimizationProvider{Provider: provider}
	minimize := runner.MinimizeSpec{Backend: observed, ArtifactPath: campaign.Artifacts[0], OutputRoot: t.TempDir(), AttemptBudget: 64}
	baseline, err := runner.Minimize(t.Context(), minimize)
	if err != nil {
		t.Fatal(err)
	}
	evidence := baseline.Artifact.Manifest.Minimization
	if !baseline.Changed || len(baseline.Accepted) == 0 || evidence == nil || evidence.Schema != "gomad3.minimization/choice-prefix/v1" || evidence.FinalForcedDecisions == 0 || evidence.FinalForcedDecisions >= evidence.OriginalForcedDecisions || evidence.Predicate.SimulationReplay != "not_present" || baseline.Artifact.Manifest.Outcome.FailureSignature != manifest.Outcome.FailureSignature {
		t.Fatalf("effective forcing not reduced with essential prefix preserved: %+v evidence=%+v", baseline, evidence)
	}
	if baseline.Artifact.Manifest.Target.Backend.Provenance.SHA256 != manifest.Target.Backend.Provenance.SHA256 || baseline.Artifact.Manifest.Target.Backend.Evidence.SHA256 != manifest.Target.Backend.Evidence.SHA256 {
		t.Fatal("minimization changed fixed model/fault inputs or complete backend observations")
	}
	if len(observed.controls) == 0 || observed.controls[0] != 0 {
		t.Fatalf("missing rejected no-forcing attempt: %v", observed.controls)
	}
	for _, reduction := range baseline.Accepted {
		if reduction.Kind != "schedule_suffix" {
			t.Fatalf("invalid prefix reduction: %+v", reduction)
		}
	}
	replay, err := runner.Replay(t.Context(), runner.ReplaySpec{Backend: provider, ArtifactPath: baseline.Artifact.Path})
	if err != nil || !replay.Match || replay.ChoiceReplayStatus != runner.ChoiceReplayExact {
		t.Fatalf("source-less fresh replay: %+v err=%v", replay, err)
	}
	for _, interruption := range []string{"repeat", "after accepted", "after publication"} {
		t.Run(interruption, func(t *testing.T) {
			tracked := &minimizationProvider{Provider: provider}
			request := minimize
			request.Backend, request.OutputRoot = tracked, t.TempDir()
			blockedPublication := filepath.Join(request.OutputRoot, filepath.Base(baseline.Artifact.Path))
			if interruption == "after accepted" {
				if err := os.WriteFile(blockedPublication, []byte("publication interruption"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if interruption == "after publication" {
				request.Backend = provider
				request.Replayer = &minimizationPublicationReplayer{provider: provider, interrupt: true}
			}
			result, err := runner.Minimize(t.Context(), request)
			if interruption != "repeat" {
				if interruption == "after publication" && !errors.Is(err, errMinimizeInterrupted) || interruption == "after accepted" && err == nil {
					t.Fatalf("missing interruption: %v", err)
				}
				if interruption == "after accepted" {
					if err := os.Rename(blockedPublication, blockedPublication+"-unavailable"); err != nil {
						t.Fatal(err)
					}
				}
				request.Resume = true
				if request.Replayer != nil {
					request.Replayer = &minimizationPublicationReplayer{provider: provider}
				}
				result, err = runner.Minimize(t.Context(), request)
			}
			if err != nil || result.Attempts != baseline.Attempts || result.Artifact.Manifest.RecordHash != baseline.Artifact.Manifest.RecordHash || !reflect.DeepEqual(result.Accepted, baseline.Accepted) {
				t.Fatalf("repeat/resume differs: %+v err=%v", result, err)
			}
			if interruption != "after publication" && !reflect.DeepEqual(tracked.controls, observed.controls) {
				t.Fatalf("committed attempts repeated: got=%v want=%v", tracked.controls, observed.controls)
			}
		})
	}
	parent, err = artifact.OpenArtifact(campaign.Artifacts[0])
	if err != nil {
		t.Fatal(err)
	}
	if parent.Manifest().RecordHash != manifest.RecordHash {
		t.Fatal("minimization mutated parent")
	}
	if err := parent.Close(); err != nil {
		t.Fatal(err)
	}
	t.Logf("parent=%s final=%s attempts=%d original-forcing=%d final-forcing=%d controls=%v backend-evidence=%s", manifest.RecordHash, baseline.Artifact.Manifest.RecordHash, baseline.Attempts, evidence.OriginalForcedDecisions, evidence.FinalForcedDecisions, observed.controls, baseline.Artifact.Manifest.Target.Backend.Evidence.SHA256)
}
