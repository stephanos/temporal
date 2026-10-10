package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad_wasm/toolchain"
)

type runtimeQualification struct {
	SeedIndex, Seed, Instances, ExactReplays                               uint64
	Module, Helper, Runtime, Model                                         string
	SourceSHA256, RuntimeImplementationSHA256, CompilerSHA256, GoVersion   string
	HostGOOS, HostGOARCH, Collector, Profile                               string
	Engine                                                                 EngineIdentity
	Clock                                                                  ClockPolicy
	Limits                                                                 Limits
	ChoiceSHA256, DiagnosticSHA256, StdoutSHA256, StderrSHA256, WASISHA256 string
	TerminalSHA256                                                         string
	ExitCode                                                               uint32
	Summary                                                                choice.Summary
	LastDiagnostic                                                         choice.DiagnosticRecord
}

func TestCooperativeRuntimeQualification(t *testing.T) {
	indexValue := os.Getenv("GOMAD_WASM_QUALIFICATION_INDEX")
	if indexValue == "" {
		t.Skip("run qualify-runtime for the mandatory 32 seeds x 100 fresh-instance gate")
	}
	index, err := strconv.ParseUint(indexValue, 10, 64)
	if err != nil || index >= 32 {
		t.Fatal("qualification index must be 0..31")
	}
	root := os.Getenv("GOMAD_WASM_QUALIFICATION_ROOT")
	if !filepath.IsAbs(root) {
		t.Fatal("qualification requires an absolute retained evidence root")
	}
	request := cooperativeRequest(t, "../testdata/runtime_probes")
	sourceSHA256 := qualificationSourceSHA256(t)
	request.Runtime.Seed = index * 1000003
	request.Config.Environment = append(request.Config.Environment, "EXTRA_ENTROPY=no_")
	var baseline Result
	for instance := range 100 {
		observed, err := Run(t.Context(), request)
		if err != nil || observed.Termination != "exit" || observed.ExitCode == nil || (*observed.ExitCode != 0 && *observed.ExitCode != 17) || !observed.Reaped || observed.Cancelled || observed.WatchdogTimeout {
			t.Fatalf("seed %d fresh instance %d: %s %s err=%v stderr=%s", request.Runtime.Seed, instance, observed.Termination, observed.Message, err, observed.Stderr)
		}
		for _, witness := range []string{"entropy ", "map ", "runq ", "timer callbacks ", "timer parking", "note negative park wake", "collector automatic finalizer cleanup 1 1", "canary "} {
			if !bytes.Contains(observed.Stdout, []byte(witness)) {
				t.Fatalf("seed %d instance %d missing %q", request.Runtime.Seed, instance, witness)
			}
		}
		if observed.ChoiceTrace.Summary.Runnable == 0 || observed.ChoiceTrace.Summary.SelectPoll == 0 || observed.ChoiceTrace.Summary.SelectResult == 0 || observed.ChoiceTrace.Summary.Terminal != choice.TerminalComplete || len(observed.DiagnosticTrace.Records) != len(observed.ChoiceTrace.Records) {
			t.Fatalf("seed %d instance %d incomplete hook witnesses: %+v", request.Runtime.Seed, instance, observed.ChoiceTrace.Summary)
		}
		last := observed.DiagnosticTrace.Records[len(observed.DiagnosticTrace.Records)-1]
		if last.Allocations == 0 || last.GCCycle == 0 || last.RunqDraws == 0 || last.SelectDraws == 0 || last.RuntimeRandDraws == 0 || last.TimerDraws == 0 {
			t.Fatalf("seed %d instance %d missing collector/seeded-stream evidence: %+v", request.Runtime.Seed, instance, last)
		}
		if instance == 0 {
			baseline = observed
		} else if *observed.ExitCode != *baseline.ExitCode || !bytes.Equal(observed.Stdout, baseline.Stdout) || !bytes.Equal(observed.Stderr, baseline.Stderr) || !bytes.Equal(observed.Transcript, baseline.Transcript) || !bytes.Equal(observed.ChoiceTrace.Bytes, baseline.ChoiceTrace.Bytes) || !bytes.Equal(observed.DiagnosticTrace.Bytes, baseline.DiagnosticTrace.Bytes) || observed.ChoiceTrace.Summary != baseline.ChoiceTrace.Summary || observed.runtimeTerminal != baseline.runtimeTerminal {
			t.Fatalf("seed %d fresh instance %d diverged: stdout=%x/%x wasi=%x/%x choices=%x/%x diagnostics=%x/%x", request.Runtime.Seed, instance, sha256.Sum256(observed.Stdout), sha256.Sum256(baseline.Stdout), sha256.Sum256(observed.Transcript), sha256.Sum256(baseline.Transcript), observed.ChoiceTrace.SHA256, baseline.ChoiceTrace.SHA256, observed.DiagnosticTrace.SHA256, baseline.DiagnosticTrace.SHA256)
		}
		if difference, err := choice.DiffDiagnostics(baseline.DiagnosticTrace.Bytes, observed.DiagnosticTrace.Bytes); err != nil || difference != nil {
			t.Fatalf("seed %d instance %d diagnostic divergence: %+v %v", request.Runtime.Seed, instance, difference, err)
		}
	}
	tape, err := choice.ProjectReplayPlan(baseline.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModeReplay, &tape
	request.Runtime.Diagnostics = false
	exact, err := Run(t.Context(), request)
	if err != nil || exact.Termination != "exit" || exact.ExitCode == nil || *exact.ExitCode != *baseline.ExitCode || !exact.Reaped || !bytes.Equal(exact.Stdout, baseline.Stdout) || !bytes.Equal(exact.Stderr, baseline.Stderr) || !bytes.Equal(exact.Transcript, baseline.Transcript) || !bytes.Equal(exact.ChoiceTrace.Bytes, baseline.ChoiceTrace.Bytes) {
		t.Fatalf("seed %d forced full-tape replay: %s %s err=%v stderr=%s", request.Runtime.Seed, exact.Termination, exact.Message, err, exact.Stderr)
	}
	report := runtimeQualification{SeedIndex: index, Seed: request.Runtime.Seed, Instances: 100, ExactReplays: 1, Module: request.ModuleSHA256, Helper: request.HelperSHA256, Runtime: request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, Model: ImplementationSHA256(), ChoiceSHA256: fmt.Sprintf("%x", baseline.ChoiceTrace.SHA256), DiagnosticSHA256: fmt.Sprintf("%x", baseline.DiagnosticTrace.SHA256), StdoutSHA256: fmt.Sprintf("%x", sha256.Sum256(baseline.Stdout)), StderrSHA256: fmt.Sprintf("%x", sha256.Sum256(baseline.Stderr)), WASISHA256: fmt.Sprintf("%x", sha256.Sum256(baseline.Transcript)), ExitCode: *baseline.ExitCode, Summary: baseline.ChoiceTrace.Summary, LastDiagnostic: baseline.DiagnosticTrace.Records[len(baseline.DiagnosticTrace.Records)-1]}
	version, err := exec.Command(os.Getenv("GOMAD3_STOCK_GO"), "version").Output()
	if err != nil || !strings.Contains(string(version), "go1.27.1 ") {
		t.Fatalf("qualification compiler version: %q %v", version, err)
	}
	report.SourceSHA256 = sourceSHA256
	report.RuntimeImplementationSHA256 = toolchain.ImplementationSHA256()
	report.CompilerSHA256 = fileHash(t, os.Getenv("GOMAD3_STOCK_GO"))
	report.GoVersion = strings.TrimSpace(string(version))
	report.HostGOOS, report.HostGOARCH = runtime.GOOS, runtime.GOARCH
	report.Engine, report.Clock, report.Limits = request.Engine, request.Config.Clock, request.Config.Limits
	report.Collector, report.Profile = "nogreenteagc", request.Config.Profile
	report.TerminalSHA256 = fmt.Sprintf("%x", sha256.Sum256(baseline.runtimeTerminal[:]))
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, fmt.Sprintf("seed-%02d", index))
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	for name, payload := range map[string][]byte{"report.json": data, "stdout": baseline.Stdout, "stderr": baseline.Stderr, "choices.bin": baseline.ChoiceTrace.Bytes, "diagnostics.bin": baseline.DiagnosticTrace.Bytes, "wasi.jsonl": baseline.Transcript, "finish.bin": baseline.runtimeTerminal[:]} {
		if err := os.WriteFile(filepath.Join(directory, name), payload, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("qualification %s", data)
}

func TestCooperativeRuntimeQualificationAggregate(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_QUALIFICATION_ROOT")
	if root == "" {
		t.Skip("run qualify-runtime for the retained aggregate gate")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("qualification requires an absolute retained evidence root")
	}
	sourceSHA256 := qualificationSourceSHA256(t)
	choices, outputs, outcomes := map[string]bool{}, map[string]bool{}, map[uint32]bool{}
	domains := map[string]map[string]bool{"map ": {}, "runq ": {}, "timer callbacks ": {}, "canary ": {}}
	var first runtimeQualification
	for index := range 32 {
		data, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("seed-%02d/report.json", index)))
		if err != nil {
			t.Fatal(err)
		}
		var report runtimeQualification
		if err := json.Unmarshal(data, &report); err != nil {
			t.Fatal(err)
		}
		if report.SeedIndex != uint64(index) || report.Seed != uint64(index)*1000003 || report.Instances != 100 || report.ExactReplays != 1 || report.Model != ImplementationSHA256() || report.SourceSHA256 != sourceSHA256 || report.RuntimeImplementationSHA256 != toolchain.ImplementationSHA256() || report.Helper != fileHash(t, helperPath(t)) || report.CompilerSHA256 != fileHash(t, os.Getenv("GOMAD3_STOCK_GO")) || report.Engine != PinnedEngine() || report.HostGOOS != runtime.GOOS || report.HostGOARCH != runtime.GOARCH || report.Profile != CooperativeProfile || report.Collector != "nogreenteagc" || !strings.Contains(report.GoVersion, "go1.27.1 ") {
			t.Fatalf("missing or stale qualification shard %d", index)
		}
		if index == 0 {
			first = report
		} else if report.Module != first.Module || report.Helper != first.Helper || report.Runtime != first.Runtime || report.GoVersion != first.GoVersion || report.Clock != first.Clock || report.Limits != first.Limits {
			t.Fatal("qualification shards are not one frozen candidate")
		}
		choices[report.ChoiceSHA256], outputs[report.StdoutSHA256], outcomes[report.ExitCode] = true, true, true
		payloads := map[string][]byte{}
		for name, expected := range map[string]string{"stdout": report.StdoutSHA256, "stderr": report.StderrSHA256, "choices.bin": report.ChoiceSHA256, "diagnostics.bin": report.DiagnosticSHA256, "wasi.jsonl": report.WASISHA256, "finish.bin": report.TerminalSHA256} {
			payload, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("seed-%02d", index), name))
			if err != nil || fmt.Sprintf("%x", sha256.Sum256(payload)) != expected {
				t.Fatalf("qualification shard %d payload %s changed: %v", index, name, err)
			}
			payloads[name] = payload
		}
		if err := validateQualificationPayloads(report, payloads); err != nil {
			t.Fatalf("qualification shard %d payload malformed: %v", index, err)
		}
		for prefix, observations := range domains {
			found := false
			for _, line := range strings.Split(string(payloads["stdout"]), "\n") {
				if strings.HasPrefix(line, prefix) {
					observations[line] = true
					found = true
				}
			}
			if !found {
				t.Fatalf("qualification shard %d lacks %s", index, prefix)
			}
		}
	}
	if len(choices) < 2 || len(outputs) < 2 || !outcomes[0] || !outcomes[17] {
		t.Fatal("representative seeds did not reach distinct choices, outputs and canary outcomes")
	}
	for name, observations := range domains {
		if len(observations) < 2 {
			t.Fatalf("representative seeds did not vary the %s domain", name)
		}
	}
	t.Logf("32 seeds x 100 fresh instances: 3200 complete record observations and 32 fresh forced exact replays; choices=%d outputs=%d outcomes=%d module=%s helper=%s runtime=%s model=%s", len(choices), len(outputs), len(outcomes), first.Module, first.Helper, first.Runtime, first.Model)
}

