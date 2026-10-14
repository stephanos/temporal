package qualification

import (
	"errors"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
)

// BudgetEvaluation retains the agreed 16 GiB host protocol. The resident peak is
// an intentionally conservative upper bound, NOT measured simultaneous RSS.
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
	warmResidents := map[string]uint64{}
	for _, sample := range samples {
		if sample.HelperWallNanos == 0 || sample.HostPeakRSSBytes == 0 {
			return budget, errors.New("actual helper wall and process peak RSS are required for budget")
		}
		if err := sample.checkHelperPeakRSS(); err != nil {
			return budget, err
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
			resident, _, ok := strings.Cut(sample.Incarnation, "/")
			if !ok {
				resident = sample.Incarnation
			}
			warmResidents[resident] = max(warmResidents[resident], sample.HostPeakRSSBytes)
		} else {
			budget.ColdHelperPeakRSSBytes = max(budget.ColdHelperPeakRSSBytes, sample.HostPeakRSSBytes)
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
	for _, peak := range warmResidents {
		if ^uint64(0)-budget.ConservativeResidentHelperPeakBytes < peak {
			return budget, errors.New("resident helper peak bound overflow")
		}
		budget.ConservativeResidentHelperPeakBytes += peak
	}
	budget.ColdHelperExceededOriginal4GiBLimit = budget.ColdHelperPeakRSSBytes > 4<<30
	for _, check := range []struct {
		observed, limit uint64
		name            string
	}{{budget.PreparationWallNanos, budget.PreparationLimitNanos, "preparation wall"}, {budget.LargestHelperWallNanos, budget.ExecutionLimitNanos, "helper execution/replay wall"}, {budget.CompilerPeakRSSBytes, budget.CompilerPeakLimitBytes, "individual compiler peak RSS"}, {budget.WarmHelperPeakRSSBytes, budget.HelperPeakLimitBytes, "individual warm helper peak RSS"}, {budget.ColdHelperPeakRSSBytes, budget.ColdHelperPeakLimitBytes, "individual cold helper peak RSS"}, {budget.ConservativeResidentHelperPeakBytes, budget.ResidentHelperPeakLimitBytes, "conservative resident helper peak upper bound"}, {budget.OperatorPeakRSSBytes, budget.OperatorPeakRecommendedLimitBytes, "operator lifetime peak RSS"}} {
		if check.observed > check.limit {
			budget.Violations = append(budget.Violations, check.name)
		}
	}
	budget.WithinBudget = len(budget.Violations) == 0
	return budget, nil
}
