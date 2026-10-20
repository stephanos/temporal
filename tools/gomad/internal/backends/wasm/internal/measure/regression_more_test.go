package measure

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	runnerbackend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/toolchain"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
)

func TestHelperMigrationQualificationRejectsMissingBaselineCosts(t *testing.T) {
	for _, fixture := range []struct {
		name, cost string
		reject     bool
	}{
		{"missing", "", true},
		{"zero", `,"WallNanos":0`, true},
		{"negative", `,"WallNanos":-1`, true},
		{"positive", `,"WallNanos":1`, false},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			baseline := t.TempDir()
			for _, name := range []string{"scheduling", "virtual-time"} {
				data := `{}`
				if name == "scheduling" {
					data = `{"Guests":{},"Runs":[{"Guest":"choices","Variant":"strict","Seed":0,"Instance":0,"Termination":"exit"` + fixture.cost + `}]}`
				}
				if err := os.WriteFile(filepath.Join(baseline, name+"-report.json"), []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			// wasmmeasure migration runs this preflight before any guest or
			// evidence; a costless baseline is refused before module checks.
			_, modules, err := migrationBaselinePreflight(baseline)
			costRejected := err != nil && strings.Contains(err.Error(), "baseline wall time must be positive")
			if fixture.reject != costRejected || !fixture.reject && err != nil {
				t.Fatalf("baseline cost preflight reject=%v: %v", fixture.reject, err)
			}
			if !fixture.reject {
				err := migrationBaselineModule(filepath.Join(baseline, "modules", "choices.wasm"), modules["choices"])
				if err == nil || !strings.Contains(err.Error(), "baseline guest module identity unavailable or changed") {
					t.Fatalf("missing baseline module admitted: %v", err)
				}
			}
		})
	}
}

func TestMatchedPreparationMemoryBoundaryBeforeBothOwners(t *testing.T) {
	for _, owner := range []string{"original", "candidate"} {
		t.Run(owner, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "meminfo")
			for _, test := range []struct {
				data string
				want bool
			}{
				{"MemAvailable: 8388607 kB\n", false},
				{"MemAvailable: 8388608 kB\n", true},
				{"MemAvailable: 8388609 kB\n", true},
			} {
				if err := os.WriteFile(path, []byte(test.data), 0600); err != nil {
					t.Fatal(err)
				}
				called := false
				result, err := matchedPrepareAdmitted("linux", path, "1", func() (wasi.Result, error) {
					called = true
					return wasi.Result{Termination: owner}, nil
				})
				if called != test.want || (err == nil) != test.want || (result.Termination == owner) != test.want {
					t.Fatalf("owner=%s observation=%q dispatched=%t want=%t error=%v", owner, test.data, called, test.want, err)
				}
			}
		})
	}
}

func diagnosticFixture(t *testing.T) []byte {
	t.Helper()
	phases := map[string]any{}
	for _, name := range strings.Fields("module_read module_hash engine_init cache_identity cache_read envelope_verify deserialize compile cache_write instantiate execute") {
		phases[name] = map[string]uint64{"count": 0, "wall_nanos": 0, "user_nanos": 0, "system_nanos": 0, "cpu_samples": 0, "peak_rss_bytes_high_water": 0}
	}
	phases["execute"] = map[string]uint64{"count": 1, "wall_nanos": 100, "user_nanos": 70, "system_nanos": 10, "cpu_samples": 1, "peak_rss_bytes_high_water": 1024}
	abi := map[string]any{}
	for _, name := range strings.Fields("clock scheduler select observation idle other_local remote_callback") {
		abi[name] = map[string]uint64{"count": 0, "wall_nanos": 0, "instantiate_wall_nanos": 0, "execute_wall_nanos": 0}
	}
	abi["clock"] = map[string]uint64{"count": 1, "wall_nanos": 30, "instantiate_wall_nanos": 0, "execute_wall_nanos": 30}
	nested := map[string]any{}
	for _, name := range strings.Fields("alternative_hash encoding transcript_hash flush remote_write remote_read_wait") {
		nested[name] = map[string]uint64{"count": 0, "wall_nanos": 0}
	}
	data, err := json.Marshal(map[string]any{"schema": "gomad.wasm-profile/v1", "diagnostic_only": true, "backend": "cranelift-jit", "host_os": "linux", "host_arch": "aarch64", "phases": phases, "abi": abi, "nested": nested, "residual": map[string]uint64{"instantiate_wall_nanos": 0, "execute_wall_nanos": 70}, "abi_overlap_errors": 0})
	if err != nil {
		t.Fatal(err)
	}
	return append(append([]byte("gomad-wasm-profile-v1 "), data...), '\n')
}

func TestDiagnosticReportRejectsMalformedOrUnattributedMeasurements(t *testing.T) {
	valid := diagnosticFixture(t)
	report, err := decodeDiagnosticReport(valid)
	if err != nil || report.Phases["execute"].UserNanos != 70 || report.ABI["clock"].ExecuteWallNanos != 30 || report.Residual.ExecuteWallNanos != 70 {
		t.Fatalf("literal phase profile: report=%+v err=%v", report, err)
	}
	for _, test := range []struct {
		name string
		data []byte
	}{
		{"missing", nil},
		{"oversized", bytes.Repeat([]byte{'x'}, 48<<10+1)},
		{"duplicate-report", append(append([]byte(nil), valid...), valid...)},
		{"trailing", append(append([]byte(nil), valid...), []byte("unexpected\n")...)},
		{"null", []byte("gomad-wasm-profile-v1 null\n")},
		{"wrong-schema", bytes.Replace(valid, []byte("gomad.wasm-profile/v1"), []byte("gomad.wasm-profile/v2"), 1)},
		{"unknown-top", bytes.Replace(valid, []byte(`"schema":`), []byte(`"extra":0,"schema":`), 1)},
		{"nested-duplicate", bytes.Replace(valid, []byte(`"user_nanos":70`), []byte(`"user_nanos":1,"user_nanos":70`), 1)},
		{"escaped-duplicate", bytes.Replace(valid, []byte(`"user_nanos":70`), []byte(`"\u0075ser_nanos":1,"user_nanos":70`), 1)},
		{"missing-phase", bytes.Replace(valid, []byte(`"module_read":`), []byte(`"absent":`), 1)},
		{"missing-measurement", bytes.Replace(valid, []byte(`"user_nanos":70,`), nil, 1)},
		{"unknown-measurement", bytes.Replace(valid, []byte(`"user_nanos":70`), []byte(`"user_nanos":70,"invented":0`), 1)},
		{"not-diagnostic", bytes.Replace(valid, []byte(`"diagnostic_only":true`), []byte(`"diagnostic_only":false`), 1)},
		{"not-jit", bytes.Replace(valid, []byte("cranelift-jit"), []byte("interpreter"), 1)},
		{"negative", bytes.Replace(valid, []byte(`"user_nanos":70`), []byte(`"user_nanos":-1`), 1)},
		{"too-many-cpu-samples", bytes.Replace(valid, []byte(`"cpu_samples":1`), []byte(`"cpu_samples":2`), 1)},
		{"overlapping-abi", bytes.Replace(valid, []byte(`"abi_overlap_errors":0`), []byte(`"abi_overlap_errors":1`), 1)},
		{"wrong-residual", bytes.Replace(valid, []byte(`"execute_wall_nanos":70`), []byte(`"execute_wall_nanos":71`), 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeDiagnosticReport(test.data); err == nil {
				t.Fatalf("accepted %s diagnostic", test.name)
			}
		})
	}
}

func operationCountRecord(id uint64, tag byte, body string) string {
	// Independent fixture framing; payloads below are hand-derived from protocol.md.
	data := []byte{tag, 0, 0, 0}
	data = binary.LittleEndian.AppendUint64(data, id)
	data = binary.LittleEndian.AppendUint32(data, uint32(len(body)))
	return string(append(data, body...))
}

func TestCanonicalRemoteOperationCountsRejectsInvalid(t *testing.T) {
	valid := "GMWT\x01\x00\x00\x00" + operationCountRecord(1, 8, "\x01"+strings.Repeat("\x00", 11))
	for name, data := range map[string]string{
		"truncated":        valid[:len(valid)-1],
		"trailing":         valid + "\x00",
		"nonincreasing":    valid + valid[8:],
		"unknown tag":      "GMWT\x01\x00\x00\x00" + operationCountRecord(1, 255, ""),
		"oversize payload": "GMWT\x01\x00\x00\x00" + operationCountRecord(1, 8, "\x01\x00\x00\x00\xff\xff\xff\xff"),
		"oversize array":   "GMWT\x01\x00\x00\x00" + operationCountRecord(1, 32, "\xff\xff\xff\xff"),
		"retired JSONL":    `{"type":"call","id":1,"op":"sched_yield","input":{}}` + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := countCanonicalRemoteOperations([]byte(data), uint64(len(data))); err == nil {
				t.Fatal("invalid retained binary trace accepted")
			}
		})
	}
	if _, err := countCanonicalRemoteOperations([]byte(valid), uint64(len(valid)-1)); err == nil {
		t.Fatal("retained byte bound ignored")
	}
}

func TestOperationDiagnosticPersistsPrivateTerminalAndRawEvidence(t *testing.T) {
	root := t.TempDir()
	result := measurementResult(wasi.Result{Stdout: []byte("stdout"), Transcript: []byte("canonical\n")}, [16]byte{1, 2, 3, 4}, nil)
	result.ChoiceTrace.Bytes = []byte("choice")
	testPersistDiagnosticRawResult(t, root, "record", result)
	for name, expected := range map[string][]byte{"finish.bin": measurementTerminalBytes(result)[:], "canonical.bin": result.Transcript, "choice.bin": result.ChoiceTrace.Bytes} {
		actual, err := os.ReadFile(filepath.Join(root, "record-"+name))
		if err != nil || !bytes.Equal(actual, expected) {
			t.Fatalf("raw %s not preserved: %q, %v", name, actual, err)
		}
	}
	var stored wasi.Result
	testReadDiagnosticJSON(t, filepath.Join(root, "record-observed-result.json"), &stored)
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
	var storedTerminal [16]byte
	copy(storedTerminal[:], terminal)
	stored.SetRuntimeTerminal(storedTerminal)
	if sameEvidence(result, stored) != "" {
		t.Fatal("actual terminal sidecar failed exact roundtrip")
	}
}

func TestHistoricalJSONPublicProjectionRejectsChangedEvidence(t *testing.T) {
	exit := uint32(0)
	actual := measurementResult(wasi.Result{Termination: "exit", ExitCode: &exit, Stdout: []byte("stdout"), Stderr: []byte("stderr"), NetworkTranscript: []byte("network"), TranscriptSHA256: [32]byte{1}, Calls: 7, TimeAdvances: 2, EnvironmentIdentity: "model", FuelRemaining: 13, VirtualTimeElapsedNanos: 17}, [16]byte{1, 2, 3}, nil)
	actual.ChoiceTrace.Bytes = []byte("choices")
	data, err := json.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}
	var historical wasi.Result
	if err := json.Unmarshal(data, &historical); err != nil {
		t.Fatal(err)
	}
	if sameEvidence(historical, actual) != "terminal" {
		t.Fatal("historical JSON falsely preserved private terminal")
	}
	if difference := historicalJSONPublicProjectionDifference(historical, actual); difference != "" {
		t.Fatalf("equal public projection rejected: %s", difference)
	}
	for name, mutate := range map[string]func(*wasi.Result){
		"stdout": func(r *wasi.Result) { r.Stdout = []byte("changed") }, "stderr": func(r *wasi.Result) { r.Stderr = []byte("changed") },
		"wasi": func(r *wasi.Result) { r.TranscriptSHA256[0]++ }, "choice": func(r *wasi.Result) { r.ChoiceTrace.Bytes = []byte("changed") },
		"network": func(r *wasi.Result) { r.NetworkTranscript = []byte("changed") }, "termination": func(r *wasi.Result) { r.Termination = "invalid" },
		"exit": func(r *wasi.Result) { code := uint32(1); r.ExitCode = &code }, "calls": func(r *wasi.Result) { r.Calls++ },
		"advances": func(r *wasi.Result) { r.TimeAdvances++ }, "environment": func(r *wasi.Result) { r.EnvironmentIdentity = "changed" },
		"fuel": func(r *wasi.Result) { r.FuelRemaining++ }, "virtual": func(r *wasi.Result) { r.VirtualTimeElapsedNanos++ },
	} {
		t.Run(name, func(t *testing.T) {
			changed := actual
			mutate(&changed)
			if historicalJSONPublicProjectionDifference(historical, changed) == "" {
				t.Fatal("changed public evidence accepted")
			}
		})
	}
	if actual.RuntimeTerminal() != [16]byte{1, 2, 3} || historical.RuntimeTerminal() != [16]byte{} {
		t.Fatal("projection mutated original terminal evidence")
	}
}

