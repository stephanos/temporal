package qualification

import (
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	hostexec "go.temporal.io/server/tools/gomad3/internal/host/exec"
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
