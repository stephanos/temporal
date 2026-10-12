package qualification

import (
	"errors"
	"math"

	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/hostexec"
	"go.temporal.io/server/tools/gomad3/record"
)

// Sample contains observed values only. ReplayableBugSHA256 is populated only
// after a retained failure artifact actually matches on replay. CPU is combined
// helper user+system time, including engine work, not inferred guest CPU.
type Sample struct {
	AuthorizationToTerminalNanos                                                                 uint64
	Iteration                                                                                    uint64
	Incarnation                                                                                  string
	WASITraceBytes, ChoiceTraceBytes, DiagnosticTraceBytes, StorageTraceBytes, NetworkTraceBytes uint64
	HelperWallNanos, EnginePrepareNanos, CacheLoadNanos, CacheWriteNanos                         uint64
	Phase                                                                                        string
	PreparationWallNanos, EngineCompileNanos, InstantiationNanos, ExecutionWallNanos             uint64
	CPUNanos, GuestPeakMemoryBytes, HostPeakRSSBytes, FuelUsed, TraceBytes, VirtualTimeNanos     uint64
	CompileCacheHit                                                                              bool
	OutcomeSHA256, ReplayableBugSHA256                                                           record.SHA256
}

type Cost struct {
	CompilerCPUNanos, HelperCPUNanos, CompilerPeakRSSBytes, HelperPeakRSSBytes                 uint64
	InitialPreparationWallNanos, RepeatedPreparationWallNanos                                  uint64
	OperatorCPUNanos, OperatorPeakRSSBytes                                                     uint64
	Executions, Replays, DistinctOutcomes, ReplayableBugs                                      uint64
	PreparationWallNanos, EngineCompileNanos, InstantiationNanos, ExecutionWallNanos, CPUNanos uint64
	GuestPeakMemoryBytes, HostPeakRSSBytes, FuelUsed, TraceBytes, CompileCacheHits             uint64
	VirtualToWallRatio, ReplayableBugsPerComputeHour                                           float64
	GuestVirtualToExecutionWallRatio                                                           float64
}

func Summarize(samples []Sample) (Cost, error) {
	var cost Cost
	outcomes, bugs := map[record.SHA256]bool{}, map[record.SHA256]bool{}
	var virtual uint64
	add := func(destination *uint64, value uint64) error {
		if math.MaxUint64-*destination < value {
			return errors.New("WASM cost counter overflow")
		}
		*destination += value
		return nil
	}
	for _, sample := range samples {
		if sample.Phase != "execute" && sample.Phase != "replay" || sample.ExecutionWallNanos == 0 || sample.CPUNanos == 0 || sample.GuestPeakMemoryBytes == 0 || sample.HostPeakRSSBytes == 0 {
			return Cost{}, errors.New("actual WASM wall, CPU and guest/host peak measurements are required")
		}
		if sample.Phase == "replay" {
			cost.Replays++
		} else {
			cost.Executions++
			if _, err := record.ParseSHA256(string(sample.OutcomeSHA256)); err != nil {
				return Cost{}, err
			}
			outcomes[sample.OutcomeSHA256] = true
			if sample.ReplayableBugSHA256 != "" {
				if _, err := record.ParseSHA256(string(sample.ReplayableBugSHA256)); err != nil {
					return Cost{}, err
				}
				bugs[sample.ReplayableBugSHA256] = true
			}
			for _, counter := range []struct {
				target *uint64
				value  uint64
			}{{&cost.PreparationWallNanos, sample.PreparationWallNanos}, {&cost.EngineCompileNanos, sample.EngineCompileNanos}, {&cost.InstantiationNanos, sample.InstantiationNanos}, {&cost.ExecutionWallNanos, sample.ExecutionWallNanos}, {&cost.FuelUsed, sample.FuelUsed}, {&cost.TraceBytes, sample.TraceBytes}, {&virtual, sample.VirtualTimeNanos}} {
				if err := add(counter.target, counter.value); err != nil {
					return Cost{}, err
				}
			}
			if sample.CompileCacheHit {
				cost.CompileCacheHits++
			}
		}
		if err := add(&cost.CPUNanos, sample.CPUNanos); err != nil {
			return Cost{}, err
		}
		cost.GuestPeakMemoryBytes = max(cost.GuestPeakMemoryBytes, sample.GuestPeakMemoryBytes)
		cost.HostPeakRSSBytes = max(cost.HostPeakRSSBytes, sample.HostPeakRSSBytes)
	}
	cost.HelperCPUNanos, cost.HelperPeakRSSBytes = cost.CPUNanos, cost.HostPeakRSSBytes
	cost.RepeatedPreparationWallNanos = cost.PreparationWallNanos
	if cost.Executions == 0 {
		return Cost{}, errors.New("WASM cost needs actual executions")
	}
	cost.DistinctOutcomes, cost.ReplayableBugs = uint64(len(outcomes)), uint64(len(bugs))
	cost.VirtualToWallRatio = float64(virtual) / float64(cost.ExecutionWallNanos)
	cost.GuestVirtualToExecutionWallRatio = cost.VirtualToWallRatio
	cost.ReplayableBugsPerComputeHour = float64(cost.ReplayableBugs) * 3_600_000_000_000 / float64(cost.CPUNanos)
	return cost, nil
}

// helperPeakRSS prefers the helper's own image high-water mark. On Linux the
// wait4 ru_maxrss of a child started with CLONE_VM|CLONE_VFORK (os/exec) also
// folds in the operator's peak at execve, so it is only a fallback upper bound.
func helperPeakRSS(telemetry *wasi.Telemetry, usage *hostexec.ResourceUsage) uint64 {
	if telemetry != nil && telemetry.ResourceUsageAvailable && telemetry.PeakRSSBytes != 0 {
		return telemetry.PeakRSSBytes
	}
	if usage != nil {
		return usage.PeakRSSBytes
	}
	return 0
}
