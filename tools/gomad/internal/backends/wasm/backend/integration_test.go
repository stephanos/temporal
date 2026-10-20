package backend

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	runnerbackend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/runner"
	"go.temporal.io/server/tools/gomad/internal/runner/inspect"
	campaignoptions "go.temporal.io/server/tools/gomad/internal/runner/options"
)

func integrationProvider(t *testing.T) *Provider {
	t.Helper()
	compiler := os.Getenv("GOMAD_STOCK_GO")
	if compiler == "" {
		t.Fatal("explicit stock compiler is required for integration gate")
	}
	helper := os.Getenv("GOMAD_WASM_TEST_HELPER")
	if helper == "" {
		helper = "../wasmhost/target/release/gomad-wasmhost"
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
	result, err := runner.Explore(t.Context(), runner.CampaignSpec{Backend: provider, Seeds: "7", Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 3 * time.Minute, TerminateGrace: time.Millisecond * 100, OnFailure: runner.PolicyFirst, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, Artifacts: t.TempDir(), RunnerBuild: "task4-integration", Coverage: campaignoptions.CoverageNone, Target: target.Spec{Backend: Name, Kind: target.KindGoTest, Source: "./io_failure", WorkingDir: fixture, BuildTags: []string{"gomad_fixture", "test_dep"}, Args: []string{"-test.run=^TestDeterministicIOFailure$", "-test.v"}}})
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
	module := filepath.Join(compilationCachePath(t, provider, third), "module.wasm")
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

func TestBuildSelectionChangesInvalidateCompilationCache(t *testing.T) {
	provider := integrationProvider(t)
	source, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(source, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module cache.fixture\n\ngo 1.27.1\n\nrequire cache.fixture/optional v0.0.0\n")
	dependencyMod := "module cache.fixture/optional\n\ngo 1.26\n"
	dependencySource := "package optional\nfunc Name() string { return \"optional provider\" }\n"
	proxyVersion := "proxy/cache.fixture/optional/@v/v0.0.0"
	write(proxyVersion+".mod", dependencyMod)
	write(proxyVersion+".info", `{"Version":"v0.0.0","Time":"2026-10-15T00:00:00Z"}`)
	var moduleZip bytes.Buffer
	archive := zip.NewWriter(&moduleZip)
	for name, body := range map[string]string{"go.mod": dependencyMod, "optional.go": dependencySource} {
		file, err := archive.Create("cache.fixture/optional@v0.0.0/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(file, body); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	write(proxyVersion+".zip", moduleZip.String())
	t.Setenv("GOPROXY", "file://"+filepath.ToSlash(filepath.Join(source, "proxy")))
	t.Setenv("GOMODCACHE", filepath.Join(source, "modules"))
	t.Setenv("GOSUMDB", "off")
	download := func() {
		t.Helper()
		if _, err := command(t.Context(), provider.options.CompilerPath, []string{"mod", "download", "-modcacherw", "cache.fixture/optional"}, source, buildEnvironment(), 1<<20); err != nil {
			t.Fatal(err)
		}
	}
	download()
	write("main.go", "package main\nfunc main(){ println(selected) }\n")
	write("selected_full.go", "//go:build !fixture_slim\n\npackage main\nimport \"cache.fixture/optional\"\nvar selected = optional.Name()\n")
	write("selected_slim.go", "//go:build fixture_slim\n\npackage main\nvar selected = \"slim\"\n")
	prepare := func(name string, tags ...string) (target.Prepared, provenance) {
		t.Helper()
		prepared, err := provider.Prepare(t.Context(), target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: source, PreparationRoot: filepath.Join(t.TempDir(), name), BuildTags: tags})
		if err != nil {
			t.Fatal(err)
		}
		pr, err := decodeProvenance(prepared.BackendPayloads[0].Data)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(pr.BuildTags, prepared.BuildTags) || pr.BuildKey != prepared.BuildKey || pr.ModuleSHA256 != string(prepared.SHA256) {
			t.Fatal("prepared target and provenance disagree on the build selection")
		}
		return prepared, pr
	}
	has := func(pr provenance, file string) bool {
		want := filepath.Join(source, file)
		return slices.ContainsFunc(pr.Sources, func(source sourceIdentity) bool { return source.Path == want })
	}
	full, fullProvenance := prepare("full", "test_dep")
	slim, slimProvenance := prepare("slim", "fixture_slim", "test_dep")
	if full.BuildKey == slim.BuildKey || full.SHA256 == slim.SHA256 || !slices.Equal(slim.BuildTags, []string{"fixture_slim", "test_dep"}) {
		t.Fatal("changed dependency selection reused a compilation cache entry")
	}
	provider.options.BuildTags = []string{"fixture_slim", "test_dep"}
	installed, installedProvenance := prepare("installed", "test_dep")
	if installed.BuildKey != slim.BuildKey || !reflect.DeepEqual(installedProvenance, slimProvenance) {
		t.Fatal("installed build tags did not select and seal the same source closure")
	}
	provider.options.BuildTags = nil
	dependencyFile := "modules/cache.fixture/optional@v0.0.0/optional.go"
	if !has(fullProvenance, dependencyFile) || !has(fullProvenance, "selected_full.go") || has(fullProvenance, "selected_slim.go") {
		t.Fatal("full selection provenance does not name the optional provider closure")
	}
	if has(slimProvenance, dependencyFile) || has(slimProvenance, "selected_full.go") || !has(slimProvenance, "selected_slim.go") {
		t.Fatal("slim selection provenance does not name its actual closure")
	}
	// A tag that selects no different file still keys a separate entry.
	tagOnly, tagOnlyProvenance := prepare("tag-only", "fixture_unused", "test_dep")
	if tagOnly.BuildKey == full.BuildKey || !reflect.DeepEqual(tagOnlyProvenance.Sources, fullProvenance.Sources) {
		t.Fatal("tag-only selection change shared the build key or altered the source closure")
	}
	dependencyModFile := "modules/cache/download/cache.fixture/optional/@v/v0.0.0.mod"
	changedDependencyMod := "module cache.fixture/optional\n\ngo 1.27.1\n"
	write(dependencyModFile, changedDependencyMod)
	if err := os.Remove(filepath.Join(source, "go.sum")); err != nil {
		t.Fatal(err)
	}
	download()
	language, languageProvenance := prepare("language", "test_dep")
	if language.BuildKey == full.BuildKey || language.BuildKey == slim.BuildKey {
		t.Fatal("changed dependency language version reused a compilation cache entry")
	}
	wantSources := append([]sourceIdentity(nil), fullProvenance.Sources...)
	for i := range wantSources {
		if wantSources[i].Path == filepath.Join(source, dependencyModFile) {
			wantSources[i].SHA256 = record.HashBytes([]byte(changedDependencyMod))
		} else if wantSources[i].Path == filepath.Join(source, "go.sum") {
			wantSources[i].SHA256 = record.HashBytes([]byte(stringMustRead(t, wantSources[i].Path)))
		}
	}
	if !has(fullProvenance, dependencyModFile) || !reflect.DeepEqual(languageProvenance.Sources, wantSources) {
		t.Fatal("dependency language version change did not bind exactly the changed module and checksum inputs")
	}
	write(dependencyModFile, dependencyMod)
	if err := os.Remove(filepath.Join(source, "go.sum")); err != nil {
		t.Fatal(err)
	}
	download()
	again, _ := prepare("again", "test_dep")
	if again.BuildKey != full.BuildKey || again.SHA256 != full.SHA256 {
		t.Fatal("unchanged selection did not reuse its verified cache identity")
	}
	spec := target.Spec{Backend: Name, Kind: target.KindGoRun, Source: "."}
	for _, prepared := range []target.Prepared{full, slim, tagOnly, language} {
		if err := provider.ValidatePrepared(spec, prepared, nil); err != nil {
			t.Fatal(err)
		}
	}
	relabeled := slim
	relabeled.BuildTags = full.BuildTags
	if err := provider.ValidatePrepared(spec, relabeled, nil); err == nil {
		t.Fatal("prepared target relabeled with another build selection was accepted")
	}
}

// The slim guest tag must cut exactly the optional-provider seams it names:
// the actual guest closure equals the default closure with those import edges
// removed, so no other package or init registration silently disappears.
func TestSlimGuestClosureCutsOnlyOptionalProviderSeams(t *testing.T) {
	compiler := os.Getenv("GOMAD_STOCK_GO")
	if compiler == "" {
		t.Fatal("explicit stock compiler is required for integration gate")
	}
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	environment := buildEnvironment()
	base := func(path string) string { path, _, _ = strings.Cut(path, " ["); return path }
	list := func(tags []string) (map[string][]string, []string, []byte) {
		t.Helper()
		data, err := command(t.Context(), compiler, []string{"list", "-mod=readonly", "-pgo=off", "-deps", "-test", "-json=ImportPath,Imports,DepOnly,Dir,GoFiles,EmbedFiles,TestGoFiles,XTestGoFiles,TestEmbedFiles,XTestEmbedFiles,SFiles,SysoFiles,HFiles,Standard,Module", "-tags", strings.Join(tags, ","), "./tests"}, repo, environment, 128<<20)
		if err != nil {
			t.Fatal(err)
		}
		graph, roots := map[string][]string{}, []string{}
		decoder := json.NewDecoder(bytes.NewReader(data))
		for {
			var listed struct {
				ImportPath string
				Imports    []string
				DepOnly    bool
			}
			if err := decoder.Decode(&listed); errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				t.Fatal(err)
			}
			graph[listed.ImportPath] = listed.Imports
			if !listed.DepOnly {
				roots = append(roots, listed.ImportPath)
			}
		}
		return graph, roots, data
	}
	const server = "go.temporal.io/server/"
	seams := map[string][]string{
		server + "common/archiver/provider": {server + "common/archiver/gcloud", server + "common/archiver/s3store"},
		server + "temporal":                 {server + "common/membership/ringpop"},
		server + "tests/testcore":           {server + "common/persistence/sql/sqlplugin/mysql", server + "common/persistence/sql/sqlplugin/postgresql"},
	}
	cut := func(from, to string) bool { return slices.Contains(seams[base(from)], base(to)) }
	reach := func(graph map[string][]string, roots []string, cut func(string, string) bool) map[string]bool {
		seen, packages, pending := map[string]bool{}, map[string]bool{}, append([]string{}, roots...)
		for len(pending) != 0 {
			next := pending[len(pending)-1]
			pending = pending[:len(pending)-1]
			if seen[next] {
				continue
			}
			seen[next], packages[base(next)] = true, true
			for _, imported := range graph[next] {
				if _, listed := graph[imported]; listed && (cut == nil || !cut(next, imported)) {
					pending = append(pending, imported)
				}
			}
		}
		return packages
	}
	fullGraph, fullRoots, fullListing := list([]string{"disable_grpc_modules", "sqlite3_dotlk", "test_dep"})
	slimGraph, slimRoots, slimListing := list(TemporalGuestBuildTags())
	full, predicted, slim := reach(fullGraph, fullRoots, nil), reach(fullGraph, fullRoots, cut), reach(slimGraph, slimRoots, nil)
	for from, targets := range seams {
		for _, to := range targets {
			if !full[from] || !full[to] || slim[to] || !slim[from] {
				t.Fatalf("seam %s -> %s is not the measured optional-provider edge", from, to)
			}
		}
	}
	if !reflect.DeepEqual(predicted, slim) {
		var unexpected []string
		for path := range predicted {
			if !slim[path] {
				unexpected = append(unexpected, "-"+path)
			}
		}
		for path := range slim {
			if !predicted[path] {
				unexpected = append(unexpected, "+"+path)
			}
		}
		sort.Strings(unexpected)
		t.Fatalf("slim guest closure differs from the cut seams: %v", unexpected)
	}
	for _, required := range []string{
		server + "common/archiver/provider", server + "common/archiver/filestore", server + "common/membership/static",
		server + "common/persistence/sql/sqlplugin/sqlite", server + "common/persistence/visibility/store/elasticsearch",
		"github.com/olivere/elastic/v7", server + "common/persistence/cassandra", "go.temporal.io/auto-scaled-workers/wci/workercomponent",
	} {
		if !slim[required] {
			t.Fatalf("slim guest closure dropped retained package %s", required)
		}
	}
	_, includeDirectory, err := compilerInventory(t.Context(), compiler, repo, environment)
	if err != nil {
		t.Fatal(err)
	}
	moduleInput := func(listing []byte, module string) bool {
		sources, err := sourceInventory(listing, includeDirectory)
		if err != nil {
			t.Fatal(err)
		}
		return slices.ContainsFunc(sources, func(source sourceIdentity) bool {
			return strings.Contains(filepath.ToSlash(source.Path), "/cache/download/"+module+"/@v/") && strings.HasSuffix(source.Path, ".mod")
		})
	}
	for _, module := range []string{"cloud.google.com/go/storage", "github.com/temporalio/ringpop-go", "github.com/go-sql-driver/mysql", "github.com/jackc/pgx/v5"} {
		if !moduleInput(fullListing, module) || moduleInput(slimListing, module) {
			t.Fatalf("preparation inputs do not follow the %s exclusion", module)
		}
	}
	t.Logf("default ./tests guest closure %d packages; slim %d packages", len(full), len(slim))
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
	receipt := filepath.Join(compilationCachePath(t, isolated, first), "receipt.json")
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
	campaign, err := runner.Explore(t.Context(), runner.CampaignSpec{Backend: provider, Seeds: "7", Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 3 * time.Minute, TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyFirst, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, Artifacts: t.TempDir(), RunnerBuild: "task4-integration", Coverage: campaignoptions.CoverageNone, Target: target.Spec{Backend: Name, Kind: target.KindGoTest, Source: "./io_failure", WorkingDir: fixture, BuildTags: []string{"gomad_fixture", "test_dep"}, Args: []string{"-test.run=^TestDeterministicIOFailure$", "-test.v"}}})
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
			_, err = runner.Replay(t.Context(), runner.ReplaySpec{Backend: counted, ArtifactPath: published.Path})
			var incompatible *runner.ReplayPreflightError
			if !errors.As(err, &incompatible) {
				t.Fatalf("changed identity did not produce a classified replay refusal: %v", err)
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
			for _, reserved := range []string{"storage.json", "manifest.json"} {
				data, err := os.ReadFile(filepath.Join(campaign.Artifacts[0], reserved))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(copyRoot, reserved), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			corrupted := filepath.Join(copyRoot, filepath.FromSlash(file))
			data, err := os.ReadFile(corrupted)
			if err != nil {
				t.Fatal(err)
			}
			data[0] ^= 1
			if err := os.WriteFile(corrupted, data, 0600); err != nil {
				t.Fatal(err)
			}
			counted := &countedProvider{Provider: provider}
			if _, err := runner.Replay(t.Context(), runner.ReplaySpec{Backend: counted, ArtifactPath: copyRoot}); err == nil || errors.Is(err, artifact.ErrUnsupportedFormat) {
				t.Fatalf("corrupt payload accepted or refused for its lineage instead: %v", err)
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
	campaign, err := runner.Explore(t.Context(), runner.CampaignSpec{Backend: counted, Seeds: "7", Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 3 * time.Minute, TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyFirst, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, Artifacts: filepath.Join(root, "artifacts"), RunnerBuild: "task4-integration", Coverage: campaignoptions.CoverageNone, Target: target.Spec{Backend: Name, Kind: target.KindGoTest, Source: "./io_failure", WorkingDir: fixture, BuildTags: []string{"gomad_fixture", "test_dep"}, Args: []string{"-test.run=^TestDeterministicIOFailure$", "-test.v"}}})
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
	config := runner.CampaignSpec{Backend: &resumeBoundaryProvider{Provider: provider}, Seeds: "7-8", Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 3 * time.Minute, TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyAll, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, Artifacts: t.TempDir(), RunnerBuild: "task4-integration", Coverage: campaignoptions.CoverageNone, ProgressInterval: time.Millisecond, Target: target.Spec{Backend: Name, Kind: target.KindGoTest, Source: "./io_failure", WorkingDir: fixture, BuildTags: []string{"gomad_fixture", "test_dep"}, Args: []string{"-test.run=^TestDeterministicIOFailure$", "-test.v"}}, Progress: func(event runner.CampaignEvent) error {
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
	compilerLink := filepath.Join(root, "compiler-link")
	if err := os.Symlink(compilerRoot, compilerLink); err != nil {
		t.Fatal(err)
	}
	options.CompilerPath = filepath.Join(compilerLink, "bin/go")
	options.Config.Profile = wasi.CooperativeProfile
	options.RuntimeRoot = filepath.Join(root, "runtime-owner")
	if err := os.MkdirAll(options.RuntimeRoot, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"gomad_semantics.go.txt", "gomad_choicewire_generated.go.txt"} {
		data, err := os.ReadFile(filepath.Join("../../../model/runtimesource", name))
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
	legacyEnvironment := buildEnvironment()
	legacyEnvironment = append(legacyEnvironment, "GOROOT="+compilerLink)
	legacyCompiler, legacyInclude, err := compilerInventory(t.Context(), options.CompilerPath, source, legacyEnvironment)
	if err != nil {
		t.Fatal(err)
	}
	overlay, _, replacements, err := provider.runtimeOverlay(t.Context(), source, legacyEnvironment, legacyInclude)
	if err != nil {
		t.Fatal(err)
	}
	legacyList, err := command(t.Context(), options.CompilerPath, []string{"list", "-mod=readonly", "-pgo=off", "-deps", "-json", "-overlay", overlay, "-tags", "test_dep", "."}, source, legacyEnvironment, 128<<20)
	if err != nil {
		t.Fatal(err)
	}
	legacySources, err := sourceInventory(legacyList, legacyInclude, replacements)
	if err != nil {
		t.Fatal(err)
	}
	runtimeInputs, err := provider.runtimeInputs(overlay)
	if err != nil {
		t.Fatal(err)
	}
	legacyProvenance, err := decodeProvenance(first.BackendPayloads[0].Data)
	if err != nil {
		t.Fatal(err)
	}
	legacyProvenance.Sources = append(append(legacySources, legacyCompiler...), runtimeInputs...)
	sort.Slice(legacyProvenance.Sources, func(i, j int) bool { return legacyProvenance.Sources[i].Path < legacyProvenance.Sources[j].Path })
	legacyKey, err := buildKey(legacyProvenance)
	if err != nil {
		t.Fatal(err)
	}
	if first.BuildKey == legacyKey {
		t.Fatal("canonical cooperative preparation reused the uninstrumented alias-path cache identity")
	}
	t.Logf("legacy alias build=%s canonical build=%s", legacyKey, first.BuildKey)
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
	campaign, err := runner.Explore(t.Context(), runner.CampaignSpec{Backend: provider, Seeds: seeds, Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 10 * time.Minute, TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyFirst, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, ChoiceTraceLimit: 1 << 20, Diagnostics: true, Artifacts: filepath.Join(root, "artifacts"), RunnerBuild: "task5-cooperative", Coverage: campaignoptions.CoverageNone, Target: spec})
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
	options.HelperPath = filepath.Join(t.TempDir(), "gomad-wasmhost")
	if err := os.WriteFile(options.HelperPath, helperBytes, 0700); err != nil {
		t.Fatal(err)
	}
	options.CompileCacheRoot = t.TempDir()
	if err := os.Chmod(options.CompileCacheRoot, 0700); err != nil {
		t.Fatal(err)
	}
	options.Config.Profile = wasi.CooperativeProfile
	options.Config.Clock.ReadStepNanos = 0
	options.RuntimeRoot, err = filepath.Abs("../../../model/runtimesource")
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
		Artifacts: t.TempDir(), RunnerBuild: "fn156-exploration", Coverage: campaignoptions.CoverageChoice,
		KeepSuccesses: runner.KeepSuccessesAll, SuccessArtifactLimit: 16, SuccessBytesLimit: 256 << 20,
		Target: target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: root, BuildTags: []string{"test_dep"}},
	}
}

// committedRounds reads a published exploration campaign's committed rounds
// through the public Inspect report: each round's index, encoded round
// length and identity, segment identity and logical ordinals.
func committedRounds(t *testing.T, campaign string) []inspect.ExplorationRoundInspection {
	t.Helper()
	report, err := inspect.Inspect(campaign, inspect.InspectOptions{})
	if err != nil || report.Exploration == nil || len(report.Exploration.Rounds) == 0 {
		t.Fatalf("missing committed rounds: %#v %v", report.Exploration, err)
	}
	return report.Exploration.Rounds
}

// verifyLogicalExecutions checks through Inspect that a published campaign
// committed every logical ordinal below want exactly once.
func verifyLogicalExecutions(t *testing.T, campaign string, want uint64) {
	t.Helper()
	report, err := inspect.Inspect(campaign, inspect.InspectOptions{})
	if err != nil || report.Campaign == nil || uint64(len(report.Campaign.Executions)) != want {
		t.Fatalf("logical executions = %#v want=%d err=%v", report.Campaign, want, err)
	}
	seen := map[uint64]bool{}
	for _, execution := range report.Campaign.Executions {
		ordinal := execution.SelectionOrdinal
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
	if err != nil || !reflect.DeepEqual(first.ChoiceExploration, second.ChoiceExploration) || !reflect.DeepEqual(committedRounds(t, first.CampaignPath), committedRounds(t, second.CampaignPath)) {
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
	rootRound := committedRounds(t, first.CampaignPath)[0]
	if rootRound.Index != 0 || rootRound.RoundBytes == 0 {
		t.Fatalf("missing root round for byte-capacity envelope: %#v", rootRound)
	}
	bounded.Artifacts = t.TempDir()
	bounded.MaxChoiceDepth = spec.MaxChoiceDepth
	// Admit the encoded root round, but no larger forced prefix.
	bounded.MaxExplorationBytes = rootRound.RoundBytes
	limited, err = runner.Explore(t.Context(), bounded)
	if err != nil || limited.StopReason != runner.StopExplorationCapacity || limited.Attempted != 1 || limited.ChoiceExploration == nil {
		t.Fatalf("byte envelope was not reported: %+v err=%v", limited, err)
	}
	capacity := limited.ChoiceExploration
	if capacity.MaxExplorationBytes != bounded.MaxExplorationBytes || capacity.OmittedByCapacity == 0 || capacity.Pending != 0 || capacity.PendingBytes != 0 || capacity.SeenPrefixes != 1 || capacity.LogicalExecutions != 1 || capacity.CommittedRounds != 1 || capacity.DeepestPrefix != 0 || capacity.BoundedComplete {
		t.Fatalf("byte-capacity queue/state escaped its envelope: %+v", capacity)
	}
	verifyLogicalExecutions(t, limited.CampaignPath, 1)
	capacityRounds := committedRounds(t, limited.CampaignPath)
	if len(capacityRounds) != 1 || capacityRounds[0].RoundSHA256 != rootRound.RoundSHA256 || capacityRounds[0].RoundBytes != rootRound.RoundBytes {
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
	if err != nil || resumed.Attempted != baseline.Attempted || resumed.RecoveryExecutions == 0 || !reflect.DeepEqual(baseline.ChoiceExploration, resumed.ChoiceExploration) || !reflect.DeepEqual(committedRounds(t, baseline.CampaignPath), committedRounds(t, resumed.CampaignPath)) {
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
	report, inspectErr := inspect.Inspect(result.CampaignPath, inspect.InspectOptions{})
	if inspectErr != nil || report.Exploration == nil || len(report.Exploration.Rounds) != 0 {
		t.Fatalf("capacity round was committed: %#v err=%v", report.Exploration, inspectErr)
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
	if !baseline.Changed || len(baseline.Accepted) == 0 || evidence == nil || evidence.Schema != "gomad.minimization/choice-prefix/v1" || evidence.FinalForcedDecisions == 0 || evidence.FinalForcedDecisions >= evidence.OriginalForcedDecisions || evidence.Predicate.SimulationReplay != "not_present" || baseline.Artifact.Manifest.Outcome.FailureSignature != manifest.Outcome.FailureSignature {
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
