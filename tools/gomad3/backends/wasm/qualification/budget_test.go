package qualification

import (
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/hostexec"
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
	if got := helperPeakRSS(own, inherited); got != own.PeakRSSBytes {
		t.Fatalf("helper peak %d, want own image high-water %d", got, own.PeakRSSBytes)
	}
	for _, telemetry := range []*wasi.Telemetry{nil, {}, {ResourceUsageAvailable: true}, {PeakRSSBytes: 1}} {
		if got := helperPeakRSS(telemetry, inherited); got != inherited.PeakRSSBytes {
			t.Fatalf("unavailable telemetry %+v: got %d, want wait4 upper bound", telemetry, got)
		}
	}
	if got := helperPeakRSS(nil, nil); got != 0 {
		t.Fatalf("missing measurements produced %d", got)
	}
}
