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
	"runtime"
	"strings"
	"testing"
	"time"
)

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
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", root)
	repo, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(repo, root)
	if err != nil || relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		t.Fatal("raw guest profiles must be retained outside the repository")
	}
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
	temporalJSON(t, filepath.Join(root, "guest-attribution-report.json"), map[string]any{
		"diagnostic_only": true, "source_commit": strings.TrimSpace(string(commit)), "source_dirty": len(status) != 0,
		"host_goos": runtime.GOOS, "host_goarch": runtime.GOARCH, "go_version": runtime.Version(), "runs": runs,
		"required_host": "darwin/arm64", "required_host_measured": runtime.GOOS == "darwin" && runtime.GOARCH == "arm64",
		"scope":  "weighted guest stack samples; epoch function-entry/loop bias; host-call ticks excluded; no timestamp-gap charge; sampled wall times are not baselines",
		"limits": "scheduler-frame share does not quantify all unwind/rewind work; fuel checks are distributed across ordinary guest frames",
		"rerun": map[string]any{
			"working_directory": filepath.Join(repo, "tools", "gomad3"),
			"environment": map[string]string{
				"GOMAD_WASM_GUEST_PROFILE_ROOT": root,
				"GOMAD_WASM_TEST_HELPER":        os.Getenv("GOMAD_WASM_TEST_HELPER"),
				"GOMAD3_STOCK_GO":               os.Getenv("GOMAD3_STOCK_GO"),
			},
			"command": "env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -timeout 60m -run '^TestDiagnosticGuestProfile$' ./backends/wasm/wasi",
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
