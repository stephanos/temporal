package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/backend"
	simulation "go.temporal.io/server/tools/gomad3/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/installation"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/qualification"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/internal/evidence/artifact"
	captured "go.temporal.io/server/tools/gomad3/internal/evidence/artifact/capture"
	gomadqualification "go.temporal.io/server/tools/gomad3/internal/evidence/qualification"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/runner"
)

func run(arguments []string, stdout, stderr io.Writer) int {
	status, handled := Run(arguments, stdout, stderr, func() (installation.Description, error) {
		return installation.Resolve(installation.Spec{EnvironmentManifest: os.Getenv(installation.Environment)})
	})
	if !handled {
		panic("backend command test selected a shared launcher command")
	}
	return status
}

func TestQualifyValidatesSpecBeforeResolvingInstallation(t *testing.T) {
	t.Setenv(installation.Environment, "/missing/installation-must-not-be-resolved")
	invalid := filepath.Join(t.TempDir(), "invalid-spec.json")
	if err := os.WriteFile(invalid, []byte(`{"Options":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, arguments := range [][]string{{"qualify"}, {"qualify", "--spec", "/missing/spec"}, {"qualify", "--spec", invalid}, {"qualify", "--spec", invalid, "extra"}} {
		var stdout, stderr bytes.Buffer
		if status := run(arguments, &stdout, &stderr); status != 2 || strings.Contains(stderr.String(), "installation-must-not-be-resolved") || stdout.Len() != 0 {
			t.Fatalf("%q: status=%d stdout=%q stderr=%q", arguments, status, stdout.String(), stderr.String())
		}
		if strings.Contains(stderr.String(), "unavailable") {
			t.Fatalf("qualify did not validate operator spec: %s", stderr.String())
		}
	}
}

func TestFinalQualificationCostGatePreservesPublishedSupportOutcome(t *testing.T) {
	for _, test := range []struct {
		name          string
		provisional   bool
		targetSuccess bool
		costFailure   string
		budgetFailure string
		budget        *qualification.BudgetEvaluation
		status        int
	}{
		{"final success", false, true, "", "", &qualification.BudgetEvaluation{WithinBudget: true}, 0},
		{"final application failure", false, false, "", "", &qualification.BudgetEvaluation{WithinBudget: true}, 1},
		{"final missing cost measurement", false, true, "missing CPU", "", &qualification.BudgetEvaluation{WithinBudget: true}, 3},
		{"final missing budget measurement", false, true, "", "missing operator RSS", nil, 3},
		{"final absent budget", false, true, "", "", nil, 3},
		{"final exceeded budget", false, true, "", "", &qualification.BudgetEvaluation{WithinBudget: false}, 3},
		{"application failure with invalid cost", false, false, "missing CPU", "", &qualification.BudgetEvaluation{WithinBudget: true}, 3},
		{"provisional success diagnostic", true, true, "missing CPU", "missing operator RSS", nil, 0},
		{"provisional application diagnostic", true, false, "missing CPU", "missing operator RSS", nil, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := qualification.Result{
				Manifest:      qualification.Manifest{Provisional: test.provisional},
				Qualification: gomadqualification.QualificationReport{Qualified: test.targetSuccess, Deterministic: true, TargetSuccess: test.targetSuccess},
				CostFailure:   test.costFailure, BudgetFailure: test.budgetFailure, Budget: test.budget,
			}
			var stdout, stderr bytes.Buffer
			receipt := qualification.PublicationReceipt{ResultSHA256: record.HashBytes([]byte("published result")), OperatorProcessID: uint64(os.Getpid())}
			if status := emitQualificationResult(&stdout, &stderr, "/retained/result.json", result, receipt, nil); status != test.status {
				t.Fatalf("status=%d want=%d stderr=%s", status, test.status, stderr.String())
			}
			var response struct {
				Path    string                           `json:"result_path"`
				Receipt qualification.PublicationReceipt `json:"operator_receipt"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
				t.Fatalf("raw result lost before status: %v", err)
			}
			if response.Path != "/retained/result.json" || response.Receipt.ResultSHA256 != receipt.ResultSHA256 || response.Receipt.OperatorProcessID != receipt.OperatorProcessID || result.Qualification.TargetSuccess != test.targetSuccess || !result.Qualification.Deterministic || result.CostFailure != test.costFailure || result.BudgetFailure != test.budgetFailure {
				t.Fatalf("cost status changed raw support evidence or lost retained pointer/receipt: %+v", response)
			}
			if test.status == 3 && (!strings.Contains(stderr.String(), "cost/budget") || strings.Contains(stderr.String(), "target_failure")) {
				t.Fatalf("cost failure not identified separately from application failure: %s", stderr.String())
			}
		})
	}
}

