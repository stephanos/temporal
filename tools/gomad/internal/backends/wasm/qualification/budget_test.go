package qualification

import (
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
)

func TestBudgetRejectsMeasuredOverrunAndMissingResources(t *testing.T) {
	samples := []Sample{{Phase: "execute", Incarnation: "a/1", CompileCacheHit: true, HelperWallNanos: uint64(time.Second), HostPeakRSSBytes: 3 << 30}, {Phase: "execute", Incarnation: "b/1", CompileCacheHit: true, HelperWallNanos: uint64(time.Second), HostPeakRSSBytes: 2 << 30}, {Phase: "replay", Incarnation: "a/1", HelperWallNanos: 1, HostPeakRSSBytes: 6 << 30}}
	commands := []backend.CompilerCommandMetrics{{ResourceUsage: &hostexec.ResourceUsage{PeakRSSBytes: 5 << 30}}}
	observed, err := EvaluateBudget(samples, commands, uint64(time.Minute), 2, 1<<20)
	if err != nil || !observed.WithinBudget || observed.ConservativeResidentHelperPeakBytes != 5<<30 || !observed.ColdHelperExceededOriginal4GiBLimit {
		t.Fatalf("measured budget: %+v %v", observed, err)
	}
	samples[0].HostPeakRSSBytes = 7 << 30
	observed, err = EvaluateBudget(samples, commands, uint64(11*time.Minute), 2, 1<<20)
	if err != nil || observed.WithinBudget || len(observed.Violations) != 3 {
		t.Fatalf("over-budget accepted: %+v %v", observed, err)
	}
	commands[0].ResourceUsage = nil
	if _, err := EvaluateBudget(samples, commands, uint64(time.Minute), 2, 1<<20); err == nil {
		t.Fatal("missing compiler resource measurements accepted")
	}
}

func TestBudgetOperatorOverrunCannotPass(t *testing.T) {
	samples := []Sample{{Phase: "execute", CompileCacheHit: true, HelperWallNanos: 1, HostPeakRSSBytes: 1}}
	commands := []backend.CompilerCommandMetrics{{ResourceUsage: &hostexec.ResourceUsage{PeakRSSBytes: 1}}}
	budget, err := EvaluateBudget(samples, commands, 1, 1, 2<<30+1)
	if err != nil || budget.WithinBudget {
		t.Fatalf("operator peak overrun accepted: %+v %v", budget, err)
	}
	if _, err := EvaluateBudget(samples, commands, 1, 1); err == nil {
		t.Fatal("missing operator measurements accepted")
	}
}

func TestBudgetClusterPeakCoversResidentsOnce(t *testing.T) {
	commands := []backend.CompilerCommandMetrics{{ResourceUsage: &hostexec.ResourceUsage{PeakRSSBytes: 1}}}
	warm := Sample{Iteration: 1, ClusterHelper: true, ResidentInstances: []string{"a", "b"}, CompileCacheHit: true, HelperWallNanos: 1, HostPeakRSSBytes: 7 << 30}
	cold := warm
	cold.Iteration = 2
	cold.CompileCacheHit, cold.HostPeakRSSBytes = false, 8<<30
	store := Sample{Iteration: 1, Incarnation: "a/1", GuestMemoryLimitBytes: 64, GuestPeakMemoryBytes: 64}
	other := store
	other.Incarnation = "b/1"
	coldStore, coldOther := store, other
	coldStore.Iteration, coldOther.Iteration = 2, 2
	samples := []Sample{warm, cold, store, other, coldStore, coldOther}
	for _, missing := range [][]Sample{samples[:2], samples[:3]} {
		if _, err := EvaluateBudget(missing, commands, 1, 2, 1); err == nil {
			t.Fatal("cluster helper RSS admitted without every store's memory peak")
		}
	}
	budget, err := EvaluateBudget(samples, commands, 1, 2, 1)
	if err != nil || !budget.WithinBudget || budget.ConservativeResidentHelperPeakBytes != 8<<30 || budget.WarmHelperPeakRSSBytes != 7<<30 || budget.ColdHelperPeakRSSBytes != 8<<30 {
		t.Fatalf("cluster RSS counted per guest or held to single-helper cap: %+v %v", budget, err)
	}
	samples[1].HostPeakRSSBytes++
	budget, err = EvaluateBudget(samples, commands, 1, 2, 1)
	if err != nil || budget.WithinBudget || len(budget.Violations) != 1 {
		t.Fatalf("cluster resident cap not enforced: %+v %v", budget, err)
	}
	for name, mutate := range map[string]func([]Sample){
		"missing warm":       func(s []Sample) { s[0].CompileCacheHit = false },
		"missing resident":   func(s []Sample) { s[0].ResidentInstances = []string{"a"} },
		"missing residents":  func(s []Sample) { s[0].ResidentInstances = nil },
		"duplicate resident": func(s []Sample) { s[0].ResidentInstances = []string{"a", "a"} },
		"empty resident":     func(s []Sample) { s[0].ResidentInstances = []string{"a", ""} },
		"wrong resident":     func(s []Sample) { s[0].ResidentInstances = []string{"a", "c"} },
		"zero peak":          func(s []Sample) { s[0].HostPeakRSSBytes = 0 },
		"wait upper bound": func(s []Sample) {
			s[0].HostPeakRSSUpperBound = true
			s[0].HelperWaitPeakRSSBytes = s[0].HostPeakRSSBytes
		},
		"missing guest peak": func(s []Sample) { s[2].GuestPeakMemoryBytes = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			invalid := append([]Sample(nil), samples...)
			mutate(invalid)
			if _, err := EvaluateBudget(invalid, commands, 1, 2, 1); err == nil {
				t.Fatal("invalid cluster measurement accepted")
			}
		})
	}
	samples[1].HostPeakRSSBytes = 8 << 30
	samples[2].GuestPeakMemoryBytes++
	if budget, err := EvaluateBudget(samples, commands, 1, 2, 1); err != nil || budget.WithinBudget || len(budget.Violations) != 1 {
		t.Fatalf("store memory overrun accepted: %+v %v", budget, err)
	}
	samples[2].GuestPeakMemoryBytes = 64
	samples[2].AuthorizationToTerminalNanos = uint64(16 * time.Minute)
	if budget, err := EvaluateBudget(samples, commands, 1, 2, 1); err != nil || budget.LargestHelperWallNanos != uint64(16*time.Minute) || budget.WithinBudget {
		t.Fatalf("store execution overrun accepted: %+v %v", budget, err)
	}
}

