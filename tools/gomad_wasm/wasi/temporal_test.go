package wasi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/hostexec"
	"go.temporal.io/server/tools/gomad_wasm/internal/sqlitebusy"
)

func TestTemporalGuestAdmission(t *testing.T) {
	target := os.Getenv("GOMAD_WASM_TEMPORAL_TARGET")
	if target == "" {
		t.Skip("set GOMAD_WASM_TEMPORAL_TARGET to run the explicit Temporal guest gate")
	}
	clockSelection := os.Getenv("GOMAD_WASM_TEMPORAL_CLOCK")
	clock, err := temporalClockPolicy(clockSelection)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	evidence := filepath.Join(root, ".tmp", "wasm-backend-temporal")
	adapterMode := os.Getenv("GOMAD_WASM_SQLITE_ADAPTER")
	if adapterMode != "" && adapterMode != "cooperative-v1" {
		t.Fatalf("unknown SQLite adapter %q", adapterMode)
	}
	if target == "grpc-progress-diagnostic" && adapterMode != "" {
		t.Fatal("standalone gRPC diagnostic requires the original dependency graph without SQLite adapter")
	}
	projectionControl := os.Getenv("GOMAD_WASM_SQLITE_PROJECTION_CONTROL") == "1"
	if projectionControl && (adapterMode == "" || target != "contention") {
		t.Fatal("projection-only control requires explicit contention adapter selection")
	}
	if adapterMode != "" {
		evidence = filepath.Join(evidence, "cooperative-sqlite-v1")
		if projectionControl {
			evidence = filepath.Join(evidence, "projection-control")
		}
	}
	if clockSelection != "" {
		evidence = filepath.Join(evidence, clockSelection)
	}
	if err := os.MkdirAll(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	packageName, version, cwd := "", "go1.27.0", "/workspace/tests"
	selections := map[string]string{}
	tags := "disable_grpc_modules,test_dep,sqlite3_dotlk"
	switch target {
	case "sqlite":
		packageName, cwd = "./common/persistence/sql/sqlplugin/sqlite", "/workspace/common/persistence/sql/sqlplugin/sqlite"
		selections["sqlite"] = "."
	case "persistence":
		packageName, cwd = "./common/persistence/tests", "/workspace/common/persistence/tests"
		selections["cancellation"] = "^TestSQLiteTransactionContextCancellation$"
		selections["visibility"] = "^TestSQLiteVisibilitySuite$"
		selections["visibility-persistence"] = "^TestSQLiteVisibilityPersistenceSuite$"
	case "frontend":
		packageName, cwd = "./tests/gomadfunctional", "/workspace/tests/gomadfunctional"
		selections["frontend"] = "^TestFrontendSystemInfo$"
	case "namespace-diagnostic":
		packageName, cwd = "./.tmp/wasm-backend-temporal/frontend-namespace", "/workspace/namespace-diagnostic"
		selections["namespace-diagnostic"] = "^TestNamespaceRPCCause$"
	case "namespace-stack-diagnostic":
		if adapterMode != "cooperative-v1" {
			t.Fatal("namespace stack diagnostic requires the explicitly admitted cooperative SQLite adapter")
		}
		packageName, cwd = "./.tmp/wasm-backend-temporal/frontend-namespace-stack", "/workspace/namespace-diagnostic"
		selections["namespace-stack-diagnostic"] = "^TestNamespacePendingStack$"
	case "grpc-progress-diagnostic":
		packageName, cwd = "./.tmp/wasm-backend-temporal/grpc-progress", "/workspace/grpc-progress-diagnostic"
		selections["grpc-progress-diagnostic"] = "^TestGRPCUnaryProgress$"
	case "contention":
		packageName, cwd = "./.tmp/wasm-backend-temporal/sqlite-contention", "/workspace/contention"
		selections["contention"] = "^TestBusyTimeoutAllowsQueuedLockHolder$"
	case "contention-controls":
		packageName, cwd = "./.tmp/wasm-backend-temporal/sqlite-contention", "/workspace/contention"
		selections["default-timeout"] = "^TestUnreleasedLockPreservesDefaultTimeout$"
		selections["queued-cancellation"] = "^TestQueuedCancellationInterruptsBusyRetry$"
		selections["zero-timeout"] = "^TestZeroBusyTimeoutPreserved$"
		selections["custom-timeout"] = "^TestCustomBusyTimeoutPreserved$"
	case "functional":
		packageName = "./tests"
		selections["workflow"] = "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"
		selections["workflow-activity-cleanup"] = "^TestActivityApiPause_AttributesToActivityInContextMetadata$"
		selections["activity"] = "^TestActivityTestSuite$"
		selections["child"] = "^TestChildWorkflowSuite$"
		selections["update"] = "^TestWorkflowUpdateSuite$"
		selections["timers"] = "^TestUserTimersTestSuite$"
		selections["application-failure"] = "^TestActivityClientTestSuite$/^TestActivity_AttemptsExceeded$"
	case "failure":
		root = filepath.Join(root, "tools", "gomad3", "internal", "gomadtool", "conformance", "testdata")
		packageName, version, cwd = "./io_failure", "go1.27.1", "/workspace/failure"
		tags = "test_dep,gomad_fixture"
		selections["failure"] = "^TestDeterministicIOFailure$"
	default:
		t.Fatalf("unknown Temporal gate %q", target)
	}
	if strings.HasPrefix(target, "contention") || target == "namespace-diagnostic" || target == "namespace-stack-diagnostic" || target == "grpc-progress-diagnostic" {
		owner, staged := "sqlite_contention", "sqlite-contention"
		if target == "namespace-diagnostic" {
			owner, staged = "frontend_namespace", "frontend-namespace"
		}
		if target == "namespace-stack-diagnostic" {
			owner, staged = "frontend_namespace", "frontend-namespace-stack"
		}
		if target == "grpc-progress-diagnostic" {
			owner, staged = "environment", "grpc-progress"
		}
		if err := temporalStageFixture(root, filepath.Join("tools", "gomad_wasm", "testdata", owner), root, filepath.Join(".tmp", "wasm-backend-temporal", staged)); err != nil {
			t.Fatal(err)
		}
	}
	compiler := os.Getenv("GOMAD_WASM_TEMPORAL_GO")
	if !filepath.IsAbs(compiler) {
		t.Fatal("GOMAD_WASM_TEMPORAL_GO must name the absolute stock compiler")
	}
	buildEnv := temporalBuildEnvironment(compiler)
	compilerResult := temporalCommand(t, evidence, target+"-compiler", root, buildEnv, []string{compiler, "version"})
	if !strings.Contains(string(compilerResult.Stdout), version+" ") {
		t.Fatalf("wrong stock compiler: %s", compilerResult.Stdout)
	}
	listCommand := []string{compiler, "list", "-mod=readonly", "-tags", tags, "-deps", "-test", "-json", packageName}
	var projection *temporalSQLiteProjection
	var overlayArgs []string
	if adapterMode != "" {
		discovery := temporalCommand(t, evidence, target+"-discovery", root, buildEnv, listCommand)
		projection = temporalSQLiteAdapter(t, evidence, root, discovery.Stdout)
		originalGraph := temporalCommand(t, evidence, target+"-original-modules", root, buildEnv, []string{compiler, "list", "-mod=readonly", "-m", "-json", "all"})
		effectiveGraph := temporalCommand(t, evidence, target+"-projected-modules", root, buildEnv, []string{compiler, "list", "-modfile", projection.ModFile, "-mod=readonly", "-m", "-json", "all"})
		if err := temporalCompareModuleGraphs(originalGraph.Stdout, effectiveGraph.Stdout, filepath.Dir(projection.Original), projection.PrivateRoot, projection.ModFile); err != nil {
			t.Fatal(err)
		}
		overlayArgs = []string{"-modfile", projection.ModFile}
		if !projectionControl {
			overlayArgs = append(overlayArgs, "-overlay", projection.Overlay)
		}
		listCommand = append([]string{compiler, "list"}, append(overlayArgs, listCommand[2:]...)...)
	}
	listed := temporalCommand(t, evidence, target+"-inputs", root, buildEnv, listCommand)
	inputs := temporalSourceInputs(t, listed.Stdout)
	if projection != nil {
		inputs[projection.Original] = projection.OriginalSHA256
		inputs[projection.Replacement] = projection.ReplacementSHA256
		inputs[projection.Overlay] = projection.OverlaySHA256
		inputs[projection.ModFile] = fileHash(t, projection.ModFile)
		inputs[projection.SumFile] = fileHash(t, projection.SumFile)
		for _, name := range []string{target + "-original-modules.stdout", target + "-projected-modules.stdout"} {
			inputs[filepath.Join(evidence, name)] = fileHash(t, filepath.Join(evidence, name))
		}
		inputs[filepath.Join(projection.PrivateRoot, "conn.go")] = projection.OriginalSHA256
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		file := filepath.Join(root, name)
		if _, err := os.Stat(file); err == nil {
			inputs[file] = fileHash(t, file)
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { temporalVerifyInputs(t, inputs) })
	module := filepath.Join(evidence, target+".test.wasm")
	command := []string{compiler, "test", "-mod=readonly", "-tags", tags, "-c", "-o", module, packageName}
	if projection != nil {
		command = append([]string{compiler, "test"}, append(overlayArgs, command[2:]...)...)
	}
	temporalCommand(t, evidence, target+"-build", root, buildEnv, command)
	temporalVerifyInputs(t, inputs)
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	helper := helperPath(t)
	provenance := struct {
		Compiler, CompilerSHA256, ModuleSHA256, HelperSHA256 string
		BuildCommand, BuildEnvironment                       []string
		Inputs                                               map[string]string
		Imports                                              []Import
		InitialMemoryPages                                   uint64
		Engine                                               EngineIdentity
		SQLiteAdapter                                        *temporalSQLiteProjection
		DiagnosticOnly                                       bool
		ClockSelection, ClockClassification                  string
		Clock                                                ClockPolicy
		ConfigIdentities                                     map[string]string
	}{string(compilerResult.Stdout), fileHash(t, compiler), fileHash(t, module), fileHash(t, helper), command, buildEnv, inputs, imports, pages, PinnedEngine(), projection, strings.HasPrefix(target, "contention") || target == "namespace-diagnostic" || target == "namespace-stack-diagnostic" || target == "grpc-progress-diagnostic", clockSelection, "exploratory; no strict-time or production clock-policy qualification", clock, map[string]string{}}
	temporalJSON(t, filepath.Join(evidence, target+"-provenance.json"), provenance)
	config := Config{
		Profile: StockProfile, WorkingDirectory: cwd,
		Environment:         []string{"PWD=" + cwd, "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC"},
		WritableDirectories: []string{"/workspace", "/tmp", cwd},
		EntropyKey:          sha256.Sum256([]byte("gomad-wasm-temporal-admission/v1")),
		Clock:               clock,
		Limits:              Limits{OutputBytes: 8 << 20, TranscriptBytes: 128 << 20, Calls: 250000, Descriptors: 1024, Files: 10000, FilesystemBytes: 128 << 20, PendingEvents: 4096},
	}
	if target != "failure" {
		captured, err := readonlymount.CaptureReadOnlyMountInputs([]readonlymount.Mapping{{Source: filepath.Join(root, "schema", "sqlite"), Target: "/workspace/schema/sqlite"}}, readonlymount.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		config.CapturedInputs = captured
		temporalJSON(t, filepath.Join(evidence, target+"-captured.json"), captured)
	}
	names := make([]string, 0, len(selections))
	for name := range selections {
		names = append(names, name)
	}
	slices.Sort(names)
	if target == "functional" {
		names = []string{"workflow", "workflow-activity-cleanup", "activity", "child", "update", "timers", "application-failure"}
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			config.Args = []string{"temporal.test", "-test.v", "-test.count=1", "-test.timeout=20m", "-test.run=" + selections[name]}
			if target == "frontend" || target == "functional" || target == "namespace-diagnostic" || target == "namespace-stack-diagnostic" {
				config.Args = append(config.Args, "-persistenceType=sql", "-persistenceDriver=sqlite")
			}
			environment, err := NewEnvironment(config)
			if err != nil {
				t.Fatal(err)
			}
			provenance.ConfigIdentities[name] = environment.Identity()
			temporalJSON(t, filepath.Join(evidence, target+"-provenance.json"), provenance)
			request := Request{HelperPath: helper, HelperSHA256: provenance.HelperSHA256, ModulePath: module, ModuleSHA256: provenance.ModuleSHA256, Engine: provenance.Engine, Imports: imports, InitialMemoryPages: pages, MemoryBytes: 2 << 30, Fuel: 2000000000000, Config: config, Timeout: 15 * time.Minute}
			temporalJSON(t, filepath.Join(evidence, name+"-request.json"), request)
			var first Result
			repeat := 1
			if target == "failure" {
				repeat = 2
			}
			for iteration := range repeat {
				started := time.Now()
				result, err := Run(t.Context(), request)
				prefix := filepath.Join(evidence, fmt.Sprintf("%s-%d", name, iteration))
				temporalWrite(t, prefix+".stdout", result.Stdout)
				temporalWrite(t, prefix+".stderr", result.Stderr)
				temporalWrite(t, prefix+".transcript.jsonl", result.Transcript)
				metadata := result
				metadata.Stdout, metadata.Stderr, metadata.Transcript = nil, nil, nil
				temporalJSON(t, prefix+".json", struct {
					Result                                       Result
					ElapsedNanos                                 int64
					StdoutSHA256, StderrSHA256, TranscriptSHA256 string
					Error                                        string
				}{metadata, time.Since(started).Nanoseconds(), temporalHash(result.Stdout), temporalHash(result.Stderr), temporalHash(result.Transcript), fmt.Sprint(err)})
				if err != nil {
					t.Fatal(err)
				}
				if result.EnvironmentIdentity != provenance.ConfigIdentities[name] {
					t.Fatal("guest configuration identity differs from admitted provenance")
				}
				wantExit := uint32(0)
				if target == "failure" || projectionControl {
					wantExit = 1
				}
				if (target == "namespace-diagnostic" || target == "namespace-stack-diagnostic") && result.ExitCode != nil && *result.ExitCode <= 1 {
					wantExit = *result.ExitCode
				}
				if result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != wantExit || !result.Reaped {
					t.Fatalf("guest %s: termination=%s exit=%v reaped=%v message=%s; complete output retained in %s", name, result.Termination, result.ExitCode, result.Reaped, result.Message, prefix)
				}
				if err := temporalCheckOutput(result.Stdout, selections[name], target == "failure" || projectionControl || (target == "namespace-diagnostic" || target == "namespace-stack-diagnostic") && wantExit == 1); err != nil {
					t.Fatal(err)
				}
				if target == "failure" && !bytes.Contains(result.Stdout, []byte(`deterministic failure after reading "pending"`)) {
					t.Fatal("deliberate application failure was not reached")
				}
				if iteration > 0 && (!bytes.Equal(first.Stdout, result.Stdout) || !bytes.Equal(first.Stderr, result.Stderr) || !bytes.Equal(first.Transcript, result.Transcript)) {
					t.Fatal("fresh deliberate failure output or transcript diverged")
				}
				first = result
				t.Logf("target=%s guest=%s iteration=%d exit=%d environment=%s module=%s transcript=%s memory=%d fuel-remaining=%d", target, name, iteration, wantExit, result.EnvironmentIdentity, request.ModuleSHA256, temporalHash(result.Transcript), result.PeakMemoryBytes, result.FuelRemaining)
			}
		})
	}
}

func temporalClockPolicy(selection string) (ClockPolicy, error) {
	step := uint64(1000000)
	switch selection {
	case "":
	case "read-step-1us-v1":
		step = 1000
	default:
		return ClockPolicy{}, fmt.Errorf("unknown Temporal exploratory clock policy %q", selection)
	}
	return ClockPolicy{EpochNanos: 946684800000000000, ReadStepNanos: step}, nil
}

func temporalVerifyInputs(t *testing.T, inputs map[string]string) {
	t.Helper()
	for name, expected := range inputs {
		if actual := fileHash(t, name); actual != expected {
			t.Fatalf("source input changed during admission: %s: expected %s, got %s", name, expected, actual)
		}
	}
}

func TestTemporalClockPolicy(t *testing.T) {
	for _, test := range []struct {
		selection string
		step      uint64
	}{
		{"", 1000000},
		{"read-step-1us-v1", 1000},
		{"read-step-1ms-v1", 0},
		{"1000", 0},
		{"read-step-1us-v2", 0},
		{"read-step-1us-v1 ", 0},
		{"\xff", 0},
	} {
		t.Run(test.selection, func(t *testing.T) {
			clock, err := temporalClockPolicy(test.selection)
			if test.step == 0 {
				if err == nil || clock != (ClockPolicy{}) {
					t.Fatal("accepted an arbitrary clock policy")
				}
				return
			}
			if err != nil || clock != (ClockPolicy{EpochNanos: 946684800000000000, ReadStepNanos: test.step}) {
				t.Fatalf("clock policy = %+v, error = %v", clock, err)
			}
		})
	}
}

func TestTemporalClockConfigurationIdentity(t *testing.T) {
	var identities []string
	for _, selection := range []string{"", "read-step-1us-v1"} {
		clock, err := temporalClockPolicy(selection)
		if err != nil {
			t.Fatal(err)
		}
		config := testConfig()
		config.Clock = clock
		environment, err := NewEnvironment(config)
		if err != nil {
			t.Fatal(err)
		}
		identities = append(identities, environment.Identity())
	}
	if identities[0] == identities[1] {
		t.Fatal("clock selection was omitted from the configuration identity")
	}
}

func TestTemporalGuestOutput(t *testing.T) {
	for _, test := range []struct {
		name, selection, output string
		failure, valid          bool
	}{
		{"empty", "^TestFrontendSystemInfo$", "", false, false},
		{"zero-match", "^TestFrontendSystemInfo$", "testing: warning: no tests to run\nPASS\n", false, false},
		{"skip", "^TestFrontendSystemInfo$", "--- SKIP: TestFrontendSystemInfo (0.00s)\nPASS\n", false, false},
		{"different", "^TestFrontendSystemInfo$", "--- PASS: TestFrontendSystemInfoOther (0.00s)\nPASS\n", false, false},
		{"pass", "^TestFrontendSystemInfo$", "--- PASS: TestFrontendSystemInfo (0.00s)\nPASS\n", false, true},
		{"grpc-zero-match", "^TestGRPCUnaryProgress$", "testing: warning: no tests to run\nPASS\n", false, false},
		{"grpc-skip", "^TestGRPCUnaryProgress$", "--- SKIP: TestGRPCUnaryProgress (0.00s)\nPASS\n", false, false},
		{"grpc-pass", "^TestGRPCUnaryProgress$", "--- PASS: TestGRPCUnaryProgress (0.00s)\nPASS\n", false, true},
		{"stack-zero-match", "^TestNamespacePendingStack$", "testing: warning: no tests to run\nPASS\n", false, false},
		{"stack-skip", "^TestNamespacePendingStack$", "--- SKIP: TestNamespacePendingStack (0.00s)\nPASS\n", false, false},
		{"stack-pass", "^TestNamespacePendingStack$", "--- PASS: TestNamespacePendingStack (0.00s)\nPASS\n", false, true},
		{"all-suite-skipped", "^TestActivityTestSuite$", "--- PASS: TestActivityTestSuite (0.00s)\n    --- SKIP: TestActivityTestSuite/TestActivity (0.00s)\nPASS\n", false, false},
		{"suite", "^TestActivityTestSuite$", "--- PASS: TestActivityTestSuite (0.00s)\n    --- PASS: TestActivityTestSuite/TestActivity (0.00s)\nPASS\n", false, true},
		{"failure", "^TestDeterministicIOFailure$", "--- FAIL: TestDeterministicIOFailure (0.00s)\nFAIL\n", true, true},
		{"wrong-failure", "^TestDeterministicIOFailure$", "--- FAIL: TestOther (0.00s)\nFAIL\n", true, false},
		{"nested", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$", "    --- PASS: TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds (0.00s)\nPASS\n", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := temporalCheckOutput([]byte(test.output), test.selection, test.failure); (err == nil) != test.valid {
				t.Fatalf("output validity: expected %v, got %v", test.valid, err)
			}
		})
	}
}

