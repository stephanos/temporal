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
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	hostfs "go.temporal.io/server/tools/gomad3/internal/host/fs"
)

const diagnosticHostCPUInfoLimit = 1 << 20

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
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", root)
	for _, directory := range []string{"modules", "variants"} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0700); err != nil {
			t.Fatal(err)
		}
	}
	module, optimized := filepath.Join(root, "modules", "variants-baseline.wasm"), filepath.Join(root, "modules", "variants-satconv-signext.wasm")
	for _, build := range []struct{ module, gowasm string }{{module, ""}, {optimized, "satconv,signext"}} {
		t.Setenv("GOWASM", build.gowasm)
		_, seconds := prototypeBuild(t, repo, "test", "./tests", build.module, "disable_grpc_modules", "test_dep", "sqlite3_dotlk")
		t.Logf("built GOWASM=%q guest in %.1fs", build.gowasm, seconds)
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
			_, _, warmFailures[index] = runGuestVariant(t, evidence, selection.name+"-"+variant.name+"-prepare", warm, false)
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
					result, profile, failure := runGuestVariant(t, evidence, name, request, true)
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
		"diagnostic_only": true, "source_commit": strings.TrimSpace(string(commit)), "source_dirty": len(status) != 0,
		"host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "go_version": runtime.Version(), "seeds": seeds, "warm_repetitions": repetitions,
		"required_host": "linux/arm64", "required_host_measured": requiredHostMeasured, "developmental": !requiredHostMeasured, "host_cpuinfo": cpuinfo,
		"measurements": reports, "recommendations": recommendations,
		"build_settings": map[string]string{"GOOS": "wasip1", "GOARCH": "wasm", "GOEXPERIMENT": "nogreenteagc", "baseline_GOWASM": "", "variant_GOWASM": "satconv,signext"},
		"scope":          "unsampled diagnostic-profile timing; full transcript retention for exact behavior comparisons; residual includes guest plus Wasmtime/trampoline work, not guest-only CPU",
		"equality":       "exact choice, WASI, network, stdout/stderr, outcome and virtual time; only GC environ reply outputs masked; module/environment/helper/engine identities separate",
		"decision_rule":  "adopt only when all seeds have identical successful evidence and median execute gain strictly exceeds both baseline and candidate run-to-run ranges; adoption remains a follow-up decision",
		"rerun":          map[string]any{"working_directory": filepath.Join(repo, "tools", "gomad3"), "environment": map[string]string{"GOMAD_WASM_GUEST_PROFILE_ROOT": baseRoot, "GOMAD_WASM_TEST_HELPER": helper, "GOMAD_WASM_FUEL_OFF_HELPER": fuelHelper, "GOMAD3_STOCK_GO": os.Getenv("GOMAD3_STOCK_GO")}, "command": "env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -timeout 180m -run '^TestDiagnosticGuestVariants$' ./internal/backends/wasm/wasi"},
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
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", root)
	for _, directory := range []string{"modules", "evidence"} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0700); err != nil {
			t.Fatal(err)
		}
	}
	module := filepath.Join(root, "modules", "tests.wasm")
	_, seconds := prototypeBuild(t, repo, "test", "./tests", module, "disable_grpc_modules", "test_dep", "sqlite3_dotlk")
	t.Logf("built Temporal guest in %.1fs", seconds)
	evidence := filepath.Join(root, "evidence")
	persistence := []string{"-persistenceType=sql", "-persistenceDriver=sqlite"}
	warm := prototypeRequest(t, module, prototypeTemporalConfig(t, repo, "/workspace/tests", append([]string{"-test.run=^$"}, persistence...)...), 0, 30*time.Minute)
	warm.diagnosticConfigurationRevision = PinnedEngine().Configuration + "-guest-profile"
	runDiagnosticProfile(t, evidence, "guest-profile-prepare", warm, true, false)
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
			before := runDiagnosticProfile(t, evidence, name+"-unsampled", request, true, true)
			request.guestProfile = &guestProfileConfig{Path: filepath.Join(evidence, name+"-guest.firefox.json"), IntervalMicros: 1000, MaxSamples: 10000}
			after := runDiagnosticProfile(t, evidence, name+"-sampled", request, true, true)
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
		"diagnostic_only": true, "source_commit": strings.TrimSpace(string(commit)), "source_dirty": len(status) != 0,
		"host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "go_version": runtime.Version(), "runs": runs,
		"required_host": "linux/arm64", "required_host_measured": requiredHostMeasured, "developmental": !requiredHostMeasured, "host_cpuinfo": cpuinfo,
		"scope":  "weighted guest stack samples; epoch function-entry/loop bias; host-call ticks excluded; no timestamp-gap charge; sampled wall times are not baselines",
		"limits": "scheduler-frame share does not quantify all unwind/rewind work; fuel checks are distributed across ordinary guest frames",
		"rerun": map[string]any{
			"working_directory": filepath.Join(repo, "tools", "gomad3"),
			"environment": map[string]string{
				"GOMAD_WASM_GUEST_PROFILE_ROOT": baseRoot,
				"GOMAD_WASM_TEST_HELPER":        os.Getenv("GOMAD_WASM_TEST_HELPER"),
				"GOMAD3_STOCK_GO":               os.Getenv("GOMAD3_STOCK_GO"),
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