func TestDiagnosticPerformanceHost(t *testing.T) {
	const armCPU = "processor: 0\nFeatures: fp asimd aes crc32\nCPU architecture: 8\n"
	const x86CPU = "processor: 0\nvendor_id: GenuineIntel\nflags: fpu sse sse2\n"
	for _, test := range []struct {
		name, goos, goarch, cpuinfo string
		want                        bool
	}{
		{"Linux arm64", "linux", "arm64", armCPU, true},
		{"Darwin arm64", "darwin", "arm64", armCPU, false},
		{"Linux amd64", "linux", "amd64", x86CPU, false},
		{"emulated arm64 on x86", "linux", "arm64", x86CPU, false},
		{"missing host witness", "linux", "arm64", "", false},
		{"32-bit ARM", "linux", "arm64", "Features: fp neon\nCPU architecture: 8\n", false},
		{"oversized host witness", "linux", "arm64", armCPU + strings.Repeat("\n", diagnosticHostCPUInfoLimit), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := diagnosticPerformanceHost(test.goos, test.goarch, test.cpuinfo); got != test.want {
				t.Fatalf("performance host accepted=%t, want %t", got, test.want)
			}
		})
	}
}

func TestDiagnosticGuestFrameBuckets(t *testing.T) {
	for _, test := range []struct{ bucket, names string }{
		{"gc", "runtime.gcBgMarkWorker runtime.gcDrainN runtime.scanobject runtime.markrootBlock runtime.gcAssistAlloc1 runtime.bgsweep runtime.sweepone runtime.wbBufFlush1 runtime.gcWriteBarrier runtime.bulkBarrierPreWrite runtime.writebarrierptr runtime.scanObject runtime.scanObjectsSmall runtime.greyobject runtime.findObject runtime.tryDeferToSpanScan runtime.scanblock runtime.scanstack runtime.scanframeworker"},
		{"scheduler", "runtime.schedule runtime.findRunnable runtime.gopark runtime.park_m runtime.goready runtime.mcall runtime.gogo runtime.handleEvent runtime.beforeIdle wasm_pc_f_loop runtime.wasm_pc_f_loop"},
		{"allocation", "runtime.mallocgcSmallScanNoHeader runtime.makeslice runtime.growslice runtime.newobject runtime.newarray runtime.makemap runtime.mapassign"},
		{"test", "go.temporal.io/server/tests.(*ActivityTestSuite).TestActivity go.temporal.io/server/tests/testcore.NewTestServer go.temporal.io_server_tests.__ActivityTestSuite_.TestActivity go.temporal.io_server_tests_testcore.NewTestServer"},
		{"server", "go.temporal.io/server/service/history.(*Engine).Execute go.temporal.io_server_service_history.__historyEngineImpl_.StartWorkflowExecution"},
		{"runtime", "runtime.memmove runtime/atomic.Store internal/runtime/maps.(*Map).get internal_runtime_maps.memHashFallback"},
		{"stdlib", "encoding/json.Marshal net/http.(*Client).Do testing.tRunner reflect.Value.Call sync.(*Mutex).Lock encoding_json.Marshal sync.__Mutex_.Lock sort.insertionSort_func syscall.fd_write syscall.path_open sort.pdqsort_func"},
		{"other", "github.com/stretchr/testify/require.NoError main.main type:.eq.go.temporal.io/server.Task <wasm-function[1]> github.com_ncruces_go_sqlite3_wasm_v6.__Module_._sqlite3VdbeExec go.temporal.io_api_workflowservice_v1.__workflowServiceClient_.RespondWorkflowTaskCompleted golang.org_x_net_http2.__Framer_.ReadFrame"},
	} {
		for _, name := range strings.Fields(test.names) {
			t.Run(name, func(t *testing.T) {
				if got := guestStackBucket([]string{name}); got != test.bucket {
					t.Fatalf("bucket=%s, want %s", got, test.bucket)
				}
			})
		}
	}
	for _, test := range []struct {
		frames []string
		want   string
	}{
		{[]string{"runtime.mallocgc", "runtime.gcAssistAlloc", "go.temporal.io/server/tests.Test"}, "allocation"},
		{[]string{"runtime.gcDrain", "runtime.schedule"}, "gc"},
		{[]string{"runtime.schedule", "runtime.gcDrain"}, "scheduler"},
		{[]string{"runtime.schedule.func1", "go.temporal.io/server/tests.Test"}, "scheduler"},
		{[]string{"go.temporal.io/server/tests.Test", "runtime.gcDrain"}, "test"},
		{[]string{"runtime.memmove", "go.temporal.io/server/tests.Test"}, "runtime"},
		{[]string{"<wasm-function[1]>", "runtime.gcDrain"}, "gc"},
		{nil, "other"},
	} {
		if got := guestStackBucket(test.frames); got != test.want {
			t.Fatalf("frames %v: bucket=%s, want %s", test.frames, got, test.want)
		}
	}
}

func diagnosticGuestFixture() []byte {
	return []byte(`{
 "meta":{"interval":1},
 "threads":[{
  "samples":{"length":4,"stack":[1,0,2,null],"weight":[3,2,5,7],"weightType":"samples","timeDeltas":[0,100000,1,100000],"threadCPUDelta":[0,100000000,0,100000000]},
  "stackTable":{"length":3,"frame":[2,0,1],"prefix":[null,0,0]},
  "frameTable":{"length":3,"func":[1,2,0]},
  "funcTable":{"length":3,"name":[2,0,1]},
  "stringArray":["runtime.gcDrain","runtime.schedule","go.temporal.io/server/tests.Test"]
 },{
  "samples":{"length":1,"stack":[0],"weight":null,"weightType":"samples"},
  "stackTable":{"length":1,"frame":[0],"prefix":[null]},
  "frameTable":{"length":1,"func":[0]},
  "funcTable":{"length":1,"name":[0]},
  "stringArray":["encoding/json.Marshal"]
 }]
}`)
}

func TestDiagnosticGuestAttributionRejectsMalformedTables(t *testing.T) {
	for _, test := range []struct{ name, before, after string }{
		{"sample length", `"length":4`, `"length":3`},
		{"weight length", `[3,2,5,7]`, `[3,2,5]`},
		{"negative weight", `[3,2,5,7]`, `[-3,2,5,7]`},
		{"weight overflow", `[3,2,5,7]`, `[9223372036854775807,9223372036854775807,5,7]`},
		{"allocation weight", `"weightType":"samples"`, `"weightType":"bytes"`},
		{"stack length", `"frame":[2,0,1]`, `"frame":[2,0]`},
		{"prefix length", `"prefix":[null,0,0]`, `"prefix":[null,0]`},
		{"stack index", `"stack":[1,0,2,null]`, `"stack":[3,0,2,null]`},
		{"negative stack", `"stack":[1,0,2,null]`, `"stack":[-1,0,2,null]`},
		{"cycle", `"prefix":[null,0,0]`, `"prefix":[1,0,0]`},
		{"frame index", `"frame":[2,0,1]`, `"frame":[2,3,1]`},
		{"function index", `"func":[1,2,0]`, `"func":[3,2,0]`},
		{"name index", `"name":[2,0,1]`, `"name":[3,0,1]`},
		{"no interval", `"interval":1`, `"interval":0`},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := strings.Replace(string(diagnosticGuestFixture()), test.before, test.after, 1)
			_, _, err := decodeGuestAttribution([]byte(data))
			if err == nil {
				t.Fatal("malformed profile accepted")
			}
		})
	}
	for _, data := range []string{`{}`, `{`, `{"meta":{"interval":1},"threads":[]}`} {
		_, _, err := decodeGuestAttribution([]byte(data))
		if err == nil {
			t.Fatal("malformed profile accepted")
		}
	}
}

func TestDiagnosticVariantArtifactSuffixes(t *testing.T) {
	root := t.TempDir()
	_, _, failure := testRunGuestVariant(t, root, "invalid-request", wasi.Request{}, false)
	if failure == "" {
		t.Fatal("invalid request unexpectedly executed")
	}
	files, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, file := range files {
		names = append(names, file.Name())
	}
	want := []string{
		"invalid-request-canonical.bin", "invalid-request-choice.bin",
		"invalid-request-guest-stderr.txt", "invalid-request-guest-stdout.txt",
		"invalid-request-helper-stderr.txt", "invalid-request-network.jsonl",
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("artifact names = %v, want %v", names, want)
	}
}

func TestDiagnosticVariantEvidencePreservesDerivedEnvironmentRecords(t *testing.T) {
	config := standaloneConfig()
	config.Environment = append(config.Environment, "GOGC=200")
	baseline, variant := newTestEnvironment(t, standaloneConfig()), newTestEnvironment(t, config)
	for _, environment := range []*wasi.Environment{baseline, variant} {
		testCall(t, environment, 1, "environ_sizes_get", `{}`)
		testCall(t, environment, 2, "environ_get", `{}`)
		testCall(t, environment, 3, "args_get", `{}`)
	}
	before := wasi.Result{Transcript: baseline.Transcript()}
	after := wasi.Result{Transcript: variant.Transcript()}
	unmasked, err := projectVariantEvidence(after, config, "")
	if err != nil {
		t.Fatal(err)
	}
	want, err := projectVariantEvidence(before, standaloneConfig(), "")
	if err != nil || reflect.DeepEqual(want, unmasked) {
		t.Fatal("unmasked derived environment change was lost", err)
	}
	want, err = projectVariantEvidence(before, standaloneConfig(), "200")
	if err != nil {
		t.Fatal(err)
	}
	got, err := projectVariantEvidence(after, config, "200")
	if err != nil || !reflect.DeepEqual(want, got) {
		t.Fatal("GC environment was not masked", err)
	}
	for _, replacement := range []struct {
		name   string
		offset int
		value  byte
	}{
		{"environment operation", 8, 3},
		{"environment errno", 10, 1},
		{"argument operation", 40, 2},
		{"call id", 44, 9},
	} {
		changed := after
		changed.Transcript = bytes.Clone(after.Transcript)
		changed.Transcript[replacement.offset] = replacement.value
		projection, err := projectVariantEvidence(changed, config, "200")
		if err != nil || reflect.DeepEqual(want, projection) {
			t.Fatalf("mask hid non-environment change: %+v", replacement)
		}
	}
}

func TestDiagnosticVariantEvidencePreservesClockRuns(t *testing.T) {
	transcript := []byte("GMWT\x01\x00\x00\x00\x21\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x1c\x00\x00\x00\x01\x00\x00\x00\x07\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x03\x00\x00\x00\x00\x00\x00\x00")
	for _, mask := range []bool{false, true} {
		gcVariant := ""
		if mask {
			gcVariant = "200"
		}
		want, err := projectVariantEvidence(wasi.Result{Transcript: transcript}, standaloneConfig(), gcVariant)
		if err != nil {
			t.Fatal(err)
		}
		for _, change := range []struct {
			name   string
			offset int
			value  byte
		}{
			{"id", 12, 2}, {"clock id", 24, 0}, {"precision", 28, 6}, {"position", 36, 1}, {"count", 44, 4},
		} {
			t.Run(change.name+"/mask="+strconv.FormatBool(mask), func(t *testing.T) {
				data := bytes.Clone(transcript)
				data[change.offset] = change.value
				got, err := projectVariantEvidence(wasi.Result{Transcript: data}, standaloneConfig(), gcVariant)
				if err != nil || reflect.DeepEqual(want, got) {
					t.Fatal("clock evidence change was hidden", err)
				}
			})
		}
	}
}

