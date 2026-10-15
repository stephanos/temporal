package qualification

import (
	"errors"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
)

// BudgetEvaluation retains the agreed 16 GiB host protocol. The resident peak
// is a cluster process measurement or a conservative sum of single helpers.
type BudgetEvaluation struct {
	SerialHeavyPreparation                                                        bool
	MaxResidentGuests, MaxVisibleContinuations                                    uint64
	PreparationLimitNanos, ExecutionLimitNanos                                    uint64
	CompilerPeakLimitBytes, HelperPeakLimitBytes, ResidentHelperPeakLimitBytes    uint64
	ColdHelperPeakLimitBytes, ColdHelperPeakRSSBytes, WarmHelperPeakRSSBytes      uint64
	ColdHelperExceededOriginal4GiBLimit                                           bool
	OperatorPeakRSSBytes, OperatorPeakRecommendedLimitBytes                       uint64
	OperatorRecommendationExceeded                                                bool
	PreparationWallNanos, LargestHelperWallNanos                                  uint64
	CompilerPeakRSSBytes, HelperPeakRSSBytes, ConservativeResidentHelperPeakBytes uint64
	// Helper wait4 upper bounds are reported here, not as cold or warm peaks.
	HelperUpperBoundSamples, HelperUpperBoundPeakRSSBytes uint64
	WithinBudget                                          bool
	Violations                                            []string
}

