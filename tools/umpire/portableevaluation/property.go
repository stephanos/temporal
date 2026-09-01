package portableevaluation

import (
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

type patternEvaluation struct {
	matched     bool
	coordinates []*umpirespb.ModelCoordinate
	links       []*umpirespb.EvidenceLink
}

func (i *interpreter) evaluateProperties(
	link *umpirespb.ImplementationLinkResult,
) ([]*umpirespb.PropertyResult, umpirespb.EvaluationStatus, *evaluationFailure) {
	properties := make([]*umpirespb.PropertyResult, 0, len(i.contract.GetProperties()))
	for _, property := range i.contract.GetProperties() {
		result, failure := i.evaluateProperty(property, link)
		properties = append(properties, result)
		if failure != nil {
			return properties, umpirespb.EVALUATION_STATUS_INCOMPLETE, failure
		}
	}

	hasViolation := false
	for _, property := range properties {
		switch property.GetStatus() {
		case umpirespb.SEMANTIC_STATUS_SATISFIED:
		case umpirespb.SEMANTIC_STATUS_VIOLATED:
			hasViolation = true
		default:
			return properties, umpirespb.EVALUATION_STATUS_INCOMPLETE, nil
		}
	}
	if hasViolation {
		return properties, umpirespb.EVALUATION_STATUS_VIOLATED, nil
	}
	return properties, umpirespb.EVALUATION_STATUS_SATISFIED, nil
}

func (i *interpreter) evaluateProperty(
	property *umpirespb.Property,
	link *umpirespb.ImplementationLinkResult,
) (*umpirespb.PropertyResult, *evaluationFailure) {
	result := &umpirespb.PropertyResult{Property: proto.CloneOf(property.GetDefinition())}
	for _, clause := range property.GetClauses() {
		clauseResult, failure := i.evaluateClause(property, clause, link)
		result.Clauses = append(result.Clauses, clauseResult)
		if failure != nil {
			result.Status = semanticStatus(failure.class)
			result.Diagnostics = append(result.Diagnostics, failure.diagnostic())
			return result, failure
		}
	}
	result.Status = umpirespb.SEMANTIC_STATUS_SATISFIED
	for _, clause := range result.GetClauses() {
		if clause.GetStatus() == umpirespb.SEMANTIC_STATUS_VIOLATED {
			result.Status = umpirespb.SEMANTIC_STATUS_VIOLATED
			break
		}
	}
	return result, nil
}

func (i *interpreter) evaluateClause(
	property *umpirespb.Property,
	clause *umpirespb.PropertyClause,
	link *umpirespb.ImplementationLinkResult,
) (*umpirespb.PropertyClauseResult, *evaluationFailure) {
	result := &umpirespb.PropertyClauseResult{
		Property:           proto.CloneOf(property.GetDefinition()),
		ClauseDefinitionId: clause.GetDefinitionId(),
		Status:             umpirespb.SEMANTIC_STATUS_SATISFIED,
	}
	for _, step := range link.GetTrace().GetSteps() {
		if failure := i.work.charge(i.ctx.Err(), umpirespb.WORK_UNIT_KIND_CLAUSE_STEP_PAIR, 1); failure != nil {
			return failedClause(result, failure), failure
		}
		trigger, failure := i.evaluatePattern(link, step, clause.GetPerStepImplies().GetTrigger())
		if failure != nil {
			return failedClause(result, failure), failure
		}
		appendPatternSupport(result, trigger)
		if !trigger.matched {
			continue
		}
		required, failure := i.evaluatePattern(link, step, clause.GetPerStepImplies().GetRequired())
		if failure != nil {
			return failedClause(result, failure), failure
		}
		appendPatternSupport(result, required)
		if !required.matched {
			result.Status = umpirespb.SEMANTIC_STATUS_VIOLATED
		}
	}
	return result, nil
}

func failedClause(
	result *umpirespb.PropertyClauseResult,
	failure *evaluationFailure,
) *umpirespb.PropertyClauseResult {
	result.Status = semanticStatus(failure.class)
	result.Diagnostics = append(result.Diagnostics, failure.diagnostic())
	return result
}

func (i *interpreter) evaluatePattern(
	link *umpirespb.ImplementationLinkResult,
	step *umpirespb.ModelTraceStep,
	pattern *umpirespb.Pattern,
) (*patternEvaluation, *evaluationFailure) {
	result := &patternEvaluation{}
	for _, candidate := range stepValues(link.GetTrace(), step, pattern.GetField()) {
		if failure := i.work.charge(i.ctx.Err(), umpirespb.WORK_UNIT_KIND_PATTERN_VALUE_CANDIDATE, 1); failure != nil {
			return nil, failure
		}
		if !proto.Equal(candidate.value.GetDefinition(), pattern.GetDefinition()) {
			continue
		}
		matched, failure := matchPatternValue(pattern, candidate.value.GetValue(), i.contract.GetLimits().GetMaxNatural())
		if failure != nil {
			failure.coord = proto.CloneOf(candidate.coordinate)
			failure.related = []string{pattern.GetDefinition().GetDefinitionId()}
			return nil, failure
		}
		result.coordinates = appendUniqueCoordinate(result.coordinates, candidate.coordinate)
		if evidenceLink := applicationEvidenceLink(link.GetApplications(), candidate.coordinate); evidenceLink != nil {
			result.links = appendUniqueEvidenceLink(result.links, evidenceLink)
		}
		result.matched = result.matched || matched
	}
	return result, nil
}

func matchPatternValue(
	pattern *umpirespb.Pattern,
	value *umpirespb.Value,
	maxNatural string,
) (bool, *evaluationFailure) {
	switch operator := pattern.GetOperator().(type) {
	case *umpirespb.Pattern_EqualsText:
		text, ok := value.GetValue().(*umpirespb.Value_Text)
		if !ok {
			return false, typeFailure(pattern.GetDefinition().GetDefinitionId(), nil, "equals_text requires Text")
		}
		return text.Text == operator.EqualsText.GetValue(), nil
	case *umpirespb.Pattern_NaturalAtMost:
		natural, ok := value.GetValue().(*umpirespb.Value_Natural)
		if !ok {
			return false, typeFailure(pattern.GetDefinition().GetDefinitionId(), nil, "natural_at_most requires Natural")
		}
		bound := operator.NaturalAtMost.GetBound()
		if !validNatural(natural.Natural) || !validNatural(bound) {
			return false, unsupportedFailure(
				umpirespb.DIAGNOSTIC_CODE_NONCANONICAL_NATURAL,
				[]string{pattern.GetDefinition().GetDefinitionId()},
				"natural_at_most received a noncanonical Natural",
			)
		}
		if compareNatural(natural.Natural, maxNatural) > 0 || compareNatural(bound, maxNatural) > 0 {
			return false, unsupportedFailure(
				umpirespb.DIAGNOSTIC_CODE_NATURAL_OUT_OF_RANGE,
				[]string{pattern.GetDefinition().GetDefinitionId()},
				"natural_at_most exceeds the contract Natural bound",
			)
		}
		return compareNatural(natural.Natural, bound) <= 0, nil
	default:
		return false, unsupportedFailure(
			umpirespb.DIAGNOSTIC_CODE_UNSUPPORTED_OPERATOR,
			[]string{pattern.GetDefinition().GetDefinitionId()},
			"Property pattern operator is unsupported",
		)
	}
}

func stepValues(
	trace *umpirespb.ModelTrace,
	step *umpirespb.ModelTraceStep,
	field umpirespb.TraceField,
) []coordinateValue {
	coordinate := func(position int64) *umpirespb.ModelCoordinate {
		return &umpirespb.ModelCoordinate{Field: field, Step: step.GetPosition(), Position: position}
	}
	switch field {
	case umpirespb.TRACE_FIELD_INITIAL_STATE:
		return []coordinateValue{{
			coordinate: &umpirespb.ModelCoordinate{Field: umpirespb.TRACE_FIELD_INITIAL_STATE},
			value:      trace.GetInitialState(),
		}}
	case umpirespb.TRACE_FIELD_PRIOR_STATE:
		return []coordinateValue{{coordinate: coordinate(0), value: step.GetPriorState()}}
	case umpirespb.TRACE_FIELD_SELECTED_ACTION:
		return []coordinateValue{{coordinate: coordinate(0), value: step.GetSelectedAction()}}
	case umpirespb.TRACE_FIELD_MODEL_OUTCOME:
		return []coordinateValue{{coordinate: coordinate(0), value: step.GetModelOutcome()}}
	case umpirespb.TRACE_FIELD_RESULTING_STATE:
		return []coordinateValue{{coordinate: coordinate(0), value: step.GetResultingState()}}
	case umpirespb.TRACE_FIELD_OBSERVATION:
		values := make([]coordinateValue, 0, len(step.GetObservations()))
		for index, observation := range step.GetObservations() {
			values = append(values, coordinateValue{coordinate: coordinate(int64(index + 1)), value: observation})
		}
		return values
	default:
		return nil
	}
}

func applicationEvidenceLink(
	applications []*umpirespb.RenameExactApplication,
	coordinate *umpirespb.ModelCoordinate,
) *umpirespb.EvidenceLink {
	for _, application := range applications {
		if proto.Equal(application.GetCoordinate(), coordinate) {
			return application.GetSourceEvidenceLink()
		}
	}
	return nil
}

func appendPatternSupport(result *umpirespb.PropertyClauseResult, pattern *patternEvaluation) {
	for _, coordinate := range pattern.coordinates {
		result.Coordinates = appendUniqueCoordinate(result.Coordinates, coordinate)
	}
	for _, link := range pattern.links {
		result.EvidenceLinks = appendUniqueEvidenceLink(result.EvidenceLinks, link)
	}
}

func appendUniqueCoordinate(
	coordinates []*umpirespb.ModelCoordinate,
	candidate *umpirespb.ModelCoordinate,
) []*umpirespb.ModelCoordinate {
	for _, coordinate := range coordinates {
		if proto.Equal(coordinate, candidate) {
			return coordinates
		}
	}
	return append(coordinates, proto.CloneOf(candidate))
}

func appendUniqueEvidenceLink(
	links []*umpirespb.EvidenceLink,
	candidate *umpirespb.EvidenceLink,
) []*umpirespb.EvidenceLink {
	for _, link := range links {
		if proto.Equal(link, candidate) {
			return links
		}
	}
	return append(links, proto.CloneOf(candidate))
}