func TestDiagnosticVariantEvidencePreservesInputsAndFailures(t *testing.T) {
	for _, test := range []struct {
		name         string
		tag          byte
		input        string
		inputOffsets []int
	}{
		{"random_get", 7, "\x02\x00\x00\x00\x04\x00\x00\x00\x00\x00\x00\x00", []int{0, 4}},
		{"fd_write", 8, "\x01\x00\x00\x00\x02\x00\x00\x00ab", []int{0, 9}},
		{"unsupported", 34, "\x03\x00\x00\x00bad\x0b\x00\x00\x00" + `{"value":1}`, []int{2 + 4, 20}},
		{"environ_get", 6, "", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			transcript := []byte("GMWT\x01\x00\x00\x00" + operationCountRecord(1, test.tag, test.input+"\x07\x00\x00\x00refused"))
			transcript[9] = 1
			want, err := projectVariantEvidence(wasi.Result{Transcript: transcript}, standaloneConfig(), "200")
			if err != nil {
				t.Fatal(err)
			}
			for _, offset := range []int{10, len(transcript) - 1} {
				data := bytes.Clone(transcript)
				data[offset] ^= 1
				got, err := projectVariantEvidence(wasi.Result{Transcript: data}, standaloneConfig(), "200")
				if err != nil || reflect.DeepEqual(want, got) {
					t.Fatal("failure evidence was hidden", err)
				}
			}
			for _, offset := range test.inputOffsets {
				data := bytes.Clone(transcript)
				data[8+16+offset] ^= 1
				got, err := projectVariantEvidence(wasi.Result{Transcript: data}, standaloneConfig(), "200")
				if err != nil || reflect.DeepEqual(want, got) {
					t.Fatal("operation input was hidden", err)
				}
			}
		})
	}
}

