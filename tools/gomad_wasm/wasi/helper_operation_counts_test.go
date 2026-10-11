package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/hostfs"
	runnerbackend "go.temporal.io/server/tools/gomad3/runner/backend"
)

type canonicalOperationCounts struct {
	Calls      uint64            `json:"calls"`
	Local      uint64            `json:"local"`
	Remote     uint64            `json:"remote"`
	Operations map[string]uint64 `json:"remote_operations"`
}

func countCanonicalRemoteOperations(data []byte, maximum uint64) (canonicalOperationCounts, error) {
	counts := canonicalOperationCounts{Operations: map[string]uint64{}}
	if uint64(len(data)) > maximum || len(data) > 0 && data[len(data)-1] != '\n' {
		return counts, invalid("retained canonical byte bound or framing")
	}
	var last, pending uint64
	for len(data) > 0 {
		end := bytes.IndexByte(data, '\n')
		line := data[:end]
		data = data[end+1:]
		if pending == 0 {
			var call Call
			if err := decodeInput(line, &call, "type", "id", "op", "input"); err != nil {
				return counts, err
			}
			if call.Type != "call" || call.ID <= last || call.Op == "" {
				return counts, invalid("retained call sequence")
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(call.Input, &fields); err != nil || fields == nil {
				return counts, invalid("retained input object")
			}
			fd := "no-root-fd"
			if raw, ok := fields["fd"]; ok {
				var value uint32
				if bytes.Equal(raw, []byte("null")) || json.Unmarshal(raw, &value) != nil {
					return counts, invalid("retained fd type")
				}
				fd = fmt.Sprintf("fd=%d", value)
			}
			local := strings.HasPrefix(call.Op, "runtime_") && call.Op != "runtime_net_stream" || call.Op == "clock_time_get" || call.Op == "sched_yield"
			if call.Op == "poll_oneoff" {
				var subscriptions []map[string]json.RawMessage
				if raw, ok := fields["subscriptions"]; ok && len(raw) > 0 && raw[0] == '[' && json.Unmarshal(raw, &subscriptions) == nil {
					local = true
					for _, subscription := range subscriptions {
						var kind uint64
						raw, ok := subscription["type"]
						if !ok || bytes.Equal(raw, []byte("null")) || json.Unmarshal(raw, &kind) != nil || kind != 0 {
							local = false
							break
						}
					}
				}
			}
			counts.Calls++
			if local {
				counts.Local++
			} else {
				counts.Remote++
				namespace := "wasi_snapshot_preview1"
				if call.Op == "runtime_net_stream" {
					namespace = "gomad_wasm_v2"
				}
				counts.Operations[namespace+"/"+call.Op+"/"+fd]++
			}
			last, pending = call.ID, call.ID
		} else {
			if err := validateJSON(line); err != nil {
				return counts, err
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(line, &fields); err != nil {
				return counts, invalid("retained reply object")
			}
			keys := []string{"type", "id", "errno", "output"}
			for _, key := range []string{"unsupported", "message"} {
				if _, ok := fields[key]; ok {
					keys = append(keys, key)
				}
			}
			var reply Reply
			if err := decodeFields(line, fields, &reply, keys...); err != nil {
				return counts, err
			}
			if reply.Type != "reply" || reply.ID != pending {
				return counts, invalid("retained reply sequence")
			}
			pending = 0
		}
	}
	if pending != 0 {
		return counts, invalid("unfinished retained call")
	}
	return counts, nil
}

func operationCountPair(id int, op, input string) string {
	return fmt.Sprintf("{\"type\":\"call\",\"id\":%d,\"op\":%q,\"input\":%s}\n{\"type\":\"reply\",\"id\":%d,\"errno\":0,\"output\":{}}\n", id, op, input, id)
}

func TestCanonicalRemoteOperationCounts(t *testing.T) {
	data := []byte(operationCountPair(1, "clock_time_get", `{"clock_id":0}`) + operationCountPair(3, "sched_yield", `{}`) + operationCountPair(4, "poll_oneoff", `{"subscriptions":[{"type":0}]}`) + operationCountPair(5, "poll_oneoff", `{"subscriptions":[{"type":0},{"type":1,"fd":4}]}`) + operationCountPair(6, "fd_write", `{"fd":1}`) + operationCountPair(7, "fd_write", `{"fd":2}`) + operationCountPair(8, "fd_pwrite", `{"fd":4}`) + operationCountPair(9, "path_open", `{"fd":3}`))
	original := bytes.Clone(data)
	got, err := countCanonicalRemoteOperations(data, uint64(len(data)))
	if err != nil || got.Calls != 8 || got.Local != 3 || got.Remote != 5 || got.Operations["wasi_snapshot_preview1/poll_oneoff/no-root-fd"] != 1 || got.Operations["wasi_snapshot_preview1/fd_write/fd=1"] != 1 || got.Operations["wasi_snapshot_preview1/fd_write/fd=2"] != 1 || got.Operations["wasi_snapshot_preview1/fd_pwrite/fd=4"] != 1 || got.Operations["wasi_snapshot_preview1/path_open/fd=3"] != 1 {
		t.Fatalf("actual canonical operation counts = %+v, %v", got, err)
	}
	if !bytes.Equal(original, data) {
		t.Fatal("counter mutated canonical input")
	}
}

func TestCanonicalRemoteOperationCountsRejectsInvalid(t *testing.T) {
	valid := operationCountPair(1, "fd_write", `{"fd":1}`)
	for name, data := range map[string]string{
		"duplicate":             strings.Replace(valid, `"fd":1`, `"fd":1,"f\u0064":2`, 1),
		"nested duplicate":      strings.Replace(valid, `"fd":1`, `"fd":1,"x":{"a":0,"a":1}`, 1),
		"null fd":               strings.Replace(valid, `"fd":1`, `"fd":null`, 1),
		"fd type":               strings.Replace(valid, `"fd":1`, `"fd":"1"`, 1),
		"fd overflow":           strings.Replace(valid, `"fd":1`, `"fd":4294967296`, 1),
		"unknown call key":      strings.Replace(valid, `"op":"fd_write"`, `"op":"fd_write","extra":0`, 1),
		"wrong reply":           strings.Replace(valid, `"type":"reply","id":1`, `"type":"reply","id":2`, 1),
		"unfinished":            strings.Split(valid, "\n")[0] + "\n",
		"nonincreasing":         valid + valid,
		"wrong tag":             strings.Replace(valid, `"type":"call"`, `"type":"unknown"`, 1),
		"malformed":             "{\n",
		"missing final newline": strings.TrimSuffix(valid, "\n"),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := countCanonicalRemoteOperations([]byte(data), uint64(len(data))); err == nil {
				t.Fatal("invalid retained canonical trace accepted")
			}
		})
	}
	if _, err := countCanonicalRemoteOperations([]byte(valid), uint64(len(valid)-1)); err == nil {
		t.Fatal("retained byte bound ignored")
	}
}

