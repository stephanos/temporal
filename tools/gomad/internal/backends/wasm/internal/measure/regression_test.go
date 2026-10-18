package measure

import (
	"bytes"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	wasi "go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
)

// The compiled owner identities must exist in every build; capturing them is
// `wasmmeasure compiled-identity-owners`.
func TestDiagnosticCompiledIdentityOwners(t *testing.T) {
	owners, err := compiledOwnerIdentities()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("compiled identity owners: %v", owners)
}

func TestCanonicalRemoteOperationClassification(t *testing.T) {
	for _, sample := range []struct {
		tag   byte
		body  string
		local bool
	}{
		{2, "", true}, {3, "", true}, {4, "", true}, {5, "", true}, {6, "", true}, {7, strings.Repeat("\x00", 12), true},
		{32, strings.Repeat("\x00", 8), true},
		{32, "\x01\x00\x00\x00" + strings.Repeat("\x00", 8) + "\x01\x04\x00\x00\x00" + strings.Repeat("\x00", 4), false},
		{13, "\x04\x00\x00\x00", false},
	} {
		data := []byte("GMWT\x01\x00\x00\x00" + operationCountRecord(1, sample.tag, sample.body))
		counts, err := countCanonicalRemoteOperations(data, uint64(len(data)))
		if err != nil || (counts.Local == 1) != sample.local || counts.Local+counts.Remote != 1 {
			t.Fatalf("tag=%d = %+v, %v", sample.tag, counts, err)
		}
	}
}

func TestCanonicalRemoteOperationCounts(t *testing.T) {
	clock := "\x00\x00\x00\x00" + strings.Repeat("\x00", 8) + "\x01" + strings.Repeat("\x00", 7) + "\x02" + strings.Repeat("\x00", 7)
	clockPoll := "\x01\x00\x00\x00" + strings.Repeat("\x00", 35)
	mixedPoll := "\x02\x00\x00\x00" + strings.Repeat("\x00", 39) + "\x01\x04\x00\x00\x00" + strings.Repeat("\x00", 4)
	data := []byte("GMWT\x01\x00\x00\x00" + operationCountRecord(1, 33, clock) + operationCountRecord(3, 2, "") + operationCountRecord(4, 32, clockPoll) + operationCountRecord(5, 32, mixedPoll) + operationCountRecord(6, 8, "\x01"+strings.Repeat("\x00", 11)) + operationCountRecord(7, 8, "\x02"+strings.Repeat("\x00", 11)) + operationCountRecord(8, 9, "\x04"+strings.Repeat("\x00", 19)) + operationCountRecord(9, 26, "\x03"+strings.Repeat("\x00", 35)))
	original := bytes.Clone(data)
	got, err := countCanonicalRemoteOperations(data, uint64(len(data)))
	if err != nil || got.Calls != 9 || got.Local != 4 || got.Remote != 5 || got.Operations["wasi_snapshot_preview1/poll_oneoff/no-root-fd"] != 1 || got.Operations["wasi_snapshot_preview1/fd_write/fd=1"] != 1 || got.Operations["wasi_snapshot_preview1/fd_write/fd=2"] != 1 || got.Operations["wasi_snapshot_preview1/fd_pwrite/fd=4"] != 1 || got.Operations["wasi_snapshot_preview1/path_open/fd=3"] != 1 {
		t.Fatalf("actual binary operation counts = %+v, %v", got, err)
	}
	if !bytes.Equal(original, data) {
		t.Fatal("counter mutated binary input")
	}
}

func TestDiagnosticAdmissionCommand(t *testing.T) {
	features := []string{"diagnostic-profile", "guest-profile"}
	command := []string{"/pinned/cargo", "build", "--manifest-path", "/crate/Cargo.toml", "--release", "--locked", "--jobs=1", "--no-default-features", "--message-format=json", "--features", "diagnostic-profile,guest-profile"}
	build := terminalExitHelperBuild{Crate: "/crate", Command: command, Inputs: map[string]string{"/pinned/cargo": "digest"}}
	if !diagnosticHelperCommand(build, features) {
		t.Fatal("exact command rejected")
	}
	for index := range command {
		changed := build
		changed.Command = slices.Clone(command)
		changed.Command[index] = "wrong"
		if diagnosticHelperCommand(changed, features) {
			t.Fatalf("changed command element %d admitted", index)
		}
	}
}

