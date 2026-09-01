package portableevaluation

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/evaluationcontract"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"google.golang.org/protobuf/proto"
)

// Request contains the exact admitted contract and one bounded Raw Evidence snapshot.
type Request struct {
	Contract          *umpirespb.EvaluationContract
	RawEvidence       artifactv2.RawEvidence
	OperationalStatus umpirespb.OperationalStatus
	CleanupStatus     umpirespb.CleanupStatus
}

// Evaluate interprets one contract without consulting Lean or a model registry.
func Evaluate(ctx context.Context, request Request) *umpirespb.EvaluationResult {
	if ctx == nil {
		ctx = context.Background()
	}
	interpreter := newInterpreter(ctx, request)
	return interpreter.evaluate()
}

type interpreter struct {
	ctx      context.Context
	cancel   context.CancelFunc
	request  Request
	contract *umpirespb.EvaluationContract
	result   *umpirespb.EvaluationResult
	work     workTracker
}

func newInterpreter(ctx context.Context, request Request) *interpreter {
	result := &umpirespb.EvaluationResult{
		ToolingStatus:     umpirespb.TOOLING_STATUS_SUCCEEDED,
		OperationalStatus: normalizeOperationalStatus(request.OperationalStatus),
		CleanupStatus:     normalizeCleanupStatus(request.CleanupStatus),
		Decision:          umpirespb.CANARY_DECISION_INCONCLUSIVE,
		SemanticStatus:    umpirespb.EVALUATION_STATUS_INCOMPLETE,
	}
	return &interpreter{ctx: ctx, request: request, result: result}
}

func (i *interpreter) evaluate() *umpirespb.EvaluationResult {
	contract, failure := admitContract(i.request.Contract)
	if failure != nil {
		i.failBeforeObservation(umpirespb.TOOLING_STATUS_INVALID_CONTRACT, failure)
		return i.finish()
	}
	i.contract = contract
	i.result.Version = proto.CloneOf(contract.GetVersion())
	i.result.ContractChecksum = append([]byte(nil), contract.GetArtifactChecksum()...)
	i.result.RunIdentity = i.request.RawEvidence.RunIdentity
	i.result.KnownGaps = collectKnownGaps(i.request.RawEvidence.KnownGaps, contract.GetKnownGaps())
	i.work.limit = contract.GetLimits().GetMaxEvaluationWork()

	duration := time.Duration(contract.GetLimits().GetMaxTotalDurationMilliseconds()) * time.Millisecond
	i.ctx, i.cancel = context.WithTimeout(i.ctx, duration)
	defer i.cancel()

	if failure = i.validateInput(); failure != nil {
		if failure.canceled {
			i.failBeforeObservation(umpirespb.TOOLING_STATUS_CANCELED, failure)
		} else if failure.code == umpirespb.DIAGNOSTIC_CODE_LIMIT_REACHED {
			i.failObservation(failure)
		} else {
			i.failBeforeObservation(umpirespb.TOOLING_STATUS_INVALID_INPUT, failure)
		}
		return i.finish()
	}

	observation, failure := i.evaluateObservation()
	if failure != nil {
		if failure.canceled {
			i.result.ToolingStatus = umpirespb.TOOLING_STATUS_CANCELED
		}
		i.failObservation(failure)
		return i.finish()
	}
	i.result.Observation = observation

	link, failure := i.applyLink(observation.GetTrace(), observation.GetEvidenceLinks())
	if failure != nil {
		if failure.canceled {
			i.result.ToolingStatus = umpirespb.TOOLING_STATUS_CANCELED
		}
		i.failLink(failure)
		return i.finish()
	}
	i.result.ImplementationLink = link

	properties, status, failure := i.evaluateProperties(link)
	i.result.Properties = properties
	i.result.SemanticStatus = status
	if failure != nil {
		i.result.Diagnostics = append(i.result.Diagnostics, failure.diagnostic())
		if failure.canceled {
			i.result.ToolingStatus = umpirespb.TOOLING_STATUS_CANCELED
		}
	}
	i.result.Decision = localDecision(i.result)
	return i.finish()
}

