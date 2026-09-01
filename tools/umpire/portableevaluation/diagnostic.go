package portableevaluation

import (
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

type evaluationFailure struct {
	class    umpirespb.DiagnosticClass
	code     umpirespb.DiagnosticCode
	related  []string
	coord    *umpirespb.ModelCoordinate
	limit    *umpirespb.Limit
	observed int64
	detail   string
	canceled bool
}

func (f *evaluationFailure) diagnostic() *umpirespb.Diagnostic {
	if f == nil {
		return nil
	}
	return &umpirespb.Diagnostic{
		DiagnosticClass:      f.class,
		Code:                 f.code,
		RelatedDefinitionIds: append([]string(nil), f.related...),
		Coordinate:           proto.CloneOf(f.coord),
		AppliedLimit:         proto.CloneOf(f.limit),
		ObservedCount:        f.observed,
		Detail:               f.detail,
	}
}

func malformedFailure(detail string) *evaluationFailure {
	return &evaluationFailure{
		class:  umpirespb.DIAGNOSTIC_CLASS_INVALID,
		code:   umpirespb.DIAGNOSTIC_CODE_MALFORMED_CONTRACT,
		detail: detail,
	}
}

func invalidInputFailure(detail string) *evaluationFailure {
	return &evaluationFailure{
		class:  umpirespb.DIAGNOSTIC_CLASS_INVALID,
		code:   umpirespb.DIAGNOSTIC_CODE_TYPE_MISMATCH,
		detail: detail,
	}
}

func canceledFailure(err error) *evaluationFailure {
	return &evaluationFailure{
		class:    umpirespb.DIAGNOSTIC_CLASS_UNKNOWN,
		code:     umpirespb.DIAGNOSTIC_CODE_LIMIT_REACHED,
		detail:   err.Error(),
		canceled: true,
	}
}

func limitFailure(detail, unit string, limit, observed int64) *evaluationFailure {
	return &evaluationFailure{
		class:    umpirespb.DIAGNOSTIC_CLASS_UNKNOWN,
		code:     umpirespb.DIAGNOSTIC_CODE_LIMIT_REACHED,
		limit:    &umpirespb.Limit{Value: limit, Unit: unit},
		observed: observed,
		detail:   detail,
	}
}

func observationStatus(class umpirespb.DiagnosticClass) umpirespb.ObservationStatus {
	switch class {
	case umpirespb.DIAGNOSTIC_CLASS_CONFLICT:
		return umpirespb.OBSERVATION_STATUS_CONFLICT
	case umpirespb.DIAGNOSTIC_CLASS_UNSUPPORTED:
		return umpirespb.OBSERVATION_STATUS_UNSUPPORTED
	default:
		return umpirespb.OBSERVATION_STATUS_UNKNOWN
	}
}

func linkStatus(class umpirespb.DiagnosticClass) umpirespb.ImplementationLinkStatus {
	switch class {
	case umpirespb.DIAGNOSTIC_CLASS_CONFLICT:
		return umpirespb.IMPLEMENTATION_LINK_STATUS_CONFLICT
	case umpirespb.DIAGNOSTIC_CLASS_UNSUPPORTED:
		return umpirespb.IMPLEMENTATION_LINK_STATUS_UNSUPPORTED
	case umpirespb.DIAGNOSTIC_CLASS_INVALID:
		return umpirespb.IMPLEMENTATION_LINK_STATUS_INVALID
	default:
		return umpirespb.IMPLEMENTATION_LINK_STATUS_UNKNOWN
	}
}

func semanticStatus(class umpirespb.DiagnosticClass) umpirespb.SemanticStatus {
	switch class {
	case umpirespb.DIAGNOSTIC_CLASS_CONFLICT:
		return umpirespb.SEMANTIC_STATUS_CONFLICT
	case umpirespb.DIAGNOSTIC_CLASS_UNSUPPORTED, umpirespb.DIAGNOSTIC_CLASS_INVALID:
		return umpirespb.SEMANTIC_STATUS_UNSUPPORTED
	default:
		return umpirespb.SEMANTIC_STATUS_UNKNOWN
	}
}