func TestDiagnosticVariantEvidenceRefusesMalformedBinary(t *testing.T) {
	valid := []byte("GMWT\x01\x00\x00\x00\x06\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
	unknown := bytes.Clone(valid)
	unknown[8] = 255
	oversized := bytes.Clone(valid)
	binary.LittleEndian.PutUint32(oversized[20:], 8<<20)
	for _, transcript := range [][]byte{
		[]byte("{\"type\":\"call\",\"id\":1,\"op\":\"environ_get\",\"input\":{}}\n{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{}}\n"),
		valid[:len(valid)-1], append(bytes.Clone(valid), 0), unknown, oversized,
	} {
		for _, mask := range []bool{false, true} {
			gcVariant := ""
			if mask {
				gcVariant = "200"
			}
			if _, err := projectVariantEvidence(wasi.Result{Transcript: transcript}, standaloneConfig(), gcVariant); err == nil {
				t.Fatalf("malformed binary transcript accepted with mask=%v: %x", mask, transcript)
			}
		}
	}
}

func TestDiagnosticVariantEvidenceSeparatesIdentities(t *testing.T) {
	code := uint32(0)
	before := wasi.Result{Termination: "exit", ExitCode: &code, Stdout: []byte("out"), Stderr: []byte("err"), Transcript: []byte{}, NetworkTranscript: []byte("network"), EnvironmentIdentity: "baseline", VirtualTimeElapsedNanos: 12}
	before.ChoiceTrace.Bytes = []byte("choice")
	after := before
	after.EnvironmentIdentity, after.FuelRemaining = "variant", 99
	baseRequest := wasi.Request{ModuleSHA256: "base-module", HelperSHA256: "base-helper", Engine: wasi.PinnedEngine()}
	variantRequest := wasi.Request{ModuleSHA256: "variant-module", HelperSHA256: "variant-helper", Engine: wasi.PinnedEngine()}
	variantRequest.SetDiagnosticConfigurationRevision(wasi.PinnedEngine().Configuration + "-fuel-off")
	want, err := projectVariantEvidence(before, standaloneConfig(), "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := projectVariantEvidence(after, standaloneConfig(), "")
	if err != nil || !reflect.DeepEqual(want, got) {
		t.Fatal("identity or fuel accounting entered behavior equality", err)
	}
	if reflect.DeepEqual(variantIdentities(baseRequest, before), variantIdentities(variantRequest, after)) {
		t.Fatal("variant identities were not reported separately")
	}
	for _, field := range []string{"module_sha256", "helper_sha256", "environment_identity", "engine_revision"} {
		t.Run(field, func(t *testing.T) {
			request, result := baseRequest, before
			switch field {
			case "module_sha256":
				request.ModuleSHA256 = "other-module"
			case "helper_sha256":
				request.HelperSHA256 = "other-helper"
			case "environment_identity":
				result.EnvironmentIdentity = "other-environment"
			case "engine_revision":
				request.SetDiagnosticConfigurationRevision(wasi.PinnedEngine().Configuration + "-fuel-off")
			}
			if reflect.DeepEqual(variantIdentities(baseRequest, before), variantIdentities(request, result)) {
				t.Fatal("changed identity was omitted")
			}
			projection, err := projectVariantEvidence(result, standaloneConfig(), "")
			if err != nil || !reflect.DeepEqual(want, projection) {
				t.Fatal("identity change altered behavior equality", err)
			}
		})
	}
	for name, mutate := range map[string]func(*wasi.Result){
		"choice":  func(r *wasi.Result) { r.ChoiceTrace.Bytes = []byte("other") },
		"wasi":    func(r *wasi.Result) { r.Transcript = []byte("other") },
		"network": func(r *wasi.Result) { r.NetworkTranscript = []byte("other") },
		"stdout":  func(r *wasi.Result) { r.Stdout = []byte("other") },
		"stderr":  func(r *wasi.Result) { r.Stderr = []byte("other") },
		"outcome": func(r *wasi.Result) { r.Termination = "trap" },
		"message": func(r *wasi.Result) { r.Message = "boundary failure" },
		"exit":    func(r *wasi.Result) { other := uint32(1); r.ExitCode = &other },
		"time":    func(r *wasi.Result) { r.VirtualTimeElapsedNanos++ },
	} {
		t.Run(name, func(t *testing.T) {
			changed := before
			mutate(&changed)
			projection, err := projectVariantEvidence(changed, standaloneConfig(), "")
			if err == nil && reflect.DeepEqual(want, projection) {
				t.Fatal("behavior change was hidden")
			}
		})
	}
}

func TestDiagnosticVariantEvidencePreservesDerivedInputs(t *testing.T) {
	for _, gcVariant := range []string{"", "200", "off"} {
		t.Run("gc="+gcVariant, func(t *testing.T) {
			want, err := projectVariantEvidence(wasi.Result{}, standaloneConfig(), gcVariant)
			if err != nil {
				t.Fatal(err)
			}
			for name, mutate := range map[string]func(*wasi.Config){
				"argv":        func(c *wasi.Config) { c.Args = append(c.Args, "other") },
				"environment": func(c *wasi.Config) { c.Environment = append(c.Environment, "USER_INPUT=other") },
				"entropy":     func(c *wasi.Config) { c.EntropyKey[0]++ },
				"clock step":  func(c *wasi.Config) { c.Clock.ReadStepNanos++ },
				"clock epoch": func(c *wasi.Config) { c.Clock.EpochNanos++ },
			} {
				t.Run(name, func(t *testing.T) {
					config := standaloneConfig()
					mutate(&config)
					got, err := projectVariantEvidence(wasi.Result{}, config, gcVariant)
					if err != nil || reflect.DeepEqual(want, got) {
						t.Fatal("derived input change was hidden", err)
					}
				})
			}
		})
	}
}

func TestDiagnosticVariantEvidenceMasksOnlyDeclaredGCInputs(t *testing.T) {
	for _, gcVariant := range []string{"200", "400", "off"} {
		t.Run(gcVariant, func(t *testing.T) {
			want, err := projectVariantEvidence(wasi.Result{}, standaloneConfig(), gcVariant)
			if err != nil {
				t.Fatal(err)
			}
			config := standaloneConfig()
			config.Environment = append(config.Environment, "GOGC="+gcVariant)
			if gcVariant == "off" {
				config.Environment = append(config.Environment, "GOMEMLIMIT=100B")
			}
			got, err := projectVariantEvidence(wasi.Result{}, config, gcVariant)
			if err != nil || !reflect.DeepEqual(want, got) {
				t.Fatal("declared GC policy change was not masked", err)
			}
			if gcVariant != "off" {
				config.Environment = append(config.Environment, "GOMEMLIMIT=100B")
				got, err := projectVariantEvidence(wasi.Result{}, config, gcVariant)
				if err != nil || reflect.DeepEqual(want, got) {
					t.Fatal("unexpected memory policy change was hidden", err)
				}
			}
		})
	}
}

func TestDiagnosticVariantRecommendationRequiresReliableGain(t *testing.T) {
	for _, test := range []struct {
		name                string
		baseline, candidate []uint64
		failed, identical   bool
		want                string
	}{
		{"gain", []uint64{98, 99, 100, 101, 102}, []uint64{78, 79, 80, 81, 82}, false, true, "adopt"},
		{"spread", []uint64{90, 99, 100, 101, 110}, []uint64{80, 89, 90, 91, 100}, false, true, "investigate"},
		{"slower", []uint64{98, 99, 100, 101, 102}, []uint64{108, 109, 110, 111, 112}, false, true, "reject"},
		{"mismatch", []uint64{98, 99, 100, 101, 102}, []uint64{78, 79, 80, 81, 82}, false, false, "reject"},
		{"failed", []uint64{98, 99, 100, 101, 102}, nil, true, false, "reject"},
		{"missing", []uint64{100}, nil, false, true, "investigate"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := variantRecommendation(test.baseline, test.candidate, test.failed, test.identical); got != test.want {
				t.Fatalf("recommendation=%s want=%s", got, test.want)
			}
		})
	}
}

func TestTerminalExitStagesHardlinkedCargoHelper(t *testing.T) {
	root := t.TempDir()
	artifact, alias := filepath.Join(root, "cargo-helper"), filepath.Join(root, "cargo-deps-alias")
	data := []byte("#!/bin/sh\nexit 0\n")
	if err := os.WriteFile(artifact, data, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(artifact, alias); err != nil {
		t.Fatal(err)
	}
	if file, _, err := hostfs.OpenPath(artifact); err == nil {
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		t.Fatal("hardlinked compiler output unexpectedly admitted")
	}
	helper := filepath.Join(root, "admitted-helper")
	testTerminalExitStageHelper(t, artifact, helper, testFileHash(t, artifact))
	file, info, err := hostfs.OpenPath(helper)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 || !bytes.Equal(testMustTerminalExitRead(t, helper), data) || !bytes.Equal(testMustTerminalExitRead(t, artifact), data) {
		t.Fatal("staging changed helper bytes, executable mode, or Cargo artifact")
	}
}

func TestTerminalExitCompilerEnvironmentControls(t *testing.T) {
	for _, test := range []struct {
		name        string
		environment []string
		want        bool
	}{
		{"pinned", []string{"RUSTC=/pinned/rustc", "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER="}, true},
		{"missing-compiler", []string{"RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER="}, false},
		{"wrong-compiler", []string{"RUSTC=/other/rustc", "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER="}, false},
		{"conflicting-compiler", []string{"RUSTC=/pinned/rustc", "RUSTC=/other/rustc", "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER="}, false},
		{"missing-wrapper-override", []string{"RUSTC=/pinned/rustc", "RUSTC_WORKSPACE_WRAPPER="}, false},
		{"compiler-wrapper", []string{"RUSTC=/pinned/rustc", "RUSTC_WRAPPER=/wrapper", "RUSTC_WORKSPACE_WRAPPER="}, false},
		{"workspace-wrapper", []string{"RUSTC=/pinned/rustc", "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER=/wrapper"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			build := terminalExitHelperBuild{RustcPath: "/pinned/rustc", Environment: test.environment, Inputs: map[string]string{"/pinned/rustc": strings.Repeat("a", 64)}}
			if got := terminalExitCompilerEnvironment(build); got != test.want {
				t.Fatalf("compiler environment admitted=%v, want %v", got, test.want)
			}
		})
	}
}

func terminalExitCargoHelper(data []byte) (string, error) {
	return terminalExitCargoHelperFeatures(data, []string{})
}

func TestTerminalExitCargoArtifactControls(t *testing.T) {
	for _, test := range []struct {
		name, output string
		want         bool
	}{
		{"release", `{"reason":"compiler-artifact","target":{"name":"gomad-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":[]} {"reason":"build-finished","success":true}`, true},
		{"debug-build", `{"reason":"compiler-artifact","target":{"name":"gomad-wasmhost"},"profile":{"opt_level":"0","debug_assertions":true},"executable":"/release/helper","features":[]} {"reason":"build-finished","success":true}`, false},
		{"diagnostic-feature", `{"reason":"compiler-artifact","target":{"name":"gomad-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":["diagnostic-profile"]} {"reason":"build-finished","success":true}`, false},
		{"missing-features", `{"reason":"compiler-artifact","target":{"name":"gomad-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper"} {"reason":"build-finished","success":true}`, false},
		{"failed-build", `{"reason":"compiler-artifact","target":{"name":"gomad-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":[]} {"reason":"build-finished","success":false}`, false},
		{"missing-finish", `{"reason":"compiler-artifact","target":{"name":"gomad-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":[]}`, false},
		{"missing-helper", `{"reason":"build-finished","success":true}`, false},
		{"malformed", `{`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper, err := terminalExitCargoHelper([]byte(test.output))
			if (err == nil) != test.want || test.want && helper != "/release/helper" {
				t.Fatalf("Cargo artifact = %q, %v", helper, err)
			}
		})
	}
}

func TestDiagnosticModuleSourceBindings(t *testing.T) {
	root := t.TempDir()
	compiler := filepath.Join(root, "stock/bin/go")
	overlay := filepath.Join(root, "overlay.json")
	paths := []string{compiler, filepath.Join(root, "stock/VERSION"), filepath.Join(root, "go.mod"), filepath.Join(root, "go.sum"), filepath.Join(root, "fixture.go"), filepath.Join(root, "fixture.h"), filepath.Join(root, "fixture.syso"), filepath.Join(root, "replacement.go"), filepath.Join(root, "added-replacement.go"), filepath.Join(root, "embedded.txt")}
	for _, tool := range []string{"compile", "link", "asm"} {
		paths = append(paths, filepath.Join(root, "stock/pkg/tool", runtime.GOOS+"_"+runtime.GOARCH, tool))
	}
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		testTemporalWrite(t, path, []byte("original"))
	}
	// added.go exists only in the overlay, as the cooperative runtime/net glue does.
	added := filepath.Join(root, "added.go")
	testTemporalJSON(t, overlay, struct{ Replace map[string]string }{map[string]string{filepath.Join(root, "fixture.go"): filepath.Join(root, "replacement.go"), added: filepath.Join(root, "added-replacement.go")}})
	listed, err := json.Marshal(struct {
		Dir                                        string
		GoFiles, HFiles, SysoFiles, TestEmbedFiles []string
	}{root, []string{"fixture.go", "added.go"}, []string{"fixture.h"}, []string{"fixture.syso"}, []string{"embedded.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	inputs := testDiagnosticModuleSourceInputs(t, listed, compiler, overlay, root)
	if _, ok := inputs[added]; ok {
		t.Fatal("overlay-only file bound as an on-disk input")
	}
	paths = append(paths, overlay)
	for _, path := range paths {
		if inputs[path] != testFileHash(t, path) {
			t.Fatalf("producer omitted %s", path)
		}
		original := testMustTerminalExitRead(t, path)
		testTemporalWrite(t, path, []byte("changed"))
		if err := diagnosticVerifyInputs(inputs); err == nil {
			t.Fatalf("changed compiler/source/tool admitted: %s", path)
		}
		testTemporalWrite(t, path, original)
	}
}

func TestDiagnosticRecoveryCompilerEnvironment(t *testing.T) {
	t.Setenv("GOMAXPROCS", "1")
	t.Setenv("GOTMPDIR", "/private/compiler-temp")
	environment, err := temporalBuildEnvironment(measurementTestContext(t), "/pinned/bin/go")
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"GOMAXPROCS=1", "GOTMPDIR=/private/compiler-temp", "GOENV=off", "GOTOOLCHAIN=local", "GOWORK=off", "GOFLAGS="} {
		if !slices.Contains(environment, expected) {
			t.Fatalf("compiler environment lost %s", expected)
		}
	}
}

func TestDiagnosticAdmissionInputs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "input")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	inputs := map[string]string{path: testFileHash(t, path)}
	if err := diagnosticVerifyInputs(inputs); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := diagnosticVerifyInputs(inputs); err == nil {
		t.Fatal("changed input admitted")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := diagnosticVerifyInputs(inputs); err == nil {
		t.Fatal("missing input admitted")
	}
}

func TestDiagnosticClusterPairAdmission(t *testing.T) {
	if root := os.Getenv("GOMAD_DIAGNOSTIC_PAIR_CHILD"); root != "" {
		phase, drift := os.Getenv("GOMAD_DIAGNOSTIC_PAIR_PHASE"), os.Getenv("GOMAD_DIAGNOSTIC_PAIR_DRIFT")
		input, memory := filepath.Join(root, "module"), filepath.Join(root, "meminfo")
		testTemporalWrite(t, input, []byte("admitted"))
		testTemporalWrite(t, memory, []byte("MemAvailable: 8388608 kB\n"))
		inputs := map[string]string{input: testFileHash(t, input)}
		if drift == "before" {
			testTemporalWrite(t, input, []byte("changed"))
		}
		if drift == "resource" {
			testTemporalWrite(t, memory, []byte("MemAvailable: 8388607 kB\n"))
		}
		observer := temporalPairObserver{
			Admission: func(_ context.Context, evidence, label string, requests []wasi.Request, before bool) error {
				if evidence != root || label != phase || len(requests) != 2 {
					t.Fatal("pair boundary lost")
				}
				if err := diagnosticVerifyInputs(inputs); err != nil {
					return err
				}
				if before {
					if _, err := matchedPrepareAdmitted("linux", memory, "1", func() (wasi.Result, error) { return wasi.Result{}, nil }); err != nil {
						return err
					}
				}
				return nil
			},
			Observe: func(_ context.Context, evidence, label string, requests []wasi.Request, results []wasi.Result, failures []error) error {
				if len(results) != 2 || len(failures) != 2 {
					t.Fatal("pair evidence lost")
				}
				testTemporalWrite(t, filepath.Join(root, "observed"), []byte(label))
				return nil
			},
		}
		testTemporalObservedPair(t, root, phase, []wasi.Request{{}, {}}, []temporalPairObserver{observer}, func() ([]wasi.Result, []error) {
			testTemporalWrite(t, filepath.Join(root, "dispatched"), []byte(phase))
			if drift == "after" {
				testTemporalWrite(t, input, []byte("changed"))
			}
			return []wasi.Result{{}, {}}, []error{nil, nil}
		})
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"cluster-crash", "cluster-reopened", "cluster-prefix-replay", "cluster-full-replay"} {
		for _, drift := range []string{"none", "before", "after", "resource"} {
			t.Run(phase+"/"+drift, func(t *testing.T) {
				root := t.TempDir()
				ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
				defer cancel()
				child := exec.CommandContext(ctx, executable, "-test.run=^TestDiagnosticClusterPairAdmission$")
				child.Env = append(os.Environ(), "GOMAD_DIAGNOSTIC_PAIR_CHILD="+root, "GOMAD_DIAGNOSTIC_PAIR_PHASE="+phase, "GOMAD_DIAGNOSTIC_PAIR_DRIFT="+drift)
				output, err := child.CombinedOutput()
				if (err == nil) != (drift == "none") {
					t.Fatalf("phase admission: %v\n%s", err, output)
				}
				if ctx.Err() != nil {
					t.Fatal("controlled pair admission timed out", ctx.Err())
				}
				if drift != "none" {
					reason := "diagnostic input changed"
					if drift == "resource" {
						reason = "at least 8GiB MemAvailable"
					}
					if !strings.Contains(string(output), reason) {
						t.Fatalf("pair failed without admission rejection: %s", output)
					}
				}
				_, dispatchErr := os.Stat(filepath.Join(root, "dispatched"))
				if (dispatchErr == nil) != (drift == "none" || drift == "after") {
					t.Fatal("unadmitted pair dispatched")
				}
				_, observedErr := os.Stat(filepath.Join(root, "observed"))
				if (observedErr == nil) != (drift == "none") {
					t.Fatal("unadmitted pair published measurements")
				}
			})
		}
	}
}

func TestDiagnosticAdmissionModuleBindings(t *testing.T) {
	request := wasi.Request{ModulePath: "/module", ModuleSHA256: "module-sha", Runtime: &wasi.RuntimeControl{Choice: &runnerbackend.ChoiceRequest{}}}
	request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey = "runtime-key"
	module := diagnosticModuleAdmission{Schema: "gomad-diagnostic-module-build/v1", Module: request.ModulePath, ModuleSHA256: request.ModuleSHA256, RuntimeKey: "runtime-key", RuntimeSHA256: toolchain.ImplementationSHA256(), ModelSHA256: wasi.ImplementationSHA256()}
	if !diagnosticModuleMatches(module, request) {
		t.Fatal("matching module rejected")
	}
	for _, name := range []string{"schema", "module", "digest", "runtime-key", "runtime-source", "model"} {
		t.Run(name, func(t *testing.T) {
			changed := module
			switch name {
			case "schema":
				changed.Schema = "gomad-wasm.provenance/v1"
			case "module":
				changed.Module = "/other"
			case "digest":
				changed.ModuleSHA256 = "stale"
			case "runtime-key":
				changed.RuntimeKey = "stale"
			case "runtime-source":
				changed.RuntimeSHA256 = "stale"
			case "model":
				changed.ModelSHA256 = "stale"
			}
			if diagnosticModuleMatches(changed, request) {
				t.Fatal("stale module admitted")
			}
		})
	}
	request.Runtime = nil
	if diagnosticModuleMatches(module, request) {
		t.Fatal("missing runtime admitted")
	}
}

func TestDiagnosticAdmissionHelperEvidence(t *testing.T) {
	if root := os.Getenv("GOMAD_DIAGNOSTIC_ADMISSION_CHILD"); root != "" {
		a := &diagnosticAdmission{Repo: root, Helpers: map[string]diagnosticHelperAdmission{}, Inputs: map[string]string{}}
		features := []string{"diagnostic-profile", "guest-profile"}
		if os.Getenv("GOMAD_DIAGNOSTIC_ADMISSION_CLUSTER") == "1" {
			features = []string{"diagnostic-profile"}
		}
		testAddHelper(a, t, filepath.Join(root, "admitted-helper"), filepath.Join(root, "helper-admission.json"), features)
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []string{"valid", "cluster-valid", "cluster-extra-feature", "artifact", "helper", "source", "compiler", "receipt", "wrapper", "wrong-compiler", "old-toolchain", "missing-feature", "extra-feature", "raw-command"} {
		t.Run(mutation, func(t *testing.T) {
			root := t.TempDir()
			crate := filepath.Join(root, "tools/gomad/internal/backends/wasm/wasmhost")
			if err := os.MkdirAll(filepath.Join(crate, "src"), 0700); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"src/main.rs", "Cargo.toml", "Cargo.lock", "rust-toolchain.toml", "../Makefile"} {
				testTemporalWrite(t, filepath.Join(crate, name), []byte(name))
			}
			rustc, cargo := filepath.Join(root, "rustc"), filepath.Join(root, "cargo")
			artifact, helper := filepath.Join(root, "artifact"), filepath.Join(root, "admitted-helper")
			for _, path := range []string{rustc, cargo, artifact, helper} {
				if err := os.WriteFile(path, []byte("fixture; never executed"), 0755); err != nil {
					t.Fatal(err)
				}
			}
			features := []string{"diagnostic-profile", "guest-profile"}
			if mutation == "cluster-valid" {
				features = []string{"diagnostic-profile"}
			}
			command := []string{cargo, "build", "--manifest-path", filepath.Join(crate, "Cargo.toml"), "--release", "--locked", "--jobs=1", "--no-default-features", "--message-format=json", "--features", strings.Join(features, ",")}
			encoded, err := json.Marshal(artifact)
			if err != nil {
				t.Fatal(err)
			}
			encodedFeatures, err := json.Marshal(features)
			if err != nil {
				t.Fatal(err)
			}
			testTemporalWrite(t, filepath.Join(root, "cargo-build.stdout"), []byte(`{"reason":"compiler-artifact","target":{"name":"gomad-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":`+string(encoded)+`,"features":`+string(encodedFeatures)+`} {"reason":"build-finished","success":true}`))
			rawCommand := slices.Clone(command)
			if mutation == "raw-command" {
				rawCommand[6] = "--jobs=8"
			}
			testTemporalJSON(t, filepath.Join(root, "cargo-build.json"), struct {
				Command  []string
				ExitCode int
			}{rawCommand, 0})
			rustcVersion, cargoVersion := "rustc 1.99.1 fixture", "cargo 1.99.1 fixture"
			if mutation == "old-toolchain" {
				rustcVersion, cargoVersion = "rustc 1.94.1 fixture", "cargo 1.94.1 fixture"
			}
			testTemporalWrite(t, filepath.Join(root, "rustc-version.stdout"), []byte(rustcVersion))
			testTemporalWrite(t, filepath.Join(root, "cargo-version.stdout"), []byte(cargoVersion))
			inputs := testTerminalExitRustInputs(t, crate)
			for _, path := range []string{rustc, cargo, filepath.Join(root, "cargo-build.stdout"), filepath.Join(root, "cargo-build.json"), filepath.Join(root, "rustc-version.stdout"), filepath.Join(root, "cargo-version.stdout")} {
				inputs[path] = testFileHash(t, path)
			}
			build := terminalExitHelperBuild{Crate: crate, Helper: artifact, HelperSHA256: testFileHash(t, artifact), RustcPath: rustc, Inputs: inputs, Command: command, Environment: []string{"RUSTC=" + rustc, "RUSTC_WRAPPER=", "RUSTC_WORKSPACE_WRAPPER="}, Features: features, RustcVersion: rustcVersion, CargoVersion: cargoVersion}
			switch mutation {
			case "wrapper":
				build.Environment[1] = "RUSTC_WRAPPER=/wrong"
			case "wrong-compiler":
				build.Environment[0] = "RUSTC=/wrong"
			case "missing-feature":
				build.Features = []string{"guest-profile"}
			case "extra-feature":
				build.Features = append(build.Features, "unexpected")
			}
			receipt := filepath.Join(root, "helper-build.json")
			testTemporalJSON(t, receipt, build)
			testTemporalJSON(t, filepath.Join(root, "helper-admission.json"), terminalExitHelperAdmission{BuildReceipt: receipt, BuildReceiptSHA256: testFileHash(t, receipt), CargoArtifact: artifact, CargoArtifactSHA256: testFileHash(t, artifact), Helper: helper, HelperSHA256: testFileHash(t, helper)})
			changed := map[string]string{"artifact": artifact, "helper": helper, "source": filepath.Join(crate, "src/main.rs"), "compiler": rustc, "receipt": receipt}[mutation]
			if changed != "" {
				testTemporalWrite(t, changed, []byte("tampered"))
			}
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			child := exec.CommandContext(ctx, executable, "-test.run=^TestDiagnosticAdmissionHelperEvidence$")
			child.Env = append(os.Environ(), "GOMAD_DIAGNOSTIC_ADMISSION_CHILD="+root)
			if strings.HasPrefix(mutation, "cluster-") {
				child.Env = append(child.Env, "GOMAD_DIAGNOSTIC_ADMISSION_CLUSTER=1")
			}
			output, err := child.CombinedOutput()
			if (err == nil) != (mutation == "valid" || mutation == "cluster-valid") {
				t.Fatalf("admission for %s: %v\n%s", mutation, err, output)
			}
		})
	}
}

func TestDiagnosticAdmissionCargoFeatures(t *testing.T) {
	for _, features := range [][]string{{"diagnostic-profile"}, {"diagnostic-profile", "guest-profile"}, {"diagnostic-fuel-off", "diagnostic-profile"}} {
		for _, test := range []struct {
			name     string
			actual   []string
			accepted bool
		}{
			{"exact", features, true},
			{"missing", features[:len(features)-1], false},
			{"missing-first", features[1:], false},
			{"extra", append(append([]string{}, features...), "unexpected"), false},
		} {
			t.Run(features[0]+"/"+test.name, func(t *testing.T) {
				encoded, err := json.Marshal(test.actual)
				if err != nil {
					t.Fatal(err)
				}
				data := []byte(`{"reason":"compiler-artifact","target":{"name":"gomad-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":` + string(encoded) + `} {"reason":"build-finished","success":true}`)
				_, err = terminalExitCargoHelperFeatures(data, features)
				if (err == nil) != test.accepted {
					t.Fatalf("features %v admitted=%t", test.actual, err == nil)
				}
				if _, err := terminalExitCargoHelper(data); err == nil && len(test.actual) != 0 {
					t.Fatal("release wrapper admitted diagnostic features")
				}
			})
		}
	}
}

func TestDiagnosticClusterPairProfiles(t *testing.T) {
	requests := []wasi.Request{{HelperSHA256: "helper", ModuleSHA256: "module"}, {HelperSHA256: "helper", ModuleSHA256: "module"}}
	exit := uint32(0)
	valid := bytes.Replace(diagnosticFixture(t), []byte(`"encoding":{"count":0,"wall_nanos":0}`), []byte(`"encoding":{"count":1,"wall_nanos":5}`), 1)
	for _, test := range []struct {
		name      string
		second    wasi.Result
		complete  bool
		measured  int
		wall      uint64
		wantError bool
	}{
		{"both measured", measurementResult(wasi.Result{Termination: "exit", ExitCode: &exit}, [16]byte{}, valid), true, 2, 200, false},
		{"killed helper unavailable", wasi.Result{Termination: "crash", Reaped: true}, false, 1, 100, false},
		{"partial killed report unavailable", measurementResult(wasi.Result{Termination: "crash", Reaped: true}, [16]byte{}, []byte("gomad-wasm-profile-v1 {")), false, 1, 100, false},
		{"successful helper missing profile", wasi.Result{Termination: "exit", ExitCode: &exit}, false, 1, 100, true},
		{"successful helper malformed profile", measurementResult(wasi.Result{Termination: "exit", ExitCode: &exit}, [16]byte{}, []byte("invalid\n")), false, 1, 100, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			results := []wasi.Result{measurementResult(wasi.Result{Termination: "exit", ExitCode: &exit}, [16]byte{}, valid), test.second}
			pair, err := decodeClusterDiagnosticPair(requests, results)
			if (err != nil) != test.wantError || pair.Complete != test.complete || pair.Measured != test.measured || pair.MeasuredSum.Phases["execute"].WallNanos != test.wall {
				t.Fatalf("pair=%+v err=%v", pair, err)
			}
			if pair.Instances[0].Profile == nil || pair.Instances[0].Profile.Phases["execute"].UserNanos != 70 || pair.Instances[0].Profile.ABI["clock"].ExecuteWallNanos != 30 || pair.Instances[0].Profile.Residual.ExecuteWallNanos != 70 {
				t.Fatal("per-helper buckets lost")
			}
			if pair.MeasuredSum.Phases["execute"].PeakRSSBytesHighWater != 1024 {
				t.Fatal("process high-water values were added as concurrent RSS")
			}
			if pair.MeasuredSum.Phases["execute"].UserNanos != uint64(test.measured)*70 || pair.MeasuredSum.Phases["execute"].SystemNanos != uint64(test.measured)*10 || pair.MeasuredSum.ABI["clock"].ExecuteWallNanos != uint64(test.measured)*30 || pair.MeasuredSum.Nested["encoding"].WallNanos != uint64(test.measured)*5 || pair.MeasuredSum.Residual.ExecuteWallNanos != uint64(test.measured)*70 {
				t.Fatal("measured phase, ABI, nested or residual buckets not summed")
			}
			if !test.complete && (pair.Instances[1].Profile != nil || pair.Instances[1].Unavailable == "") {
				t.Fatal("missing helper measurements presented as zero or available")
			}
		})
	}
}

