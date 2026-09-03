package portableevaluation

import (
	"context"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/evaluationcontract"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/testplan"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

// PortableRequest binds one authorized plan to the exact closed runtime Evidence.
type PortableRequest struct {
	Plan                         testplan.AuthorizedPlan
	RawEvidence                  artifactv2.RawEvidence
	ExpectedRunIdentity          string
	ExpectedExperiment           artifactv2.ArtifactBinding
	ExpectedRuntimeConfiguration artifactv2.ArtifactBinding
	ExpectedRun                  artifactv2.ArtifactBinding
	ExpectedClosures             []artifactv2.SourceClosure
	OperationalStatus            umpirespb.OperationalStatus
	CleanupStatus                umpirespb.CleanupStatus
}

// EvaluatePortable reuses the portable interpreter for one authorized typed plan.
func EvaluatePortable(ctx context.Context, request PortableRequest) *umpirespb.ExecutionResult {
	plan := request.Plan.Plan()
	if plan == nil {
		return nil
	}
	contract := portableContract(plan, request.ExpectedExperiment, request.ExpectedRuntimeConfiguration)
	interpreter := newInterpreter(ctx, Request{
		RawEvidence:         request.RawEvidence,
		ExpectedRunIdentity: request.ExpectedRunIdentity,
		ExpectedRun:         request.ExpectedRun,
		ExpectedClosures:    request.ExpectedClosures,
		OperationalStatus:   request.OperationalStatus,
		CleanupStatus:       request.CleanupStatus,
	})
	interpreter.directPlanTrace = plan.GetVerification().GetDirectPlanTrace() != nil
	evaluated := interpreter.evaluateAdmitted(contract)
	result, ok := boundedExecutionResult(
		request.Plan,
		executionResult(plan, evaluated, interpreter.directPlanTrace),
		int(plan.GetLimits().GetOutput().GetMaxResultBytes()),
	)
	if !ok {
		return request.Plan.ResultLimitExceeded()
	}
	return result
}

func boundedExecutionResult(
	plan testplan.AuthorizedPlan,
	complete *umpirespb.ExecutionResult,
	limit int,
) (*umpirespb.ExecutionResult, bool) {
	result, err := plan.ScopeResult(&umpirespb.ExecutionResult{
		RunIdentity: complete.GetRunIdentity(), ToolingStatus: complete.GetToolingStatus(),
		OperationalStatus: complete.GetOperationalStatus(),
		Observation:       &umpirespb.ObservationEvaluationResult{Status: complete.GetObservation().GetStatus()},
		TraceProjection:   &umpirespb.TraceProjectionResult{Status: complete.GetTraceProjection().GetStatus()},
		SemanticStatus:    complete.GetSemanticStatus(), CleanupStatus: complete.GetCleanupStatus(),
		Decision: complete.GetDecision(),
		Work: &umpirespb.EvaluationWork{
			Total: complete.GetWork().GetTotal(), Limit: complete.GetWork().GetLimit(),
		},
	})
	if err != nil || proto.Size(result) > limit {
		return nil, false
	}
	budget := executionResultBudget{result: result, size: proto.Size(result), limit: limit}
	if !budget.appendObservation(complete.GetObservation()) ||
		!budget.appendProjection(complete.GetTraceProjection()) ||
		!budget.appendProperties(complete.GetProperties()) ||
		!budget.appendWork(complete.GetWork()) ||
		!budget.appendEvidenceAndDiagnostics(complete.GetEvidenceLinks(), complete.GetDiagnostics()) {
		return nil, false
	}
	if proto.Size(result) != budget.size {
		return nil, false
	}
	return result, true
}

type executionResultBudget struct {
	result *umpirespb.ExecutionResult
	size   int
	limit  int
}

func (b *executionResultBudget) appendObservation(complete *umpirespb.ObservationEvaluationResult) bool {
	if trace := complete.GetTrace(); trace != nil && !b.appendNested(8, 2, b.result.GetObservation(), trace, func() {
		b.result.Observation.Trace = proto.CloneOf(trace)
	}) {
		return false
	}
	for _, value := range complete.GetEvidenceLinks() {
		if !b.appendNested(8, 3, b.result.GetObservation(), value, func() {
			b.result.Observation.EvidenceLinks = append(b.result.Observation.EvidenceLinks, proto.CloneOf(value))
		}) {
			return false
		}
	}
	for _, value := range complete.GetDiagnostics() {
		if !b.appendNested(8, 4, b.result.GetObservation(), value, func() {
			b.result.Observation.Diagnostics = append(b.result.Observation.Diagnostics, proto.CloneOf(value))
		}) {
			return false
		}
	}
	return true
}

func (b *executionResultBudget) appendProjection(complete *umpirespb.TraceProjectionResult) bool {
	if trace := complete.GetTrace(); trace != nil && !b.appendNested(9, 2, b.result.GetTraceProjection(), trace, func() {
		b.result.TraceProjection.Trace = proto.CloneOf(trace)
	}) {
		return false
	}
	for _, value := range complete.GetApplications() {
		if !b.appendNested(9, 3, b.result.GetTraceProjection(), value, func() {
			b.result.TraceProjection.Applications = append(b.result.TraceProjection.Applications, proto.CloneOf(value))
		}) {
			return false
		}
	}
	for _, value := range complete.GetDiagnostics() {
		if !b.appendNested(9, 4, b.result.GetTraceProjection(), value, func() {
			b.result.TraceProjection.Diagnostics = append(b.result.TraceProjection.Diagnostics, proto.CloneOf(value))
		}) {
			return false
		}
	}
	return true
}

func (b *executionResultBudget) appendProperties(properties []*umpirespb.PropertyResult) bool {
	for _, property := range properties {
		if !b.appendTopLevel(10, property, func() {
			b.result.Properties = append(b.result.Properties, proto.CloneOf(property))
		}) {
			return false
		}
	}
	return true
}

func (b *executionResultBudget) appendWork(work *umpirespb.EvaluationWork) bool {
	for _, charge := range work.GetCharges() {
		if !b.appendNested(14, 1, b.result.GetWork(), charge, func() {
			b.result.Work.Charges = append(b.result.Work.Charges, proto.CloneOf(charge))
		}) {
			return false
		}
	}
	return true
}

func (b *executionResultBudget) appendEvidenceAndDiagnostics(
	links []*umpirespb.EvidenceLink,
	diagnostics []*umpirespb.Diagnostic,
) bool {
	for _, link := range links {
		if !b.appendTopLevel(15, link, func() {
			b.result.EvidenceLinks = append(b.result.EvidenceLinks, proto.CloneOf(link))
		}) {
			return false
		}
	}
	for _, diagnostic := range diagnostics {
		if !b.appendTopLevel(18, diagnostic, func() {
			b.result.Diagnostics = append(b.result.Diagnostics, proto.CloneOf(diagnostic))
		}) {
			return false
		}
	}
	return true
}

func (b *executionResultBudget) appendTopLevel(
	field protowire.Number,
	item proto.Message,
	appendItem func(),
) bool {
	growth := messageFieldSize(field, proto.Size(item))
	if b.size+growth > b.limit {
		return false
	}
	appendItem()
	b.size += growth
	return true
}

func (b *executionResultBudget) appendNested(
	outerField protowire.Number,
	innerField protowire.Number,
	container proto.Message,
	item proto.Message,
	appendItem func(),
) bool {
	before := proto.Size(container)
	after := before + messageFieldSize(innerField, proto.Size(item))
	growth := messageFieldSize(outerField, after) - messageFieldSize(outerField, before)
	if b.size+growth > b.limit {
		return false
	}
	appendItem()
	b.size += growth
	return true
}

func messageFieldSize(field protowire.Number, size int) int {
	return protowire.SizeTag(field) + protowire.SizeBytes(size)
}

func portableContract(
	plan *umpirespb.PortableTestPlan,
	experiment artifactv2.ArtifactBinding,
	runtimeConfiguration artifactv2.ArtifactBinding,
) *umpirespb.EvaluationContract {
	verification := plan.GetVerification()
	return &umpirespb.EvaluationContract{
		Version:          proto.CloneOf(plan.GetVersion()),
		ContractId:       plan.GetPlanId(),
		ArtifactChecksum: append([]byte(nil), plan.GetPlanChecksum()...),
		Experiment:       protoArtifactBinding(experiment),
		RuntimeConfig:    protoArtifactBinding(runtimeConfiguration),
		Limits: &umpirespb.EvaluationLimits{
			MaxContractBytes:             evaluationcontract.MaximumContractBytes,
			MaxInputBytes:                plan.GetLimits().GetEvidence().GetMaxBytes(),
			MaxEvidenceRecords:           plan.GetLimits().GetEvidence().GetMaxRecords(),
			MaxExpressionDepth:           plan.GetLimits().GetEvaluation().GetMaxExpressionDepth(),
			MaxCollectionItems:           plan.GetLimits().GetStructural().GetMaxCollectionItems(),
			MaxNatural:                   plan.GetLimits().GetEvaluation().GetMaxNatural(),
			MaxEvaluationWork:            plan.GetLimits().GetEvaluation().GetMaxWork(),
			MaxDiagnosticBytes:           plan.GetLimits().GetOutput().GetMaxDiagnosticBytes(),
			MaxResultBytes:               evaluationcontract.MaximumResultBytes,
			MaxTotalDurationMilliseconds: plan.GetLimits().GetExecution().GetMaxTotalDurationMilliseconds(),
			MaxOperatorCount:             plan.GetLimits().GetStructural().GetMaxOperatorCount(),
		},
		Observation:        proto.CloneOf(verification.GetObservation()),
		ImplementationLink: proto.CloneOf(verification.GetRenameExactLink()),
		Properties:         cloneProperties(verification.GetProperties()),
		KnownGaps:          []*umpirespb.KnownGap{},
	}
}

func executionResult(
	plan *umpirespb.PortableTestPlan,
	result *umpirespb.EvaluationResult,
	direct bool,
) *umpirespb.ExecutionResult {
	projectionStatus := traceProjectionStatus(result.GetImplementationLink().GetStatus())
	if direct && projectionStatus == umpirespb.TRACE_PROJECTION_STATUS_APPLIED {
		projectionStatus = umpirespb.TRACE_PROJECTION_STATUS_DIRECT
	}
	return &umpirespb.ExecutionResult{
		RunIdentity:       result.GetRunIdentity(),
		ToolingStatus:     executionToolingStatus(result.GetToolingStatus()),
		OperationalStatus: executionOperationalStatus(result.GetOperationalStatus()),
		Observation:       proto.CloneOf(result.GetObservation()),
		TraceProjection: &umpirespb.TraceProjectionResult{
			Status:       projectionStatus,
			Trace:        proto.CloneOf(result.GetImplementationLink().GetTrace()),
			Applications: cloneRenameApplications(result.GetImplementationLink().GetApplications()),
			Diagnostics:  cloneDiagnostics(result.GetImplementationLink().GetDiagnostics()),
		},
		Properties:     clonePropertyResults(result.GetProperties()),
		SemanticStatus: executionEvaluationStatus(result.GetSemanticStatus()),
		CleanupStatus:  executionCleanupStatus(result.GetCleanupStatus()),
		Decision:       executionDecision(result.GetDecision()),
		Work:           proto.CloneOf(result.GetWork()),
		EvidenceLinks:  cloneEvidenceLinks(result.GetObservation().GetEvidenceLinks()),
		Diagnostics:    cloneDiagnostics(result.GetDiagnostics()),
	}
}

func protoArtifactBinding(binding artifactv2.ArtifactBinding) *umpirespb.ArtifactBinding {
	return &umpirespb.ArtifactBinding{
		FormatVersion:       binding.FormatVersion,
		ArtifactChecksum:    binding.ArtifactChecksum,
		BehaviorFingerprint: binding.BehaviorFingerprint,
		ProvenanceChecksum:  binding.ProvenanceChecksum,
	}
}

func executionToolingStatus(status umpirespb.ToolingStatus) umpirespb.ExecutionToolingStatus {
	switch status {
	case umpirespb.TOOLING_STATUS_SUCCEEDED:
		return umpirespb.EXECUTION_TOOLING_STATUS_SUCCEEDED
	case umpirespb.TOOLING_STATUS_CANCELED:
		return umpirespb.EXECUTION_TOOLING_STATUS_CANCELED
	case umpirespb.TOOLING_STATUS_INTERNAL_ERROR:
		return umpirespb.EXECUTION_TOOLING_STATUS_INTERNAL_ERROR
	default:
		return umpirespb.EXECUTION_TOOLING_STATUS_INVALID_PLAN
	}
}

func executionOperationalStatus(status umpirespb.OperationalStatus) umpirespb.ExecutionOperationalStatus {
	switch status {
	case umpirespb.OPERATIONAL_STATUS_SUCCEEDED:
		return umpirespb.EXECUTION_OPERATIONAL_STATUS_SUCCEEDED
	case umpirespb.OPERATIONAL_STATUS_FAILED:
		return umpirespb.EXECUTION_OPERATIONAL_STATUS_FAILED
	default:
		return umpirespb.EXECUTION_OPERATIONAL_STATUS_INCOMPLETE
	}
}

func traceProjectionStatus(status umpirespb.ImplementationLinkStatus) umpirespb.TraceProjectionStatus {
	switch status {
	case umpirespb.IMPLEMENTATION_LINK_STATUS_APPLIED:
		return umpirespb.TRACE_PROJECTION_STATUS_APPLIED
	case umpirespb.IMPLEMENTATION_LINK_STATUS_INVALID:
		return umpirespb.TRACE_PROJECTION_STATUS_INVALID
	case umpirespb.IMPLEMENTATION_LINK_STATUS_UNKNOWN:
		return umpirespb.TRACE_PROJECTION_STATUS_UNKNOWN
	case umpirespb.IMPLEMENTATION_LINK_STATUS_CONFLICT:
		return umpirespb.TRACE_PROJECTION_STATUS_CONFLICT
	case umpirespb.IMPLEMENTATION_LINK_STATUS_UNSUPPORTED:
		return umpirespb.TRACE_PROJECTION_STATUS_UNSUPPORTED
	default:
		return umpirespb.TRACE_PROJECTION_STATUS_NOT_EVALUATED
	}
}

func executionEvaluationStatus(status umpirespb.EvaluationStatus) umpirespb.ExecutionEvaluationStatus {
	switch status {
	case umpirespb.EVALUATION_STATUS_SATISFIED:
		return umpirespb.EXECUTION_EVALUATION_STATUS_SATISFIED
	case umpirespb.EVALUATION_STATUS_VIOLATED:
		return umpirespb.EXECUTION_EVALUATION_STATUS_VIOLATED
	default:
		return umpirespb.EXECUTION_EVALUATION_STATUS_INCOMPLETE
	}
}

func executionCleanupStatus(status umpirespb.CleanupStatus) umpirespb.ExecutionCleanupStatus {
	switch status {
	case umpirespb.CLEANUP_STATUS_COMPLETE:
		return umpirespb.EXECUTION_CLEANUP_STATUS_COMPLETE
	case umpirespb.CLEANUP_STATUS_FAILED:
		return umpirespb.EXECUTION_CLEANUP_STATUS_FAILED
	default:
		return umpirespb.EXECUTION_CLEANUP_STATUS_INCOMPLETE
	}
}

func executionDecision(decision umpirespb.CanaryDecision) umpirespb.ExecutionDecision {
	switch decision {
	case umpirespb.CANARY_DECISION_PASS:
		return umpirespb.EXECUTION_DECISION_PASS
	case umpirespb.CANARY_DECISION_FAIL:
		return umpirespb.EXECUTION_DECISION_FAIL
	default:
		return umpirespb.EXECUTION_DECISION_INCONCLUSIVE
	}
}

func cloneProperties(values []*umpirespb.Property) []*umpirespb.Property {
	return proto.CloneOf(&umpirespb.VerificationProgram{Properties: values}).GetProperties()
}

func clonePropertyResults(values []*umpirespb.PropertyResult) []*umpirespb.PropertyResult {
	return proto.CloneOf(&umpirespb.ExecutionResult{Properties: values}).GetProperties()
}

func cloneEvidenceLinks(values []*umpirespb.EvidenceLink) []*umpirespb.EvidenceLink {
	return proto.CloneOf(&umpirespb.ExecutionResult{EvidenceLinks: values}).GetEvidenceLinks()
}

func cloneRenameApplications(values []*umpirespb.RenameExactApplication) []*umpirespb.RenameExactApplication {
	return proto.CloneOf(&umpirespb.TraceProjectionResult{Applications: values}).GetApplications()
}

func cloneDiagnostics(values []*umpirespb.Diagnostic) []*umpirespb.Diagnostic {
	return proto.CloneOf(&umpirespb.ExecutionResult{Diagnostics: values}).GetDiagnostics()
}
