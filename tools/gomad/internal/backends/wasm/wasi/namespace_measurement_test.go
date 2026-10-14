package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

const namespaceMeasurementBase = "10d2a1ec1578a1789c1a010b0771f15a2e8f3a75"

type namespaceMeasurementSample struct {
	Role                                string
	Seed                                uint64
	Repetition                          int
	AuthorizedNanos                     uint64
	EvidenceSHA256, EnvironmentIdentity string
	Calls, TimeAdvances                 uint64
	GoFilesystemCalls                   map[string]uint64
	CounterPresent                      bool
}

func namespaceMeasurementCompare(before, after []namespaceMeasurementSample) error {
	seeds := []uint64{0, 1000003, 2000006, 3000009}
	if len(before) != len(seeds)*5 || len(after) != len(before) {
		return fmt.Errorf("four matched seeds with five fresh repetitions required")
	}
	for seedIndex, seed := range seeds {
		var baseline, candidate []uint64
		for repetition := range 5 {
			index := seedIndex*5 + repetition
			b, a := before[index], after[index]
			if b.Role != "baseline" || a.Role != "candidate" || b.Seed != seed || a.Seed != seed || b.Repetition != repetition || a.Repetition != repetition || b.AuthorizedNanos == 0 || a.AuthorizedNanos == 0 || !a.CounterPresent || len(a.GoFilesystemCalls) != 0 {
				return fmt.Errorf("seed %d repetition %d lacks matched timings or empty actual candidate counter", seed, repetition)
			}
			if !warmPreparationHex(b.EvidenceSHA256, 64) || a.EvidenceSHA256 != b.EvidenceSHA256 || b.EnvironmentIdentity == "" || a.EnvironmentIdentity != b.EnvironmentIdentity || a.Calls != b.Calls || a.TimeAdvances != b.TimeAdvances || b.EvidenceSHA256 != before[seedIndex*5].EvidenceSHA256 {
				return fmt.Errorf("seed %d repetition %d changed choice/WASI/network/output/terminal or model counts", seed, repetition)
			}
			baseline, candidate = append(baseline, b.AuthorizedNanos), append(candidate, a.AuthorizedNanos)
		}
		b, a := variantTiming(baseline).Median, variantTiming(candidate).Median
		// Division avoids overflowing a wall-time counter when comparing 4*a <= 3*b.
		if a > b/4*3+b%4*3/4 {
			return fmt.Errorf("seed %d authorized median %d exceeds 0.75 baseline %d", seed, a, b)
		}
	}
	return nil
}

func namespaceMeasurementCounter(data []byte, candidate bool) (map[string]uint64, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	raw, present := fields["GoFilesystemCalls"]
	if !candidate && !present {
		return nil, nil
	}
	if candidate && (!present || bytes.Equal(bytes.TrimSpace(raw), []byte("null"))) {
		return nil, fmt.Errorf("actual Result filesystem counter absent")
	}
	var counts map[string]uint64
	if err := json.Unmarshal(raw, &counts); err != nil {
		return nil, err
	}
	if candidate && len(counts) != 0 {
		return nil, fmt.Errorf("candidate Go namespace served operations: %v", counts)
	}
	return counts, nil
}

type namespaceMeasurementCommon struct {
	Schema, CurrentSourceRoot, CurrentSourceCommit, ModuleReceipt string
	Request                                                       Request
	ProducerInputs                                                map[string]string
}

type namespaceMeasurementReceipt struct {
	Schema, Role, SourceRoot, SourceCommit, ModelSHA256                                                  string
	HostBootID, HostCPUInfo, CommonPath, HelperReceipt, ArtifactPath                                     string
	ResultPath, RequestPath, FinishPath, TranscriptPath, ChoicePath, StdoutPath, StderrPath, NetworkPath string
	Seed                                                                                                 uint64
	Repetition                                                                                           int
	Replay                                                                                               bool
	WallNanos                                                                                            uint64
	RunError, Failure                                                                                    string
	CollectionValid                                                                                      bool
	Inputs                                                                                               map[string]string
}