func temporalCheckOutput(output []byte, selection string, failure bool) error {
	names := []string{strings.NewReplacer("^", "", "$", "").Replace(selection)}
	if selection == "." {
		names = []string{"TestDriverErrors", "TestDriverTimestampCompatibility", "TestDriverMemorySchema", "TestDriverTimestampPrecision", "TestQueryConverter_GetCoalesceCloseTimeExpr", "TestQueryConverter_ConvertKeywordListComparisonExpr", "TestQueryConverter_ConvertTextComparisonExpr", "TestQueryConverter_BuildSelectStmt", "TestQueryConverter_BuildCountStmt", "TestRewriteSchemaStatements_EnforcesVarcharLengthChecks"}
	}
	status := "PASS"
	if failure {
		status = "FAIL"
	}
	for _, name := range names {
		found, child := false, false
		for line := range strings.SplitSeq(string(output), "\n") {
			line = strings.TrimSpace(line)
			found = found || strings.HasPrefix(line, "--- "+status+": "+name+" (")
			child = child || strings.HasPrefix(line, "--- "+status+": "+name+"/")
		}
		if !found || (strings.HasSuffix(name, "Suite") && !child) {
			return fmt.Errorf("guest did not execute named %s outcome for %s", status, name)
		}
	}
	return nil
}

