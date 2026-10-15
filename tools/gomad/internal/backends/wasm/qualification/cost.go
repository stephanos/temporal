package qualification

import (
	"errors"
	"math"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
)

// Sample contains observed values only. ReplayableBugSHA256 is populated only
// after a retained failure artifact actually matches on replay. CPU is combined
// helper user+system time, including engine work, not inferred guest CPU.
// HelperWaitPeakRSSBytes is the raw wait4 figure. HostPeakRSSUpperBound marks a
// helper without image telemetry, such as a modeled crash or stop, whose host
// peak is only that inherited wait4 upper bound.
type Sample struct {
	AuthorizationToTerminalNanos                                                                 uint64
	Iteration                                                                                    uint64
	Incarnation                                                                                  string
	WASITraceBytes, ChoiceTraceBytes, DiagnosticTraceBytes, StorageTraceBytes, NetworkTraceBytes uint64
	HelperWallNanos, EnginePrepareNanos, CacheLoadNanos, CacheWriteNanos                         uint64
	Phase                                                                                        string
	PreparationWallNanos, EngineCompileNanos, InstantiationNanos, ExecutionWallNanos             uint64
	CPUNanos, GuestPeakMemoryBytes, HostPeakRSSBytes, FuelUsed, TraceBytes, VirtualTimeNanos     uint64
	HelperWaitPeakRSSBytes                                                                       uint64
	CompileCacheHit, HostPeakRSSUpperBound                                                       bool
	// ClusterHelper samples measure one shared process; store samples carry their memory limit.
	ClusterHelper                      bool     `json:",omitempty"`
	ResidentInstances                  []string `json:",omitempty"`
	GuestMemoryLimitBytes              uint64   `json:",omitempty"`
	OutcomeSHA256, ReplayableBugSHA256 record.SHA256
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
	// Helper wait4 upper bounds are kept out of the helper image peaks above.
	HelperUpperBoundSamples, HelperUpperBoundPeakRSSBytes uint64
}

func Summarize(samples []Sample) (Cost, error) {
	if err := validateClusterSamples(samples); err != nil {
		return Cost{}, err
	}
	var cost Cost
	outcomes, bugs := map[record.SHA256]bool{}, map[record.SHA256]bool{}
	var virtual uint64
	helperMeasured := false
	add := func(destination *uint64, value uint64) error {
		if math.MaxUint64-*destination < value {
			return errors.New("WASM cost counter overflow")
		}
		*destination += value
		return nil
	}
	for _, sample := range samples {
		if sample.Phase != "execute" && sample.Phase != "replay" {
			return Cost{}, errors.New("actual WASM execution or replay phase is required")
		}
		if sample.ClusterHelper {
			if sample.HelperWallNanos == 0 || sample.CPUNanos == 0 || sample.HostPeakRSSBytes == 0 {
				return Cost{}, errors.New("actual cluster helper wall, CPU and peak RSS are required")
			}
			if err := sample.checkHelperPeakRSS(); err != nil {
				return Cost{}, err
			}
			if err := add(&cost.CPUNanos, sample.CPUNanos); err != nil {
				return Cost{}, err
			}
			if sample.Phase == "execute" {
				if err := add(&cost.EngineCompileNanos, sample.EngineCompileNanos); err != nil {
					return Cost{}, err
				}
			}
			cost.HostPeakRSSBytes = max(cost.HostPeakRSSBytes, sample.HostPeakRSSBytes)
			if sample.Phase == "execute" && sample.CompileCacheHit {
				cost.CompileCacheHits++
			}
			helperMeasured = true
			continue
		}
		store := sample.GuestMemoryLimitBytes != 0
		if sample.ExecutionWallNanos == 0 || sample.GuestPeakMemoryBytes == 0 || (!store && (sample.CPUNanos == 0 || sample.HostPeakRSSBytes == 0)) {
			return Cost{}, errors.New("actual WASM wall, CPU and guest/host peak measurements are required")
		}
		if store {
			if sample.GuestPeakMemoryBytes > sample.GuestMemoryLimitBytes {
				return Cost{}, errors.New("guest linear memory peak exceeds instance memory limit")
			}
		} else {
			if err := sample.checkHelperPeakRSS(); err != nil {
				return Cost{}, err
			}
			helperMeasured = true
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
			if sample.CompileCacheHit && !store {
				cost.CompileCacheHits++
			}
		}
		cost.GuestPeakMemoryBytes = max(cost.GuestPeakMemoryBytes, sample.GuestPeakMemoryBytes)
		if store {
			continue
		}
		if err := add(&cost.CPUNanos, sample.CPUNanos); err != nil {
			return Cost{}, err
		}
		if sample.HostPeakRSSUpperBound {
			cost.HelperUpperBoundSamples++
			cost.HelperUpperBoundPeakRSSBytes = max(cost.HelperUpperBoundPeakRSSBytes, sample.HostPeakRSSBytes)
			continue
		}
		cost.HostPeakRSSBytes = max(cost.HostPeakRSSBytes, sample.HostPeakRSSBytes)
	}
	cost.HelperCPUNanos, cost.HelperPeakRSSBytes = cost.CPUNanos, cost.HostPeakRSSBytes
	cost.RepeatedPreparationWallNanos = cost.PreparationWallNanos
	if cost.Executions == 0 || !helperMeasured {
		return Cost{}, errors.New("WASM cost needs actual executions")
	}
	cost.DistinctOutcomes, cost.ReplayableBugs = uint64(len(outcomes)), uint64(len(bugs))
	cost.VirtualToWallRatio = float64(virtual) / float64(cost.ExecutionWallNanos)
	cost.GuestVirtualToExecutionWallRatio = cost.VirtualToWallRatio
	cost.ReplayableBugsPerComputeHour = float64(cost.ReplayableBugs) * 3_600_000_000_000 / float64(cost.CPUNanos)
	return cost, nil
}

