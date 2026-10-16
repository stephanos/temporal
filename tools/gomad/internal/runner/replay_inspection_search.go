package runner

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.temporal.io/server/tools/gomad/internal/model/cluster"
)

type WASMInspectionSearchRequest struct {
	Target        cluster.NodeHandle `json:"target"`
	Cluster       bool               `json:"-"`
	Check         string             `json:"check"`
	First         uint64             `json:"first"`
	Last          uint64             `json:"last"`
	Mode          string             `json:"mode"`
	AttemptBudget uint64             `json:"attempt_budget"`
	WallTime      time.Duration      `json:"wall_time"`
}

type WASMInspectionRange struct {
	First uint64 `json:"first"`
	Last  uint64 `json:"last"`
}

type WASMInspectionEvaluation struct {
	Choice         uint64 `json:"choice"`
	Outcome        string `json:"outcome"`
	PrefixVerified bool   `json:"prefix_verified"`
	Detail         string `json:"detail,omitempty"`
}

type WASMInspectionSearchResult struct {
	Request                   WASMInspectionSearchRequest `json:"request"`
	Target                    cluster.NodeHandle          `json:"target"`
	Status                    string                      `json:"status"`
	FirstFailure              *uint64                     `json:"first_failure,omitempty"`
	Evaluations               []WASMInspectionEvaluation  `json:"evaluations"`
	Remaining                 *WASMInspectionRange        `json:"remaining,omitempty"`
	ConditionalOnMonotonicity bool                        `json:"conditional_on_monotonicity"`
}

// Each call must create a fresh exact replay of the same artifact and destroy
// its stores before returning. The context bounds startup through cleanup.
type WASMInspectionEvaluator func(context.Context, cluster.WASMInspectionResolution, string) (WASMInspectionEvaluation, error)

func SearchWASMInspection(ctx context.Context, execution *cluster.WASMExecutionEvidence, request WASMInspectionSearchRequest, evaluate WASMInspectionEvaluator) (WASMInspectionSearchResult, error) {
	result := WASMInspectionSearchResult{Request: request}
	if ctx == nil || evaluate == nil || request.First > request.Last || request.AttemptBudget == 0 || request.WallTime <= 0 || request.Check == "" {
		return result, errors.New("inspection search requires a valid inclusive range, check and positive attempt and wall bounds")
	}
	if request.Mode == "" {
		request.Mode = "scan"
		result.Request.Mode = "scan"
	}
	if request.Mode != "scan" && request.Mode != "bisect" {
		return result, errors.New("inspection search mode must be scan or bisect")
	}
	ctx, cancel := context.WithTimeout(ctx, request.WallTime)
	defer cancel()
	resolve := func(ordinal uint64) (cluster.WASMInspectionResolution, error) {
		return cluster.ResolveWASMInspection(execution, cluster.WASMInspectionRequest{Target: request.Target, Stop: cluster.WASMInspectionStop{Kind: "choice", Value: ordinal}}, request.Cluster)
	}
	first, err := resolve(request.First)
	if err != nil {
		return result, err
	}
	if _, err := resolve(request.Last); err != nil {
		return result, err
	}
	result.Target = first.Target
	var registration *cluster.WASMInspectionRegistration
	for _, incarnation := range execution.Inspection.Incarnations {
		if incarnation.Handle != first.Target {
			continue
		}
		if !incarnation.Registry.Closed {
			return result, errors.New("inspection invariant registry is not closed")
		}
		for i := range incarnation.Registry.Entries {
			entry := &incarnation.Registry.Entries[i]
			if entry.Kind == "check" && entry.Name == request.Check && entry.AvailableOrder <= first.Boundary.Order {
				registration = entry
				break
			}
		}
	}
	if registration == nil {
		return result, fmt.Errorf("unknown or unavailable inspection check %q", request.Check)
	}
	if request.Mode == "bisect" && !registration.Monotonic {
		return result, errors.New("bisect requires a retained monotonic failure declaration")
	}
	result.ConditionalOnMonotonicity = request.Mode == "bisect"
	result.Remaining = &WASMInspectionRange{First: request.First, Last: request.Last}
	cache := map[uint64]bool{}
	run := func(ordinal uint64) (bool, bool) {
		if passed, ok := cache[ordinal]; ok {
			return passed, true
		}
		if uint64(len(result.Evaluations)) >= request.AttemptBudget || ctx.Err() != nil {
			result.Status = "incomplete"
			return false, false
		}
		resolved, err := resolve(ordinal)
		if err != nil {
			result.Status = "replay_divergence"
			result.Evaluations = append(result.Evaluations, WASMInspectionEvaluation{Choice: ordinal, Outcome: "divergence", Detail: err.Error()})
			return false, false
		}
		evaluation, err := evaluate(ctx, resolved, request.Check)
		evaluation.Choice = ordinal
		if err != nil {
			evaluation.Outcome = "error"
			evaluation.Detail = err.Error()
			if errors.Is(err, cluster.ErrWASMInstrumentationDivergence) {
				evaluation.Outcome = "divergence"
			}
		}
		if ctx.Err() != nil {
			evaluation.Outcome = "timeout"
			evaluation.Detail = ctx.Err().Error()
		}
		result.Evaluations = append(result.Evaluations, evaluation)
		if evaluation.Outcome == "divergence" || (evaluation.Outcome == "pass" || evaluation.Outcome == "fail") && !evaluation.PrefixVerified {
			result.Status = "replay_divergence"
			return false, false
		}
		if evaluation.Outcome != "pass" && evaluation.Outcome != "fail" {
			result.Status = "invariant_evaluation_failure"
			return false, false
		}
		passed := evaluation.Outcome == "pass"
		cache[ordinal] = passed
		if request.Mode == "bisect" {
			for position, value := range cache {
				if !value && passed && position < ordinal || value && !passed && position > ordinal {
					result.Status = "monotonicity_violation"
					return false, false
				}
			}
		}
		return passed, true
	}
	failure := func(ordinal uint64) {
		result.FirstFailure = new(uint64)
		*result.FirstFailure = ordinal
		result.Status = "first_failure"
		if ordinal == request.First {
			result.Status = "already_failing_at_first"
		}
		result.Remaining = nil
	}
	if request.Mode == "scan" {
		for ordinal := request.First; ; ordinal++ {
			result.Remaining = &WASMInspectionRange{First: ordinal, Last: request.Last}
			passed, conclusive := run(ordinal)
			if !conclusive {
				return result, nil
			}
			if !passed {
				failure(ordinal)
				return result, nil
			}
			if ordinal == request.Last {
				result.Status = "no_failure"
				result.Remaining = nil
				return result, nil
			}
		}
	}
	lower, upper := request.First, request.Last
	lowPass, conclusive := run(lower)
	if !conclusive {
		return result, nil
	}
	if lower == upper {
		if lowPass {
			result.Status = "no_failure"
			result.Remaining = nil
		} else {
			failure(lower)
		}
		return result, nil
	}
	highPass, conclusive := run(upper)
	if !conclusive {
		return result, nil
	}
	if !lowPass {
		failure(lower)
		return result, nil
	}
	if highPass {
		result.Status = "no_failure"
		result.Remaining = nil
		return result, nil
	}
	for upper-lower > 1 {
		result.Remaining = &WASMInspectionRange{First: lower + 1, Last: upper}
		middle := lower + (upper-lower)/2
		passed, conclusive := run(middle)
		if !conclusive {
			return result, nil
		}
		if passed {
			lower = middle
		} else {
			upper = middle
		}
	}
	failure(upper)
	return result, nil
}