func TestCostClusterHelperDoesNotDoubleCountStoreExecution(t *testing.T) {
	store := Sample{Phase: "execute", Incarnation: "a/1", ExecutionWallNanos: 2, GuestPeakMemoryBytes: 64, GuestMemoryLimitBytes: 64, VirtualTimeNanos: 8, TraceBytes: 3, OutcomeSHA256: record.HashBytes(nil)}
	store.CompileCacheHit, store.CPUNanos, store.HostPeakRSSBytes = true, 4, 7<<30
	helper := Sample{Phase: "execute", ClusterHelper: true, ResidentInstances: []string{"a"}, HelperWallNanos: 5, CPUNanos: 4, HostPeakRSSBytes: 7 << 30, CompileCacheHit: true}
	replayStore, replayHelper := store, helper
	replayStore.Phase, replayHelper.Phase = "replay", "replay"
	cost, err := Summarize([]Sample{store, helper, replayStore, replayHelper})
	if err != nil || cost.Executions != 1 || cost.Replays != 1 || cost.CPUNanos != 8 || cost.HelperCPUNanos != 8 || cost.HostPeakRSSBytes != 7<<30 || cost.ExecutionWallNanos != 2 || cost.GuestPeakMemoryBytes != 64 || cost.TraceBytes != 3 || cost.CompileCacheHits != 1 || cost.GuestVirtualToExecutionWallRatio != 4 {
		t.Fatalf("shared helper cost double counted or missing: %+v %v", cost, err)
	}
	for _, missing := range [][]Sample{{store, helper, replayStore}, {store, replayStore, replayHelper}} {
		if _, err := Summarize(missing); err == nil {
			t.Fatal("store phase admitted without its helper CPU and RSS")
		}
	}
	for name, mutate := range map[string]func(*Sample){
		"zero guest peak": func(s *Sample) { s.GuestPeakMemoryBytes = 0 },
		"over limit":      func(s *Sample) { s.GuestPeakMemoryBytes = 65 },
	} {
		t.Run(name, func(t *testing.T) {
			invalid := store
			mutate(&invalid)
			if _, err := Summarize([]Sample{invalid, helper}); err == nil {
				t.Fatal("invalid store memory measurement accepted")
			}
		})
	}
	if _, err := Summarize([]Sample{store}); err == nil {
		t.Fatal("store execution without helper CPU/RSS accepted")
	}
	for name, mutate := range map[string]func(*Sample){
		"missing CPU":  func(s *Sample) { s.CPUNanos = 0 },
		"missing wall": func(s *Sample) { s.HelperWallNanos = 0 },
		"missing RSS":  func(s *Sample) { s.HostPeakRSSBytes = 0 },
		"upper bound": func(s *Sample) {
			s.HostPeakRSSUpperBound, s.HelperWaitPeakRSSBytes = true, s.HostPeakRSSBytes
		},
	} {
		t.Run(name, func(t *testing.T) {
			invalid := helper
			mutate(&invalid)
			if _, err := Summarize([]Sample{store, invalid}); err == nil {
				t.Fatal("unmeasured cluster helper cost accepted")
			}
		})
	}
}