func EvaluateBudget(samples []Sample, commands []backend.CompilerCommandMetrics, preparationWall uint64, residentGuests uint64, operatorPeak ...uint64) (BudgetEvaluation, error) {
	budget := BudgetEvaluation{SerialHeavyPreparation: true, MaxResidentGuests: residentGuests, MaxVisibleContinuations: 1, PreparationLimitNanos: uint64(10 * time.Minute), ExecutionLimitNanos: uint64(15 * time.Minute), CompilerPeakLimitBytes: 6 << 30, HelperPeakLimitBytes: 4 << 30, ColdHelperPeakLimitBytes: 6 << 30, ResidentHelperPeakLimitBytes: 8 << 30, OperatorPeakRecommendedLimitBytes: 2 << 30, PreparationWallNanos: preparationWall, Violations: []string{}}
	if len(operatorPeak) != 0 {
		budget.OperatorPeakRSSBytes = operatorPeak[0]
		budget.OperatorRecommendationExceeded = operatorPeak[0] > budget.OperatorPeakRecommendedLimitBytes
	}
	if len(operatorPeak) != 1 || operatorPeak[0] == 0 {
		return budget, errors.New("actual operator lifetime peak RSS is required for budget")
	}
	if residentGuests == 0 || residentGuests > 2 || preparationWall == 0 || len(samples) == 0 || len(commands) == 0 {
		return budget, errors.New("measured budget requires preparation, samples and one or two resident guests")
	}
	if err := validateClusterSamples(samples); err != nil {
		return budget, err
	}
	warmResidents := map[string]uint64{}
	storeResidents, clusterResidents := map[string]bool{}, map[string]bool{}
	var singleWarmPeak, singleColdPeak uint64
	for _, sample := range samples {
		if !sample.ClusterHelper && sample.GuestMemoryLimitBytes != 0 {
			if sample.GuestPeakMemoryBytes == 0 {
				return budget, errors.New("actual guest linear memory peak is required for budget")
			}
			if sample.GuestPeakMemoryBytes > sample.GuestMemoryLimitBytes {
				budget.Violations = append(budget.Violations, "guest linear memory peak: "+sample.Incarnation)
			}
			resident, _, _ := strings.Cut(sample.Incarnation, "/")
			storeResidents[resident] = true
			budget.LargestHelperWallNanos = max(budget.LargestHelperWallNanos, sample.AuthorizationToTerminalNanos)
			continue
		}
		if sample.HelperWallNanos == 0 || sample.HostPeakRSSBytes == 0 {
			return budget, errors.New("actual helper wall and process peak RSS are required for budget")
		}
		if err := sample.checkHelperPeakRSS(); err != nil {
			return budget, err
		}
		if sample.ClusterHelper {
			if uint64(len(sample.ResidentInstances)) != residentGuests {
				return budget, errors.New("cluster helper measurement must cover every resident guest")
			}
			for _, resident := range sample.ResidentInstances {
				if len(clusterResidents) != 0 && !clusterResidents[resident] {
					return budget, errors.New("cluster helper measurements name different resident guests")
				}
			}
			for _, resident := range sample.ResidentInstances {
				clusterResidents[resident] = true
				if sample.CompileCacheHit {
					warmResidents[resident] = max(warmResidents[resident], sample.HostPeakRSSBytes)
				}
			}
			budget.ConservativeResidentHelperPeakBytes = max(budget.ConservativeResidentHelperPeakBytes, sample.HostPeakRSSBytes)
		}
		budget.LargestHelperWallNanos = max(budget.LargestHelperWallNanos, sample.HelperWallNanos)
		if sample.HostPeakRSSUpperBound {
			budget.HelperUpperBoundSamples++
			budget.HelperUpperBoundPeakRSSBytes = max(budget.HelperUpperBoundPeakRSSBytes, sample.HostPeakRSSBytes)
			continue
		}
		budget.HelperPeakRSSBytes = max(budget.HelperPeakRSSBytes, sample.HostPeakRSSBytes)
		if sample.CompileCacheHit {
			budget.WarmHelperPeakRSSBytes = max(budget.WarmHelperPeakRSSBytes, sample.HostPeakRSSBytes)
			if !sample.ClusterHelper {
				resident, _, _ := strings.Cut(sample.Incarnation, "/")
				warmResidents[resident] = max(warmResidents[resident], sample.HostPeakRSSBytes)
				singleWarmPeak = max(singleWarmPeak, sample.HostPeakRSSBytes)
			}
		} else {
			budget.ColdHelperPeakRSSBytes = max(budget.ColdHelperPeakRSSBytes, sample.HostPeakRSSBytes)
			if !sample.ClusterHelper {
				singleColdPeak = max(singleColdPeak, sample.HostPeakRSSBytes)
			}
		}
	}
	for _, command := range commands {
		if command.ResourceUsage == nil || command.ResourceUsage.PeakRSSBytes == 0 {
			return budget, errors.New("actual compiler process peak RSS is required for budget")
		}
		budget.CompilerPeakRSSBytes = max(budget.CompilerPeakRSSBytes, command.ResourceUsage.PeakRSSBytes)
	}
	if uint64(len(warmResidents)) != residentGuests {
		return budget, errors.New("actual warm helper peak measurement is required for each resident guest")
	}
	if len(clusterResidents) != 0 {
		for resident := range storeResidents {
			if !clusterResidents[resident] {
				return budget, errors.New("cluster helper measurement does not cover a resident guest")
			}
		}
	} else {
		for _, peak := range warmResidents {
			if ^uint64(0)-budget.ConservativeResidentHelperPeakBytes < peak {
				return budget, errors.New("resident helper peak bound overflow")
			}
			budget.ConservativeResidentHelperPeakBytes += peak
		}
	}
	budget.ColdHelperExceededOriginal4GiBLimit = singleColdPeak > 4<<30
	for _, check := range []struct {
		observed, limit uint64
		name            string
	}{{budget.PreparationWallNanos, budget.PreparationLimitNanos, "preparation wall"}, {budget.LargestHelperWallNanos, budget.ExecutionLimitNanos, "helper execution/replay wall"}, {budget.CompilerPeakRSSBytes, budget.CompilerPeakLimitBytes, "individual compiler peak RSS"}, {singleWarmPeak, budget.HelperPeakLimitBytes, "individual warm helper peak RSS"}, {singleColdPeak, budget.ColdHelperPeakLimitBytes, "individual cold helper peak RSS"}, {budget.ConservativeResidentHelperPeakBytes, budget.ResidentHelperPeakLimitBytes, "conservative resident helper peak upper bound"}, {budget.OperatorPeakRSSBytes, budget.OperatorPeakRecommendedLimitBytes, "operator lifetime peak RSS"}} {
		if check.observed > check.limit {
			budget.Violations = append(budget.Violations, check.name)
		}
	}
	budget.WithinBudget = len(budget.Violations) == 0
	return budget, nil
}