func namespaceMeasurementReadJSON(t *testing.T, path string, value any) {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(mustTerminalExitRead(t, path)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		t.Fatal(path, err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		t.Fatal("trailing measurement receipt", path, err)
	}
}

func namespaceMeasurementRequest(t *testing.T, common namespaceMeasurementCommon) Request {
	t.Helper()
	data, err := json.Marshal(common.Request)
	if err != nil {
		t.Fatal(err)
	}
	var request Request
	if err := json.Unmarshal(data, &request); err != nil {
		t.Fatal(err)
	}
	return request
}

func namespaceMeasurementGit(t *testing.T, evidence, repo, name string, args ...string) []byte {
	t.Helper()
	return temporalCommand(t, evidence, name, repo, nil, append([]string{"git"}, args...)).Stdout.RawBytes
}

func namespaceMeasurementSource(t *testing.T, evidence, repo, role string) (string, map[string]string) {
	t.Helper()
	commit := strings.TrimSpace(string(namespaceMeasurementGit(t, evidence, repo, "source-commit", "rev-parse", "HEAD")))
	if !warmPreparationHex(commit, 40) {
		t.Fatal("source commit missing")
	}
	status := namespaceMeasurementGit(t, evidence, repo, "source-status", "status", "--porcelain", "--", "tools/gomad")
	if len(status) != 0 {
		t.Fatal("freeze and commit collector source before measurement")
	}
	if role == "baseline" {
		namespaceMeasurementGit(t, evidence, repo, "baseline-ancestor", "merge-base", "--is-ancestor", namespaceMeasurementBase, commit)
		changed := namespaceMeasurementGit(t, evidence, repo, "baseline-production", "diff", "--name-only", namespaceMeasurementBase, commit, "--", ".")
		for _, path := range strings.Fields(string(changed)) {
			if path != "tools/gomad/internal/backends/wasm/wasi/namespace_measurement_test.go" && path != "tools/gomad/internal/backends/wasm/wasi/namespace_measurement_controls_test.go" {
				t.Fatal("frozen fn174 production/original driver changed", path)
			}
		}
	}
	inputs := map[string]string{}
	tracked := namespaceMeasurementGit(t, evidence, repo, "source-inventory", "ls-files", "-z", "tools/gomad")
	for _, name := range bytes.Split(tracked, []byte{0}) {
		if len(name) > 0 {
			path := filepath.Join(repo, string(name))
			inputs[path] = fileHash(t, path)
		}
	}
	return commit, inputs
}

func namespaceMeasurementResultCheck(result Result, raw []byte, request Request, candidate bool, runError string) error {
	if runError != "" || result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 || !result.Reaped || result.Cancelled || result.WatchdogTimeout || result.Started == nil || result.Started.Engine != request.Engine || result.Started.ModuleSHA256 != request.ModuleSHA256 || result.AuthorizationToTerminalNanos == 0 || result.Telemetry == nil || !result.Telemetry.CacheHit || result.Telemetry.CompileNanos != 0 {
		return fmt.Errorf("original Run lacks successful warm source-matched authorization/terminal evidence: %s", runError)
	}
	if !bytes.Contains(result.Stdout, []byte("=== RUN   TestActivityTestSuite")) || !bytes.Contains(result.Stdout, []byte("--- PASS: TestActivityTestSuite")) {
		return fmt.Errorf("unchanged activity selection did not pass")
	}
	if _, err := namespaceMeasurementCounter(raw, candidate); err != nil {
		return err
	}
	if sha256.Sum256(result.Transcript) != result.TranscriptSHA256 {
		return fmt.Errorf("raw WASI bytes differ from actual Run digest")
	}
	if _, err := DecodeTranscript(result.Transcript); err != nil {
		return err
	}
	return nil
}

func namespaceMeasurementLoad(t *testing.T, path string, common namespaceMeasurementCommon, expected Request) (namespaceMeasurementReceipt, Result, namespaceMeasurementSample) {
	t.Helper()
	var receipt namespaceMeasurementReceipt
	namespaceMeasurementReadJSON(t, path, &receipt)
	temporalVerifyInputs(t, receipt.Inputs)
	if receipt.Schema != "gomad3.wasm-namespace-measurement-sample/v1" || receipt.Role != "baseline" && receipt.Role != "candidate" || !receipt.CollectionValid || receipt.RunError != "" || receipt.Failure != "" || receipt.ModelSHA256 == "" || receipt.HostBootID == "" || !diagnosticPerformanceHost("linux", "arm64", receipt.HostCPUInfo) {
		t.Fatal("failed or incomplete original namespace measurement collection")
	}
	for _, bound := range []string{receipt.CommonPath, receipt.ResultPath, receipt.RequestPath, receipt.FinishPath, receipt.TranscriptPath, receipt.ChoicePath, receipt.StdoutPath, receipt.StderrPath, receipt.NetworkPath, receipt.HelperReceipt, receipt.ArtifactPath} {
		if !filepath.IsAbs(bound) || !warmPreparationHex(receipt.Inputs[bound], 64) {
			t.Fatal("missing raw/source/build binding", bound)
		}
	}
	var retainedCommon namespaceMeasurementCommon
	namespaceMeasurementReadJSON(t, receipt.CommonPath, &retainedCommon)
	if !reflect.DeepEqual(common, retainedCommon) {
		t.Fatal("collection substituted common module/config/producer")
	}
	if err := warmPreparationProducerCheck(warmPreparationSeries{Inputs: receipt.Inputs, ModuleReceipt: common.ModuleReceipt}, common.ModuleReceipt, common.ProducerInputs); err != nil {
		t.Fatal(err)
	}
	var request Request
	namespaceMeasurementReadJSON(t, receipt.RequestPath, &request)
	if !reflect.DeepEqual(request, expected) {
		t.Fatal("actual source request differs from paired specification")
	}
	var result Result
	raw := mustTerminalExitRead(t, receipt.ResultPath)
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	finish := mustTerminalExitRead(t, receipt.FinishPath)
	if len(finish) != len(result.runtimeTerminal) {
		t.Fatal("private finish sidecar absent")
	}
	copy(result.runtimeTerminal[:], finish)
	for _, stream := range []struct {
		path string
		data []byte
	}{{receipt.TranscriptPath, result.Transcript}, {receipt.ChoicePath, result.ChoiceTrace.Bytes}, {receipt.StdoutPath, result.Stdout}, {receipt.StderrPath, result.Stderr}, {receipt.NetworkPath, result.NetworkTranscript}} {
		if !bytes.Equal(mustTerminalExitRead(t, stream.path), stream.data) {
			t.Fatal("raw original Run stream changed", stream.path)
		}
	}
	if err := namespaceMeasurementResultCheck(result, raw, request, receipt.Role == "candidate", receipt.RunError); err != nil {
		t.Fatal(err)
	}
	evidence, err := warmPreparationBehavior(result, request.Config, false)
	if err != nil {
		t.Fatal(err)
	}
	counts, err := namespaceMeasurementCounter(raw, receipt.Role == "candidate")
	if err != nil {
		t.Fatal(err)
	}
	sample := namespaceMeasurementSample{Role: receipt.Role, Seed: receipt.Seed, Repetition: receipt.Repetition, AuthorizedNanos: result.AuthorizationToTerminalNanos, EvidenceSHA256: evidence, EnvironmentIdentity: result.EnvironmentIdentity, Calls: result.Calls, TimeAdvances: result.TimeAdvances, GoFilesystemCalls: counts, CounterPresent: receipt.Role == "candidate"}
	return receipt, result, sample
}

// This file is also compiled against the unmodified frozen fn174 production tree.
// Its baseline driver uses that tree's Run and PinnedEngine, with no remoteModel override.
func TestNamespaceMeasurementSample(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_NAMESPACE_SAMPLE_ROOT")
	if root == "" {
		t.Skip("explicit source-matched sample collection only")
	}
	commonPath, helperReceipt, cache := os.Getenv("GOMAD_WASM_NAMESPACE_COMMON"), os.Getenv("GOMAD_WASM_NAMESPACE_HELPER_RECEIPT"), os.Getenv("GOMAD_WASM_NAMESPACE_CACHE")
	for _, path := range []string{root, commonPath, helperReceipt, cache} {
		if !filepath.IsAbs(path) {
			t.Fatal("absolute sample/config/build/cache bindings required")
		}
	}
	role := os.Getenv("GOMAD_WASM_NAMESPACE_ROLE")
	if role != "baseline" && role != "candidate" {
		t.Fatal("explicit baseline/candidate source role required")
	}
	seed, err := strconv.ParseUint(os.Getenv("GOMAD_WASM_NAMESPACE_SEED"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	repetition, err := strconv.Atoi(os.Getenv("GOMAD_WASM_NAMESPACE_REPETITION"))
	if err != nil || repetition < -1 || repetition > 4 {
		t.Fatal("warmup (-1) or measured repetition (0..4) required")
	}
	cpu, host := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	boot, err := hostfs.ReadBounded("/proc/sys/kernel/random/boot_id", 128)
	if !host || err != nil || strings.TrimSpace(string(boot)) == "" {
		t.Fatal("real Linux/arm64 host required", err)
	}
	evidence, err := diagnosticRunRoot(root, "namespace-sample-")
	if err != nil {
		t.Fatal(err)
	}
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	commit, inputs := namespaceMeasurementSource(t, evidence, repo, role)
	var common namespaceMeasurementCommon
	namespaceMeasurementReadJSON(t, commonPath, &common)
	if common.Schema != "gomad3.wasm-namespace-measurement-common/v1" || common.Request.Config.Profile != CooperativeProfile || common.Request.Config.Storage != nil || common.Request.Config.NetworkV2 != nil || common.Request.Runtime == nil || common.Request.Runtime.Choice == nil || common.Request.DigestTranscript || common.Request.Broker != nil {
		t.Fatal("original standalone cooperative full-evidence request required")
	}
	request := namespaceMeasurementRequest(t, common)
	request.Engine, request.CompileCache, request.Runtime.Seed = PinnedEngine(), cache, seed
	var admission terminalExitHelperAdmission
	namespaceMeasurementReadJSON(t, helperReceipt, &admission)
	request.HelperPath, request.HelperSHA256 = admission.Helper, admission.HelperSHA256
	crate := filepath.Join(repo, "tools/gomad/internal/backends/wasm/wasmhost")
	maps.Copy(inputs, terminalExitHelperInputs(t, helperReceipt, request, crate))
	if role == "candidate" {
		if repo != common.CurrentSourceRoot || commit != common.CurrentSourceCommit {
			t.Fatal("candidate source changed from common producer validation")
		}
		if !maps.Equal(terminalExitModuleInputs(t, common.ModuleReceipt, request), common.ProducerInputs) {
			t.Fatal("current producer closure changed")
		}
	}
	temporalVerifyInputs(t, common.ProducerInputs)
	maps.Copy(inputs, common.ProducerInputs)
	inputs[commonPath] = fileHash(t, commonPath)
	if replayPath := os.Getenv("GOMAD_WASM_NAMESPACE_REPLAY_TAPE"); replayPath != "" {
		if !filepath.IsAbs(replayPath) {
			t.Fatal("replay tape must be absolute")
		}
		var tape choice.ReplayPlan
		namespaceMeasurementReadJSON(t, replayPath, &tape)
		request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModeReplay, &tape
		inputs[replayPath] = fileHash(t, replayPath)
	}
	artifact, _, _ := warmPreparationArtifact(t, request)
	inputs[artifact] = fileHash(t, artifact)
	receipt := namespaceMeasurementReceipt{Schema: "gomad3.wasm-namespace-measurement-sample/v1", Role: role, SourceRoot: repo, SourceCommit: commit, ModelSHA256: ImplementationSHA256(), HostBootID: strings.TrimSpace(string(boot)), HostCPUInfo: cpu, CommonPath: commonPath, HelperReceipt: helperReceipt, ArtifactPath: artifact, Seed: seed, Repetition: repetition, Replay: request.Runtime.Choice.Mode == choice.ModeReplay, Inputs: inputs}
	receipt.RequestPath = filepath.Join(evidence, "request.json")
	temporalJSON(t, receipt.RequestPath, request)
	inputs[receipt.RequestPath] = fileHash(t, receipt.RequestPath)
	started := time.Now()
	result, runErr := matchedPrepareAdmitted("linux", "/proc/meminfo", "1", func() (Result, error) { return Run(t.Context(), request) })
	receipt.WallNanos = uint64(time.Since(started).Nanoseconds())
	if runErr != nil {
		receipt.RunError = runErr.Error()
	}
	receipt.ResultPath, receipt.FinishPath = filepath.Join(evidence, "result.json"), filepath.Join(evidence, "finish.bin")
	temporalJSON(t, receipt.ResultPath, result)
	temporalWrite(t, receipt.FinishPath, result.runtimeTerminal[:])
	receipt.TranscriptPath, receipt.ChoicePath = filepath.Join(evidence, "wasi.bin"), filepath.Join(evidence, "choice.bin")
	receipt.StdoutPath, receipt.StderrPath, receipt.NetworkPath = filepath.Join(evidence, "stdout.bin"), filepath.Join(evidence, "stderr.bin"), filepath.Join(evidence, "network.bin")
	for _, stream := range []struct {
		path string
		data []byte
	}{{receipt.TranscriptPath, result.Transcript}, {receipt.ChoicePath, result.ChoiceTrace.Bytes}, {receipt.StdoutPath, result.Stdout}, {receipt.StderrPath, result.Stderr}, {receipt.NetworkPath, result.NetworkTranscript}} {
		temporalWrite(t, stream.path, stream.data)
	}
	for _, path := range []string{receipt.ResultPath, receipt.FinishPath, receipt.TranscriptPath, receipt.ChoicePath, receipt.StdoutPath, receipt.StderrPath, receipt.NetworkPath} {
		inputs[path] = fileHash(t, path)
	}
	if err := namespaceMeasurementResultCheck(result, mustTerminalExitRead(t, receipt.ResultPath), request, role == "candidate", receipt.RunError); err != nil {
		receipt.Failure = err.Error()
	}
	afterBoot, err := hostfs.ReadBounded("/proc/sys/kernel/random/boot_id", 128)
	afterCPU, afterHost := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	if err != nil || !bytes.Equal(boot, afterBoot) || !afterHost || cpu != afterCPU {
		receipt.Failure = "host changed during collection"
	}
	if afterCommit, _ := namespaceMeasurementSource(t, evidence, repo, role); afterCommit != commit {
		receipt.Failure = "source changed during collection"
	}
	temporalVerifyInputs(t, inputs)
	terminalExitHelperInputs(t, helperReceipt, request, crate)
	if role == "candidate" {
		terminalExitModuleInputs(t, common.ModuleReceipt, request)
	}
	receipt.CollectionValid = receipt.Failure == "" && receipt.RunError == ""
	receiptPath := filepath.Join(evidence, "sample.json")
	temporalJSON(t, receiptPath, receipt)
	output := os.Getenv("GOMAD_WASM_NAMESPACE_SAMPLE_INDEX")
	if !filepath.IsAbs(output) {
		t.Fatal("absolute controller-owned sample index required")
	}
	temporalJSON(t, output, map[string]string{"receipt": receiptPath, "sha256": fileHash(t, receiptPath)})
	if !receipt.CollectionValid {
		t.Fatalf("namespace collection refused: %s %s; retained %s", receipt.RunError, receipt.Failure, evidence)
	}
}

func TestNamespaceActivityMeasurement(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_NAMESPACE_MEASUREMENT_ROOT")
	if root == "" {
		t.Skip("explicit real Linux/arm64 paired activity measurement only")
	}
	paths := map[string]string{
		"root": root, "baseline_source": os.Getenv("GOMAD_WASM_NAMESPACE_BASELINE_SOURCE"),
		"baseline_helper_receipt":  os.Getenv("GOMAD_WASM_NAMESPACE_BASELINE_HELPER_RECEIPT"),
		"candidate_helper_receipt": os.Getenv("GOMAD_WASM_NAMESPACE_HELPER_RECEIPT"),
		"baseline_cache":           os.Getenv("GOMAD_WASM_NAMESPACE_BASELINE_CACHE"),
		"candidate_cache":          os.Getenv("GOMAD_WASM_NAMESPACE_CACHE"),
		"module":                   os.Getenv("GOMAD_WASM_NAMESPACE_MODULE"), "module_receipt": os.Getenv("GOMAD_WASM_NAMESPACE_MODULE_RECEIPT"),
		"stock_go": os.Getenv("GOMAD_WASM_NAMESPACE_STOCK_GO"),
	}
	for name, path := range paths {
		if !filepath.IsAbs(path) {
			t.Fatal("explicit absolute paired collection input required", name)
		}
	}
	if _, host := diagnosticPerformanceHostEvidence("/proc/cpuinfo"); !host {
		t.Fatal("real Linux/arm64 performance host required")
	}
	if err := matchedBaselineCache(paths["baseline_cache"], paths["candidate_cache"]); err != nil {
		t.Fatal(err)
	}
	evidence, err := diagnosticRunRoot(root, "namespace-paired-")
	if err != nil {
		t.Fatal(err)
	}
	version := temporalCommand(t, evidence, "stock-go-version", evidence, append(os.Environ(), "GOROOT=", "GOTOOLCHAIN=local"), []string{paths["stock_go"], "version"})
	if !strings.HasPrefix(string(version.Stdout.RawBytes), "go version go1.27.1 linux/arm64") {
		t.Fatal("source drivers require stock Go 1.27.1 on the real Linux/arm64 host")
	}
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", evidence)
	t.Setenv("GOMAD3_STOCK_GO", paths["stock_go"])
	var candidateAdmission terminalExitHelperAdmission
	namespaceMeasurementReadJSON(t, paths["candidate_helper_receipt"], &candidateAdmission)
	t.Setenv("GOMAD_WASM_TEST_HELPER", candidateAdmission.Helper)
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	if repo == paths["baseline_source"] {
		t.Fatal("separate immutable baseline source and driver required")
	}
	config := prototypeTemporalConfig(t, repo, "/workspace/tests", "-test.run=^TestActivityTestSuite$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	request := prototypeRequest(t, paths["module"], config, 0, 30*time.Minute)
	request.DigestTranscript, request.CompileCache = false, paths["candidate_cache"]
	commit, sourceInputs := namespaceMeasurementSource(t, evidence, repo, "candidate")
	namespaceMeasurementSource(t, evidence, paths["baseline_source"], "baseline")
	common := namespaceMeasurementCommon{Schema: "gomad3.wasm-namespace-measurement-common/v1", CurrentSourceRoot: repo, CurrentSourceCommit: commit, ModuleReceipt: paths["module_receipt"], Request: request, ProducerInputs: terminalExitModuleInputs(t, paths["module_receipt"], request)}
	commonPath := filepath.Join(evidence, "common.json")
	temporalJSON(t, commonPath, common)
	inputs := maps.Clone(sourceInputs)
	maps.Copy(inputs, common.ProducerInputs)
	inputs[commonPath], inputs[paths["stock_go"]] = fileHash(t, commonPath), fileHash(t, paths["stock_go"])
	var baseline, candidate []namespaceMeasurementSample
	var baselineResults []Result
	var rawReceipts []string
	var prior [2]namespaceMeasurementReceipt
	var order []map[string]any
	collect := func(role string, seed uint64, repetition int, replayPath string) (namespaceMeasurementSample, Result) {
		index := 0
		owner := repo
		if role == "candidate" {
			index = 1
		} else {
			owner = paths["baseline_source"]
		}
		tag := fmt.Sprintf("seed%d-%s-rep%d", seed, role, repetition)
		if replayPath != "" {
			tag += "-replay"
		}
		sampleRoot := filepath.Join(evidence, tag)
		if err := os.Mkdir(sampleRoot, 0700); err != nil {
			t.Fatal(err)
		}
		indexPath := filepath.Join(sampleRoot, "index.json")
		environment := append(os.Environ(),
			"GOROOT=", "GOTOOLCHAIN=local", "GOFLAGS=", "GOMAXPROCS=2", "GOENV=off", "GOWORK=off", "GOOS=", "GOARCH=", "GOEXPERIMENT=", "GOWASM=",
			"GOMAD_WASM_NAMESPACE_MEASUREMENT_ROOT=", "GOMAD_WASM_NAMESPACE_SAMPLE_ROOT="+sampleRoot,
			"GOMAD_WASM_NAMESPACE_COMMON="+commonPath, "GOMAD_WASM_NAMESPACE_ROLE="+role,
			"GOMAD_WASM_NAMESPACE_HELPER_RECEIPT="+paths[role+"_helper_receipt"],
			"GOMAD_WASM_NAMESPACE_CACHE="+paths[role+"_cache"],
			"GOMAD_WASM_NAMESPACE_SEED="+strconv.FormatUint(seed, 10),
			"GOMAD_WASM_NAMESPACE_REPETITION="+strconv.Itoa(repetition),
			"GOMAD_WASM_NAMESPACE_REPLAY_TAPE="+replayPath, "GOMAD_WASM_NAMESPACE_SAMPLE_INDEX="+indexPath)
		command := []string{paths["stock_go"], "test", "-p", "1", "-tags=test_dep", "-count=1", "-timeout=35m", "-run=^TestNamespaceMeasurementSample$", "./internal/backends/wasm/wasi"}
		temporalCommand(t, evidence, tag, filepath.Join(owner, "tools/gomad"), environment, command)
		var sampleIndex map[string]string
		namespaceMeasurementReadJSON(t, indexPath, &sampleIndex)
		path := sampleIndex["receipt"]
		if !filepath.IsAbs(path) || sampleIndex["sha256"] != fileHash(t, path) {
			t.Fatal("source collector index changed")
		}
		var admission terminalExitHelperAdmission
		namespaceMeasurementReadJSON(t, paths[role+"_helper_receipt"], &admission)
		expected := namespaceMeasurementRequest(t, common)
		expected.HelperPath, expected.HelperSHA256, expected.Engine, expected.CompileCache = admission.Helper, admission.HelperSHA256, PinnedEngine(), paths[role+"_cache"]
		expected.Runtime.Seed = seed
		if replayPath != "" {
			var tape choice.ReplayPlan
			namespaceMeasurementReadJSON(t, replayPath, &tape)
			expected.Runtime.Choice.Mode, expected.Runtime.Choice.Tape = choice.ModeReplay, &tape
		}
		bound, result, sample := namespaceMeasurementLoad(t, path, common, expected)
		if bound.Role != role || bound.SourceRoot != owner || bound.Seed != seed || bound.Repetition != repetition || bound.Replay != (replayPath != "") {
			t.Fatal("source collector substituted role/source/seed/repetition/replay")
		}
		if index == 1 && bound.ModelSHA256 != ImplementationSHA256() {
			t.Fatal("current Run model binding changed")
		}
		if prior[index].SourceCommit != "" && (bound.SourceCommit != prior[index].SourceCommit || bound.ModelSHA256 != prior[index].ModelSHA256 || bound.HostBootID != prior[index].HostBootID || bound.HostCPUInfo != prior[index].HostCPUInfo) {
			t.Fatal("source/model/host changed within series")
		}
		if other := prior[1-index]; other.SourceCommit != "" && (bound.HostBootID != other.HostBootID || bound.HostCPUInfo != other.HostCPUInfo) {
			t.Fatal("baseline/candidate did not share real host")
		}
		terminalExitHelperInputs(t, bound.HelperReceipt, expected, filepath.Join(owner, "tools/gomad/internal/backends/wasm/wasmhost"))
		prior[index] = bound
		inputs[path], inputs[indexPath] = fileHash(t, path), fileHash(t, indexPath)
		maps.Copy(inputs, bound.Inputs)
		rawReceipts = append(rawReceipts, path)
		order = append(order, map[string]any{"role": role, "seed": seed, "repetition": repetition, "replay": replayPath != "", "receipt": path, "authorized_nanos": sample.AuthorizedNanos, "wall_nanos": bound.WallNanos})
		temporalJSON(t, filepath.Join(evidence, "collection-order.json"), order)
		return sample, result
	}
	var medians []map[string]any
	for seedIndex, seed := range []uint64{0, 1000003, 2000006, 3000009} {
		_, warmBefore := collect("baseline", seed, -1, "")
		_, warmAfter := collect("candidate", seed, -1, "")
		if difference := sameEvidence(warmBefore, warmAfter); difference != "" {
			t.Fatal("source-matched warmup evidence changed", difference)
		}
		for repetition := range 5 {
			var samples [2]namespaceMeasurementSample
			var results [2]Result
			for turn := range 2 {
				index := turn
				if (seedIndex*5+repetition)%2 != 0 {
					index = 1 - turn
				}
				role := "baseline"
				if index == 1 {
					role = "candidate"
				}
				samples[index], results[index] = collect(role, seed, repetition, "")
			}
			if difference := sameEvidence(results[0], results[1]); difference != "" {
				t.Fatal("same-module paired evidence changed", seed, repetition, difference)
			}
			if difference := sameEvidence(warmBefore, results[0]); difference != "" {
				t.Fatal("baseline warmup/sample changed", difference)
			}
			if difference := sameEvidence(warmAfter, results[1]); difference != "" {
				t.Fatal("candidate warmup/sample changed", difference)
			}
			baseline, candidate = append(baseline, samples[0]), append(candidate, samples[1])
			baselineResults = append(baselineResults, results[0])
		}
		first := baselineResults[seedIndex*5]
		tape, err := choice.ProjectReplayPlan(first.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
		if err != nil {
			t.Fatal(err)
		}
		tapePath := filepath.Join(evidence, fmt.Sprintf("seed%d-replay-plan.json", seed))
		temporalJSON(t, tapePath, tape)
		inputs[tapePath] = fileHash(t, tapePath)
		for _, role := range []string{"baseline", "candidate"} {
			_, replayed := collect(role, seed, 0, tapePath)
			if difference := sameEvidence(first, replayed); difference != "" || replayed.EnvironmentIdentity != first.EnvironmentIdentity || replayed.Calls != first.Calls || replayed.TimeAdvances != first.TimeAdvances {
				t.Fatal("fresh exact baseline-tape replay changed", seed, role, difference)
			}
		}
		var b, a []uint64
		for i := seedIndex * 5; i < (seedIndex+1)*5; i++ {
			b, a = append(b, baseline[i].AuthorizedNanos), append(a, candidate[i].AuthorizedNanos)
		}
		medians = append(medians, map[string]any{"seed": seed, "baseline_authorized_nanos": variantTiming(b), "candidate_authorized_nanos": variantTiming(a)})
	}
	failure := ""
	if err := namespaceMeasurementCompare(baseline, candidate); err != nil {
		failure = err.Error()
	}
	if prior[0].ModelSHA256 == prior[1].ModelSHA256 {
		failure = "namespace image source must rotate model identity; historical model was substituted"
	}
	temporalVerifyInputs(t, inputs)
	terminalExitModuleInputs(t, common.ModuleReceipt, request)
	for _, source := range []struct{ root, role, commit string }{{repo, "candidate", commit}, {paths["baseline_source"], "baseline", prior[0].SourceCommit}} {
		frozen, _ := namespaceMeasurementSource(t, evidence, source.root, source.role)
		if frozen != source.commit {
			failure = "source changed after paired collection"
		}
	}
	temporalJSON(t, filepath.Join(evidence, "measurement.json"), map[string]any{"schema": "gomad3.wasm-namespace-activity-measurement/v1", "accepted": failure == "", "failure": failure, "baseline_source": prior[0], "candidate_source": prior[1], "baseline": baseline, "candidate": candidate, "per_seed_nanos": medians, "raw_receipts": rawReceipts, "collection_order": order, "inputs": inputs, "boundary": "five fresh helpers per matched seed on existing warm caches; actual Run authorization through validated terminal; full raw evidence and per-seed full-tape replay; alternating paired source drivers; no cold-compilation measurement", "baseline_base": namespaceMeasurementBase})
	if failure != "" {
		t.Fatalf("namespace activity requirement not met: %s; retained %s", failure, evidence)
	}
}
