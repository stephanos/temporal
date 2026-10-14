package conformance

import (
	"fmt"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/check"
	umpirerealization "go.temporal.io/server/tools/umpire/realization"
)

// performedOutcome is what the Model says the instruction performing one witness step returns.
// Accepted steps complete successfully; rejected steps return their realization's gRPC code.
type performedOutcome struct {
	code     string
	rejected bool
}

// compilePerformedOutcomes correlates each performed step of the Query witness to the instruction
// node the realization emits for that occurrence. The realized rejection-code export is the sole
// source of the expected gRPC code.
func compilePerformedOutcomes(realizer *check.Realizer, key check.ClaimKey, r *umpirespb.Realization,
	source *testpilotspb.Case, at *umpirespb.Position,
) (map[coordinate]performedOutcome, error) {
	// The framework also admits generic realizations whose outcomes do not use Temporal's shared
	// accepted/rejected vocabulary. The rejection-code projection opts a realization into this
	// Temporal-specific check.
	codes := umpirerealization.RejectionCodes(r)
	if len(codes) == 0 {
		return nil, nil
	}
	bindings := map[string]coordinate{}
	for _, script := range r.GetScripts() {
		for _, item := range script.GetItems() {
			for _, performance := range item.GetPerforms() {
				// Only RPC results have the gRPC status-code contract this check enforces. Other
				// performed commands (for example, delivery controls) have domain-specific outcomes.
				if performance.GetCommand().GetRpc() == nil {
					continue
				}
				action := realizer.ClassKey(performance.GetStep())
				bound := coordinate{entrypoint: script.GetId(), instruction: performance.GetCommand().GetId()}
				if earlier, duplicate := bindings[action]; duplicate && earlier != bound {
					return nil, located(performance.GetPosition(), "action class %s is performed by both %s/%s and %s/%s", action,
						earlier.entrypoint, earlier.instruction, bound.entrypoint, bound.instruction)
				}
				bindings[action] = bound
			}
		}
	}
	if len(bindings) == 0 {
		return nil, nil
	}
	query, err := realizer.Find(key)
	if err != nil {
		return nil, err
	}
	answer, err := query.Answer()
	if err != nil {
		return nil, err
	}
	if answer.Witness == nil {
		return nil, located(at, "query %s has no witness whose performed steps can be correlated to the Case", key.Name)
	}
	nodes := map[coordinate]bool{}
	for _, entrypoint := range source.GetProgram().GetEntrypoints() {
		for _, instruction := range entrypoint.GetInstructions() {
			nodes[coordinate{entrypoint: entrypoint.GetEntrypointId(), instruction: instruction.GetInstructionId()}] = true
		}
	}
	occurrences := map[string]int{}
	out := map[coordinate]performedOutcome{}
	for _, step := range answer.Witness.Steps {
		bound, performed := bindings[step.Action.Value]
		if !performed {
			continue
		}
		occurrences[step.Action.Value]++
		if ordinal := occurrences[step.Action.Value]; ordinal > 1 {
			bound.instruction += fmt.Sprintf("-%d", ordinal)
		}
		if !nodes[bound] {
			return nil, located(at, "query %s performs step %s at %s/%s, which is no instruction of the Case", key.Name,
				step.Action.Value, bound.entrypoint, bound.instruction)
		}
		expected, err := expectedPerformedOutcome(step.Outcome.Value, codes)
		if err != nil {
			return nil, located(at, "query %s performs step %s with %v", key.Name, step.Action.Value, err)
		}
		out[bound] = expected
	}
	return out, nil
}

func expectedPerformedOutcome(outcome string, codes map[umpirespb.RejectionCode_Rejection]string) (performedOutcome, error) {
	if outcome == "accepted" {
		return performedOutcome{code: "OK"}, nil
	}
	reason, rejected := strings.CutPrefix(outcome, "rejected-")
	if !rejected {
		return performedOutcome{}, fmt.Errorf("outcome %s, which is neither accepted nor rejected", outcome)
	}
	var rejection umpirespb.RejectionCode_Rejection
	for number, name := range umpirespb.RejectionCode_Rejection_name {
		if normalizedRejection(name) == normalizedRejection(reason) {
			rejection = umpirespb.RejectionCode_Rejection(number)
			break
		}
	}
	code, known := codes[rejection]
	if rejection == umpirespb.RejectionCode_REJECTION_UNSPECIFIED || !known {
		return performedOutcome{}, fmt.Errorf("rejection %s, which the realization maps to no gRPC code", reason)
	}
	return performedOutcome{code: strings.ToUpper(code), rejected: true}, nil
}

func normalizedRejection(value string) string {
	value = strings.TrimPrefix(strings.ToUpper(value), "REJECTION_")
	return strings.NewReplacer("_", "", "-", "").Replace(value)
}

func (e performedOutcome) matches(observed *testpilotspb.InstructionOutcome) bool {
	if !e.rejected {
		return observed.GetStatus() == testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED
	}
	return observed.GetStatus() == testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE &&
		normalizedProtocolCode(e.code) == normalizedProtocolCode(observed.GetProtocolCode())
}

func normalizedProtocolCode(value string) string {
	return strings.NewReplacer("_", "", "-", "").Replace(strings.ToUpper(value))
}

func observedCode(outcome *testpilotspb.InstructionOutcome) string {
	if outcome.GetStatus() == testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED {
		return "OK"
	}
	if outcome.GetProtocolCode() != "" {
		return strings.ToUpper(outcome.GetProtocolCode())
	}
	name, known := testpilotspb.InstructionOutcomeStatus_name[int32(outcome.GetStatus())]
	if !known {
		return outcome.GetStatus().String()
	}
	return strings.TrimPrefix(name, "INSTRUCTION_OUTCOME_STATUS_")
}