func TestDiagnosticClusterPairRejectsOverflow(t *testing.T) {
	valid := diagnosticFixture(t)
	overflow := bytes.Replace(valid, []byte(`"count":1`), []byte(`"count":18446744073709551615`), 1)
	_, err := decodeClusterDiagnosticPair([]wasi.Request{{}, {}}, []wasi.Result{measurementResult(wasi.Result{}, [16]byte{}, overflow), measurementResult(wasi.Result{}, [16]byte{}, valid)})
	if err == nil {
		t.Fatal("overflowing summed helper counters accepted")
	}
}

// Env-gated drivers hash their own source; a moved owner file would otherwise
// fail them only when the real measurement is attempted.
func TestDiagnosticDriverSourcesExist(t *testing.T) {
	repo, err := filepath.Abs("../../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{filepath.Join(repo, warmPreparationDriverSource)}
	for _, name := range diagnosticClusterCollectorSources {
		path := filepath.Join(repo, "tools/gomad/internal/backends/wasm/wasi", name)
		paths = append(paths, path)
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Fatal("driver source binding names a missing file", err)
		}
	}
}

func TestWarmPreparationDarwinMemoryAdmission(t *testing.T) {
	for _, test := range []struct {
		name, stats string
		want        bool
	}{
		{"at-bound", "Mach Virtual Memory Statistics: (page size of 16384 bytes)\nPages free: 262144.\nPages inactive: 262144.\n", true},
		{"below-bound", "Mach Virtual Memory Statistics: (page size of 16384 bytes)\nPages free: 262143.\nPages inactive: 262144.\n", false},
		{"missing-inactive", "Mach Virtual Memory Statistics: (page size of 16384 bytes)\nPages free: 524288.\n", false},
		{"duplicate-free", "Mach Virtual Memory Statistics: (page size of 16384 bytes)\nPages free: 262144.\nPages free: 262144.\nPages inactive: 262144.\n", false},
		{"malformed-header", "16384\nPages free: 524288.\nPages inactive: 0.\n", false},
		{"invalid-page-size", "Mach Virtual Memory Statistics: (page size of 0 bytes)\nPages free: 524288.\nPages inactive: 0.\n", false},
		{"overflow", "Mach Virtual Memory Statistics: (page size of 16384 bytes)\nPages free: 18446744073709551615.\nPages inactive: 1.\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			called := false
			_, err := warmPreparationDarwinAdmitted(test.stats, func() (wasi.Result, error) {
				called = true
				return wasi.Result{Termination: "exit"}, nil
			})
			if (err == nil) != test.want || called != test.want {
				t.Fatalf("memory admission=%v dispatched=%t, want %t", err, called, test.want)
			}
		})
	}
}

func TestWarmPreparationCurrentHostEvidence(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("current host collector requires available procfs or sysctl boot/CPU evidence")
	}
	boot, cpuinfo, err := warmPreparationHostEvidence(t.Context())
	if err != nil || boot == "" || !warmPreparationHostCheck(wasi.PinnedEngine(), cpuinfo) {
		t.Fatal("current host evidence did not bind the engine", err)
	}
	changed := wasi.PinnedEngine()
	changed.HostArch = "unrelated-architecture"
	if warmPreparationHostCheck(changed, cpuinfo) {
		t.Fatal("actual host evidence admitted a different engine architecture")
	}
}

func TestWarmPreparationCurrentProducerClosureControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(map[string]string)
		want   bool
	}{
		{"same-owner-validated-producer", func(map[string]string) {}, true},
		{"missing-current-source", func(inputs map[string]string) { delete(inputs, "/current-source.go") }, false},
		{"changed-current-source", func(inputs map[string]string) { inputs["/current-source.go"] = strings.Repeat("f", 64) }, false},
		{"changed-current-producer", func(inputs map[string]string) { inputs["/module-receipt.json"] = strings.Repeat("f", 64) }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			series := warmPreparationFixture()
			series.Inputs["/current-source.go"] = strings.Repeat("b", 64)
			current := maps.Clone(series.Inputs)
			test.mutate(series.Inputs)
			if err := warmPreparationProducerCheck(series, series.ModuleReceipt, current); (err == nil) != test.want {
				t.Fatalf("current producer closure=%v want %v", err, test.want)
			}
		})
	}
}

func TestWarmPreparationCollectionControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*warmPreparationSeries, *warmPreparationCollection, map[string][]byte)
		want   bool
	}{
		{"successful-original-run", func(*warmPreparationSeries, *warmPreparationCollection, map[string][]byte) {}, true},
		{"collection-failed-five-good-samples", func(_ *warmPreparationSeries, c *warmPreparationCollection, _ map[string][]byte) {
			c.CollectionValid = false
		}, false},
		{"reported-failure", func(_ *warmPreparationSeries, c *warmPreparationCollection, _ map[string][]byte) {
			c.Failure = "warmup changed"
		}, false},
		{"original-run-error", func(_ *warmPreparationSeries, c *warmPreparationCollection, _ map[string][]byte) {
			c.UntimedRunError = "original Run failed"
		}, false},
		{"changed-source-binding", func(s *warmPreparationSeries, _ *warmPreparationCollection, _ map[string][]byte) {
			s.SourceCommit = strings.Repeat("b", 40)
		}, false},
		{"changed-sample-binding", func(s *warmPreparationSeries, _ *warmPreparationCollection, _ map[string][]byte) {
			s.Runs[1].WallNanos++
		}, false},
		{"dropped-warmup", func(s *warmPreparationSeries, _ *warmPreparationCollection, _ map[string][]byte) { s.Warmup = nil }, false},
		{"dropped-marker", func(s *warmPreparationSeries, _ *warmPreparationCollection, files map[string][]byte) {
			delete(files, s.CollectionPath)
		}, false},
		{"unbound-marker", func(s *warmPreparationSeries, _ *warmPreparationCollection, _ map[string][]byte) {
			delete(s.Inputs, s.CollectionPath)
		}, false},
		{"dropped-canonical", func(_ *warmPreparationSeries, c *warmPreparationCollection, files map[string][]byte) {
			delete(files, c.CanonicalPath)
		}, false},
		{"tampered-canonical", func(_ *warmPreparationSeries, c *warmPreparationCollection, files map[string][]byte) {
			files[c.CanonicalPath] = []byte("tampered")
		}, false},
		{"rehashed-wrong-canonical", func(s *warmPreparationSeries, c *warmPreparationCollection, files map[string][]byte) {
			files[c.CanonicalPath] = []byte("tampered")
			s.Inputs[c.CanonicalPath] = digest(files[c.CanonicalPath])
		}, false},
		{"rehashed-wrong-finish", func(s *warmPreparationSeries, c *warmPreparationCollection, files map[string][]byte) {
			files[c.FinishPath][0]++
			s.Inputs[c.FinishPath] = digest(files[c.FinishPath])
		}, false},
		{"dropped-original-observation", func(s *warmPreparationSeries, _ *warmPreparationCollection, files map[string][]byte) {
			delete(files, s.Runs[0].RawObservationPath)
		}, false},
		{"tampered-original-observation", func(s *warmPreparationSeries, _ *warmPreparationCollection, files map[string][]byte) {
			files[s.Runs[0].RawObservationPath] = []byte("{}")
		}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			series := warmPreparationFixture()
			series.Role, series.CollectionPath = "baseline", "/collection.json"
			code := uint32(0)
			result := measurementResult(wasi.Result{Termination: "exit", ExitCode: &code, Reaped: true, Started: &wasi.Started{Engine: series.Engine, ModuleSHA256: series.ModuleSHA256}, Transcript: []byte("canonical"), TranscriptSHA256: sha256.Sum256([]byte("canonical"))}, [16]byte{1}, nil)
			result.ChoiceTrace.Bytes = []byte("choices")
			files := map[string][]byte{}
			put := func(path string, data []byte) { files[path] = data; series.Inputs[path] = digest(data) }
			encode := func(value any) []byte {
				data, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				return data
			}
			for i := range series.Runs {
				run := &series.Runs[i]
				run.EvidenceSHA256 = warmPreparationEvidence(result)
				run.RawObservationPath, run.ObservationPath = fmt.Sprintf("/run-%d-raw.json", i), fmt.Sprintf("/run-%d-projected.json", i)
				run.FinishPath = fmt.Sprintf("/run-%d-finish", i)
				observation := terminalExitObservation{TerminalReceivedAt: time.Unix(1, 0), SupervisorReturnedAt: time.Unix(2, 0), TerminalToReapNanos: uint64(time.Second), Result: result}
				observation.Host.Termination, observation.Host.GroupGone = "exit", true
				put(run.RawObservationPath, encode(observation))
				put(run.ObservationPath, encode(warmPreparationObserve(observation)))
				put(run.FinishPath, bytes.Clone(measurementTerminalBytes(result)[:]))
			}
			warmup := series.Runs[0]
			series.Warmup = &warmup
			collection := warmPreparationCollection{Schema: "gomad.wasm-warm-preparation-collection/v1", CollectionValid: true, BindingSHA256: warmPreparationCollectionBinding(series), UntimedEvidenceSHA256: warmPreparationEvidence(result), ResultPath: "/untimed-result.json", CanonicalPath: "/untimed-canonical", ChoicePath: "/untimed-choice", FinishPath: "/untimed-finish", SummaryNanos: warmPreparationSummary(series)}
			put(collection.ResultPath, encode(result))
			put(collection.CanonicalPath, result.Transcript)
			put(collection.ChoicePath, result.ChoiceTrace.Bytes)
			put(collection.FinishPath, bytes.Clone(measurementTerminalBytes(result)[:]))
			test.mutate(&series, &collection, files)
			marker := encode(collection)
			if test.name != "dropped-marker" {
				files[series.CollectionPath] = marker
			}
			if test.name != "unbound-marker" {
				series.Inputs[series.CollectionPath] = digest(marker)
			}
			err := warmPreparationCollectionCheck(series, func(path string) ([]byte, error) {
				data, ok := files[path]
				if !ok {
					return nil, os.ErrNotExist
				}
				return data, nil
			})
			if (err == nil) != test.want {
				t.Fatalf("collection admission=%v want %v", err, test.want)
			}
		})
	}
}

func TestWarmPreparationAdmissionControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*warmPreparationSeries)
		want   bool
	}{
		{"five-warm-at-bound", func(*warmPreparationSeries) {}, true},
		{"linux-arm64", func(s *warmPreparationSeries) {
			s.Engine.HostOS, s.Engine.HostArch = "linux", "aarch64"
			s.HostCPUInfo = "Features: fp asimd\nCPU architecture: 8\n"
		}, true},
		{"linux-amd64", func(s *warmPreparationSeries) {
			s.Engine.HostOS, s.Engine.HostArch = "linux", "x86_64"
			s.HostCPUInfo = `{"KernelOS":"Linux","Machine":"x86_64","KernelVersion":"Linux fixture","CPUInfo":"vendor_id: GenuineIntel\nflags: fpu sse sse2\n","Translation":"0"}`
		}, true},
		{"darwin-amd64", func(s *warmPreparationSeries) {
			s.Engine.HostArch = "x86_64"
			s.HostCPUInfo = `{"KernelOS":"Darwin","Machine":"x86_64","KernelVersion":"Darwin fixture","CPUInfo":"hw.optional.arm64: 0\nmachdep.cpu.brand_string: Intel Core i7","Translation":"0"}`
		}, true},
		{"other-os-and-architecture", func(s *warmPreparationSeries) {
			s.Engine.HostOS, s.Engine.HostArch = "freebsd", "riscv64"
			s.HostCPUInfo = `{"KernelOS":"FreeBSD","Machine":"riscv64","KernelVersion":"FreeBSD fixture","CPUInfo":"riscv64","Translation":"0"}`
		}, true},
		{"emulated-linux-host", func(s *warmPreparationSeries) {
			s.Engine.HostOS, s.Engine.HostArch = "linux", "aarch64"
			s.HostCPUInfo = `{"KernelOS":"Linux","Machine":"aarch64","KernelVersion":"Linux fixture","CPUInfo":"vendor_id: GenuineIntel\nflags: fpu sse sse2\n","Translation":"0"}`
		}, false},
		{"translated-host", func(s *warmPreparationSeries) {
			s.HostCPUInfo = `{"KernelOS":"Darwin","Machine":"arm64","KernelVersion":"Darwin fixture","CPUInfo":"hw.optional.arm64: 1\nmachdep.cpu.brand_string: Apple M2","Translation":"1"}`
		}, false},
		{"wrong-engine-os", func(s *warmPreparationSeries) { s.Engine.HostOS = "linux" }, false},
		{"wrong-engine-arch", func(s *warmPreparationSeries) { s.Engine.HostArch = "x86_64" }, false},
		{"missing-cpu-evidence", func(s *warmPreparationSeries) { s.HostCPUInfo = "" }, false},
		{"invented-cpu-evidence", func(s *warmPreparationSeries) { s.HostCPUInfo = "host" }, false},
		{"oversized-cpu-evidence", func(s *warmPreparationSeries) { s.HostCPUInfo = strings.Repeat("x", diagnosticHostCPUInfoLimit+1) }, false},
		{"four-runs", func(s *warmPreparationSeries) { s.Runs = s.Runs[:4] }, false},
		{"cold-run", func(s *warmPreparationSeries) { s.Runs[2].Telemetry.CacheHit = false }, false},
		{"compiled-run", func(s *warmPreparationSeries) { s.Runs[2].Telemetry.CompileNanos = 1 }, false},
		{"prepare-over-bound", func(s *warmPreparationSeries) {
			for i := range s.Runs {
				s.Runs[i].Telemetry.PrepareNanos++
			}
		}, false},
		{"one-slow-prepare-median-still-bound", func(s *warmPreparationSeries) { s.Runs[0].Telemetry.PrepareNanos++ }, true},
		{"slow-terminal", func(s *warmPreparationSeries) { s.Runs[2].TerminalToReapNanos++ }, false},
		{"missing-terminal", func(s *warmPreparationSeries) { s.Runs[2].TerminalToReapNanos = 0 }, false},
		{"unreaped", func(s *warmPreparationSeries) { s.Runs[2].Reaped = false }, false},
		{"failed-exit", func(s *warmPreparationSeries) { s.Runs[2].Exited = false }, false},
		{"missing-phase", func(s *warmPreparationSeries) { delete(s.Runs[2].Phases, "deserialize") }, false},
		{"missing-phase-sample", func(s *warmPreparationSeries) { s.Runs[2].Phases["deserialize"] = diagnosticPhase{} }, false},
		{"unstable-evidence", func(s *warmPreparationSeries) { s.Runs[2].EvidenceSHA256 = strings.Repeat("f", 64) }, false},
		{"emulated-host", func(s *warmPreparationSeries) { s.HostCPUInfo = "vendor_id: GenuineIntel\n" }, false},
		{"missing-host-binding", func(s *warmPreparationSeries) { s.HostBootID = "" }, false},
		{"missing-engine-binding", func(s *warmPreparationSeries) { s.Engine.Configuration = "" }, false},
		{"missing-source-binding", func(s *warmPreparationSeries) { s.SourceCommit = "" }, false},
		{"missing-receipts", func(s *warmPreparationSeries) { s.Inputs = nil }, false},
		{"wrong-artifact-size", func(s *warmPreparationSeries) { s.ArtifactBytes = s.EnvelopeBytes }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			series := warmPreparationFixture()
			test.mutate(&series)
			if err := warmPreparationValidate(series, true); (err == nil) != test.want {
				t.Fatalf("admission=%v, want %v", err, test.want)
			}
		})
	}
}

func TestWarmPreparationComparisonControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*warmPreparationSeries)
		want   bool
	}{
		{"separate-engine-helper-model-bindings", func(s *warmPreparationSeries) {
			s.Engine.Configuration += "-historical"
			s.HelperSHA256 = strings.Repeat("f", 64)
			s.ModelSHA256 = "historical-model"
			s.DriverSHA256 = strings.Repeat("f", 64)
			s.SourceCommit = strings.Repeat("b", 40)
		}, true},
		{"different-host-platform", func(s *warmPreparationSeries) {
			s.Engine.HostOS, s.Engine.HostArch = "linux", "aarch64"
			s.HostCPUInfo = "Features: fp asimd\nCPU architecture: 8\n"
		}, false},
		{"different-host-boot", func(s *warmPreparationSeries) { s.HostBootID = "other-boot" }, false},
		{"different-module", func(s *warmPreparationSeries) { s.ModuleSHA256 = strings.Repeat("f", 64) }, false},
		{"different-seed", func(s *warmPreparationSeries) { s.Seed++ }, false},
		{"different-config", func(s *warmPreparationSeries) { s.ConfigSHA256 = strings.Repeat("f", 64) }, false},
		{"different-runtime", func(s *warmPreparationSeries) { s.RuntimeIdentity = "other-runtime" }, false},
		{"different-current-producer", func(s *warmPreparationSeries) { s.Inputs[s.ModuleReceipt] = strings.Repeat("f", 64) }, false},
		{"different-evidence", func(s *warmPreparationSeries) {
			s.ComparisonSHA256 = strings.Repeat("f", 64)
			for i := range s.Runs {
				s.Runs[i].EvidenceSHA256 = strings.Repeat("f", 64)
			}
		}, false},
		{"different-raw-encoding-same-validated-behavior", func(s *warmPreparationSeries) {
			for i := range s.Runs {
				s.Runs[i].EvidenceSHA256 = strings.Repeat("f", 64)
			}
		}, true},
		{"missing-validated-comparison", func(s *warmPreparationSeries) { s.ComparisonSHA256 = "" }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			before, after := warmPreparationFixture(), warmPreparationFixture()
			before.Role = "baseline"
			test.mutate(&before)
			if err := warmPreparationCompare(before, after); (err == nil) != test.want {
				t.Fatalf("comparison=%v, want %v", err, test.want)
			}
		})
	}
}