func validateClusterSamples(samples []Sample) error {
	type execution struct {
		phase     string
		iteration uint64
	}
	helpers := map[execution]map[string]bool{}
	stores := map[execution]map[string]bool{}
	for _, sample := range samples {
		key := execution{sample.Phase, sample.Iteration}
		if sample.ClusterHelper {
			if err := sample.checkHelperPeakRSS(); err != nil {
				return err
			}
			residents := map[string]bool{}
			for _, resident := range sample.ResidentInstances {
				residents[resident] = true
			}
			if previous := helpers[key]; previous != nil {
				if len(previous) != len(residents) {
					return errors.New("cluster helper measurements name different resident guests")
				}
				for resident := range previous {
					if !residents[resident] {
						return errors.New("cluster helper measurements name different resident guests")
					}
				}
			}
			helpers[key] = residents
		} else if sample.GuestMemoryLimitBytes != 0 {
			resident, incarnation, found := strings.Cut(sample.Incarnation, "/")
			if !found || resident == "" || incarnation == "" {
				return errors.New("store memory measurement requires an incarnation")
			}
			if stores[key] == nil {
				stores[key] = map[string]bool{}
			}
			stores[key][resident] = true
		}
	}
	for key, residents := range helpers {
		for resident := range residents {
			if !stores[key][resident] {
				return errors.New("cluster helper measurement lacks resident store memory peak")
			}
		}
	}
	for key, residents := range stores {
		for resident := range residents {
			if !helpers[key][resident] {
				return errors.New("store measurement lacks matching cluster helper CPU and RSS")
			}
		}
	}
	return nil
}

// recordHelperPeakRSS prefers the helper's own image high-water mark. On Linux
// the wait4 ru_maxrss of a child started with CLONE_VM|CLONE_VFORK (os/exec)
// also folds in the operator's peak at execve, so without telemetry the wait
// figure is retained only as a marked upper bound.
func (sample *Sample) recordHelperPeakRSS(telemetry *wasi.Telemetry, usage *hostexec.ResourceUsage) {
	sample.HostPeakRSSBytes, sample.HelperWaitPeakRSSBytes, sample.HostPeakRSSUpperBound = 0, 0, false
	if usage != nil {
		sample.HelperWaitPeakRSSBytes = usage.PeakRSSBytes
	}
	if telemetry != nil && telemetry.ResourceUsageAvailable && telemetry.PeakRSSBytes != 0 {
		sample.HostPeakRSSBytes = telemetry.PeakRSSBytes
		return
	}
	sample.HostPeakRSSBytes = sample.HelperWaitPeakRSSBytes
	sample.HostPeakRSSUpperBound = sample.HostPeakRSSBytes != 0
}

// checkHelperPeakRSS rejects an image high water above the same process's wait
// figure (VmHWM <= ru_maxrss on Linux) and an upper bound not taken from wait4.
func (sample Sample) checkHelperPeakRSS() error {
	if sample.ClusterHelper {
		if sample.HostPeakRSSUpperBound || sample.HostPeakRSSBytes == 0 || len(sample.ResidentInstances) == 0 {
			return errors.New("actual cluster helper peak RSS and resident instances are required")
		}
		residents := make(map[string]bool, len(sample.ResidentInstances))
		for _, resident := range sample.ResidentInstances {
			if resident == "" || residents[resident] {
				return errors.New("cluster helper resident instances must be nonempty and unique")
			}
			residents[resident] = true
		}
	}
	if sample.HostPeakRSSUpperBound && sample.HostPeakRSSBytes != sample.HelperWaitPeakRSSBytes {
		return errors.New("helper peak RSS upper bound differs from its wait figure")
	}
	if sample.HelperWaitPeakRSSBytes != 0 && sample.HostPeakRSSBytes > sample.HelperWaitPeakRSSBytes {
		return errors.New("helper image peak RSS exceeds its wait figure")
	}
	return nil
}