func qualificationSourceSHA256(t *testing.T) string {
	t.Helper()
	manifest := os.Getenv("GOMAD_WASM_QUALIFICATION_MANIFEST")
	if !filepath.IsAbs(manifest) {
		t.Fatal("qualification requires an absolute frozen source manifest")
	}
	data, err := os.ReadFile(manifest)
	if err != nil || len(data) == 0 {
		t.Fatalf("read frozen source manifest: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		digest, path, found := strings.Cut(line, "  ")
		if !found || !filepath.IsAbs(path) || digest != fileHash(t, path) {
			t.Fatalf("qualification source changed: %q", line)
		}
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func TestQualificationPayloadPreservesTerminalPeak(t *testing.T) {
	a, b := sha256.Sum256([]byte("a")), sha256.Sum256([]byte("b"))
	decision, err := choice.CanonicalDecision(0, choice.KindRunnable, 20, false, [][32]byte{a, b}, a, 0)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	diagnostic, err := choice.BuildDiagnosticTrace([]choice.DiagnosticRecord{{Ordinal: 0}}, 8192)
	if err != nil {
		t.Fatal(err)
	}
	trace.Summary.PeakGoroutines = 5
	finish := make([]byte, 16)
	binary.BigEndian.PutUint32(finish, 1)
	binary.BigEndian.PutUint32(finish[4:], 5)
	report := runtimeQualification{Summary: trace.Summary, LastDiagnostic: diagnostic.Records[0]}
	payloads := map[string][]byte{"choices.bin": trace.Bytes, "diagnostics.bin": diagnostic.Bytes, "finish.bin": finish}
	if err := validateQualificationPayloads(report, payloads); err != nil {
		t.Fatalf("valid terminal peak rejected: %v", err)
	}
	binary.BigEndian.PutUint32(finish[4:], 4)
	if err := validateQualificationPayloads(report, payloads); err == nil {
		t.Fatal("altered retained terminal peak accepted")
	}
}

func validateQualificationPayloads(report runtimeQualification, payloads map[string][]byte) error {
	trace, err := choice.DecodeStoredTrace(choice.Profile, payloads["choices.bin"], choice.TerminalMetadata{State: report.Summary.Terminal, Limit: 1 << 20, Records: report.Summary.Records, SHA256: sha256.Sum256(payloads["choices.bin"])})
	if err != nil {
		return err
	}
	finish := payloads["finish.bin"]
	if len(finish) != 16 || binary.BigEndian.Uint32(finish) != 1 || binary.BigEndian.Uint32(finish[4:]) == 0 {
		return fmt.Errorf("terminal frame malformed")
	}
	trace.Summary.PeakGoroutines = binary.BigEndian.Uint32(finish[4:])
	if err != nil || len(trace.Records) == 0 || trace.Summary != report.Summary || trace.Summary.Terminal != choice.TerminalComplete {
		return fmt.Errorf("choices malformed: %v", err)
	}
	diagnostic, err := choice.DecodeDiagnosticTrace(payloads["diagnostics.bin"])
	if err != nil || len(diagnostic.Records) == 0 || len(diagnostic.Records) != len(trace.Records) || diagnostic.Records[len(diagnostic.Records)-1] != report.LastDiagnostic {
		return fmt.Errorf("diagnostics malformed: %v", err)
	}
	return nil
}