func TestHelperPeakRSSPrefersHelperImageHighWater(t *testing.T) {
	// Linux wait4 folds the operator's peak at execve into a vfork child.
	inherited := &hostexec.ResourceUsage{UserNanos: 1, SystemNanos: 1, PeakRSSBytes: 4924080128}
	own := &wasi.Telemetry{ResourceUsageAvailable: true, PeakRSSBytes: 1269350400}
	var sample Sample
	sample.recordHelperPeakRSS(own, inherited)
	if sample.HostPeakRSSBytes != own.PeakRSSBytes || sample.HostPeakRSSUpperBound || sample.HelperWaitPeakRSSBytes != inherited.PeakRSSBytes || sample.checkHelperPeakRSS() != nil {
		t.Fatalf("helper peak %+v, want own image high-water %d", sample, own.PeakRSSBytes)
	}
	for _, telemetry := range []*wasi.Telemetry{nil, {}, {ResourceUsageAvailable: true}, {PeakRSSBytes: 1}} {
		sample.recordHelperPeakRSS(telemetry, inherited)
		if sample.HostPeakRSSBytes != inherited.PeakRSSBytes || !sample.HostPeakRSSUpperBound || sample.checkHelperPeakRSS() != nil {
			t.Fatalf("unavailable telemetry %+v: got %+v, want marked wait4 upper bound", telemetry, sample)
		}
	}
	sample.recordHelperPeakRSS(nil, nil)
	if sample.HostPeakRSSBytes != 0 || sample.HostPeakRSSUpperBound {
		t.Fatalf("missing measurements produced %+v", sample)
	}
	// VmHWM <= ru_maxrss for the same process; a larger image figure is not trusted.
	sample.recordHelperPeakRSS(&wasi.Telemetry{ResourceUsageAvailable: true, PeakRSSBytes: inherited.PeakRSSBytes + 1}, inherited)
	if sample.checkHelperPeakRSS() == nil {
		t.Fatal("helper image peak above its wait figure accepted")
	}
}

func TestHelperUpperBoundsStayOutOfImagePeaks(t *testing.T) {
	commands := []backend.CompilerCommandMetrics{{ResourceUsage: &hostexec.ResourceUsage{PeakRSSBytes: 1 << 30}}}
	warm := Sample{Phase: "execute", Incarnation: "a/1", CompileCacheHit: true, HelperWallNanos: 1, ExecutionWallNanos: 1, CPUNanos: 1, GuestPeakMemoryBytes: 1, OutcomeSHA256: record.HashBytes(nil)}
	warm.recordHelperPeakRSS(&wasi.Telemetry{ResourceUsageAvailable: true, PeakRSSBytes: 1 << 30}, &hostexec.ResourceUsage{PeakRSSBytes: 3 << 30})
	cold := warm
	cold.Incarnation, cold.CompileCacheHit = "a/2", false
	cold.recordHelperPeakRSS(&wasi.Telemetry{ResourceUsageAvailable: true, PeakRSSBytes: 2 << 30}, &hostexec.ResourceUsage{PeakRSSBytes: 3 << 30})
	// A modeled crash or stop ends without telemetry; its wait figure includes
	// the operator's peak inherited at execve.
	crashed := warm
	crashed.Incarnation, crashed.CompileCacheHit = "a/3", false
	crashed.recordHelperPeakRSS(nil, &hostexec.ResourceUsage{PeakRSSBytes: 7 << 30})
	samples := []Sample{warm, cold, crashed}
	budget, err := EvaluateBudget(samples, commands, 1, 1, 1<<20)
	if err != nil || !budget.WithinBudget || budget.ColdHelperPeakRSSBytes != 2<<30 || budget.WarmHelperPeakRSSBytes != 1<<30 || budget.HelperPeakRSSBytes != 2<<30 || budget.HelperUpperBoundSamples != 1 || budget.HelperUpperBoundPeakRSSBytes != 7<<30 {
		t.Fatalf("upper bound counted as helper image peak: %+v %v", budget, err)
	}
	cost, err := Summarize(samples)
	if err != nil || cost.HostPeakRSSBytes != 2<<30 || cost.HelperPeakRSSBytes != 2<<30 || cost.HelperUpperBoundSamples != 1 || cost.HelperUpperBoundPeakRSSBytes != 7<<30 {
		t.Fatalf("upper bound counted as helper image cost: %+v %v", cost, err)
	}
	for _, mutate := range []func(*Sample){
		func(sample *Sample) { sample.HostPeakRSSBytes = 3<<30 + 1 },
		func(sample *Sample) { sample.HostPeakRSSUpperBound = true },
	} {
		tampered := append([]Sample{}, samples...)
		mutate(&tampered[1])
		if _, err := EvaluateBudget(tampered, commands, 1, 1, 1<<20); err == nil {
			t.Fatalf("inconsistent helper peak accepted by budget: %+v", tampered[1])
		}
		if _, err := Summarize(tampered); err == nil {
			t.Fatalf("inconsistent helper peak accepted by cost: %+v", tampered[1])
		}
	}
}