func temporalBuildEnvironment(compiler string) []string {
	environment := []string{}
	for _, name := range []string{"HOME", "TMPDIR", "GOCACHE", "GOPATH", "GOMODCACHE"} {
		if value, found := os.LookupEnv(name); found {
			environment = append(environment, name+"="+value)
		}
	}
	return append(environment, "PATH="+filepath.Dir(compiler)+":"+os.Getenv("PATH"), "GOOS=wasip1", "GOARCH=wasm", "CGO_ENABLED=0", "GOWORK=off", "GOENV=off", "GOFLAGS=", "GOEXPERIMENT=nogreenteagc", "GOTOOLCHAIN=local")
}

type temporalSQLiteProjection struct {
	Policy, Module, Version                                  string
	Original, OriginalSHA256, Replacement, ReplacementSHA256 string
	Overlay, OverlaySHA256                                   string
	PrivateRoot, ModFile, SumFile                            string
	OriginalInventory, PrivateInventory                      readonlymount.CapturedInputsManifest
}

func temporalSQLiteAdapter(t *testing.T, evidence, root string, listed []byte) *temporalSQLiteProjection {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(listed))
	var projection *temporalSQLiteProjection
	for {
		var pkg struct {
			ImportPath, Dir string
			GoFiles         []string
			Module          *struct {
				Path, Version, GoMod string
				Replace              json.RawMessage
			}
		}
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if pkg.ImportPath != "github.com/ncruces/go-sqlite3" {
			continue
		}
		if projection != nil || pkg.Module == nil || !filepath.IsAbs(pkg.Dir) || !slices.Contains(pkg.GoFiles, "conn.go") {
			t.Fatal("SQLite adapter requires one selected pinned conn.go source and its module identity")
		}
		original := filepath.Join(pkg.Dir, "conn.go")
		source, err := os.ReadFile(original)
		if err != nil {
			t.Fatal(err)
		}
		dependency := sqlitebusy.Dependency{Path: pkg.Module.Path, Version: pkg.Module.Version, Replaced: len(pkg.Module.Replace) > 0 && string(pkg.Module.Replace) != "null"}
		adapter, err := sqlitebusy.Prepare(dependency, source)
		if err != nil {
			t.Fatal(err)
		}
		cachedMod, err := os.ReadFile(pkg.Module.GoMod)
		if err != nil {
			t.Fatal(err)
		}
		moduleMod, err := os.ReadFile(filepath.Join(pkg.Dir, "go.mod"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(cachedMod, moduleMod) {
			t.Fatal("cached SQLite module metadata differs from captured module go.mod")
		}
		t.Cleanup(func() {
			if fileHash(t, pkg.Module.GoMod) != temporalHash(cachedMod) {
				t.Fatal("original SQLite cached module metadata changed")
			}
		})
		captured := temporalCaptureModule(t, pkg.Dir)
		privateParent := filepath.Join(evidence, "private-modules")
		if err := os.MkdirAll(privateParent, 0700); err != nil {
			t.Fatal(err)
		}
		privateRoot := filepath.Join(privateParent, strings.TrimPrefix(string(captured.Manifest.SHA256), "sha256:"))
		if _, err := os.Stat(privateRoot); os.IsNotExist(err) {
			if err := temporalMaterializeModule(privateRoot, captured); err != nil {
				t.Fatal(err)
			}
		} else if err != nil {
			t.Fatal(err)
		}
		privateCaptured := temporalCaptureModule(t, privateRoot)
		if !bytes.Equal(captured.Descriptor, privateCaptured.Descriptor) {
			t.Fatal("private SQLite module does not match original complete captured inventory")
		}
		verify := func() {
			for _, location := range []string{pkg.Dir, privateRoot} {
				if current := temporalCaptureModule(t, location); !bytes.Equal(captured.Descriptor, current.Descriptor) {
					t.Fatalf("SQLite module inventory changed during admission: %s", location)
				}
			}
		}
		t.Cleanup(verify)
		for _, name := range []string{"go.mod", "go.sum"} {
			contents, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				t.Fatal(err)
			}
			if name == "go.mod" {
				contents = append(contents, []byte("\nreplace github.com/ncruces/go-sqlite3 v0.35.6 => "+strconv.Quote(privateRoot)+"\n")...)
			}
			output := "temporal.mod"
			if name == "go.sum" {
				output = "temporal.sum"
			}
			temporalWrite(t, filepath.Join(evidence, output), contents)
		}
		temporalJSON(t, filepath.Join(evidence, "sqlite-original-capture.json"), captured)
		if fileHash(t, filepath.Join(privateRoot, "conn.go")) != adapter.OriginalSHA256 {
			t.Fatal("private SQLite source differs from admitted source")
		}
		replacement := filepath.Join(evidence, "sqlite-conn-cooperative.go")
		temporalWrite(t, replacement, adapter.Source)
		overlay := filepath.Join(evidence, "sqlite-overlay.json")
		temporalJSON(t, overlay, struct{ Replace map[string]string }{map[string]string{filepath.Join(privateRoot, "conn.go"): replacement}})
		projection = &temporalSQLiteProjection{Policy: adapter.Policy, Module: dependency.Path, Version: dependency.Version, Original: original, OriginalSHA256: adapter.OriginalSHA256, Replacement: replacement, ReplacementSHA256: adapter.ReplacementSHA256, Overlay: overlay, OverlaySHA256: fileHash(t, overlay), PrivateRoot: privateRoot, ModFile: filepath.Join(evidence, "temporal.mod"), SumFile: filepath.Join(evidence, "temporal.sum"), OriginalInventory: captured.Manifest, PrivateInventory: privateCaptured.Manifest}
	}
	if projection == nil {
		t.Fatal("SQLite adapter selected without the pinned SQLite dependency")
	}
	return projection
}