func TestQualificationResponseDoesNotSerializeTheRetainedResult(t *testing.T) {
	result := qualification.Result{
		Manifest:      qualification.Manifest{Provisional: true, Unsupported: []string{strings.Repeat("unbounded retained evidence", 1<<16)}},
		Qualification: gomadqualification.QualificationReport{Qualified: true, Deterministic: true, TargetSuccess: true},
	}
	var stdout, stderr bytes.Buffer
	if status := emitQualificationResult(&stdout, &stderr, "/retained/result.json", result, qualification.PublicationReceipt{}, nil); status != 0 {
		t.Fatalf("provisional response: status=%d stderr=%s", status, stderr.String())
	}
	if stdout.Len() > 8<<10 || bytes.Contains(stdout.Bytes(), []byte(`"result":`)) {
		t.Fatalf("qualification encoded retained evidence after observation: response bytes=%d", stdout.Len())
	}
}

func TestQualificationResponseRetainsReceiptBeforePublicationFailure(t *testing.T) {
	result := qualification.Result{Qualification: gomadqualification.QualificationReport{Qualified: true, Deterministic: true, TargetSuccess: true}, Budget: &qualification.BudgetEvaluation{WithinBudget: true}}
	for _, receipt := range []qualification.PublicationReceipt{
		{},
		{ResultSHA256: record.HashBytes([]byte("published result")), OperatorProcessID: uint64(os.Getpid())},
	} {
		var stdout, stderr bytes.Buffer
		if status := emitQualificationResult(&stdout, &stderr, "/retained/result.json", result, receipt, errors.New("missing publication self observation")); status != 3 {
			t.Fatalf("invalid receipt was cost acceptance: status=%d stderr=%s", status, stderr.String())
		}
		var response struct {
			Path    string                           `json:"result_path"`
			Receipt qualification.PublicationReceipt `json:"operator_receipt"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &response); err != nil || response.Path != "/retained/result.json" || response.Receipt.ResultSHA256 != receipt.ResultSHA256 || response.Receipt.OperatorProcessID != receipt.OperatorProcessID {
			t.Fatalf("publication failure lost raw retained evidence: response=%s error=%v", stdout.String(), err)
		}
		if !strings.Contains(stderr.String(), "publication") || strings.Contains(stderr.String(), "target_failure") {
			t.Fatalf("publication failure classified as an application bug: %s", stderr.String())
		}
	}
}

func TestQualificationResponseBoundsAbsolutePathAndEncodedBytes(t *testing.T) {
	result := qualification.Result{Manifest: qualification.Manifest{Provisional: true}, Qualification: gomadqualification.QualificationReport{Qualified: true, Deterministic: true, TargetSuccess: true}}
	for _, path := range []string{"relative/result.json", "/" + strings.Repeat("x", 1024)} {
		var stdout, stderr bytes.Buffer
		if status := emitQualificationResult(&stdout, &stderr, path, result, qualification.PublicationReceipt{}, nil); status != 3 || stdout.Len() != 0 {
			t.Fatalf("unbounded/relative response path accepted: status=%d bytes=%d", status, stdout.Len())
		}
	}
	var stdout, stderr bytes.Buffer
	if status := emitQualificationResult(&stdout, &stderr, "/"+strings.Repeat("<", 1023), result, qualification.PublicationReceipt{}, nil); status != 0 || stdout.Len() > 8<<10 {
		t.Fatalf("escaped bounded path exceeded response limit: status=%d bytes=%d stderr=%s", status, stdout.Len(), stderr.String())
	}
}

func TestQualificationResponseBoundsPublicationErrorOutput(t *testing.T) {
	result := qualification.Result{Qualification: gomadqualification.QualificationReport{Qualified: true, Deterministic: true, TargetSuccess: true}}
	var stdout, stderr bytes.Buffer
	status := emitQualificationResult(&stdout, &stderr, "/retained/result.json", result, qualification.PublicationReceipt{}, errors.New(strings.Repeat("publication path diagnostic ", 1024)))
	if status != 3 || stderr.Len() > 2<<10 || !strings.Contains(stderr.String(), "publication") || !strings.Contains(stderr.String(), "truncated") {
		t.Fatalf("unbounded post-observation error: status=%d stderr bytes=%d", status, stderr.Len())
	}
}

func TestReportRejectsInvalidResultWithoutResolvingInstallation(t *testing.T) {
	t.Setenv(installation.Environment, "/missing/installation-must-not-be-resolved")
	invalid := filepath.Join(t.TempDir(), "invalid-result.json")
	if err := os.WriteFile(invalid, []byte(`{"schema":"invented"}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		arguments []string
		status    int
	}{
		{[]string{"report"}, 2},
		{[]string{"report", "--result", "/missing/qualification-result"}, 3},
		{[]string{"report", "--result", invalid}, 3},
		{[]string{"report", "--result", "/missing/qualification-result", "extra"}, 2},
	} {
		var stdout, stderr bytes.Buffer
		if status := run(test.arguments, &stdout, &stderr); status != test.status || strings.Contains(stderr.String(), "installation-must-not-be-resolved") || stdout.Len() != 0 {
			t.Fatalf("%q: status=%d stdout=%q stderr=%q", test.arguments, status, stdout.String(), stderr.String())
		}
	}
}

func TestOperatorReportsRetainedQualificationWithoutInstallation(t *testing.T) {
	path := os.Getenv("GOMAD_WASM_QUALIFICATION_RESULT")
	if path == "" {
		t.Skip("explicit retained qualification result selects the source-less report gate")
	}
	t.Setenv(installation.Environment, "/missing/source-less-report-installation")
	var stdout, stderr bytes.Buffer
	if status := run([]string{"report", "--result", path}, &stdout, &stderr); status != 0 {
		t.Fatalf("source-less report: status=%d stderr=%s", status, stderr.String())
	}
	var result qualification.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Qualification.Deterministic || len(result.Qualification.Executions) < 2 || result.Cost == nil || result.CostFailure != "" || result.Cost.Executions < 2 || result.Cost.Replays < 2 || result.Cost.CPUNanos == 0 || result.Cost.GuestPeakMemoryBytes == 0 || result.Cost.HostPeakRSSBytes == 0 || result.Manifest.IntegratedSourceSHA256 == "" {
		t.Fatalf("report lacks actual repeated support/cost evidence: %+v", result)
	}
	for _, execution := range result.Qualification.Executions {
		if execution.Replay == nil || !execution.Replay.Match {
			t.Fatalf("report omitted actual fresh replay: %+v", execution)
		}
	}
}

func TestInstalledOperatorReplaysRetainedClusterWithInstalledHelper(t *testing.T) {
	path := os.Getenv("GOMAD_WASM_CLUSTER_ARTIFACT")
	if path == "" {
		t.Skip("explicit retained cluster Artifact selects the detached operator gate")
	}
	compiler := os.Getenv("GOMAD3_STOCK_GO")
	if compiler == "" {
		t.Fatal("explicit stock compiler required to build the operator")
	}
	installed := os.Getenv(installation.Environment)
	if !filepath.IsAbs(installed) {
		t.Fatalf("%s must name the installation whose helper replays the cluster", installation.Environment)
	}
	opened, err := artifact.OpenArtifact(path)
	if err != nil {
		t.Fatal(err)
	}
	manifest := opened.Manifest()
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	if manifest.Target.Backend == nil || manifest.Target.Backend.Evidence == nil || manifest.Target.Backend.Evidence.Schema != "gomad3.wasm-simulation-retained/v1" {
		t.Fatal("test input is not typed retained cluster evidence")
	}
	root := t.TempDir()
	binary := filepath.Join(root, "gomad")
	build := exec.CommandContext(t.Context(), compiler, "build", "-p=1", "-trimpath", "-o", binary, "../../../../cmd/gomad")
	build.Env = operatorEnvironment()
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build operator: %v\n%s", err, output)
	}
	want := 1
	if manifest.Outcome.Domain == "success" {
		want = 0
	}
	output := invokeOperator(t, binary, want, "--installation", installed, "replay", path)
	if !bytes.Contains(output, []byte("reproduced=true")) || !bytes.Contains(output, []byte("choice-replay=none")) {
		t.Fatalf("retained cluster did not reproduce its sealed lifecycle: %s", output)
	}
	// Retained helper bytes are never executed, so replay without an
	// installed helper fails closed.
	invokeOperator(t, binary, 3, "--installation", "/missing/detached-cluster-installation", "replay", path)
	for _, flag := range []string{"--verify-only", "--observed=" + filepath.Join(root, "observed"), "--toolchain-root=/missing/native"} {
		invokeOperator(t, binary, 2, "--installation", "/missing/detached-cluster-installation", "replay", flag, path)
	}
	corrupt := filepath.Join(root, "corrupt-artifact")
	if err := os.CopyFS(corrupt, os.DirFS(path)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(corrupt, 0700); err != nil {
		t.Fatal(err)
	}
	payloadPath := filepath.Join(corrupt, filepath.FromSlash(manifest.Target.Backend.Evidence.File))
	payload, err := os.ReadFile(payloadPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) == 0 {
		t.Fatal("retained cluster payload is empty")
	}
	payload[len(payload)-1] ^= 1
	if err := os.WriteFile(payloadPath, payload, 0600); err != nil {
		t.Fatal(err)
	}
	invokeOperator(t, binary, 3, "--installation", installed, "replay", corrupt)
}

func TestInstalledOperatorDiscoversInspectsAndReplaysWithoutBuildInputs(t *testing.T) {
	compiler := os.Getenv("GOMAD3_STOCK_GO")
	if compiler == "" {
		t.Skip("explicit stock Go compiler selects the installed operator guest gate")
	}
	root := t.TempDir()
	binary := filepath.Join(root, "gomad")
	build := exec.CommandContext(t.Context(), compiler, "build", "-p=1", "-trimpath", "-o", binary, "../../../../cmd/gomad")
	build.Env = operatorEnvironment()
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build installed operator: %v\n%s", err, output)
	}
	helperPath := os.Getenv("GOMAD_WASM_TEST_HELPER")
	if helperPath == "" {
		helperPath = "../wasmhost/target/release/gomad3-wasmhost"
	}
	helper, err := filepath.Abs(helperPath)
	if err != nil {
		t.Fatal(err)
	}
	pinnedHelper := helper
	fixture, err := os.ReadFile("../../native/conformance/testdata/io_failure/io_failure_test.go")
	if err != nil {
		t.Fatal(err)
	}
	capturedFixture, err := os.ReadFile("testdata/captured_test.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []string{wasi.StockProfile, wasi.CooperativeProfile} {
		t.Run(profile, func(t *testing.T) {
			directory := t.TempDir()
			// Renamed away below to prove source-less operation, so this subtest owns a private clone.
			compilerRoot := stockgo.Clone(t, filepath.Join(directory, "go"))
			source, inputs := filepath.Join(directory, "source"), filepath.Join(directory, "inputs")
			for _, path := range []string{source, inputs} {
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			}
			for name, data := range map[string][]byte{"go.mod": []byte("module gomad.wasm.operator.fixture\n\ngo 1.27.1\n"), "io_failure_test.go": fixture, "captured_test.go": capturedFixture} {
				if err := os.WriteFile(filepath.Join(source, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(inputs, "control"), []byte("immutable operator input"), 0600); err != nil {
				t.Fatal(err)
			}
			captured, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: inputs, Target: "/inputs"}}, captured.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			config := installation.DefaultConfig(profile)
			config.CapturedInputs = captured
			configPath := filepath.Join(directory, "config.json")
			data, err := json.Marshal(config)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(configPath, data, 0600); err != nil {
				t.Fatal(err)
			}
			manifestPath := filepath.Join(directory, "installation.json")
			arguments := []string{"install", "--output", manifestPath, "--compiler", filepath.Join(compilerRoot, "bin/go"), "--helper", pinnedHelper, "--cache", filepath.Join(directory, "cache"), "--compile-cache", filepath.Join(directory, "engine-cache"), "--profile", profile, "--config", configPath}
			if profile == wasi.CooperativeProfile {
				runtimeRoot := filepath.Join(directory, "runtime")
				if err := os.Mkdir(runtimeRoot, 0700); err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"gomad_semantics.go.txt", "gomad_choicewire_generated.go.txt"} {
					data, err := os.ReadFile(filepath.Join("../../../model/runtimesource", name))
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(runtimeRoot, name), data, 0600); err != nil {
						t.Fatal(err)
					}
				}
				arguments = append(arguments, "--runtime-root", runtimeRoot)
			}
			invokeOperator(t, binary, 0, arguments...)
			discovered := invokeOperator(t, binary, 0, "--installation", manifestPath, "discover")
			if !bytes.Contains(discovered, []byte(profile)) {
				t.Fatalf("discovery omitted profile: %s", discovered)
			}
			explore := []string{"--installation", manifestPath, "explore", "--json", "--seeds=7", "--parallel=1", "--artifacts", filepath.Join(directory, "artifacts"), "--working-dir", source, "--build-tag=gomad_fixture", "--build-tag=test_dep"}
			if profile == wasi.CooperativeProfile {
				explore = append(explore, "--choices", "--diagnostics")
			}
			explore = append(explore, "go-test", ".", "--", "-test.run=^(TestDeterministicIOFailure|TestCapturedInputFailure)$", "-test.v")
			output := invokeOperator(t, binary, 1, explore...)
			var artifactPath string
			decoder := json.NewDecoder(bytes.NewReader(output))
			for {
				var event struct {
					Type          string `json:"type"`
					Path          string `json:"path"`
					ReplayCommand string `json:"replay_command"`
				}
				if err := decoder.Decode(&event); errors.Is(err, io.EOF) {
					break
				} else if err != nil {
					t.Fatal(err)
				}
				if event.Type == "artifact" {
					artifactPath = event.Path
					if !strings.HasPrefix(event.ReplayCommand, "gomad --installation ") {
						t.Fatalf("incorrect replay command: %s", event.ReplayCommand)
					}
				}
			}
			if artifactPath == "" {
				t.Fatalf("no discovered artifact: %s", output)
			}
			invokeOperator(t, binary, 0, "inspect", "--json", artifactPath)
			opened, err := artifact.OpenArtifact(artifactPath)
			if err != nil {
				t.Fatal(err)
			}
			manifest := opened.Manifest()
			guestOutput, err := opened.ReadPayload("stdout", 8<<20)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(guestOutput, []byte(`intentional failure after captured input "immutable operator input"`)) || !bytes.Contains(guestOutput, []byte(`deterministic failure after reading "pending"`)) {
				t.Fatalf("guest did not exercise both unchanged failure and captured input: %s", guestOutput)
			}
			if profile == wasi.StockProfile && manifest.ReplayMode != record.ReplayObserved || profile == wasi.CooperativeProfile && (manifest.ReplayMode != record.ReplayExact || manifest.ChoiceProfile == nil) {
				t.Fatalf("incorrect profile capabilities: %#v", manifest)
			}
			if err := opened.Close(); err != nil {
				t.Fatal(err)
			}
			var resultPath string
			if frozen := os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_SHA256"); frozen != "" {
				specPath := filepath.Join(directory, "qualification-spec.json")
				specification := map[string]any{
					"Manifest":     map[string]any{"Provisional": os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST") == "", "IntegratedSourceManifestPath": os.Getenv("GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST"), "ID": "installed-operator-canary", "Workload": "fixture", "IntegratedSourceSHA256": frozen, "Seed": 7, "Repeat": 2, "Unsupported": []string{"production Temporal support", "cross-host exact portability", "weak-memory coverage"}},
					"Campaign":     map[string]any{"Target": map[string]any{"Backend": "wasm", "Kind": "go-test", "Source": ".", "WorkingDir": source, "PreparationRoot": filepath.Join(directory, "qualification-preparation"), "BuildTags": []string{"gomad_fixture", "test_dep"}, "Args": []string{"-test.run=^(TestDeterministicIOFailure|TestCapturedInputFailure)$", "-test.v"}}, "ExecutionTimeout": uint64(time.Minute), "OverallTimeout": uint64(3 * time.Minute), "TerminateGrace": uint64(100 * time.Millisecond), "OnFailure": "all", "FailureBudget": 1, "OutputLimit": 8 << 20, "WorldTransitionLimit": 1 << 20, "Artifacts": filepath.Join(directory, "qualification-artifacts"), "RunnerBuild": "installed-operator-qualification"},
					"ArtifactRoot": filepath.Join(directory, "qualification-artifacts"),
				}
				data, err := json.Marshal(specification)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(specPath, data, 0600); err != nil {
					t.Fatal(err)
				}
				qualified := invokeOperator(t, binary, 1, "--installation", manifestPath, "qualify", "--spec", specPath)
				var observed struct {
					Path    string                           `json:"result_path"`
					Receipt qualification.PublicationReceipt `json:"operator_receipt"`
				}
				if err := json.Unmarshal(qualified, &observed); err != nil {
					t.Fatal(err)
				}
				result, err := qualification.OpenResult(observed.Path)
				if err != nil {
					t.Fatal(err)
				}
				if !result.Qualification.Deterministic || result.Qualification.TargetSuccess || result.Cost == nil || result.CostFailure != "" || result.Cost.Executions != 2 || result.Cost.Replays != 2 {
					t.Fatalf("installed actual qualification missing measured replay: %+v", result)
				}
				if result.OperatorPreparationBefore == nil || result.OperatorExecutionAfter == nil || result.OperatorPreparationBefore.PeakRSSBytes == 0 || result.OperatorExecutionAfter.PeakRSSBytes == 0 {
					t.Fatalf("qualification omitted operator preparation-through-execution accounting: %+v", result)
				}
				receipt, err := qualification.OpenPublicationReceipt(observed.Path)
				if err != nil {
					t.Fatal(err)
				}
				if len(qualified) > 8<<10 || receipt.ResultSHA256 != observed.Receipt.ResultSHA256 || receipt.OperatorProcessID != observed.Receipt.OperatorProcessID || receipt.OperatorAfter == nil || observed.Receipt.OperatorAfter == nil || *receipt.OperatorAfter != *observed.Receipt.OperatorAfter {
					t.Fatalf("bounded response changed actual publication receipt: %+v", observed)
				}
				tail, err := qualification.ValidatePublicationReceipt(observed.Path, result, receipt)
				if err != nil {
					t.Fatalf("separate reader rejected actual publication receipt: %v", err)
				}
				resultPath = observed.Path
				t.Logf("operator qualification profile=%s manifest=%+v cost=%+v samples=%+v publication=%+v terminal-self-cpu-tail=%d", profile, result.Manifest, *result.Cost, result.Samples, receipt, tail)
			}
			unavailable := []string{compilerRoot, source, inputs, configPath, filepath.Join(directory, "cache")}
			if profile == wasi.CooperativeProfile {
				unavailable = append(unavailable, filepath.Join(directory, "runtime"))
			}
			for _, path := range unavailable {
				if err := os.Rename(path, path+"-unavailable"); err != nil {
					t.Fatal(err)
				}
			}
			if resultPath != "" {
				invokeOperator(t, binary, 0, "--installation", "/missing/source-less-report-manifest", "report", "--result", resultPath)
			}
			invokeOperator(t, binary, 0, "--installation", manifestPath, "discover")
			invokeOperator(t, binary, 0, "--installation", manifestPath, "replay", "--verify-only", artifactPath)
			replay := invokeOperator(t, binary, 1, "--installation", manifestPath, "replay", artifactPath)
			if !bytes.Contains(replay, []byte("reproduced=true")) {
				t.Fatalf("fresh replay did not match: %s", replay)
			}
			cacheRoot := filepath.Join(directory, "engine-cache")
			entries, err := os.ReadDir(cacheRoot)
			if err != nil {
				t.Fatal(err)
			}
			var compiledEntry string
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), ".gomad-cache-v3") {
					if compiledEntry != "" {
						t.Fatal("same retained module generated multiple compiled-code identities")
					}
					compiledEntry = filepath.Join(cacheRoot, entry.Name())
				}
			}
			if compiledEntry == "" {
				t.Fatal("installed helper did not publish a qualified compiled-code cache entry")
			}
			compiled, err := os.ReadFile(compiledEntry)
			if err != nil {
				t.Fatal(err)
			}
			entryInfo, err := os.Stat(compiledEntry)
			if err != nil || entryInfo.Mode().Perm() != 0600 {
				t.Fatalf("compiled code is not private regular storage: %v %v", entryInfo, err)
			}
			poisoned := append([]byte(nil), compiled...)
			poisoned[len(poisoned)-1] ^= 1
			if err := os.WriteFile(compiledEntry, poisoned, 0600); err != nil {
				t.Fatal(err)
			}
			invokeOperator(t, binary, 3, "--installation", manifestPath, "replay", artifactPath)
			if err := os.WriteFile(compiledEntry, compiled, 0600); err != nil {
				t.Fatal(err)
			}
			held := compiledEntry + ".held"
			if err := os.Rename(compiledEntry, held); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(held, compiledEntry); err != nil {
				t.Fatal(err)
			}
			invokeOperator(t, binary, 3, "--installation", manifestPath, "replay", artifactPath)
			if err := os.Remove(compiledEntry); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(held, compiledEntry); err != nil {
				t.Fatal(err)
			}
			invokeOperator(t, binary, 1, "--installation", manifestPath, "replay", artifactPath)
			t.Logf("private compiled code profile=%s identity=%s bytes=%d envelope=%s poison/symlink replay rejected, restored fresh replay matched", profile, record.HashBytes(compiled), len(compiled), filepath.Base(compiledEntry))
			var installationIdentity installation.Description
			if err := json.Unmarshal(discovered, &installationIdentity); err != nil {
				t.Fatal(err)
			}
			t.Logf("profile=%s manifest=%s module=%s record=%s helper=%s compiler=%s model=%s runtime=%s; source/compiler/runtime/captured-input originals unavailable; fresh replay=%s", profile, installationIdentity.SHA256, manifest.Target.SHA256, manifest.RecordHash, installationIdentity.Manifest.Options.HelperSHA256, installationIdentity.Manifest.Options.CompilerSHA256, installationIdentity.Manifest.Options.ModelSHA256, installationIdentity.Manifest.RuntimeImplementationSHA256, bytes.TrimSpace(replay))
		})
	}
}