func admitContract(contract *umpirespb.EvaluationContract) (*umpirespb.EvaluationContract, *evaluationFailure) {
	if contract == nil {
		return nil, malformedFailure("contract is required")
	}
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(contract)
	if err != nil {
		return nil, malformedFailure(fmt.Sprintf("encode contract: %v", err))
	}
	admitted, err := evaluationcontract.Admit(encoded)
	if err != nil {
		return nil, malformedFailure(err.Error())
	}
	return admitted, nil
}

func (i *interpreter) validateInput() *evaluationFailure {
	if err := i.ctx.Err(); err != nil {
		return canceledFailure(err)
	}
	encoded, err := artifactv2.CanonicalRawEvidenceBytes(i.request.RawEvidence)
	if err != nil {
		return invalidInputFailure(fmt.Sprintf("encode Raw Evidence: %v", err))
	}
	if int64(len(encoded)) > i.contract.GetLimits().GetMaxInputBytes() {
		return limitFailure("input bytes exceed the contract Limit", "input-bytes",
			i.contract.GetLimits().GetMaxInputBytes(), int64(len(encoded)))
	}
	if int64(len(i.request.RawEvidence.Facts)) > i.contract.GetLimits().GetMaxEvidenceRecords() {
		return limitFailure("Evidence records exceed the contract Limit", "evidence-records",
			i.contract.GetLimits().GetMaxEvidenceRecords(), int64(len(i.request.RawEvidence.Facts)))
	}
	if err := artifactv2.ValidateRawEvidence(i.request.RawEvidence); err != nil {
		return invalidInputFailure(err.Error())
	}
	if err := artifactv2.VerifyRawEvidenceProvenanceChecksum(i.request.RawEvidence); err != nil {
		return invalidInputFailure(err.Error())
	}
	if err := artifactv2.VerifyRawEvidenceArtifactChecksum(i.request.RawEvidence); err != nil {
		return invalidInputFailure(err.Error())
	}
	if !artifactBindingMatches(i.contract.GetExperiment(), i.request.RawEvidence.Experiment) ||
		!artifactBindingMatches(i.contract.GetRuntimeConfig(), i.request.RawEvidence.RuntimeConfiguration) {
		return &evaluationFailure{
			class: umpirespb.DIAGNOSTIC_CLASS_CONFLICT, code: umpirespb.DIAGNOSTIC_CODE_CORRELATION,
			detail: "Raw Evidence artifact bindings do not match the contract",
		}
	}
	return nil
}

func artifactBindingMatches(expected *umpirespb.ArtifactBinding, actual artifactv2.ArtifactBinding) bool {
	return expected != nil && expected.GetFormatVersion() == actual.FormatVersion &&
		expected.GetArtifactChecksum() == actual.ArtifactChecksum &&
		expected.GetBehaviorFingerprint() == actual.BehaviorFingerprint &&
		expected.GetProvenanceChecksum() == actual.ProvenanceChecksum
}

func (i *interpreter) failBeforeObservation(status umpirespb.ToolingStatus, failure *evaluationFailure) {
	i.result.ToolingStatus = status
	i.result.Observation = &umpirespb.ObservationEvaluationResult{
		Status: umpirespb.OBSERVATION_STATUS_UNKNOWN, Diagnostics: []*umpirespb.Diagnostic{failure.diagnostic()},
	}
	i.result.ImplementationLink = &umpirespb.ImplementationLinkResult{
		Status: umpirespb.IMPLEMENTATION_LINK_STATUS_NOT_EVALUATED,
	}
	i.result.Diagnostics = []*umpirespb.Diagnostic{failure.diagnostic()}
}

func (i *interpreter) failObservation(failure *evaluationFailure) {
	i.result.Observation = &umpirespb.ObservationEvaluationResult{
		Status: observationStatus(failure.class), Diagnostics: []*umpirespb.Diagnostic{failure.diagnostic()},
	}
	i.result.ImplementationLink = &umpirespb.ImplementationLinkResult{
		Status: umpirespb.IMPLEMENTATION_LINK_STATUS_NOT_EVALUATED,
	}
	i.result.Diagnostics = []*umpirespb.Diagnostic{failure.diagnostic()}
}