func TestCanonicalRemoteOperationClassification(t *testing.T) {
	for _, sample := range []struct {
		op, input string
		local     bool
	}{
		{"runtime_decision", `{}`, true}, {"runtime_net_stream", `{}`, false},
		{"poll_oneoff", `{"subscriptions":[]}`, true}, {"poll_oneoff", `{}`, false},
		{"poll_oneoff", `{"subscriptions":null}`, false}, {"poll_oneoff", `{"subscriptions":[{"type":null}]}`, false},
		{"poll_oneoff", `{"subscriptions":[{"type":0.0}]}`, false}, {"poll_oneoff", `{"subscriptions":[0]}`, false},
	} {
		data := []byte(operationCountPair(1, sample.op, sample.input))
		counts, err := countCanonicalRemoteOperations(data, uint64(len(data)))
		if err != nil || (counts.Local == 1) != sample.local || counts.Local+counts.Remote != 1 {
			t.Fatalf("%s/%s = %+v, %v", sample.op, sample.input, counts, err)
		}
	}
}

func TestHelperCanonicalOperationRetentionDiagnostic(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_OPERATION_COUNT_ROOT")
	if root == "" {
		t.Skip("counts diagnostic requires an exclusive guest slot")
	}
	control := os.Getenv("GOMAD_WASM_OPERATION_CONTROL_ROOT")
	module, cache, runtimeIdentity := os.Getenv("GOMAD_WASM_MATCHED_MODULE"), os.Getenv("GOMAD_WASM_MATCHED_CACHE"), os.Getenv("GOMAD_WASM_MATCHED_TOOLCHAIN_ID")
	for _, path := range []string{root, control, module, cache} {
		if !filepath.IsAbs(path) {
			t.Fatal("counts diagnostic requires absolute paths")
		}
	}
	if runtimeIdentity == "" {
		t.Fatal("explicit legacy runtime required")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	repo, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: runtimeIdentity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(runtimeIdentity))}
	for _, suite := range []struct{ name, run string }{
		{"workflow", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"},
		{"child", "^TestChildWorkflowSuite$"}, {"timers", "^TestUserTimersTestSuite$"},
	} {
		for _, seed := range []uint64{0, 1000003} {
			name := suite.name + "-seed" + stringSeed(seed)
			if !t.Run(name, func(t *testing.T) {
				config := prototypeTemporalConfig(t, repo, "/workspace/tests", "-test.run="+suite.run, "-persistenceType=sql", "-persistenceDriver=sqlite")
				config.Profile, config.Clock.ReadStepNanos = CooperativeProfile, 0
				helper := helperPath(t)
				request := Request{HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 4 << 30, Fuel: 1 << 50, Config: config, Timeout: 5 * time.Minute, CompileCache: cache, DigestTranscript: false, Runtime: &RuntimeControl{Seed: seed, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: choice.MaximumTraceBytes}}}
				reference := suite.name + "-reference-seed" + stringSeed(seed)
				var historical Result
				readDiagnosticJSON(t, filepath.Join(control, reference+"-observed-result.json"), &historical)
				var metadata struct {
					Helper  string `json:"helper_sha256"`
					Module  string `json:"module_sha256"`
					Model   string `json:"model_sha256"`
					Runtime string `json:"runtime_identity"`
				}
				readDiagnosticJSON(t, filepath.Join(control, reference+"-metrics.json"), &metadata)
				if metadata.Helper != request.HelperSHA256 || metadata.Module != request.ModuleSHA256 || metadata.Model != ImplementationSHA256() || metadata.Runtime != runtimeIdentity {
					t.Fatal("current control identities mismatch")
				}
				var before, retained Result
				for _, owner := range []string{"control", "retained"} {
					current := request
					current.DigestTranscript = owner == "control"
					started := time.Now()
					result, runErr := matchedPrepareAdmitted(runtime.GOOS, "/proc/meminfo", os.Getenv("GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD"), func() (Result, error) { return Run(t.Context(), current) })
					persistOperationDiagnostic(t, root, name+"-"+owner, current, result, time.Since(started).Nanoseconds())
					if runErr != nil {
						t.Fatal(runErr)
					}
					requireExit(t, name+"-"+owner, result, 0)
					if result.Telemetry == nil || !result.Telemetry.CacheHit || result.Telemetry.CompileNanos != 0 {
						t.Fatal("warm diagnostic cache miss; not accepted as warm")
					}
					if owner == "control" {
						before = result
					} else {
						retained = result
					}
				}
				requireDiagnosticEvidence(t, before, retained)
				if difference := historicalJSONPublicProjectionDifference(historical, before); difference != "" {
					t.Fatalf("historical JSON PUBLIC projection changed (no historical private-terminal proof): %s", difference)
				}
				counts, err := countCanonicalRemoteOperations(retained.Transcript, 64<<20)
				if err != nil {
					t.Fatal(err)
				}
				var diagnostic struct {
					Profile struct {
						ABI struct {
							Remote struct {
								Count uint64 `json:"count"`
							} `json:"remote_callback"`
						} `json:"abi"`
					} `json:"helper_profile"`
				}
				readDiagnosticJSON(t, filepath.Join(control, suite.name+"-diagnostic-seed"+stringSeed(seed)+"-metrics.json"), &diagnostic)
				if counts.Remote != diagnostic.Profile.ABI.Remote.Count {
					t.Fatalf("actual remote classification=%d diagnostic callback count=%d", counts.Remote, diagnostic.Profile.ABI.Remote.Count)
				}
				temporalJSON(t, filepath.Join(root, name+"-operation-counts.json"), counts)
			}) {
				return
			}
		}
	}
}

func readDiagnosticJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := hostfs.ReadBounded(path, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateJSON(data); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func persistOperationDiagnostic(t *testing.T, root, name string, request Request, result Result, wall int64) {
	t.Helper()
	runtimeIdentity := ""
	if request.Runtime != nil && request.Runtime.Choice != nil {
		runtimeIdentity = request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey
	}
	temporalJSON(t, filepath.Join(root, name+"-metrics.json"), map[string]any{"diagnostic_only": true, "retention_mode": !request.DigestTranscript, "retained_limit_scope": "64MiB postexecution only; configured transport reservation unchanged", "module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "model_sha256": ImplementationSHA256(), "runtime_identity": runtimeIdentity, "run_wall_nanos": wall, "authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos, "phase_metrics": result.Telemetry, "wait_resource_usage": result.HelperResourceUsage, "termination": result.Termination, "message": result.Message, "retained_bytes": len(result.Transcript)})
	persistDiagnosticRawResult(t, root, name, result)
}

func TestOperationDiagnosticPersistsPrivateTerminalAndRawEvidence(t *testing.T) {
	root := t.TempDir()
	result := Result{Stdout: []byte("stdout"), Transcript: []byte("canonical\n"), runtimeTerminal: [16]byte{1, 2, 3, 4}}
	result.ChoiceTrace.Bytes = []byte("choice")
	persistDiagnosticRawResult(t, root, "record", result)
	for name, expected := range map[string][]byte{"finish.bin": result.runtimeTerminal[:], "canonical.jsonl": result.Transcript, "choice.bin": result.ChoiceTrace.Bytes} {
		actual, err := os.ReadFile(filepath.Join(root, "record-"+name))
		if err != nil || !bytes.Equal(actual, expected) {
			t.Fatalf("raw %s not preserved: %q, %v", name, actual, err)
		}
	}
	var stored Result
	readDiagnosticJSON(t, filepath.Join(root, "record-observed-result.json"), &stored)
	if !bytes.Equal(stored.Transcript, result.Transcript) || !bytes.Equal(stored.ChoiceTrace.Bytes, result.ChoiceTrace.Bytes) {
		t.Fatal("full exported result lost evidence")
	}
	if sameEvidence(result, stored) != "terminal" {
		t.Fatal("JSON unexpectedly claims private terminal preservation")
	}
	terminal, err := os.ReadFile(filepath.Join(root, "record-finish.bin"))
	if err != nil {
		t.Fatal(err)
	}
	copy(stored.runtimeTerminal[:], terminal)
	if sameEvidence(result, stored) != "" {
		t.Fatal("actual terminal sidecar failed exact roundtrip")
	}
}

func persistDiagnosticRawResult(t *testing.T, root, name string, result Result) {
	t.Helper()
	if len(result.Transcript) > 64<<20 {
		t.Fatal("postexecution retained transcript capacity exceeded; actual costs retained")
	}
	temporalWrite(t, filepath.Join(root, name+"-canonical.jsonl"), result.Transcript)
	temporalWrite(t, filepath.Join(root, name+"-choice.bin"), result.ChoiceTrace.Bytes)
	temporalWrite(t, filepath.Join(root, name+"-finish.bin"), result.runtimeTerminal[:])
	temporalJSON(t, filepath.Join(root, name+"-observed-result.json"), result)
}

func historicalJSONPublicProjectionDifference(historical, actual Result) string {
	actual.runtimeTerminal = [16]byte{}
	if difference := sameEvidence(historical, actual); difference != "" {
		return difference
	}
	if historical.Calls != actual.Calls || historical.TimeAdvances != actual.TimeAdvances || historical.EnvironmentIdentity != actual.EnvironmentIdentity || historical.FuelRemaining != actual.FuelRemaining || historical.VirtualTimeElapsedNanos != actual.VirtualTimeElapsedNanos {
		return "public counters or identity"
	}
	return ""
}

func TestHistoricalJSONPublicProjectionRejectsChangedEvidence(t *testing.T) {
	exit := uint32(0)
	actual := Result{Termination: "exit", ExitCode: &exit, Stdout: []byte("stdout"), Stderr: []byte("stderr"), NetworkTranscript: []byte("network"), TranscriptSHA256: [32]byte{1}, Calls: 7, TimeAdvances: 2, EnvironmentIdentity: "model", FuelRemaining: 13, VirtualTimeElapsedNanos: 17, runtimeTerminal: [16]byte{1, 2, 3}}
	actual.ChoiceTrace.Bytes = []byte("choices")
	data, err := json.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}
	var historical Result
	if err := json.Unmarshal(data, &historical); err != nil {
		t.Fatal(err)
	}
	if sameEvidence(historical, actual) != "terminal" {
		t.Fatal("historical JSON falsely preserved private terminal")
	}
	if difference := historicalJSONPublicProjectionDifference(historical, actual); difference != "" {
		t.Fatalf("equal public projection rejected: %s", difference)
	}
	for name, mutate := range map[string]func(*Result){
		"stdout": func(r *Result) { r.Stdout = []byte("changed") }, "stderr": func(r *Result) { r.Stderr = []byte("changed") },
		"wasi": func(r *Result) { r.TranscriptSHA256[0]++ }, "choice": func(r *Result) { r.ChoiceTrace.Bytes = []byte("changed") },
		"network": func(r *Result) { r.NetworkTranscript = []byte("changed") }, "termination": func(r *Result) { r.Termination = "invalid" },
		"exit": func(r *Result) { code := uint32(1); r.ExitCode = &code }, "calls": func(r *Result) { r.Calls++ },
		"advances": func(r *Result) { r.TimeAdvances++ }, "environment": func(r *Result) { r.EnvironmentIdentity = "changed" },
		"fuel": func(r *Result) { r.FuelRemaining++ }, "virtual": func(r *Result) { r.VirtualTimeElapsedNanos++ },
	} {
		t.Run(name, func(t *testing.T) {
			changed := actual
			mutate(&changed)
			if historicalJSONPublicProjectionDifference(historical, changed) == "" {
				t.Fatal("changed public evidence accepted")
			}
		})
	}
	if actual.runtimeTerminal != [16]byte{1, 2, 3} || historical.runtimeTerminal != [16]byte{} {
		t.Fatal("projection mutated original terminal evidence")
	}
}