func temporalCaptureModule(t *testing.T, source string) readonlymount.CapturedInputs {
	t.Helper()
	captured, err := readonlymount.CaptureReadOnlyMountInputs([]readonlymount.Mapping{{Source: source, Target: "/module"}}, readonlymount.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return captured
}

func temporalCommand(t *testing.T, evidence, name, root string, environment, command []string) hostexec.Result {
	t.Helper()
	started := time.Now()
	result, err := hostexec.Run(context.Background(), hostexec.Request{Command: command, Dir: root, Env: environment, Timeout: 15 * time.Minute, TerminateGrace: time.Second, OutputLimit: 64 << 20})
	temporalWrite(t, filepath.Join(evidence, name+".stdout"), result.Stdout)
	temporalWrite(t, filepath.Join(evidence, name+".stderr"), result.Stderr)
	temporalJSON(t, filepath.Join(evidence, name+".json"), struct {
		Command      []string
		ExitCode     int
		ElapsedNanos int64
	}{command, result.ExitCode, time.Since(started).Nanoseconds()})
	if err != nil || result.Termination != hostexec.TerminationExit || result.ExitCode != 0 || !result.GroupGone {
		t.Fatalf("infrastructure command %v: %v, termination=%s exit=%d\n%s\n%s", command, err, result.Termination, result.ExitCode, result.Stdout, result.Stderr)
	}
	return result
}

func temporalSourceInputs(t *testing.T, data []byte) map[string]string {
	t.Helper()
	inputs := map[string]string{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var pkg struct {
			Dir                                                              string
			GoFiles, CgoFiles, SFiles, EmbedFiles, TestGoFiles, XTestGoFiles []string
		}
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		for _, names := range [][]string{pkg.GoFiles, pkg.CgoFiles, pkg.SFiles, pkg.EmbedFiles, pkg.TestGoFiles, pkg.XTestGoFiles} {
			for _, name := range names {
				if !filepath.IsAbs(name) {
					name = filepath.Join(pkg.Dir, name)
				}
				inputs[name] = fileHash(t, name)
			}
		}
	}
	return inputs
}

func temporalHash(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func temporalJSON(t *testing.T, name string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	temporalWrite(t, name, append(data, '\n'))
}

func temporalWrite(t *testing.T, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(name, data, 0600); err != nil {
		t.Fatal(err)
	}
}