func (i *interpreter) failLink(failure *evaluationFailure) {
	i.result.ImplementationLink = &umpirespb.ImplementationLinkResult{
		Status: linkStatus(failure.class), Diagnostics: []*umpirespb.Diagnostic{failure.diagnostic()},
	}
	i.result.Diagnostics = []*umpirespb.Diagnostic{failure.diagnostic()}
}

func (i *interpreter) finish() *umpirespb.EvaluationResult {
	i.result.Work = i.work.result()
	if i.result.Observation == nil {
		i.result.Observation = &umpirespb.ObservationEvaluationResult{Status: umpirespb.OBSERVATION_STATUS_UNKNOWN}
	}
	if i.result.ImplementationLink == nil {
		i.result.ImplementationLink = &umpirespb.ImplementationLinkResult{
			Status: umpirespb.IMPLEMENTATION_LINK_STATUS_NOT_EVALUATED,
		}
	}
	if i.contract != nil && i.ctx.Err() != nil && i.result.ToolingStatus == umpirespb.TOOLING_STATUS_SUCCEEDED {
		failure := canceledFailure(i.ctx.Err())
		i.result.ToolingStatus = umpirespb.TOOLING_STATUS_CANCELED
		i.result.SemanticStatus = umpirespb.EVALUATION_STATUS_INCOMPLETE
		i.result.Diagnostics = append(i.result.Diagnostics, failure.diagnostic())
	}
	if i.contract != nil {
		i.boundDiagnostics(i.contract.GetLimits().GetMaxDiagnosticBytes())
	}
	i.result.Decision = localDecision(i.result)
	return i.boundedResult()
}

func (i *interpreter) boundDiagnostics(limit int64) {
	i.result.Diagnostics = boundedDiagnostics(i.result.GetDiagnostics(), limit)
	i.result.Observation.Diagnostics = boundedDiagnostics(i.result.GetObservation().GetDiagnostics(), limit)
	i.result.ImplementationLink.Diagnostics = boundedDiagnostics(i.result.GetImplementationLink().GetDiagnostics(), limit)
	for _, property := range i.result.GetProperties() {
		property.Diagnostics = boundedDiagnostics(property.GetDiagnostics(), limit)
		for _, clause := range property.GetClauses() {
			clause.Diagnostics = boundedDiagnostics(clause.GetDiagnostics(), limit)
		}
	}
}

func boundedDiagnostics(diagnostics []*umpirespb.Diagnostic, limit int64) []*umpirespb.Diagnostic {
	result := make([]*umpirespb.Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		bounded := proto.CloneOf(diagnostic)
		if int64(proto.Size(bounded)) > limit && int64(len(bounded.GetDetail())) > limit {
			bounded.Detail = bounded.GetDetail()[:int(limit)]
			for !utf8.ValidString(bounded.GetDetail()) {
				bounded.Detail = bounded.GetDetail()[:len(bounded.GetDetail())-1]
			}
		}
		if int64(proto.Size(bounded)) > limit {
			bounded.Detail = ""
		}
		if int64(proto.Size(bounded)) > limit {
			bounded.RelatedDefinitionIds = nil
			bounded.Coordinate = nil
			bounded.AppliedLimit = nil
			bounded.ObservedCount = 0
		}
		if int64(proto.Size(bounded)) <= limit {
			result = append(result, bounded)
		}
	}
	return result
}

func (i *interpreter) boundedResult() *umpirespb.EvaluationResult {
	if i.contract == nil {
		return i.result
	}
	limit := i.contract.GetLimits().GetMaxResultBytes()
	observed := int64(proto.Size(i.result))
	if observed <= limit {
		return i.result
	}
	failure := limitFailure("Evaluation Result exceeds the contract Limit", "result-bytes", limit, observed)
	diagnostic := boundedDiagnostics([]*umpirespb.Diagnostic{failure.diagnostic()},
		i.contract.GetLimits().GetMaxDiagnosticBytes())
	result := &umpirespb.EvaluationResult{
		Version: proto.CloneOf(i.result.GetVersion()), ContractChecksum: append([]byte(nil), i.result.GetContractChecksum()...),
		RunIdentity: i.result.GetRunIdentity(), ToolingStatus: umpirespb.TOOLING_STATUS_INTERNAL_ERROR,
		OperationalStatus: i.result.GetOperationalStatus(), CleanupStatus: i.result.GetCleanupStatus(),
		Observation: &umpirespb.ObservationEvaluationResult{
			Status: umpirespb.OBSERVATION_STATUS_UNKNOWN, Diagnostics: diagnostic,
		},
		ImplementationLink: &umpirespb.ImplementationLinkResult{
			Status: umpirespb.IMPLEMENTATION_LINK_STATUS_NOT_EVALUATED,
		},
		SemanticStatus: umpirespb.EVALUATION_STATUS_INCOMPLETE,
		Decision:       umpirespb.CANARY_DECISION_INCONCLUSIVE,
		Work:           proto.CloneOf(i.result.GetWork()), Diagnostics: diagnostic,
	}
	if int64(proto.Size(result)) <= limit {
		return result
	}
	minimal := &umpirespb.EvaluationResult{Decision: umpirespb.CANARY_DECISION_INCONCLUSIVE}
	return minimal
}