func TestDiagnosticClusterPairWithoutMeasurements(t *testing.T) {
	pair, err := decodeClusterDiagnosticPair([]wasi.Request{{}, {}}, []wasi.Result{{Termination: "crash", Reaped: true}, {Termination: "crash", Reaped: true}})
	if err != nil || pair.Complete || pair.Measured != 0 || !reflect.DeepEqual(pair.MeasuredSum, (*diagnosticReport)(nil)) {
		t.Fatalf("unavailable measurements acquired zero-filled buckets: pair=%+v err=%v", pair, err)
	}
}

func TestDiagnosticGuestAttributionWeightedThreadTables(t *testing.T) {
	_, attribution, err := decodeGuestAttribution(diagnosticGuestFixture())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]uint64{"gc": 3, "scheduler": 5, "allocation": 0, "test": 2, "server": 0, "runtime": 0, "stdlib": 1, "other": 0}
	if attribution.SampleRows != 5 || attribution.GuestWeight != 11 || attribution.EmptyStackWeight != 7 || !maps.Equal(want, attribution.Buckets) {
		t.Fatalf("weighted attribution=%+v, want rows=5 guest=11 empty=7 buckets=%v", attribution, want)
	}
}

func TestDiagnosticModuleBuildCommands(t *testing.T) {
	for _, test := range []struct {
		kind        string
		tags        []string
		list, build []string
	}{
		{"test", []string{"test_dep"}, []string{"/go", "list", "-p=1", "-deps", "-test", "-json", "-overlay", "/overlay", "-tags", "test_dep", "./fixture"}, []string{"/go", "test", "-c", "-p=1", "-trimpath", "-overlay", "/overlay", "-o", "/module", "-tags", "test_dep", "./fixture"}},
		{"build", nil, []string{"/go", "list", "-p=1", "-deps", "-json", "-overlay", "/overlay", "./fixture"}, []string{"/go", "build", "-p=1", "-trimpath", "-buildvcs=false", "-overlay", "/overlay", "-o", "/module", "./fixture"}},
	} {
		listed, built, err := diagnosticModuleCommands("/go", "/overlay", "/module", "./fixture", test.kind, test.tags)
		if err != nil || !slices.Equal(listed, test.list) || !slices.Equal(built, test.build) {
			t.Fatalf("%s compiler commands: %v %v %v", test.kind, listed, built, err)
		}
	}
	if _, _, err := diagnosticModuleCommands("/go", "/overlay", "/module", "./fixture", "run", nil); err == nil {
		t.Fatal("non-build compiler operation admitted")
	}
}

func TestDiagnosticPerformanceCurrentHostAdmission(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		t.Skip("current-host admission requires Linux/arm64")
	}
	cpuinfo, admitted := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	if cpuinfo == "" || !admitted {
		t.Fatal("current Linux/arm64 host CPU witness was not admitted")
	}
}

func TestDiagnosticPerformanceHostEvidenceBounds(t *testing.T) {
	const armCPU = "Features: fp asimd\nCPU architecture: 8\n"
	for _, size := range []int{diagnosticHostCPUInfoLimit, diagnosticHostCPUInfoLimit + 1} {
		path := filepath.Join(t.TempDir(), "cpuinfo")
		data := armCPU + strings.Repeat("\n", size-len(armCPU))
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		cpuinfo, admitted := diagnosticPerformanceHostEvidence(path)
		if size == diagnosticHostCPUInfoLimit {
			if cpuinfo != data {
				t.Fatal("host witness at the byte bound was lost")
			}
		} else if cpuinfo != "" || admitted {
			t.Fatal("oversized host witness was retained or admitted")
		}
	}
}

