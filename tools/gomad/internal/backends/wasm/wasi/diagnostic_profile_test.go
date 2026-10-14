package wasi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/pprof"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/stockgo"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
)

const diagnosticHostCPUInfoLimit = 1 << 20

type diagnosticClusterInstance struct {
	Profile     *diagnosticReport `json:"profile"`
	Unavailable string            `json:"unavailable,omitempty"`
}

type diagnosticClusterPair struct {
	Instances   []diagnosticClusterInstance `json:"instances"`
	Measured    int                         `json:"measured_helpers"`
	Complete    bool                        `json:"complete"`
	MeasuredSum *diagnosticReport           `json:"measured_sum"`
}

func decodeClusterDiagnosticPair(requests []Request, results []Result) (diagnosticClusterPair, error) {
	pair := diagnosticClusterPair{Instances: make([]diagnosticClusterInstance, len(results)), Complete: true}
	if len(requests) != len(results) || len(results) == 0 {
		pair.Complete = false
		return pair, fmt.Errorf("cluster request/result cardinality")
	}
	var failures []string
	var overflow bool
	add := func(a, b uint64) uint64 {
		if b > math.MaxUint64-a {
			overflow = true
			failures = append(failures, "cluster diagnostic sum overflow")
			return 0
		}
		return a + b
	}
	for index, result := range results {
		profile, err := decodeDiagnosticReport(result.helperStderr)
		if err != nil {
			pair.Complete = false
			pair.Instances[index].Unavailable = err.Error()
			if result.Termination != "crash" || !result.Reaped || len(result.helperStderr) > 16<<10 {
				failures = append(failures, fmt.Sprintf("helper %d: %v", index, err))
			}
			continue
		}
		if pair.Measured != 0 && (profile.HostOS != pair.MeasuredSum.HostOS || profile.HostArch != pair.MeasuredSum.HostArch) {
			pair.Complete, pair.MeasuredSum = false, nil
			return pair, fmt.Errorf("cluster diagnostic host mismatch")
		}
		if pair.MeasuredSum == nil {
			pair.MeasuredSum = &diagnosticReport{Schema: "gomad3.wasm-profile/v1", DiagnosticOnly: true, Backend: "cranelift-jit", Phases: map[string]diagnosticPhase{}, ABI: map[string]diagnosticABI{}, Nested: map[string]diagnosticNested{}}
		}
		pair.Instances[index].Profile = &profile
		pair.Measured++
		pair.MeasuredSum.HostOS, pair.MeasuredSum.HostArch = profile.HostOS, profile.HostArch
		for name, bucket := range profile.Phases {
			sum := pair.MeasuredSum.Phases[name]
			sum.Count, sum.WallNanos = add(sum.Count, bucket.Count), add(sum.WallNanos, bucket.WallNanos)
			sum.UserNanos, sum.SystemNanos = add(sum.UserNanos, bucket.UserNanos), add(sum.SystemNanos, bucket.SystemNanos)
			sum.CPUSamples = add(sum.CPUSamples, bucket.CPUSamples)
			sum.PeakRSSBytesHighWater = max(sum.PeakRSSBytesHighWater, bucket.PeakRSSBytesHighWater)
			pair.MeasuredSum.Phases[name] = sum
		}
		for name, bucket := range profile.ABI {
			sum := pair.MeasuredSum.ABI[name]
			sum.Count, sum.WallNanos = add(sum.Count, bucket.Count), add(sum.WallNanos, bucket.WallNanos)
			sum.InstantiateWallNanos, sum.ExecuteWallNanos = add(sum.InstantiateWallNanos, bucket.InstantiateWallNanos), add(sum.ExecuteWallNanos, bucket.ExecuteWallNanos)
			pair.MeasuredSum.ABI[name] = sum
		}
		for name, bucket := range profile.Nested {
			sum := pair.MeasuredSum.Nested[name]
			sum.Count, sum.WallNanos = add(sum.Count, bucket.Count), add(sum.WallNanos, bucket.WallNanos)
			pair.MeasuredSum.Nested[name] = sum
		}
		pair.MeasuredSum.Residual.InstantiateWallNanos = add(pair.MeasuredSum.Residual.InstantiateWallNanos, profile.Residual.InstantiateWallNanos)
		pair.MeasuredSum.Residual.ExecuteWallNanos = add(pair.MeasuredSum.Residual.ExecuteWallNanos, profile.Residual.ExecuteWallNanos)
	}
	if len(failures) != 0 {
		pair.Complete = false
		if overflow {
			pair.MeasuredSum = nil
		}
		return pair, fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return pair, nil
}

func TestDiagnosticClusterPairProfiles(t *testing.T) {
	requests := []Request{{HelperSHA256: "helper", ModuleSHA256: "module"}, {HelperSHA256: "helper", ModuleSHA256: "module"}}
	exit := uint32(0)
	valid := bytes.Replace(diagnosticFixture(t), []byte(`"encoding":{"count":0,"wall_nanos":0}`), []byte(`"encoding":{"count":1,"wall_nanos":5}`), 1)
	for _, test := range []struct {
		name      string
		second    Result
		complete  bool
		measured  int
		wall      uint64
		wantError bool
	}{
		{"both measured", Result{Termination: "exit", ExitCode: &exit, helperStderr: valid}, true, 2, 200, false},
		{"killed helper unavailable", Result{Termination: "crash", Reaped: true}, false, 1, 100, false},
		{"partial killed report unavailable", Result{Termination: "crash", Reaped: true, helperStderr: []byte("gomad-wasm-profile-v1 {")}, false, 1, 100, false},
		{"successful helper missing profile", Result{Termination: "exit", ExitCode: &exit}, false, 1, 100, true},
		{"successful helper malformed profile", Result{Termination: "exit", ExitCode: &exit, helperStderr: []byte("invalid\n")}, false, 1, 100, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			results := []Result{{Termination: "exit", ExitCode: &exit, helperStderr: valid}, test.second}
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

func TestDiagnosticClusterPairWithoutMeasurements(t *testing.T) {
	pair, err := decodeClusterDiagnosticPair([]Request{{}, {}}, []Result{{Termination: "crash", Reaped: true}, {Termination: "crash", Reaped: true}})
	if err != nil || pair.Complete || pair.Measured != 0 || !reflect.DeepEqual(pair.MeasuredSum, (*diagnosticReport)(nil)) {
		t.Fatalf("unavailable measurements acquired zero-filled buckets: pair=%+v err=%v", pair, err)
	}
}

func TestDiagnosticClusterPairRejectsOverflow(t *testing.T) {
	valid := diagnosticFixture(t)
	overflow := bytes.Replace(valid, []byte(`"count":1`), []byte(`"count":18446744073709551615`), 1)
	_, err := decodeClusterDiagnosticPair([]Request{{}, {}}, []Result{{helperStderr: overflow}, {helperStderr: valid}})
	if err == nil {
		t.Fatal("overflowing summed helper counters accepted")
	}
}

func TestDiagnosticClusterBaseline(t *testing.T) {
	base := os.Getenv("GOMAD_WASM_TEMPORAL_RECOVERY_ROOT")
	if base == "" {
		t.Skip("set GOMAD_WASM_TEMPORAL_RECOVERY_ROOT for the unsampled cluster baseline")
	}
	helper := os.Getenv("GOMAD_WASM_TEST_HELPER")
	if !filepath.IsAbs(base) || filepath.Clean(base) != base || base == string(filepath.Separator) || !filepath.IsAbs(helper) || !filepath.IsAbs(os.Getenv("GOMAD3_STOCK_GO")) {
		t.Fatal("baseline root, diagnostic-profile helper and stock Go compiler must be explicit absolute paths")
	}
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(repo, base)
	if err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		t.Fatal("raw cluster baseline evidence must be retained outside the repository")
	}
	cpuinfo, admitted := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	if !admitted {
		t.Fatal("cluster baseline requires a real Linux/arm64 CPU witness")
	}
	prerequisites := map[string]string{}
	for index := 1; index <= 12; index++ {
		name := fmt.Sprintf(".flow/tasks/fn-154-gomad-simpler-virtual-network-with.%d.json", index)
		data, err := hostfs.ReadBounded(filepath.Join(repo, name), 1<<20)
		var task struct {
			Status string `json:"status"`
		}
		if err != nil || json.Unmarshal(data, &task) != nil || task.Status != "done" {
			t.Fatalf("cluster baseline requires completed post-fn-154 prerequisite %s", name)
		}
		prerequisites[name] = digest(data)
	}
	commit, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	status, err := exec.Command("git", "-C", repo, "status", "--porcelain").Output()
	if err != nil {
		t.Fatal(err)
	}
	inputs := map[string]string{}
	for _, name := range []string{"diagnostic_profile_test.go", "integration_test.go"} {
		path, err := filepath.Abs(name)
		if err != nil {
			t.Fatal(err)
		}
		inputs[path] = fileHash(t, path)
	}
	parentExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	parentSHA256 := fileHash(t, parentExecutable)
	root, err := diagnosticRunRoot(base, "cluster-baseline-")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("retaining cluster baseline in %s", root)
	t.Setenv("GOMAD_WASM_TEMPORAL_RECOVERY_ROOT", root)
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", root)
	// A killed helper cannot prove its diagnostic feature by emitting a final report.
	probe := cooperativeRequest(t, "../toolchain/testdata/choices")
	probeResult := runDiagnosticClusterFeatureProbe(t, root, probe)
	request, evidence := temporalRecoveryRequest(t, "temporal_broker")
	var pairs []map[string]any
	observe := func(t *testing.T, evidence, label string, requests []Request, results []Result, runErrors []error) {
		pair, decodeErr := decodeClusterDiagnosticPair(requests, results)
		var instances []map[string]any
		for index, result := range results {
			name := requests[index].Broker.Node.ID + "-" + label
			if len(result.helperStderr) > 64<<10 {
				t.Fatal("cluster helper stderr capture exceeds transport bound")
			}
			temporalWrite(t, filepath.Join(evidence, name+"-helper-stderr.txt"), result.helperStderr)
			failure := ""
			if runErrors[index] != nil {
				failure = runErrors[index].Error()
			}
			instances = append(instances, map[string]any{"node": requests[index].Broker.Node.ID, "seed": requests[index].Runtime.Seed,
				"identities": variantIdentities(requests[index], result), "termination": result.Termination, "exit_code": result.ExitCode,
				"run_error": failure, "profile": pair.Instances[index], "phase_metrics": result.Telemetry,
				"wait_resource_usage": result.HelperResourceUsage, "authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos})
		}
		measurement := map[string]any{"label": label, "instances": instances, "measured_sum": pair.MeasuredSum,
			"measured_helpers": pair.Measured, "expected_helpers": len(results), "complete": pair.Complete}
		if decodeErr != nil {
			measurement["diagnostic_error"] = decodeErr.Error()
			t.Error(decodeErr)
		}
		pairs = append(pairs, measurement)
		temporalJSON(t, filepath.Join(evidence, label+"-diagnostic-profile.json"), measurement)
	}
	profilePath := filepath.Join(root, "cluster-parent.pprof")
	profile, err := os.OpenFile(profilePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := pprof.StartCPUProfile(profile); err != nil {
		_ = profile.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pprof.StopCPUProfile()
		if err := profile.Close(); err != nil {
			t.Error(err)
		}
		summary, summaryErr := exec.Command(stockgo.Command(), "tool", "pprof", "-top", profilePath).CombinedOutput()
		temporalWrite(t, filepath.Join(root, "cluster-parent-pprof-top.txt"), summary)
		if summaryErr != nil {
			t.Error(summaryErr)
		}
		temporalVerifyInputs(t, inputs)
		summaryFailure := ""
		if summaryErr != nil {
			summaryFailure = summaryErr.Error()
		}
		environment := map[string]string{"GOMAD_WASM_TEMPORAL_RECOVERY_ROOT": base, "GOMAD_WASM_TEST_HELPER": helper, "GOMAD3_STOCK_GO": os.Getenv("GOMAD3_STOCK_GO"), "PATH": os.Getenv("PATH")}
		for _, key := range []string{"GOMAD_WASM_TEMPORAL_RECOVERY_COMPILE_CACHE", "GOMAD_WASM_TEMPORAL_RECOVERY_DIAGNOSTIC_BUDGET", "GOMAD_WASM_TEMPORAL_RECOVERY_EXPECT_MODULE_SHA256", "GOMAD_WASM_TOOLCHAIN_CACHE"} {
			environment[key] = os.Getenv(key)
		}
		temporalJSON(t, filepath.Join(root, "cluster-baseline-report.json"), map[string]any{
			"diagnostic_only": true, "source_commit": strings.TrimSpace(string(commit)), "source_dirty": len(status) != 0, "source_status": string(status), "collector_sources": inputs,
			"prerequisite_receipts": prerequisites, "host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "go_version": runtime.Version(), "host_cpuinfo": cpuinfo,
			"required_host_measured": admitted, "seeds": []uint64{7}, "selection": "TestTemporalIndependentClustersPartitionHealWholeClusterCrash", "pairs": pairs,
			"scenario_passed": !t.Failed() && len(pairs) == 4, "diagnostic_feature_probe": variantIdentities(probe, probeResult), "identities": variantIdentities(request, Result{}),
			"module_provenance_sha256": fileHash(t, filepath.Join(root, "temporal_broker-provenance.json")),
			"parent_pprof_sha256":      fileHash(t, profilePath), "parent_pprof_summary_sha256": digest(summary), "parent_pprof_summary_error": summaryFailure, "parent_executable_sha256": parentSHA256,
			"parent_pprof_summary_command": []string{stockgo.Command(), "tool", "pprof", "-top", profilePath},
			"scope":                        "unsampled diagnostic-profile helpers; parent CPU profile includes original scenario, recovery, replay and evidence publication; module preparation and feature probe excluded",
			"bucket_scope":                 "measured_sum excludes unavailable helper profiles; nested buckets overlap ABI/phase spans; RSS high-water is maximum per-process high-water, not concurrent cluster RSS; summed helper wall spans overlap in real time",
			"rerun": map[string]any{"working_directory": filepath.Join(repo, "tools", "gomad"), "environment": environment,
				"command": "env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -timeout 90m -run '^TestDiagnosticClusterBaseline$' ./internal/backends/wasm/wasi"},
		})
	})
	temporalIndependentClustersPartitionHealWholeClusterCrash(t, request, evidence, observe)
}

func runDiagnosticClusterFeatureProbe(t *testing.T, root string, request Request) Result {
	t.Helper()
	result, err := Run(t.Context(), request)
	temporalWrite(t, filepath.Join(root, "diagnostic-feature-probe-helper-stderr.txt"), result.helperStderr)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "diagnostic feature probe", result, 0)
	profile, err := decodeDiagnosticReport(result.helperStderr)
	if err != nil {
		t.Fatal(err)
	}
	temporalJSON(t, filepath.Join(root, "diagnostic-feature-probe-profile.json"), profile)
	return result
}

func diagnosticPerformanceHost(goos, goarch, cpuinfo string) bool {
	if goos != "linux" || goarch != "arm64" || len(cpuinfo) > diagnosticHostCPUInfoLimit {
		return false
	}
	var architecture, asimd bool
	for line := range strings.SplitSeq(cpuinfo, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "CPU architecture":
			architecture = architecture || strings.TrimSpace(value) == "8"
		case "Features":
			asimd = asimd || slices.Contains(strings.Fields(value), "asimd")
		}
	}
	return architecture && asimd
}

func diagnosticPerformanceHostEvidence(path string) (string, bool) {
	cpuinfo, err := hostfs.ReadBounded(path, diagnosticHostCPUInfoLimit)
	if err != nil {
		return "", false
	}
	return string(cpuinfo), diagnosticPerformanceHost(runtime.GOOS, runtime.GOARCH, string(cpuinfo))
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

func TestDiagnosticPerformanceCurrentHostAdmission(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "arm64" {
		t.Skip("current-host admission requires Linux/arm64")
	}
	cpuinfo, admitted := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	if cpuinfo == "" || !admitted {
		t.Fatal("current Linux/arm64 host CPU witness was not admitted")
	}
}

type variantEvidence struct {
	Choice, WASI, Network, Stdout, Stderr []byte
	Termination, Message                  string
	ExitCode                              *uint32
	VirtualNanos                          uint64
}

func projectVariantEvidence(result Result, maskEnvironment bool) (variantEvidence, error) {
	transcript := result.Transcript
	if maskEnvironment {
		var projected bytes.Buffer
		lines := bytes.Split(transcript, []byte{'\n'})
		if len(transcript) != 0 && (len(lines)%2 != 1 || len(lines[len(lines)-1]) != 0) {
			return variantEvidence{}, fmt.Errorf("incomplete WASI call/reply transcript")
		}
		for index := 0; index+1 < len(lines); index += 2 {
			var call Call
			var reply Reply
			if err := json.Unmarshal(lines[index], &call); err != nil {
				return variantEvidence{}, err
			}
			if err := json.Unmarshal(lines[index+1], &reply); err != nil {
				return variantEvidence{}, err
			}
			if call.Type != "call" || reply.Type != "reply" || call.ID != reply.ID {
				return variantEvidence{}, fmt.Errorf("invalid WASI call/reply pair")
			}
			output := lines[index+1]
			if call.Op == "environ_get" || call.Op == "environ_sizes_get" {
				reply.Output = json.RawMessage(`{}`)
				var err error
				output, err = json.Marshal(reply)
				if err != nil {
					return variantEvidence{}, err
				}
			}
			projected.Write(lines[index])
			projected.WriteByte('\n')
			projected.Write(output)
			projected.WriteByte('\n')
		}
		transcript = projected.Bytes()
	}
	return variantEvidence{Choice: result.ChoiceTrace.Bytes, WASI: transcript, Network: result.NetworkTranscript, Stdout: result.Stdout, Stderr: result.Stderr, Termination: result.Termination, Message: result.Message, ExitCode: result.ExitCode, VirtualNanos: result.VirtualTimeElapsedNanos}, nil
}

func variantIdentities(request Request, result Result) map[string]string {
	engine := request.Engine.Configuration
	if request.diagnosticConfigurationRevision != "" {
		engine = request.diagnosticConfigurationRevision
	}
	identities := map[string]string{"module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "environment_identity": result.EnvironmentIdentity, "engine_revision": engine, "model_sha256": ImplementationSHA256()}
	if request.Runtime != nil && request.Runtime.Choice != nil {
		identities["runtime_identity"] = request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey
	}
	return identities
}

type variantTimingRange struct {
	Median uint64 `json:"median"`
	Min    uint64 `json:"min"`
	Max    uint64 `json:"max"`
}

func variantTiming(values []uint64) variantTimingRange {
	if len(values) == 0 {
		return variantTimingRange{}
	}
	ordered := slices.Clone(values)
	slices.Sort(ordered)
	return variantTimingRange{Median: ordered[len(ordered)/2], Min: ordered[0], Max: ordered[len(ordered)-1]}
}

func variantRecommendation(baseline, candidate []uint64, failed, identical bool) string {
	if failed || !identical {
		return "reject"
	}
	if len(baseline) == 0 || len(candidate) == 0 {
		return "investigate"
	}
	before, after := variantTiming(baseline), variantTiming(candidate)
	if after.Median >= before.Median {
		return "reject"
	}
	if before.Median-after.Median > max(before.Max-before.Min, after.Max-after.Min) {
		return "adopt"
	}
	return "investigate"
}

func diagnosticRunRoot(base, prefix string) (string, error) {
	if err := os.MkdirAll(base, 0700); err != nil {
		return "", err
	}
	return os.MkdirTemp(base, prefix)
}

func TestDiagnosticGuestVariants(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_GUEST_PROFILE_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_GUEST_PROFILE_ROOT for unsampled Temporal variant measurements")
	}
	helper, fuelHelper := os.Getenv("GOMAD_WASM_TEST_HELPER"), os.Getenv("GOMAD_WASM_FUEL_OFF_HELPER")
	for _, path := range []string{root, helper, fuelHelper} {
		if !filepath.IsAbs(path) {
			t.Fatal("variant root, diagnostic-profile helper and diagnostic-fuel-off helper must be absolute")
		}
	}
	if fileHash(t, helper) == fileHash(t, fuelHelper) {
		t.Fatal("fuel-off helper must have a distinct identity")
	}
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(repo, root)
	if err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		t.Fatal("raw variant evidence must be retained outside the repository")
	}
	baseRoot := root
	root, err = diagnosticRunRoot(baseRoot, "guest-variants-")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("retaining variant evidence in %s", root)
	admission := newDiagnosticAdmission(t, repo, root)
	admission.addHelper(t, helper, os.Getenv("GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION"), []string{"diagnostic-profile"})
	admission.addHelper(t, fuelHelper, os.Getenv("GOMAD_WASM_FUEL_OFF_HELPER_ADMISSION"), []string{"diagnostic-fuel-off", "diagnostic-profile"})
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", root)
	for _, directory := range []string{"modules", "variants"} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0700); err != nil {
			t.Fatal(err)
		}
	}
	module, optimized := filepath.Join(root, "modules", "variants-baseline.wasm"), filepath.Join(root, "modules", "variants-satconv-signext.wasm")
	for _, build := range []struct{ module, gowasm string }{{module, ""}, {optimized, "satconv,signext"}} {
		admission.buildModule(t, build.module, build.gowasm)
	}
	t.Setenv("GOWASM", "")
	commit, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	status, err := exec.Command("git", "-C", repo, "status", "--porcelain").Output()
	if err != nil {
		t.Fatal(err)
	}
	seeds := []uint64{0, 1000003, 2000006}
	const repetitions = 5
	variants := []struct{ name, gc, module, helper, revision, followup string }{
		{"baseline", "", module, helper, "", "none; diagnostic control"},
		{"gc200", "200", module, helper, "", "explicit GC environment and memory policy decision; requalify affected selections"},
		{"gc400", "400", module, helper, "", "explicit GC environment and memory policy decision; requalify affected selections"},
		{"gc-off", "off", module, helper, "", "bounded GOMEMLIMIT policy and GC decision; requalify affected selections"},
		{"gowasm", "", optimized, helper, "", "production preparation currently forces GOWASM=; adopting requires preparation and build identity changes plus requalification"},
		{"fuel-off", "", module, fuelHelper, PinnedEngine().Configuration + "-fuel-off", "explicit fuel/resource containment and engine identity decision; diagnostic helper is not a release configuration"},
	}
	evidence := filepath.Join(root, "variants")
	runAdmitted := func(name string, request Request, warm bool) (Result, diagnosticReport, string) {
		admission.check(t, evidence, name+"-before", request, true)
		defer admission.check(t, evidence, name+"-after", request, false)
		return runGuestVariant(t, evidence, name, request, warm)
	}
	var reports, recommendations []map[string]any
	for _, selection := range []struct{ name, run string }{
		{"workflow", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"},
		{"activity", "^TestActivityTestSuite$"},
	} {
		requests := make([]Request, len(variants))
		warmFailures := make([]string, len(variants))
		for index, variant := range variants {
			config := prototypeTemporalConfig(t, repo, "/workspace/tests", "-test.run="+selection.run, "-persistenceType=sql", "-persistenceDriver=sqlite")
			request := prototypeRequest(t, variant.module, config, 0, 30*time.Minute)
			request.HelperPath, request.HelperSHA256 = variant.helper, fileHash(t, variant.helper)
			request.diagnosticConfigurationRevision = variant.revision
			request.DigestTranscript = false
			if variant.gc != "" {
				request.Config.Environment = append(request.Config.Environment, "GOGC="+variant.gc)
				if variant.gc == "off" {
					request.Config.Environment = append(request.Config.Environment, "GOMEMLIMIT="+strconv.FormatUint(request.MemoryBytes/4*3, 10)+"B")
				}
			}
			requests[index] = request
			warm := request
			warm.Config.Args = slices.Clone(request.Config.Args)
			for argIndex, argument := range warm.Config.Args {
				if strings.HasPrefix(argument, "-test.run=") {
					warm.Config.Args[argIndex] = "-test.run=^$"
				}
			}
			_, _, warmFailures[index] = runAdmitted(selection.name+"-"+variant.name+"-prepare", warm, false)
		}
		decisions := make([][]string, len(variants))
		for _, seed := range seeds {
			execute, residual := make([][]uint64, len(variants)), make([][]uint64, len(variants))
			failed, identical := make([]bool, len(variants)), make([]bool, len(variants))
			for index := range identical {
				identical[index] = true
			}
			var reference, maskedReference variantEvidence
			var baselineSeen bool
			var runs []map[string]any
			for repetition := 0; repetition < repetitions; repetition++ {
				// Rotate candidates to reduce a fixed ordering bias; baseline runs first.
				for slot := range variants {
					index := 0
					if slot > 0 {
						index = 1 + (slot-1+repetition)%(len(variants)-1)
					}
					variant, request := variants[index], requests[index]
					request.Runtime.Seed = seed
					name := fmt.Sprintf("%s-%s-seed%d-rep%d", selection.name, variant.name, seed, repetition)
					result, profile, failure := runAdmitted(name, request, true)
					if warmFailures[index] != "" {
						failure = "preparation: " + warmFailures[index] + "; " + failure
					}
					run := map[string]any{"variant": variant.name, "repetition": repetition, "identities": variantIdentities(request, result), "failure": failure, "termination": result.Termination, "exit_code": result.ExitCode, "memory_bytes": request.MemoryBytes, "peak_memory_bytes": result.PeakMemoryBytes, "environment": request.Config.Environment, "profile": profile, "status": "failed"}
					if failure != "" {
						failed[index] = true
					} else {
						projected, projectionErr := projectVariantEvidence(result, variant.gc != "")
						if projectionErr != nil {
							failed[index], run["failure"] = true, projectionErr.Error()
						} else {
							if index == 0 && !baselineSeen {
								reference = projected
								maskedReference, projectionErr = projectVariantEvidence(result, true)
								if projectionErr != nil {
									t.Fatal(projectionErr)
								}
								baselineSeen = true
							}
							before := reference
							if variant.gc != "" {
								before = maskedReference
							}
							matched := baselineSeen && reflect.DeepEqual(before, projected)
							identical[index] = identical[index] && matched
							if matched {
								run["status"] = "identical"
							} else if !baselineSeen {
								run["status"] = "comparison-unavailable"
							} else {
								run["status"] = "evidence-changing"
							}
							data, err := json.Marshal(projected)
							if err != nil {
								t.Fatal(err)
							}
							run["behavior_projection_sha256"] = digest(data)
							execute[index] = append(execute[index], profile.Phases["execute"].WallNanos)
							residual[index] = append(residual[index], profile.Residual.ExecuteWallNanos)
						}
					}
					runs = append(runs, run)
				}
			}
			for index, variant := range variants {
				decision := variantRecommendation(execute[0], execute[index], failed[0] || failed[index], identical[0] && identical[index])
				if index == 0 {
					decision = "control"
				}
				decisions[index] = append(decisions[index], decision)
				reports = append(reports, map[string]any{"selection": selection.name, "test_run": selection.run, "seed": seed, "variant": variant.name, "warm_repetitions_requested": repetitions, "warm_repetitions_measured": len(execute[index]), "execute_nanos": variantTiming(execute[index]), "residual_nanos": variantTiming(residual[index]), "failed": failed[index], "evidence_identical": !failed[0] && !failed[index] && identical[0] && identical[index], "recommendation": decision})
			}
			temporalJSON(t, filepath.Join(evidence, fmt.Sprintf("%s-seed%d-runs.json", selection.name, seed)), runs)
		}
		for index, variant := range variants {
			decision := "adopt"
			if slices.Contains(decisions[index], "reject") {
				decision = "reject"
			} else if slices.Contains(decisions[index], "investigate") {
				decision = "investigate"
			}
			if index == 0 {
				decision = "control"
			}
			recommendations = append(recommendations, map[string]any{"selection": selection.name, "variant": variant.name, "recommendation": decision, "followup": variant.followup})
		}
	}
	cpuinfo, requiredHostMeasured := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	temporalJSON(t, filepath.Join(root, "guest-variants-report.json"), map[string]any{
		"host_admission": admission.Receipt, "host_admission_sha256": admission.Inputs[admission.Receipt], "module_build_receipts": admission.ModuleReceipts,
		"helper_admissions": map[string]string{helper: admission.Helpers[helper].Receipt, fuelHelper: admission.Helpers[fuelHelper].Receipt},
		"diagnostic_only":   true, "source_commit": strings.TrimSpace(string(commit)), "source_dirty": len(status) != 0,
		"host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "go_version": runtime.Version(), "seeds": seeds, "warm_repetitions": repetitions,
		"required_host": "linux/arm64", "required_host_measured": requiredHostMeasured, "developmental": !requiredHostMeasured, "host_cpuinfo": cpuinfo,
		"measurements": reports, "recommendations": recommendations,
		"build_settings": map[string]string{"GOOS": "wasip1", "GOARCH": "wasm", "GOEXPERIMENT": "nogreenteagc", "baseline_GOWASM": "", "variant_GOWASM": "satconv,signext"},
		"scope":          "unsampled diagnostic-profile timing; full transcript retention for exact behavior comparisons; residual includes guest plus Wasmtime/trampoline work, not guest-only CPU",
		"equality":       "exact choice, WASI, network, stdout/stderr, outcome and virtual time; only GC environ reply outputs masked; module/environment/helper/engine identities separate",
		"decision_rule":  "adopt only when all seeds have identical successful evidence and median execute gain strictly exceeds both baseline and candidate run-to-run ranges; adoption remains a follow-up decision",
		"rerun":          map[string]any{"working_directory": filepath.Join(repo, "tools", "gomad"), "environment": map[string]string{"GOMAD_WASM_GUEST_PROFILE_ROOT": baseRoot, "GOMAD_WASM_TEST_HELPER": helper, "GOMAD_WASM_FUEL_OFF_HELPER": fuelHelper, "GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION": admission.Helpers[helper].Receipt, "GOMAD_WASM_FUEL_OFF_HELPER_ADMISSION": admission.Helpers[fuelHelper].Receipt, "GOMAD3_STOCK_GO": os.Getenv("GOMAD3_STOCK_GO")}, "command": "env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -timeout 180m -run '^TestDiagnosticGuestVariants$' ./internal/backends/wasm/wasi"},
	})
}

func runGuestVariant(t *testing.T, root, name string, request Request, warm bool) (Result, diagnosticReport, string) {
	t.Helper()
	result, runErr := matchedPrepareAdmitted(runtime.GOOS, "/proc/meminfo", os.Getenv("GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD"), func() (Result, error) {
		return Run(t.Context(), request)
	})
	temporalWrite(t, filepath.Join(root, name+"-canonical.jsonl"), result.Transcript)
	temporalWrite(t, filepath.Join(root, name+"-choice.bin"), result.ChoiceTrace.Bytes)
	temporalWrite(t, filepath.Join(root, name+"-network.jsonl"), result.NetworkTranscript)
	temporalWrite(t, filepath.Join(root, name+"-helper-stderr.txt"), result.helperStderr)
	temporalWrite(t, filepath.Join(root, name+"-guest-stdout.txt"), result.Stdout)
	temporalWrite(t, filepath.Join(root, name+"-guest-stderr.txt"), result.Stderr)
	profile, profileErr := decodeDiagnosticReport(result.helperStderr)
	if runErr != nil {
		return result, profile, runErr.Error()
	}
	if result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 {
		return result, profile, fmt.Sprintf("termination=%s exit=%v message=%s", result.Termination, result.ExitCode, result.Message)
	}
	if profileErr != nil {
		return result, profile, profileErr.Error()
	}
	if result.Telemetry == nil || warm && (!result.Telemetry.CacheHit || result.Telemetry.CompileNanos != 0) {
		return result, profile, "missing telemetry or warm cache miss"
	}
	return result, profile, ""
}

var guestProfileBuckets = strings.Fields("gc scheduler allocation test server runtime stdlib other")

type firefoxGuestProfile struct {
	Meta struct {
		Interval float64              `json:"interval"`
		Guest    guestProfileSampling `json:"gomadGuestProfile"`
	} `json:"meta"`
	Threads []firefoxGuestThread `json:"threads"`
}

type guestProfileSampling struct {
	MaxSamples     uint64 `json:"maxSamples"`
	Samples        uint64 `json:"samples"`
	Truncated      bool   `json:"truncated"`
	IntervalMicros uint64 `json:"intervalMicros"`
}

type firefoxGuestThread struct {
	Samples struct {
		Length     int     `json:"length"`
		Stack      []*int  `json:"stack"`
		Weight     []int64 `json:"weight"`
		WeightType string  `json:"weightType"`
	} `json:"samples"`
	StackTable struct {
		Length int    `json:"length"`
		Frame  []int  `json:"frame"`
		Prefix []*int `json:"prefix"`
	} `json:"stackTable"`
	FrameTable struct {
		Length int   `json:"length"`
		Func   []int `json:"func"`
	} `json:"frameTable"`
	FuncTable struct {
		Length int   `json:"length"`
		Name   []int `json:"name"`
	} `json:"funcTable"`
	StringArray []string `json:"stringArray"`
}

type guestProfileAttribution struct {
	SampleRows       uint64            `json:"sample_rows"`
	GuestWeight      uint64            `json:"guest_sample_weight"`
	EmptyStackWeight uint64            `json:"empty_stack_weight"`
	Buckets          map[string]uint64 `json:"self_sample_weight"`
}

func guestFrameBucket(name string) string {
	name = strings.TrimPrefix(name, "$")
	for _, prefix := range strings.Fields("runtime.gcBgMarkWorker runtime.gcDrain runtime.scanobject runtime.markroot runtime.gcAssistAlloc runtime.bgsweep runtime.sweepone runtime.wbBufFlush runtime.gcWriteBarrier runtime.bulkBarrier runtime.writebarrier") {
		if strings.HasPrefix(name, prefix) {
			return "gc"
		}
	}
	for _, function := range strings.Fields("runtime.schedule runtime.findRunnable runtime.gopark runtime.park_m runtime.goready runtime.mcall runtime.gogo runtime.handleEvent runtime.beforeIdle wasm_pc_f_loop runtime.wasm_pc_f_loop") {
		if name == function || strings.HasPrefix(name, function+".") || strings.HasPrefix(name, function+"(") {
			return "scheduler"
		}
	}
	for _, prefix := range strings.Fields("runtime.mallocgc runtime.makeslice runtime.growslice runtime.newobject runtime.newarray runtime.makemap runtime.mapassign") {
		if strings.HasPrefix(name, prefix) {
			return "allocation"
		}
	}
	if strings.HasPrefix(name, "go.temporal.io/server/tests.") || strings.HasPrefix(name, "go.temporal.io/server/tests/") {
		return "test"
	}
	if strings.HasPrefix(name, "go.temporal.io/server/") {
		return "server"
	}
	if strings.HasPrefix(name, "runtime.") || strings.HasPrefix(name, "runtime/") || strings.HasPrefix(name, "internal/runtime/") {
		return "runtime"
	}
	if strings.Contains(name, ".") {
		first, _, _ := strings.Cut(name, "/")
		if !strings.Contains(name, "/") {
			first, _, _ = strings.Cut(name, ".")
		}
		if first != "" && first != "main" && !strings.ContainsAny(first, ". <>[]:") {
			return "stdlib"
		}
	}
	return ""
}

func guestStackBucket(leafToRoot []string) string {
	for _, name := range leafToRoot {
		if bucket := guestFrameBucket(name); bucket != "" {
			return bucket
		}
	}
	return "other"
}

func decodeGuestAttribution(data []byte) (firefoxGuestProfile, guestProfileAttribution, error) {
	var profile firefoxGuestProfile
	attribution := guestProfileAttribution{Buckets: make(map[string]uint64, len(guestProfileBuckets))}
	for _, bucket := range guestProfileBuckets {
		attribution.Buckets[bucket] = 0
	}
	if err := json.Unmarshal(data, &profile); err != nil {
		return profile, attribution, err
	}
	if len(profile.Threads) == 0 || profile.Meta.Interval <= 0 {
		return profile, attribution, fmt.Errorf("guest profile has no threads or sampling interval")
	}
	for threadIndex, thread := range profile.Threads {
		if thread.Samples.Length != len(thread.Samples.Stack) || thread.Samples.Weight != nil && len(thread.Samples.Weight) != thread.Samples.Length || thread.Samples.WeightType != "samples" || thread.StackTable.Length != len(thread.StackTable.Frame) || thread.StackTable.Length != len(thread.StackTable.Prefix) || thread.FrameTable.Length != len(thread.FrameTable.Func) || thread.FuncTable.Length != len(thread.FuncTable.Name) {
			return profile, attribution, fmt.Errorf("thread %d: inconsistent guest profile tables or weight type", threadIndex)
		}
		for sampleIndex, stack := range thread.Samples.Stack {
			weight := int64(1)
			if thread.Samples.Weight != nil {
				weight = thread.Samples.Weight[sampleIndex]
			}
			if weight < 0 {
				return profile, attribution, fmt.Errorf("thread %d sample %d: negative sample weight", threadIndex, sampleIndex)
			}
			attribution.SampleRows++
			var frames []string
			for depth := 0; stack != nil; depth++ {
				if *stack < 0 || *stack >= thread.StackTable.Length || depth >= thread.StackTable.Length {
					return profile, attribution, fmt.Errorf("thread %d sample %d: invalid or cyclic stack", threadIndex, sampleIndex)
				}
				frame := thread.StackTable.Frame[*stack]
				if frame < 0 || frame >= thread.FrameTable.Length {
					return profile, attribution, fmt.Errorf("thread %d: invalid frame index", threadIndex)
				}
				function := thread.FrameTable.Func[frame]
				if function < 0 || function >= thread.FuncTable.Length {
					return profile, attribution, fmt.Errorf("thread %d: invalid function index", threadIndex)
				}
				name := thread.FuncTable.Name[function]
				if name < 0 || name >= len(thread.StringArray) {
					return profile, attribution, fmt.Errorf("thread %d: invalid function name index", threadIndex)
				}
				frames = append(frames, thread.StringArray[name])
				stack = thread.StackTable.Prefix[*stack]
			}
			if len(frames) == 0 {
				if math.MaxUint64-attribution.EmptyStackWeight < uint64(weight) {
					return profile, attribution, fmt.Errorf("empty stack weight overflow")
				}
				attribution.EmptyStackWeight += uint64(weight)
				continue
			}
			if math.MaxUint64-attribution.GuestWeight < uint64(weight) {
				return profile, attribution, fmt.Errorf("guest sample weight overflow")
			}
			attribution.GuestWeight += uint64(weight)
			attribution.Buckets[guestStackBucket(frames)] += uint64(weight)
		}
	}
	return profile, attribution, nil
}

func TestDiagnosticGuestProfile(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_GUEST_PROFILE_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_GUEST_PROFILE_ROOT for Temporal guest stack profiling")
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(os.Getenv("GOMAD_WASM_TEST_HELPER")) {
		t.Fatal("guest profile root and GOMAD_WASM_TEST_HELPER must be absolute")
	}
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(repo, root)
	if err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		t.Fatal("raw guest profiles must be retained outside the repository")
	}
	baseRoot := root
	root, err = diagnosticRunRoot(baseRoot, "guest-profile-")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("retaining guest profiles in %s", root)
	admission := newDiagnosticAdmission(t, repo, root)
	helper := os.Getenv("GOMAD_WASM_TEST_HELPER")
	admission.addHelper(t, helper, os.Getenv("GOMAD_WASM_GUEST_HELPER_ADMISSION"), []string{"diagnostic-profile", "guest-profile"})
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", root)
	for _, directory := range []string{"modules", "evidence"} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0700); err != nil {
			t.Fatal(err)
		}
	}
	module := filepath.Join(root, "modules", "tests.wasm")
	admission.buildModule(t, module, "")
	evidence := filepath.Join(root, "evidence")
	runAdmitted := func(name string, request Request, warm bool) Result {
		admission.check(t, evidence, name+"-before", request, true)
		defer admission.check(t, evidence, name+"-after", request, false)
		return runDiagnosticProfile(t, evidence, name, request, true, warm)
	}
	persistence := []string{"-persistenceType=sql", "-persistenceDriver=sqlite"}
	warm := prototypeRequest(t, module, prototypeTemporalConfig(t, repo, "/workspace/tests", append([]string{"-test.run=^$"}, persistence...)...), 0, 30*time.Minute)
	warm.diagnosticConfigurationRevision = PinnedEngine().Configuration + "-guest-profile"
	runAdmitted("guest-profile-prepare", warm, false)
	commit, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	status, err := exec.Command("git", "-C", repo, "status", "--porcelain").Output()
	if err != nil {
		t.Fatal(err)
	}
	var runs []map[string]any
	for _, selection := range []struct{ name, run string }{
		{"workflow", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"},
		{"activity", "^TestActivityTestSuite$"},
	} {
		for _, seed := range []uint64{0, 1000003, 2000006} {
			name := selection.name + "-seed" + stringSeed(seed)
			config := prototypeTemporalConfig(t, repo, "/workspace/tests", append([]string{"-test.run=" + selection.run}, persistence...)...)
			request := prototypeRequest(t, module, config, seed, 30*time.Minute)
			request.diagnosticConfigurationRevision = warm.diagnosticConfigurationRevision
			before := runAdmitted(name+"-unsampled", request, true)
			request.guestProfile = &guestProfileConfig{Path: filepath.Join(evidence, name+"-guest.firefox.json"), IntervalMicros: 1000, MaxSamples: 10000}
			after := runAdmitted(name+"-sampled", request, true)
			requireDiagnosticEvidence(t, before, after)
			if before.VirtualTimeElapsedNanos != after.VirtualTimeElapsedNanos || !bytes.Equal(before.DiagnosticTrace.Bytes, after.DiagnosticTrace.Bytes) {
				t.Fatal("profiling changed virtual time or runtime diagnostics")
			}
			data, err := os.ReadFile(request.guestProfile.Path)
			if err != nil {
				t.Fatal(err)
			}
			profile, attribution, err := decodeGuestAttribution(data)
			if err != nil {
				t.Fatal(err)
			}
			if attribution.GuestWeight == 0 || request.guestProfile.MaxSamples != profile.Meta.Guest.MaxSamples || request.guestProfile.IntervalMicros != profile.Meta.Guest.IntervalMicros || profile.Meta.Guest.Samples > profile.Meta.Guest.MaxSamples || profile.Meta.Guest.Samples != attribution.SampleRows {
				t.Fatalf("profile samples or metadata mismatch: metadata=%+v attribution=%+v", profile.Meta.Guest, attribution)
			}
			run := map[string]any{
				"selection": selection.name, "test_run": selection.run, "seed": seed,
				"helper_sha256": request.HelperSHA256, "module_sha256": request.ModuleSHA256,
				"engine_revision":  request.diagnosticConfigurationRevision,
				"runtime_identity": request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey,
				"model_sha256":     ImplementationSHA256(), "evidence_identical": true,
				"profile": request.guestProfile.Path, "profile_sha256": digest(data),
				"sampling": profile.Meta.Guest, "attribution": attribution,
			}
			temporalJSON(t, filepath.Join(evidence, name+"-attribution.json"), run)
			runs = append(runs, run)
			t.Logf("%s: weighted guest samples=%d buckets=%v truncated=%t", name, attribution.GuestWeight, attribution.Buckets, profile.Meta.Guest.Truncated)
		}
	}
	cpuinfo, requiredHostMeasured := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	temporalJSON(t, filepath.Join(root, "guest-attribution-report.json"), map[string]any{
		"host_admission": admission.Receipt, "host_admission_sha256": admission.Inputs[admission.Receipt], "module_build_receipts": admission.ModuleReceipts,
		"helper_admission": admission.Helpers[helper].Receipt,
		"diagnostic_only":  true, "source_commit": strings.TrimSpace(string(commit)), "source_dirty": len(status) != 0,
		"host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "go_version": runtime.Version(), "runs": runs,
		"required_host": "linux/arm64", "required_host_measured": requiredHostMeasured, "developmental": !requiredHostMeasured, "host_cpuinfo": cpuinfo,
		"scope":  "weighted guest stack samples; epoch function-entry/loop bias; host-call ticks excluded; no timestamp-gap charge; sampled wall times are not baselines",
		"limits": "scheduler-frame share does not quantify all unwind/rewind work; fuel checks are distributed across ordinary guest frames",
		"rerun": map[string]any{
			"working_directory": filepath.Join(repo, "tools", "gomad"),
			"environment": map[string]string{
				"GOMAD_WASM_GUEST_PROFILE_ROOT":     baseRoot,
				"GOMAD_WASM_TEST_HELPER":            os.Getenv("GOMAD_WASM_TEST_HELPER"),
				"GOMAD_WASM_GUEST_HELPER_ADMISSION": admission.Helpers[helper].Receipt,
				"GOMAD3_STOCK_GO":                   os.Getenv("GOMAD3_STOCK_GO"),
			},
			"command": "env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -timeout 60m -run '^TestDiagnosticGuestProfile$' ./internal/backends/wasm/wasi",
		},
	})
}