func TestWarmPreparationEvidenceControls(t *testing.T) {
	code := uint32(0)
	baseline := measurementResult(wasi.Result{Stdout: []byte("stdout"), Stderr: []byte("stderr"), NetworkTranscript: []byte("network"), TranscriptSHA256: [32]byte{1}, Termination: "exit", ExitCode: &code}, [16]byte{1}, nil)
	baseline.ChoiceTrace.Bytes = []byte("choices")
	for _, test := range []struct {
		field  string
		mutate func(*wasi.Result)
		same   bool
	}{
		{"stdout", func(r *wasi.Result) { r.Stdout = []byte("other") }, false},
		{"stderr", func(r *wasi.Result) { r.Stderr = []byte("other") }, false},
		{"network", func(r *wasi.Result) { r.NetworkTranscript = []byte("other") }, false},
		{"wasi", func(r *wasi.Result) { r.TranscriptSHA256[0]++ }, false},
		{"terminal", func(r *wasi.Result) { terminal := r.RuntimeTerminal(); terminal[0]++; r.SetRuntimeTerminal(terminal) }, false},
		{"choices", func(r *wasi.Result) { r.ChoiceTrace.Bytes = []byte("other") }, false},
		{"termination", func(r *wasi.Result) { r.Termination = "other" }, false},
		{"exit-code", func(r *wasi.Result) { other := uint32(1); r.ExitCode = &other }, false},
		{"environment-identity", func(r *wasi.Result) { r.EnvironmentIdentity = "other" }, true},
		{"started-identity", func(r *wasi.Result) { r.Started = &wasi.Started{} }, true},
	} {
		t.Run(test.field, func(t *testing.T) {
			candidate := baseline
			test.mutate(&candidate)
			if (warmPreparationEvidence(candidate) == warmPreparationEvidence(baseline)) != test.same {
				t.Fatal("behavior/identity separation lost", test.field)
			}
		})
	}
}

func TestWarmPreparationArtifactSizeControls(t *testing.T) {
	request := wasi.Request{CompileCache: t.TempDir(), ModuleSHA256: strings.Repeat("a", 64), HelperSHA256: strings.Repeat("b", 64), Engine: wasi.PinnedEngine()}
	var expected string
	var expectedEnvelope uint64
	for _, module := range []string{request.ModuleSHA256, strings.Repeat("c", 64)} {
		manifest, err := json.Marshal(map[string]any{"version": 3, "module_sha256": module, "helper_sha256": request.HelperSHA256, "configuration": request.Engine.Configuration, "host_os": request.Engine.HostOS, "host_arch": request.Engine.HostArch})
		if err != nil {
			t.Fatal(err)
		}
		header := make([]byte, 24)
		copy(header, "GOMADC03")
		binary.LittleEndian.PutUint32(header[8:12], uint32(len(manifest)))
		binary.LittleEndian.PutUint64(header[12:20], 100)
		entry := filepath.Join(request.CompileCache, digest(manifest)+".gomad-cache-v3")
		data := append(append(header, manifest...), make([]byte, 100)...)
		if err := os.WriteFile(entry, data, 0600); err != nil {
			t.Fatal(err)
		}
		if module == request.ModuleSHA256 {
			expected, expectedEnvelope = entry, uint64(len(data))
		}
	}
	entry, artifact, envelope := testWarmPreparationArtifact(t, request)
	if entry != expected || artifact != 100 || envelope != expectedEnvelope {
		t.Fatal("artifact/envelope bytes or matching module identity lost", entry, artifact, envelope)
	}
}

func TestWarmPreparationCrossEncodingBehaviorControls(t *testing.T) {
	config := standaloneConfig()
	config.Profile, config.Clock.ReadStepNanos = wasi.CooperativeProfile, 0
	environment := newTestEnvironment(t, config)
	var legacy []byte
	for i, operation := range []struct{ op, input string }{
		{"args_sizes_get", `{}`}, {"args_get", `{}`}, {"environ_sizes_get", `{}`}, {"environ_get", `{}`},
		{"random_get", `{"length":2}`}, {"random_get", `{"length":3}`},
		{"clock_time_get", `{"clock_id":1,"precision":7}`}, {"clock_time_get", `{"clock_id":1,"precision":7}`},
		{"clock_time_get", `{"clock_id":0,"precision":7}`},
		{"fd_write", `{"fd":1,"data_base64":"YWI="}`},
		{"fd_read", `{"fd":99,"length":1}`}, {"unknown", `{"value":1}`},
	} {
		request := wasi.Call{Type: "call", ID: uint64(i + 1), Op: operation.op, Input: json.RawMessage(operation.input)}
		reply, err := environment.Handle(request)
		if err != nil {
			t.Fatal(err)
		}
		requestData, _ := json.Marshal(request)
		replyData, _ := json.Marshal(reply)
		legacy = append(append(append(append(legacy, requestData...), '\n'), replyData...), '\n')
	}
	code := uint32(0)
	current := measurementResult(wasi.Result{Transcript: environment.Transcript(), Stdout: []byte("ab"), Termination: "exit", ExitCode: &code}, [16]byte{1}, nil)
	historical := current
	historical.Transcript = legacy
	want, err := warmPreparationBehavior(current, config, false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := warmPreparationBehavior(historical, config, true)
	if err != nil || got != want {
		t.Fatal("same real owner behavior changed solely because of transcript encoding", err)
	}
	for _, test := range []struct{ name, old, replacement string }{
		{"clock-position", `"timestamp":1`, `"timestamp":2`},
		{"clock-precision", `"precision":7`, `"precision":8`},
		{"operation-id", `"call","id":12`, `"call","id":13`},
		{"operation-input", `"value":1`, `"value":2`},
		{"error", `"errno":8`, `"errno":9`},
		{"output", `"written":2`, `"written":1`},
		{"random-reply", `"data_base64":`, `"extra":1,"data_base64":`},
		{"unsupported-message", `operation outside stock WASI profile: unknown`, `operation outside stock WASI profile: other`},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := historical
			changed.Transcript = bytes.Replace(legacy, []byte(test.old), []byte(test.replacement), 1)
			if bytes.Equal(changed.Transcript, legacy) {
				t.Fatal("control did not mutate its real input")
			}
			got, err := warmPreparationBehavior(changed, config, true)
			if err == nil && got == want {
				t.Fatal("cross-encoding projection hid changed behavior")
			}
		})
	}
	for _, field := range []string{"stdout", "stderr", "network", "choices", "finish", "virtual-time", "message", "termination", "exit"} {
		t.Run(field, func(t *testing.T) {
			changed := historical
			switch field {
			case "stdout":
				changed.Stdout = []byte("other")
			case "stderr":
				changed.Stderr = []byte("other")
			case "network":
				changed.NetworkTranscript = []byte("other")
			case "choices":
				changed.ChoiceTrace.Bytes = []byte("other")
			case "finish":
				terminal := changed.RuntimeTerminal()
				terminal[0]++
				changed.SetRuntimeTerminal(terminal)
			case "virtual-time":
				changed.VirtualTimeElapsedNanos++
			case "message":
				changed.Message = "other"
			case "termination":
				changed.Termination = "other"
			case "exit":
				code := uint32(1)
				changed.ExitCode = &code
			}
			got, err := warmPreparationBehavior(changed, config, true)
			if err != nil || got == want {
				t.Fatal("non-WASI evidence lost", err)
			}
		})
	}
}

func TestWarmPreparationHistoricalTranscriptOwnerProjection(t *testing.T) {
	config := standaloneConfig()
	config.Args = []string{"guest"}
	environment := newTestEnvironment(t, config)
	legacy := []byte("{\"type\":\"call\",\"id\":1,\"op\":\"args_get\",\"input\":{}}\n{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{\"values\":[\"Z3Vlc3Q=\"]}}\n")
	testCall(t, environment, 1, "args_get", `{}`)
	converted, err := warmPreparationLegacyTranscript(legacy, config)
	if err != nil || !bytes.Equal(converted, environment.Transcript()) {
		t.Fatal("historical derived arguments did not match the typed owner", err)
	}
	if _, err := wasi.DecodeTranscript(legacy); err == nil {
		t.Fatal("production decoder must not accept historical JSONL")
	}
}