func normalizeOperationalStatus(status umpirespb.OperationalStatus) umpirespb.OperationalStatus {
	switch status {
	case umpirespb.OPERATIONAL_STATUS_SUCCEEDED,
		umpirespb.OPERATIONAL_STATUS_INCOMPLETE,
		umpirespb.OPERATIONAL_STATUS_FAILED:
		return status
	default:
		return umpirespb.OPERATIONAL_STATUS_INCOMPLETE
	}
}

func normalizeCleanupStatus(status umpirespb.CleanupStatus) umpirespb.CleanupStatus {
	switch status {
	case umpirespb.CLEANUP_STATUS_COMPLETE,
		umpirespb.CLEANUP_STATUS_INCOMPLETE,
		umpirespb.CLEANUP_STATUS_FAILED:
		return status
	default:
		return umpirespb.CLEANUP_STATUS_INCOMPLETE
	}
}

func localDecision(result *umpirespb.EvaluationResult) umpirespb.CanaryDecision {
	trustworthy := result.GetToolingStatus() == umpirespb.TOOLING_STATUS_SUCCEEDED &&
		result.GetOperationalStatus() == umpirespb.OPERATIONAL_STATUS_SUCCEEDED &&
		result.GetObservation().GetStatus() == umpirespb.OBSERVATION_STATUS_ACCEPTED &&
		result.GetImplementationLink().GetStatus() == umpirespb.IMPLEMENTATION_LINK_STATUS_APPLIED &&
		result.GetCleanupStatus() == umpirespb.CLEANUP_STATUS_COMPLETE
	if !trustworthy {
		return umpirespb.CANARY_DECISION_INCONCLUSIVE
	}
	switch result.GetSemanticStatus() {
	case umpirespb.EVALUATION_STATUS_SATISFIED:
		return umpirespb.CANARY_DECISION_PASS
	case umpirespb.EVALUATION_STATUS_VIOLATED:
		return umpirespb.CANARY_DECISION_FAIL
	default:
		return umpirespb.CANARY_DECISION_INCONCLUSIVE
	}
}

func collectKnownGaps(rawGaps []artifactv2.KnownGap, contractGaps []*umpirespb.KnownGap) []*umpirespb.KnownGap {
	result := make([]*umpirespb.KnownGap, 0, len(rawGaps)+len(contractGaps))
	for _, gap := range rawGaps {
		result = append(result, &umpirespb.KnownGap{
			Kind: knownGapKind(gap.Kind), Code: gap.Code,
			Subject: stringPointerValue(gap.Subject), Detail: stringPointerValue(gap.Detail),
		})
	}
	for _, gap := range contractGaps {
		result = append(result, proto.CloneOf(gap))
	}
	return result
}

func knownGapKind(kind string) umpirespb.KnownGapKind {
	switch kind {
	case "capability-contract":
		return umpirespb.KNOWN_GAP_KIND_CAPABILITY_CONTRACT
	case "input":
		return umpirespb.KNOWN_GAP_KIND_INPUT
	case "interpretation":
		return umpirespb.KNOWN_GAP_KIND_INTERPRETATION
	case "claim":
		return umpirespb.KNOWN_GAP_KIND_CLAIM
	default:
		return umpirespb.KNOWN_GAP_KIND_UNSPECIFIED
	}
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