func TestDiagnosticRunRootPreservesEarlierInvocation(t *testing.T) {
	base := t.TempDir()
	first, err := diagnosticRunRoot(base, "guest-profile-")
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(first, "seed0-guest.firefox.json")
	if err := os.WriteFile(marker, []byte("first profile"), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := diagnosticRunRoot(base, "guest-profile-")
	if err != nil {
		t.Fatal(err)
	}
	if first == second || filepath.Dir(first) != base || filepath.Dir(second) != base {
		t.Fatalf("invocations share their evidence directory: first=%s second=%s", first, second)
	}
	file, err := os.OpenFile(filepath.Join(second, "seed0-guest.firefox.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal("fresh invocation cannot create its profile", err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(marker)
	if err != nil || string(data) != "first profile" {
		t.Fatal("earlier profile changed", err)
	}
}

func TestDiagnosticVariantEvidencePreservesOutputs(t *testing.T) {
	transcript := []byte("GMWT\x01\x00\x00\x00" + operationCountRecord(1, 8, "\x01\x00\x00\x00\x02\x00\x00\x00ab\x02\x00\x00\x00"))
	want, err := projectVariantEvidence(wasi.Result{Transcript: transcript}, standaloneConfig(), "200")
	if err != nil {
		t.Fatal(err)
	}
	transcript[len(transcript)-4] = 1
	got, err := projectVariantEvidence(wasi.Result{Transcript: transcript}, standaloneConfig(), "200")
	if err != nil || reflect.DeepEqual(want, got) {
		t.Fatal("non-environment output change was hidden", err)
	}
}

func TestDiagnosticVariantEvidencePreservesUnsupportedFlag(t *testing.T) {
	before := []byte("GMWT\x01\x00\x00\x00" + operationCountRecord(1, 6, ""))
	after := []byte("GMWT\x01\x00\x00\x00" + operationCountRecord(1, 6, "\x00\x00\x00\x00"))
	after[9] = 1
	want, err := projectVariantEvidence(wasi.Result{Transcript: before}, standaloneConfig(), "200")
	if err != nil {
		t.Fatal(err)
	}
	got, err := projectVariantEvidence(wasi.Result{Transcript: after}, standaloneConfig(), "200")
	if err != nil || reflect.DeepEqual(want, got) {
		t.Fatal("environment unsupported flag was hidden", err)
	}
}

func TestMatchedPreparationMemoryIsExplicitAndPlatformBound(t *testing.T) {
	for _, test := range []struct {
		platform, mode string
		want           bool
	}{
		{"darwin", "", true}, {"linux", "", true},
		{"darwin", "1", false}, {"linux", "1", false},
		{"linux", "0", false}, {"linux", "true", false}, {"linux", " 1", false},
	} {
		called := false
		_, err := matchedPrepareAdmitted(test.platform, "unreadable", test.mode, func() (wasi.Result, error) {
			called = true
			return wasi.Result{}, nil
		})
		if called != test.want || (err == nil) != test.want {
			t.Fatalf("platform=%s mode=%q called=%t error=%v", test.platform, test.mode, called, err)
		}
	}
}

func TestMatchedPreparationMemoryRejectsMalformedBeforeDispatch(t *testing.T) {
	for _, data := range []string{
		"", "MemTotal: 100 kB\n", "MemAvailable: 8388608 kB\nMemAvailable: 8388608 kB\n",
		"memavailable: 8388608 kB\n", "MemAvailable: -1 kB\n", "MemAvailable: +1 kB\n",
		"MemAvailable: 8388608 MB\n", "MemAvailable: 8388608 kB extra\n", "MemAvailable: 8388608\n",
		"MemAvailable: 8388608 kB\nMemAvailable : 8388608 kB\n", "MemAvailable: 8388608 kB\nMemAvailable:8388608 kB\n",
		"MemAvailable: 18446744073709551616 kB\n", "MemAvailable: 18014398509481984 kB\n",
		"MemAvailable: 8388608 kB\n" + strings.Repeat(" ", 8<<10),
	} {
		path := filepath.Join(t.TempDir(), "meminfo")
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		for _, owner := range []string{"original", "candidate"} {
			called := false
			_, err := matchedPrepareAdmitted("linux", path, "1", func() (wasi.Result, error) {
				called = true
				return wasi.Result{Termination: owner}, nil
			})
			if called || err == nil {
				t.Fatalf("owner=%s malformed observation dispatched: %q error=%v", owner, data, err)
			}
		}
	}
}

func TestMatchedSuiteSelectionCacheAndOrder(t *testing.T) {
	for _, test := range []struct {
		selection, suite string
		want             bool
	}{{"", "workflow", true}, {"", "frontend", false}, {"frontend", "frontend", true}, {"workflow, frontend", "frontend", true}, {"workflow,update", "update", true}, {"workflow,update", "timers", false}} {
		if got := matchedSuiteSelected(test.selection, test.suite); got != test.want {
			t.Fatalf("selection %q suite %s = %t", test.selection, test.suite, got)
		}
	}
	candidate := filepath.Join(t.TempDir(), "candidate")
	for _, baseline := range []string{"", "relative", candidate, candidate + "/"} {
		if matchedBaselineCache(baseline, candidate) == nil {
			t.Fatalf("baseline cache %q admitted", baseline)
		}
	}
	if err := matchedBaselineCache(filepath.Join(t.TempDir(), "baseline"), candidate); err != nil {
		t.Fatal(err)
	}
	for sample, want := range []bool{true, false, true, false} {
		if matchedBaselineTurn(sample, 0) != want || matchedBaselineTurn(sample, 1) == want {
			t.Fatalf("sample %d order is not alternating", sample)
		}
	}
}

func TestWarmPreparationDiagnosticCargoFeatureControls(t *testing.T) {
	data := []byte(`{"reason":"compiler-artifact","target":{"name":"gomad-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":["diagnostic-profile"]} {"reason":"build-finished","success":true}`)
	if _, err := terminalExitCargoHelperFeatures(data, []string{"diagnostic-profile"}); err != nil {
		t.Fatal(err)
	}
	if _, err := terminalExitCargoHelper(data); err == nil {
		t.Fatal("release-only gate weakened")
	}
	for _, features := range []string{`[]`, `["guest-profile"]`, `["diagnostic-profile","fuel-off"]`} {
		if _, err := terminalExitCargoHelperFeatures(bytes.ReplaceAll(data, []byte(`["diagnostic-profile"]`), []byte(features)), []string{"diagnostic-profile"}); err == nil {
			t.Fatal("wrong feature admitted", features)
		}
	}
}

func TestWarmPreparationFailureKeepsEveryReason(t *testing.T) {
	failed := warmPreparationFailure("", "host bindings changed during collection")
	failed = warmPreparationFailure(failed, "baseline-material-gap")
	failed = warmPreparationFailure(failed, "baseline-material-gap")
	if failed != "host bindings changed during collection; baseline-material-gap" {
		t.Fatal(failed)
	}
}

func TestWarmPreparationHistoricalClockCountsAndPositions(t *testing.T) {
	config := standaloneConfig()
	config.Profile, config.Clock.ReadStepNanos = wasi.CooperativeProfile, 0
	legacy := []byte("{\"type\":\"call\",\"id\":1,\"op\":\"clock_time_get\",\"input\":{\"clock_id\":1,\"precision\":7}}\n{\"type\":\"reply\",\"id\":1,\"errno\":0,\"output\":{\"timestamp\":1}}\n{\"type\":\"call\",\"id\":2,\"op\":\"clock_time_get\",\"input\":{\"clock_id\":1,\"precision\":7}}\n{\"type\":\"reply\",\"id\":2,\"errno\":0,\"output\":{\"timestamp\":1}}\n")
	converted, err := warmPreparationLegacyTranscript(legacy, config)
	if err != nil {
		t.Fatal(err)
	}
	records, err := wasi.DecodeTranscript(converted)
	if err != nil || len(records) != 1 || records[0].ID != 1 || records[0].Count != 2 || records[0].Now != 1 || string(records[0].Input) != `{"clock_id":1,"precision":7}` {
		t.Fatal("clock run lost original positions, IDs or counts", err, records)
	}
	for _, data := range [][]byte{
		legacy[:bytes.Index(legacy, []byte(`{"type":"call","id":2`))],
		bytes.ReplaceAll(legacy, []byte(`"timestamp":1`), []byte(`"timestamp":2`)),
		bytes.ReplaceAll(legacy, []byte(`"clock_id":1`), []byte(`"clock_id":2`)),
		bytes.TrimSuffix(legacy, []byte{'\n'}),
	} {
		changed, err := warmPreparationLegacyTranscript(data, config)
		if err == nil && bytes.Equal(converted, changed) {
			t.Fatal("clock count/position/domain or framing change was hidden")
		}
	}
}

func TestWarmPreparationObservationPrivateFinishControls(t *testing.T) {
	code := uint32(0)
	observation := terminalExitObservation{TerminalReceivedAt: time.Unix(1, 0), SupervisorReturnedAt: time.Unix(1, 1), TerminalToReapNanos: 1, Result: measurementResult(wasi.Result{Termination: "exit", ExitCode: &code}, [16]byte{1}, nil)}
	observation.Host.Termination, observation.Host.GroupGone = "exit", true
	projection := warmPreparationObserve(observation)
	if !projection.HostExited || !projection.GroupGone || projection.TerminalToReapNanos != 1 || projection.SupervisorReturnedAt != observation.SupervisorReturnedAt {
		t.Fatal("actual terminal/host measurements lost")
	}
	data, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	var imported warmPreparationObservation
	if err := json.Unmarshal(data, &imported); err != nil {
		t.Fatal(err)
	}
	if warmPreparationEvidence(imported.Result) == warmPreparationEvidence(observation.Result) {
		t.Fatal("JSON unexpectedly preserved private finish; missing sidecar must remain visible")
	}
	imported.Result.SetRuntimeTerminal(observation.Result.RuntimeTerminal())
	if warmPreparationEvidence(imported.Result) != warmPreparationEvidence(observation.Result) {
		t.Fatal("finish sidecar did not restore exact behavior")
	}
}

func TestWarmPreparationReapConsistentToleratesOnlyClockSlew(t *testing.T) {
	received := time.Unix(100, 0)
	for _, test := range []struct {
		name     string
		returned time.Time
		recorded uint64
		want     bool
	}{
		{"exact", received.Add(9335583), 9335583, true},
		{"overflowing-recorded", received.Add(time.Millisecond), uint64(time.Millisecond) + 1<<63, false},
		{"monotonic-slew", received.Add(9335583), 9335625, true},
		{"slew-bound", received.Add(time.Second + time.Millisecond), uint64(time.Second), true},
		{"beyond-bound", received.Add(time.Second + 2*time.Millisecond), uint64(time.Second), false},
		{"small-beyond-floor", received.Add(9335583 + 20000), 9335583, false},
		{"reversed", received.Add(-time.Millisecond), uint64(time.Millisecond), false},
		{"zero", received.Add(time.Millisecond), 0, false},
	} {
		if got := warmPreparationReapConsistent(received, test.returned, test.recorded); got != test.want {
			t.Errorf("%s: got %v want %v", test.name, got, test.want)
		}
	}
}

func TestWarmPreparationSummaryControls(t *testing.T) {
	series := warmPreparationFixture()
	for i, value := range []uint64{7, 2, 5, 9, 4} {
		series.Runs[i].Telemetry.PrepareNanos = value
	}
	if got := warmPreparationSummary(series)["prepare"]; got != (variantTimingRange{Median: 5, Min: 2, Max: 9}) {
		t.Fatal("median/range lost", got)
	}
	baseline := warmPreparationFixture()
	baseline.Role = "baseline"
	for i := range baseline.Runs {
		baseline.Runs[i].Telemetry.PrepareNanos++
		baseline.Runs[i].TerminalToReapNanos++
	}
	if err := warmPreparationValidate(baseline, false); err != nil {
		t.Fatal("historical timings incorrectly subject to candidate bounds", err)
	}
}
