// Package testplan owns structural admission and identity for caller-neutral portable plans.
package testplan

import (
	"bytes"
	"crypto/sha256"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

const checksumDomain = "umpire.portable-test-plan/v1"

var deterministicMarshal = proto.MarshalOptions{Deterministic: true}

// AdmittedPlan is an immutable structurally checked plan.
type AdmittedPlan struct {
	plan                 *umpirespb.PortableTestPlan
	mandatoryResultBytes int
}

// Seal fills the deterministic plan checksum and returns an admitted clone.
func Seal(plan *umpirespb.PortableTestPlan) (*umpirespb.PortableTestPlan, error) {
	if plan == nil {
		return nil, admissionError(ErrorMalformedValue, "$", "plan is required")
	}
	sealed := proto.CloneOf(plan)
	sealed.PlanChecksum = nil
	if err := validatePlan(sealed, true); err != nil {
		return nil, err
	}
	checksum, err := expectedChecksum(sealed)
	if err != nil {
		return nil, err
	}
	sealed.PlanChecksum = checksum
	if _, err := Admit(sealed); err != nil {
		return nil, err
	}
	return sealed, nil
}

// Admit checks one decoded plan value without depending on its transport encoding.
func Admit(plan *umpirespb.PortableTestPlan) (AdmittedPlan, error) {
	if err := validatePlan(plan, false); err != nil {
		return AdmittedPlan{}, err
	}
	expected, err := expectedChecksum(plan)
	if err != nil {
		return AdmittedPlan{}, err
	}
	if !bytes.Equal(plan.GetPlanChecksum(), expected) {
		return AdmittedPlan{}, admissionError(ErrorChecksum, "$.planChecksum", "plan checksum mismatch")
	}
	cloned := proto.CloneOf(plan)
	return AdmittedPlan{
		plan:                 cloned,
		mandatoryResultBytes: proto.Size(mandatoryResult(cloned)),
	}, nil
}

func expectedChecksum(plan *umpirespb.PortableTestPlan) ([]byte, error) {
	if plan == nil {
		return nil, admissionError(ErrorMalformedValue, "$", "plan is required")
	}
	preimage := proto.CloneOf(plan)
	preimage.PlanChecksum = nil
	encoded, err := deterministicMarshal.Marshal(preimage)
	if err != nil {
		return nil, admissionError(ErrorMalformedValue, "$", "marshal checksum preimage: %v", err)
	}
	input := make([]byte, 0, len(checksumDomain)+1+len(encoded))
	input = append(input, checksumDomain...)
	input = append(input, '\n')
	input = append(input, encoded...)
	sum := sha256.Sum256(input)
	return sum[:], nil
}

func mandatoryResult(plan *umpirespb.PortableTestPlan) *umpirespb.ExecutionResult {
	result := &umpirespb.ExecutionResult{
		Version:           proto.CloneOf(plan.GetVersion()),
		PlanChecksum:      bytes.Clone(plan.GetPlanChecksum()),
		RunIdentity:       strings.Repeat("0", 36),
		ToolingStatus:     umpirespb.EXECUTION_TOOLING_STATUS_INVALID_PLAN,
		OperationalStatus: umpirespb.EXECUTION_OPERATIONAL_STATUS_INCOMPLETE,
		Observation: &umpirespb.ObservationEvaluationResult{
			Status: umpirespb.OBSERVATION_STATUS_UNKNOWN,
		},
		TraceProjection: &umpirespb.TraceProjectionResult{
			Status: umpirespb.TRACE_PROJECTION_STATUS_NOT_EVALUATED,
		},
		SemanticStatus: umpirespb.EXECUTION_EVALUATION_STATUS_INCOMPLETE,
		CleanupStatus:  umpirespb.EXECUTION_CLEANUP_STATUS_INCOMPLETE,
		Decision:       umpirespb.EXECUTION_DECISION_INCONCLUSIVE,
		Work:           &umpirespb.EvaluationWork{},
		KnownGaps:      cloneMessages(plan.GetKnownGaps()),
		UnresolvedExternalObligations: cloneMessages(
			plan.GetExternalObligations(),
		),
		Diagnostics: []*umpirespb.Diagnostic{{
			DiagnosticClass:      umpirespb.DIAGNOSTIC_CLASS_INVALID,
			Code:                 umpirespb.DIAGNOSTIC_CODE_LIMIT_REACHED,
			RelatedDefinitionIds: []string{plan.GetPlanId()},
			AppliedLimit: &umpirespb.Limit{
				Value: plan.GetLimits().GetOutput().GetMaxResultBytes(),
				Unit:  "bytes",
			},
			Detail: "result byte limit cannot contain the complete semantic result",
		}},
	}
	if plan.GetExternal() != nil {
		result.ProvenanceOutcome = umpirespb.PROVENANCE_OUTCOME_EXTERNAL
		result.ClaimScope = umpirespb.CLAIM_SCOPE_PLAN_LOCAL
	} else if plan.GetModelCompiled() != nil {
		result.ProvenanceOutcome = umpirespb.PROVENANCE_OUTCOME_MODEL_VERIFIED
		result.ClaimScope = umpirespb.CLAIM_SCOPE_MODEL_BOUND
	}
	return result
}

func cloneMessages[T proto.Message](messages []T) []T {
	cloned := make([]T, len(messages))
	for index, message := range messages {
		cloned[index] = proto.CloneOf(message)
	}
	return cloned
}

// Plan returns an independent copy of the admitted value.
func (p AdmittedPlan) Plan() *umpirespb.PortableTestPlan {
	return proto.CloneOf(p.plan)
}

// Checksum returns an independent copy of the admitted identity.
func (p AdmittedPlan) Checksum() []byte {
	return bytes.Clone(p.plan.GetPlanChecksum())
}

// MandatoryResultBytes returns the bytes reserved for the non-success result envelope.
func (p AdmittedPlan) MandatoryResultBytes() int {
	return p.mandatoryResultBytes
}