func operatorEnvironment() []string {
	var environment []string
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		switch name {
		case "GOROOT", "GOMADSEED", "GOMAD3_CHILD_SEED", "GOTOOLCHAIN", "GOWORK", "GOFLAGS", installation.Environment:
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment, "GOENV=off", "GOTOOLCHAIN=local", "GOWORK=off", "GOFLAGS=", "CGO_ENABLED=0")
}

func invokeOperator(t *testing.T, binary string, want int, arguments ...string) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, binary, arguments...)
	command.Env = operatorEnvironment()
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	status := 0
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatal(err)
		}
		status = exit.ExitCode()
	}
	if status != want {
		t.Fatalf("operator %q status=%d want=%d\nstdout=%s\nstderr=%s", arguments, status, want, stdout.String(), stderr.String())
	}
	return stdout.Bytes()
}

func TestRetainedV1ReplayIsRefusedBeforeResolvingInstallation(t *testing.T) {
	path := publishRetainedClusterArtifact(t, "gomad3.wasm-simulation-retained/v1")
	resolve := func() (installation.Description, error) {
		t.Fatal("v1 retained replay resolved an installation")
		return installation.Description{}, nil
	}
	_, handled, err := replayRetainedCluster(t.Context(), runner.ReplaySpec{ArtifactPath: path}, resolve)
	var preflight *runner.ReplayPreflightError
	if !handled || !errors.Is(err, simulation.ErrRetainedSchemaUnsupported) || errors.As(err, &preflight) {
		t.Fatalf("v1 retained replay: handled %v, err %v", handled, err)
	}
	// Unsupported replay options keep their preflight precedence (exit 2).
	for _, spec := range []runner.ReplaySpec{{ArtifactPath: path, VerifyOnly: true}, {ArtifactPath: path, ObservedDir: t.TempDir()}, {ArtifactPath: path, ToolchainRoot: "/missing/native"}} {
		if _, handled, err := replayRetainedCluster(t.Context(), spec, resolve); !handled || !errors.As(err, &preflight) {
			t.Fatalf("v1 retained replay %+v: handled %v, err %v", spec, handled, err)
		}
	}
}