func TestDiagnosticGuestFrameBuckets(t *testing.T) {
	for _, test := range []struct{ bucket, names string }{
		{"gc", "runtime.gcBgMarkWorker runtime.gcDrainN runtime.scanobject runtime.markrootBlock runtime.gcAssistAlloc1 runtime.bgsweep runtime.sweepone runtime.wbBufFlush1 runtime.gcWriteBarrier runtime.bulkBarrierPreWrite runtime.writebarrierptr"},
		{"scheduler", "runtime.schedule runtime.findRunnable runtime.gopark runtime.park_m runtime.goready runtime.mcall runtime.gogo runtime.handleEvent runtime.beforeIdle wasm_pc_f_loop runtime.wasm_pc_f_loop"},
		{"allocation", "runtime.mallocgcSmallScanNoHeader runtime.makeslice runtime.growslice runtime.newobject runtime.newarray runtime.makemap runtime.mapassign"},
		{"test", "go.temporal.io/server/tests.(*ActivityTestSuite).TestActivity go.temporal.io/server/tests/testcore.NewTestServer"},
		{"server", "go.temporal.io/server/service/history.(*Engine).Execute"},
		{"runtime", "runtime.memmove runtime/atomic.Store internal/runtime/maps.(*Map).get"},
		{"stdlib", "encoding/json.Marshal net/http.(*Client).Do testing.tRunner reflect.Value.Call sync.(*Mutex).Lock"},
		{"other", "github.com/stretchr/testify/require.NoError main.main type:.eq.go.temporal.io/server.Task <wasm-function[1]>"},
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