func TestWarmPreparationHistoricalTranscriptPreservesEmptyShape(t *testing.T) {
	for _, test := range []struct {
		name   string
		args   []string
		output string
		valid  bool
	}{
		{"empty-list", []string{}, `{"values":[]}`, true},
		{"null-list", []string{}, `{"values":null}`, false},
		{"missing-list", []string{}, `{}`, false},
		{"empty-bytes", []string{""}, `{"values":[""]}`, true},
		{"null-bytes", []string{""}, `{"values":[null]}`, false},
		{"unknown-field", []string{}, `{"values":[],"extra":0}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := standaloneConfig()
			config.Args = test.args
			legacy := []byte("{\"type\":\"call\",\"id\":1,\"op\":\"args_get\",\"input\":{}}\n" +
				"{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":" + test.output + "}\n")
			converted, err := warmPreparationLegacyTranscript(legacy, config)
			if !test.valid {
				if err == nil {
					t.Fatal("historical output shape or unknown field accepted")
				}
				return
			}
			environment := newTestEnvironment(t, config)
			testCall(t, environment, 1, "args_get", `{}`)
			if err != nil || !bytes.Equal(converted, environment.Transcript()) {
				t.Fatal("historical output differs from the typed owner", err)
			}
		})
	}
}

func TestWarmPreparationCollectionRetainsBinaryArtifactName(t *testing.T) {
	root := t.TempDir()
	series := warmPreparationFixture()
	result := wasi.Result{Transcript: []byte("GMWT\x01\x00\x00\x00")}
	testPersistDiagnosticRawResult(t, root, "untimed-evidence", result)
	if failed := testWarmPreparationPublishCollection(t, root, "candidate-collection.json", &series, result, "", "unqualified control"); failed != "unqualified control" {
		t.Fatal(failed)
	}
	var collection warmPreparationCollection
	data, err := os.ReadFile(series.CollectionPath)
	if err != nil || json.Unmarshal(data, &collection) != nil {
		t.Fatal("missing collection marker", err)
	}
	if collection.CanonicalPath != filepath.Join(root, "untimed-evidence-canonical.bin") || series.Inputs[collection.CanonicalPath] != digest(result.Transcript) {
		t.Fatal("collection did not bind the binary owner's actual artifact")
	}
}

func TestWarmPreparationHistoricalTranscriptRefusesLostEvidence(t *testing.T) {
	config := standaloneConfig()
	config.Args = []string{"guest"}
	legacy := []byte("{\"type\":\"call\",\"id\":1,\"op\":\"args_get\",\"input\":{}}\n{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{\"values\":[\"Z3Vlc3Q=\"]}}\n")
	for _, test := range []struct{ name, old, replacement string }{
		{"derived-argument", "Z3Vlc3Q=", "b3RoZXI="},
		{"reply-id", `"reply","id":1`, `"reply","id":2`},
		{"missing-reply", string(legacy[bytes.IndexByte(legacy, '\n')+1:]), ""},
		{"duplicate-field", `"op":"args_get"`, `"op":"args_get","op":"args_get"`},
		{"unknown-envelope-field", `"errno":0`, `"errno":0,"extra":true`},
		{"hidden-message", `"errno":0`, `"errno":0,"message":"changed"`},
		{"noncanonical-runtime", "args_get", "runtime_choice"},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := bytes.Replace(legacy, []byte(test.old), []byte(test.replacement), 1)
			if _, err := warmPreparationLegacyTranscript(changed, config); err == nil {
				t.Fatal("historical evidence change was discarded")
			}
		})
	}
	for _, replacement := range []wasi.Config{func() wasi.Config { c := config; c.Args = []string{"other"}; return c }(), func() wasi.Config { c := config; c.Environment = []string{"OTHER=value"}; return c }()} {
		op := "args_get"
		if replacement.Environment[0] == "OTHER=value" {
			op = "environ_get"
		}
		environment := newTestEnvironment(t, config)
		request := wasi.Call{Type: "call", ID: 1, Op: op, Input: json.RawMessage(`{}`)}
		reply, err := environment.Handle(request)
		if err != nil {
			t.Fatal(err)
		}
		requestData, _ := json.Marshal(request)
		replyData, _ := json.Marshal(reply)
		data := append(append(append(requestData, '\n'), replyData...), '\n')
		if _, err := warmPreparationLegacyTranscript(data, replacement); err == nil {
			t.Fatal("derived captured configuration change was discarded", op)
		}
	}
}

func TestNamespaceMeasurementPerSeedMedianControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func([]namespaceMeasurementSample, []namespaceMeasurementSample)
		want   bool
	}{
		{"at-quarter-reduction", func(_, _ []namespaceMeasurementSample) {}, true},
		{"one-seed-over-bound", func(_, after []namespaceMeasurementSample) {
			for i := 5; i < 10; i++ {
				after[i].AuthorizedNanos++
			}
		}, false},
		{"one-outlier-not-median", func(_, after []namespaceMeasurementSample) { after[0].AuthorizedNanos = 1000 }, true},
		{"different-seed", func(_, after []namespaceMeasurementSample) { after[0].Seed++ }, false},
		{"different-repetition", func(_, after []namespaceMeasurementSample) { after[0].Repetition++ }, false},
		{"zero-timing", func(_, after []namespaceMeasurementSample) { after[0].AuthorizedNanos = 0 }, false},
		{"nonempty-zero-count-map", func(_, after []namespaceMeasurementSample) {
			after[0].GoFilesystemCalls = map[string]uint64{"fd_write": 0}
		}, false},
		{"served-go-filesystem", func(_, after []namespaceMeasurementSample) {
			after[0].GoFilesystemCalls = map[string]uint64{"fd_pwrite": 1}
		}, false},
		{"missing-real-counter", func(_, after []namespaceMeasurementSample) { after[0].CounterPresent = false }, false},
		{"changed-wasi-choice-output", func(_, after []namespaceMeasurementSample) { after[0].EvidenceSHA256 = strings.Repeat("f", 64) }, false},
		{"changed-call-count", func(_, after []namespaceMeasurementSample) { after[0].Calls++ }, false},
		{"changed-time-advance", func(_, after []namespaceMeasurementSample) { after[0].TimeAdvances++ }, false},
		{"changed-environment", func(_, after []namespaceMeasurementSample) { after[0].EnvironmentIdentity = "other" }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var before, after []namespaceMeasurementSample
			for _, seed := range []uint64{0, 1000003, 2000006, 3000009} {
				for repetition := range 5 {
					sample := namespaceMeasurementSample{Role: "baseline", Seed: seed, Repetition: repetition, AuthorizedNanos: 100, EvidenceSHA256: strings.Repeat("e", 64), EnvironmentIdentity: "environment", Calls: 10, TimeAdvances: 3}
					before = append(before, sample)
					sample.Role, sample.AuthorizedNanos, sample.CounterPresent = "candidate", 75, true
					after = append(after, sample)
				}
			}
			test.mutate(before, after)
			if err := namespaceMeasurementCompare(before, after); (err == nil) != test.want {
				t.Fatalf("comparison=%v want accepted=%v", err, test.want)
			}
		})
	}
}

func TestNamespaceMeasurementActualCounterControls(t *testing.T) {
	for _, test := range []struct {
		name, data      string
		candidate, want bool
	}{
		{"historical-absence", `{}`, false, true},
		{"candidate-empty", `{"GoFilesystemCalls":{}}`, true, true},
		{"candidate-missing", `{}`, true, false},
		{"candidate-null", `{"GoFilesystemCalls":null}`, true, false},
		{"candidate-zero-entry", `{"GoFilesystemCalls":{"fd_write":0}}`, true, false},
		{"candidate-served", `{"GoFilesystemCalls":{"fd_write":1}}`, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := namespaceMeasurementCounter([]byte(test.data), test.candidate)
			if (err == nil) != test.want {
				t.Fatalf("actual counter=%v want admitted=%v", err, test.want)
			}
		})
	}
}

func TestNamespaceMeasurementOriginalRunControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*wasi.Result, *wasi.Request, *string)
		want   bool
	}{
		{"successful-actual-warm-run", func(*wasi.Result, *wasi.Request, *string) {}, true},
		{"original-run-error", func(_ *wasi.Result, _ *wasi.Request, e *string) { *e = "original transport error" }, false},
		{"failed-suite", func(r *wasi.Result, _ *wasi.Request, _ *string) {
			r.Stdout = []byte("=== RUN   TestActivityTestSuite\n--- FAIL: TestActivityTestSuite")
		}, false},
		{"missing-selection", func(r *wasi.Result, _ *wasi.Request, _ *string) { r.Stdout = []byte("PASS") }, false},
		{"unreaped", func(r *wasi.Result, _ *wasi.Request, _ *string) { r.Reaped = false }, false},
		{"cancelled", func(r *wasi.Result, _ *wasi.Request, _ *string) { r.Cancelled = true }, false},
		{"watchdog", func(r *wasi.Result, _ *wasi.Request, _ *string) { r.WatchdogTimeout = true }, false},
		{"wrong-module", func(r *wasi.Result, _ *wasi.Request, _ *string) { r.Started.ModuleSHA256 = strings.Repeat("f", 64) }, false},
		{"wrong-engine", func(r *wasi.Result, _ *wasi.Request, _ *string) { r.Started.Engine.Configuration += "-other" }, false},
		{"cold-cache", func(r *wasi.Result, _ *wasi.Request, _ *string) { r.Telemetry.CacheHit = false }, false},
		{"compiled-run", func(r *wasi.Result, _ *wasi.Request, _ *string) { r.Telemetry.CompileNanos = 1 }, false},
		{"missing-authorized-boundary", func(r *wasi.Result, _ *wasi.Request, _ *string) { r.AuthorizationToTerminalNanos = 0 }, false},
		{"raw-digest-changed", func(r *wasi.Result, _ *wasi.Request, _ *string) { r.TranscriptSHA256[0]++ }, false},
		{"invalid-raw-transcript-rehashed", func(r *wasi.Result, _ *wasi.Request, _ *string) {
			r.Transcript = []byte("invalid")
			r.TranscriptSHA256 = sha256.Sum256(r.Transcript)
		}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := standaloneConfig()
			environment := newTestEnvironment(t, config)
			request := wasi.Request{ModuleSHA256: strings.Repeat("a", 64), Engine: wasi.PinnedEngine(), Config: config}
			code := uint32(0)
			result := wasi.Result{Termination: "exit", ExitCode: &code, Reaped: true, Started: &wasi.Started{Engine: request.Engine, ModuleSHA256: request.ModuleSHA256}, Telemetry: &wasi.Telemetry{CacheHit: true}, AuthorizationToTerminalNanos: 1, Stdout: []byte("=== RUN   TestActivityTestSuite\n--- PASS: TestActivityTestSuite"), Transcript: environment.Transcript(), TranscriptSHA256: environment.TranscriptSHA256()}
			runError := ""
			test.mutate(&result, &request, &runError)
			data, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			if err := namespaceMeasurementResultCheck(result, data, request, false, runError); (err == nil) != test.want {
				t.Fatalf("original Run admission=%v want=%v", err, test.want)
			}
		})
	}
}

func TestNamespaceMeasurementRequestKeepsCommonImmutable(t *testing.T) {
	common := namespaceMeasurementCommon{Request: wasi.Request{Runtime: &wasi.RuntimeControl{Seed: 0}}}
	copy := testNamespaceMeasurementRequest(t, common)
	copy.Runtime.Seed = 1000003
	if common.Request.Runtime.Seed != 0 {
		t.Fatal("paired driver mutated immutable common seed")
	}
}

func testAddHelper(receiver *diagnosticAdmission, t *testing.T, helper string, receipt string, features []string) {
	t.Helper()
	err := receiver.addHelper(measurementTestContext(t), helper, receipt, features)
	if err != nil {
		t.Fatal(err)
	}
}

func testCall(t *testing.T, environment *wasi.Environment, id uint64, op string, input string) wasi.Reply {
	t.Helper()
	value0, err := environment.Handle(wasi.Call{Type: "call", ID: id, Op: op, Input: json.RawMessage(input)})
	if err != nil {
		t.Fatal(err)
	}
	return value0
}

func testDiagnosticModuleSourceInputs(t *testing.T, data []byte, compiler string, overlay string, repo string) map[string]string {
	t.Helper()
	value0, err := diagnosticModuleSourceInputs(measurementTestContext(t), data, compiler, overlay, repo)
	if err != nil {
		t.Fatal(err)
	}
	return value0
}

func testFileHash(t *testing.T, name string) string {
	t.Helper()
	value0, err := fileHash(measurementTestContext(t), name)
	if err != nil {
		t.Fatal(err)
	}
	return value0
}

func testMustTerminalExitRead(t *testing.T, path string) []byte {
	t.Helper()
	value0, err := mustTerminalExitRead(measurementTestContext(t), path)
	if err != nil {
		t.Fatal(err)
	}
	return value0
}

func testNamespaceMeasurementRequest(t *testing.T, common namespaceMeasurementCommon) wasi.Request {
	t.Helper()
	value0, err := namespaceMeasurementRequest(measurementTestContext(t), common)
	if err != nil {
		t.Fatal(err)
	}
	return value0
}

func testPersistDiagnosticRawResult(t *testing.T, root string, name string, result wasi.Result) {
	t.Helper()
	err := persistDiagnosticRawResult(measurementTestContext(t), root, name, result)
	if err != nil {
		t.Fatal(err)
	}
}

func testReadDiagnosticJSON(t *testing.T, path string, target any) {
	t.Helper()
	err := readDiagnosticJSON(measurementTestContext(t), path, target)
	if err != nil {
		t.Fatal(err)
	}
}

func testRequireExit(t *testing.T, label string, result wasi.Result, codes ...uint32) {
	t.Helper()
	err := requireExit(measurementTestContext(t), label, result, codes...)
	if err != nil {
		t.Fatal(err)
	}
}

func testRunGuestVariant(t *testing.T, root string, name string, request wasi.Request, warm bool) (wasi.Result, diagnosticReport, string) {
	t.Helper()
	value0, value1, value2, err := runGuestVariant(measurementTestContext(t), root, name, request, warm)
	if err != nil {
		t.Fatal(err)
	}
	return value0, value1, value2
}

func testTemporalJSON(t *testing.T, name string, value any) {
	t.Helper()
	err := temporalJSON(measurementTestContext(t), name, value)
	if err != nil {
		t.Fatal(err)
	}
}

func testTemporalObservedPair(t *testing.T, evidence string, label string, requests []wasi.Request, observers []temporalPairObserver, dispatch func() ([]wasi.Result, []error)) ([]wasi.Result, []error) {
	t.Helper()
	value0, value1, err := temporalObservedPair(measurementTestContext(t), evidence, label, requests, observers, dispatch)
	if err != nil {
		t.Fatal(err)
	}
	return value0, value1
}

func testTemporalWrite(t *testing.T, name string, data []byte) {
	t.Helper()
	err := temporalWrite(measurementTestContext(t), name, data)
	if err != nil {
		t.Fatal(err)
	}
}

func testTerminalExitRustInputs(t *testing.T, crate string) map[string]string {
	t.Helper()
	value0, err := terminalExitRustInputs(measurementTestContext(t), crate)
	if err != nil {
		t.Fatal(err)
	}
	return value0
}

func testTerminalExitStageHelper(t *testing.T, artifact string, helper string, artifactSHA256 string) {
	t.Helper()
	err := terminalExitStageHelper(measurementTestContext(t), artifact, helper, artifactSHA256)
	if err != nil {
		t.Fatal(err)
	}
}

func testWarmPreparationArtifact(t *testing.T, request wasi.Request) (string, uint64, uint64) {
	t.Helper()
	value0, value1, value2, err := warmPreparationArtifact(measurementTestContext(t), request)
	if err != nil {
		t.Fatal(err)
	}
	return value0, value1, value2
}

func testWarmPreparationPublishCollection(t *testing.T, evidence string, name string, series *warmPreparationSeries, retained wasi.Result, runError string, failed string) string {
	t.Helper()
	value0, err := warmPreparationPublishCollection(measurementTestContext(t), evidence, name, series, retained, runError, failed)
	if err != nil {
		t.Fatal(err)
	}
	return value0
}
func measurementTestContext(t *testing.T) context.Context {
	t.Helper()
	repo, err := filepath.Abs("../../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	inputs := map[string]string{}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			inputs[key] = value
		}
	}
	scope := &runScope{runResources: &runResources{light: make(chan struct{}, 4), heavy: make(chan struct{}, 4)}, options: Options{Repo: repo, Inputs: inputs, Environ: os.Environ(), Log: io.Discard}, name: t.Name()}
	ctx := context.WithValue(t.Context(), runKey{}, scope)
	t.Cleanup(func() {
		for i := len(scope.cleanups) - 1; i >= 0; i-- {
			if err := scope.cleanups[i](context.Background()); err != nil {
				t.Error(err)
			}
		}
	})
	return ctx
}
func newTestEnvironment(t *testing.T, config wasi.Config) *wasi.Environment {
	t.Helper()
	environment, err := wasi.NewEnvironment(config)
	if err != nil {
		t.Fatal(err)
	}
	return environment
}
