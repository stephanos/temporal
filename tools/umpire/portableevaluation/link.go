package portableevaluation

import (
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

func (i *interpreter) applyLink(
	sourceTrace *umpirespb.ModelTrace,
	sourceLinks []*umpirespb.EvidenceLink,
) (*umpirespb.ImplementationLinkResult, *evaluationFailure) {
	values := traceValues(sourceTrace)
	applicationLimit := i.contract.GetImplementationLink().GetApplicationLimit()
	if int64(len(sourceTrace.GetSteps())) > applicationLimit.GetValue() {
		return nil, limitFailure(
			"Implementation Link applications exceed the contract Limit",
			applicationLimit.GetUnit(),
			applicationLimit.GetValue(),
			int64(len(sourceTrace.GetSteps())),
		)
	}

	destinationTrace := traceShape(sourceTrace)
	applications := make([]*umpirespb.RenameExactApplication, 0, len(values))
	for _, candidate := range values {
		var matches []*umpirespb.RenameExactEntry
		for _, entry := range i.contract.GetImplementationLink().GetEntries() {
			if failure := i.work.charge(i.ctx.Err(), umpirespb.WORK_UNIT_KIND_LINK_ENTRY, 1); failure != nil {
				return nil, failure
			}
			if proto.Equal(candidate.value, entry.GetSource()) {
				matches = append(matches, entry)
			}
		}
		switch len(matches) {
		case 0:
			return nil, &evaluationFailure{
				class:  umpirespb.DIAGNOSTIC_CLASS_UNKNOWN,
				code:   umpirespb.DIAGNOSTIC_CODE_MISSING_LINK_MAPPING,
				coord:  proto.CloneOf(candidate.coordinate),
				detail: "Model value has no exact Implementation Link mapping",
			}
		case 1:
		default:
			code := umpirespb.DIAGNOSTIC_CODE_DUPLICATE_LINK_MAPPING
			for _, match := range matches[1:] {
				if !proto.Equal(matches[0].GetDestination(), match.GetDestination()) {
					code = umpirespb.DIAGNOSTIC_CODE_CONTRADICTORY_LINK_MAPPING
					break
				}
			}
			return nil, &evaluationFailure{
				class:    umpirespb.DIAGNOSTIC_CLASS_CONFLICT,
				code:     code,
				coord:    proto.CloneOf(candidate.coordinate),
				observed: int64(len(matches)),
				detail:   "Model value has multiple exact Implementation Link mappings",
			}
		}

		entry := matches[0]
		if !setTraceValue(destinationTrace, candidate.coordinate, proto.CloneOf(entry.GetDestination())) {
			return nil, &evaluationFailure{
				class:  umpirespb.DIAGNOSTIC_CLASS_CONFLICT,
				code:   umpirespb.DIAGNOSTIC_CODE_EXTRA_COORDINATE,
				coord:  proto.CloneOf(candidate.coordinate),
				detail: "Implementation Link produced a coordinate outside the trace shape",
			}
		}
		sourceLink := sourceEvidenceLink(candidate.coordinate, sourceLinks)
		if sourceLink == nil {
			return nil, &evaluationFailure{
				class:  umpirespb.DIAGNOSTIC_CLASS_UNKNOWN,
				code:   umpirespb.DIAGNOSTIC_CODE_MISSING_LINK_MAPPING,
				coord:  proto.CloneOf(candidate.coordinate),
				detail: "Implementation Link application has no source Evidence Link",
			}
		}
		applications = append(applications, &umpirespb.RenameExactApplication{
			Coordinate:         proto.CloneOf(candidate.coordinate),
			Entry:              proto.CloneOf(entry),
			SourceEvidenceLink: proto.CloneOf(sourceLink),
		})
	}
	destinationTrace.TraceId = traceIdentity(destinationTrace)
	return &umpirespb.ImplementationLinkResult{
		Status:       umpirespb.IMPLEMENTATION_LINK_STATUS_APPLIED,
		Trace:        destinationTrace,
		Applications: applications,
	}, nil
}

func traceShape(source *umpirespb.ModelTrace) *umpirespb.ModelTrace {
	trace := &umpirespb.ModelTrace{Steps: make([]*umpirespb.ModelTraceStep, len(source.GetSteps()))}
	for index, step := range source.GetSteps() {
		trace.Steps[index] = &umpirespb.ModelTraceStep{
			Position:     step.GetPosition(),
			Observations: make([]*umpirespb.ModelValue, len(step.GetObservations())),
		}
	}
	return trace
}

func setTraceValue(trace *umpirespb.ModelTrace, coordinate *umpirespb.ModelCoordinate, value *umpirespb.ModelValue) bool {
	if coordinate.GetField() == umpirespb.TRACE_FIELD_INITIAL_STATE {
		trace.InitialState = value
		return true
	}
	if coordinate.GetStep() < 1 || coordinate.GetStep() > int64(len(trace.GetSteps())) {
		return false
	}
	step := trace.Steps[coordinate.GetStep()-1]
	switch coordinate.GetField() {
	case umpirespb.TRACE_FIELD_PRIOR_STATE:
		step.PriorState = value
	case umpirespb.TRACE_FIELD_SELECTED_ACTION:
		step.SelectedAction = value
	case umpirespb.TRACE_FIELD_MODEL_OUTCOME:
		step.ModelOutcome = value
	case umpirespb.TRACE_FIELD_RESULTING_STATE:
		step.ResultingState = value
	case umpirespb.TRACE_FIELD_OBSERVATION:
		if coordinate.GetPosition() < 1 || coordinate.GetPosition() > int64(len(step.GetObservations())) {
			return false
		}
		step.Observations[coordinate.GetPosition()-1] = value
	default:
		return false
	}
	return true
}

func sourceEvidenceLink(
	coordinate *umpirespb.ModelCoordinate,
	links []*umpirespb.EvidenceLink,
) *umpirespb.EvidenceLink {
	for _, link := range links {
		if proto.Equal(coordinate, link.GetCoordinate()) {
			return link
		}
	}
	if coordinate.GetField() != umpirespb.TRACE_FIELD_PRIOR_STATE {
		return nil
	}
	sourceCoordinate := &umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_INITIAL_STATE}
	if coordinate.GetStep() > 1 {
		sourceCoordinate = &umpirespb.ModelCoordinate{
			Field: umpirespb.TRACE_FIELD_RESULTING_STATE,
			Step:  coordinate.GetStep() - 1,
		}
	}
	for _, link := range links {
		if proto.Equal(sourceCoordinate, link.GetCoordinate()) {
			return link
		}
	}
	return nil
}