// publishRetainedClusterArtifact publishes a minimal Artifact whose evidence
// names schema. Its payload is never a valid retained bundle.
func publishRetainedClusterArtifact(t *testing.T, schema string) string {
	t.Helper()
	module := []byte("\x00asm\x01\x00\x00\x00")
	modulePath := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(modulePath, module, 0o700); err != nil {
		t.Fatal(err)
	}
	evidence := []byte(`{"schema":"` + schema + `"}`)
	provenance := []byte(`{}`)
	metadata := &record.BackendMetadata{Name: backend.Name, ReplayMode: record.ReplayExact, Provenance: record.BackendPayload{Schema: "provenance", File: "backend/provenance.json", SHA256: record.HashBytes(provenance), Bytes: record.Uint64String(len(provenance))}, Evidence: &record.BackendPayload{Schema: schema, File: "backend/simulation.json", SHA256: record.HashBytes(evidence), Bytes: record.Uint64String(len(evidence))}}
	world, worldPayloads := record.NoneWorld()
	exitCode := record.Uint64String(1)
	manifest := record.ExecutionRecord{
		SchemaVersion: record.SchemaVersion, ArtifactKind: record.ArtifactTargetFailure, CreatedAt: "2026-10-12T12:00:00Z", CampaignID: "retained", Seed: 7, ReplayMode: record.ReplayExact,
		Runner:    record.Runner{RecordContract: record.RecordContract, RunnerBuild: "test", HostOS: "linux", HostArch: "amd64"},
		Toolchain: record.Toolchain{GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "wasip1", TargetGOARCH: "wasm"},
		Target: record.Target{Kind: "go-run", Source: ".", SHA256: record.HashBytes(module), Size: record.Uint64String(len(module)), Argv: []string{"gomad3-target"}, BuildTags: []string{},
			Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: record.BuildInfo{GoVersion: "go1.26.4", Path: "example.com/target"}, Backend: metadata},
		IOProfile:   record.BackendIOProfile(*metadata),
		Environment: []record.Environment{{Name: "GOMAD3_IO_PROFILE", Value: backend.Name}, {Name: "GOMADSEED", Value: "7"}, {Name: "TZ", Value: "UTC"}},
		Limits:      record.Limits{ExecutionTimeoutNanos: 1, OverallTimeoutNanos: 2, OutputBytes: 64, WorldTransitionBytes: 64},
		World:       world,
		Outcome:     record.Outcome{Domain: "target", Reason: "nonzero_exit", Termination: "exit", ExitCode: &exitCode},
		Streams:     record.Streams{Stdout: record.Stream{FullSHA256: record.HashBytes(nil)}, Stderr: record.Stream{FullSHA256: record.HashBytes(nil)}},
		Host:        record.Host{StartedAt: "2026-10-12T12:00:00Z", FinishedAt: "2026-10-12T12:00:01Z", ElapsedNanos: 1},
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, artifact.ArtifactInput{Manifest: manifest, TargetPath: modulePath, World: worldPayloads, BackendPayloads: []target.BackendPayload{{Reference: metadata.Provenance, Data: provenance}, {Reference: *metadata.Evidence, Data: evidence}}})
	if err != nil {
		t.Fatal(err)
	}
	return published.Path
}
