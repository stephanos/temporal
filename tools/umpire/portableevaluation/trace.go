package portableevaluation

import (
	"crypto/sha256"
	"fmt"
	"slices"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

func sortEmissions(emissions []*emission) {
	slices.SortStableFunc(emissions, func(left, right *emission) int {
		return compareCoordinates(left.emit.GetCoordinate(), right.emit.GetCoordinate())
	})
}

func compareCoordinates(left, right *umpirespb.ModelCoordinate) int {
	leftStep, leftRank, leftPosition := coordinateOrder(left)
	rightStep, rightRank, rightPosition := coordinateOrder(right)
	if leftStep != rightStep {
		if leftStep < rightStep {
			return -1
		}
		return 1
	}
	if leftRank != rightRank {
		return leftRank - rightRank
	}
	if leftPosition < rightPosition {
		return -1
	}
	if leftPosition > rightPosition {
		return 1
	}
	return 0
}

func coordinateOrder(coordinate *umpirespb.ModelCoordinate) (int64, int, int64) {
	if coordinate.GetField() == umpirespb.TRACE_FIELD_INITIAL_STATE {
		return 0, 0, 0
	}
	rank := 0
	switch coordinate.GetField() {
	case umpirespb.TRACE_FIELD_PRIOR_STATE:
		rank = 0
	case umpirespb.TRACE_FIELD_SELECTED_ACTION:
		rank = 1
	case umpirespb.TRACE_FIELD_MODEL_OUTCOME:
		rank = 2
	case umpirespb.TRACE_FIELD_RESULTING_STATE:
		rank = 3
	case umpirespb.TRACE_FIELD_OBSERVATION:
		rank = 4
	default:
		rank = 5
	}
	return coordinate.GetStep(), rank, coordinate.GetPosition()
}

func coordinateKey(coordinate *umpirespb.ModelCoordinate) string {
	return fmt.Sprintf("%d/%d/%d", coordinate.GetField(), coordinate.GetStep(), coordinate.GetPosition())
}

func traceIdentity(trace *umpirespb.ModelTrace) string {
	cloned := proto.CloneOf(trace)
	cloned.TraceId = ""
	encoded, _ := (proto.MarshalOptions{Deterministic: true}).Marshal(cloned)
	digest := sha256.Sum256(encoded)
	return fmt.Sprintf("sha256:%x", digest)
}

type coordinateValue struct {
	coordinate *umpirespb.ModelCoordinate
	value      *umpirespb.ModelValue
}

func traceValues(trace *umpirespb.ModelTrace) []coordinateValue {
	values := []coordinateValue{{
		coordinate: &umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_INITIAL_STATE},
		value:      trace.GetInitialState(),
	}}
	for _, step := range trace.GetSteps() {
		values = append(values,
			coordinateValue{coordinate: &umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_PRIOR_STATE, Step: step.GetPosition()}, value: step.GetPriorState()},
			coordinateValue{coordinate: &umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_SELECTED_ACTION, Step: step.GetPosition()}, value: step.GetSelectedAction()},
			coordinateValue{coordinate: &umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_MODEL_OUTCOME, Step: step.GetPosition()}, value: step.GetModelOutcome()},
			coordinateValue{coordinate: &umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_RESULTING_STATE, Step: step.GetPosition()}, value: step.GetResultingState()},
		)
		for index, observation := range step.GetObservations() {
			values = append(values, coordinateValue{
				coordinate: &umpirespb.ModelCoordinate{
					Field: umpirespb.TRACE_FIELD_OBSERVATION,
					Step:  step.GetPosition(), Position: int64(index + 1),
				},
				value: observation,
			})
		}
	}
	return values
}
