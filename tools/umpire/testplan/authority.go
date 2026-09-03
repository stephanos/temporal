package testplan

import (
	"bytes"
	"context"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

// ModelProvenanceBinding is the complete model identity checked by a host verifier.
type ModelProvenanceBinding struct {
	PlanChecksum  []byte
	ModelCompiled *umpirespb.ModelCompiledPlanProvenance
}

// ModelProvenanceVerifier resolves independently trusted provenance for one requested model plan.
type ModelProvenanceVerifier func(
	context.Context,
	ModelProvenanceBinding,
) (ModelProvenanceBinding, error)

// AuthorizedPlan is an admitted plan with host-established result authority.
type AuthorizedPlan struct {
	plan                 *umpirespb.PortableTestPlan
	mandatoryResultBytes int
	provenanceOutcome    umpirespb.ProvenanceOutcome
	claimScope           umpirespb.ClaimScope
	requiredObligation   bool
}

// Authorize establishes the only result authority permitted for an admitted plan.
func Authorize(
	ctx context.Context,
	admitted AdmittedPlan,
	verifier ModelProvenanceVerifier,
) (AuthorizedPlan, error) {
	if ctx == nil {
		return AuthorizedPlan{}, admissionError(ErrorProvenance, "$", "verification context is required")
	}
	if admitted.plan == nil {
		return AuthorizedPlan{}, admissionError(ErrorProvenance, "$", "admitted plan is required")
	}
	plan := admitted.Plan()
	authorized := AuthorizedPlan{
		plan:                 plan,
		mandatoryResultBytes: admitted.mandatoryResultBytes,
		requiredObligation:   hasRequiredObligation(plan),
	}
	switch {
	case plan.GetExternal() != nil:
		authorized.provenanceOutcome = umpirespb.PROVENANCE_OUTCOME_EXTERNAL
		authorized.claimScope = umpirespb.CLAIM_SCOPE_PLAN_LOCAL
		return authorized, nil
	case plan.GetModelCompiled() != nil:
		if verifier == nil {
			return AuthorizedPlan{}, admissionError(ErrorProvenance, "$.modelCompiled", "host provenance verifier is required")
		}
		expected := modelProvenanceBinding(plan)
		verified, err := verifier(ctx, cloneModelProvenanceBinding(expected))
		if err != nil {
			return AuthorizedPlan{}, admissionError(ErrorProvenance, "$.modelCompiled", "host provenance verification failed: %w", err)
		}
		if !sameModelProvenanceBinding(expected, verified) {
			return AuthorizedPlan{}, admissionError(ErrorProvenance, "$.modelCompiled", "verified provenance does not match the requested plan")
		}
		authorized.provenanceOutcome = umpirespb.PROVENANCE_OUTCOME_MODEL_VERIFIED
		authorized.claimScope = umpirespb.CLAIM_SCOPE_MODEL_BOUND
		return authorized, nil
	default:
		return AuthorizedPlan{}, admissionError(ErrorProvenance, "$.provenance", "admitted plan has no provenance kind")
	}
}

// Plan returns an independent copy of the authorized plan.
func (p AuthorizedPlan) Plan() *umpirespb.PortableTestPlan {
	return proto.CloneOf(p.plan)
}

// MandatoryResultBytes returns the reserved bytes required by this plan's result envelope.
func (p AuthorizedPlan) MandatoryResultBytes() int {
	return p.mandatoryResultBytes
}

// ResultLimitExceeded returns the reserved typed result for output exhaustion.
func (p AuthorizedPlan) ResultLimitExceeded() *umpirespb.ExecutionResult {
	if p.plan == nil {
		return nil
	}
	return p.bindResult(mandatoryResult(p.plan))
}

// ScopeResult binds an unscoped runtime result to the authorized plan and its unresolved obligations.
func (p AuthorizedPlan) ScopeResult(result *umpirespb.ExecutionResult) (*umpirespb.ExecutionResult, error) {
	if p.plan == nil || result == nil {
		return nil, admissionError(ErrorResultAuthority, "$", "authorized plan and result are required")
	}
	if result.GetVersion() != nil && !proto.Equal(result.GetVersion(), p.plan.GetVersion()) {
		return nil, admissionError(ErrorResultAuthority, "$.version", "result version is crossed with the authorized plan")
	}
	if len(result.GetPlanChecksum()) != 0 && !bytes.Equal(result.GetPlanChecksum(), p.plan.GetPlanChecksum()) {
		return nil, admissionError(ErrorResultAuthority, "$.planChecksum", "result checksum is crossed with the authorized plan")
	}
	if result.GetProvenanceOutcome() != umpirespb.PROVENANCE_OUTCOME_UNSPECIFIED ||
		result.GetClaimScope() != umpirespb.CLAIM_SCOPE_UNSPECIFIED {
		return nil, admissionError(ErrorResultAuthority, "$.claimScope", "result authority must be assigned by the authorized plan")
	}
	if len(result.GetKnownGaps()) != 0 || len(result.GetUnresolvedExternalObligations()) != 0 {
		return nil, admissionError(ErrorResultAuthority, "$.unresolvedExternalObligations", "plan-owned gaps and obligations must not be supplied by the result producer")
	}

	return p.bindResult(result), nil
}

func (p AuthorizedPlan) bindResult(result *umpirespb.ExecutionResult) *umpirespb.ExecutionResult {
	scoped := proto.CloneOf(result)
	scoped.Version = proto.CloneOf(p.plan.GetVersion())
	scoped.PlanChecksum = bytes.Clone(p.plan.GetPlanChecksum())
	scoped.ProvenanceOutcome = p.provenanceOutcome
	scoped.ClaimScope = p.claimScope
	scoped.KnownGaps = cloneMessages(p.plan.GetKnownGaps())
	scoped.UnresolvedExternalObligations = cloneMessages(p.plan.GetExternalObligations())
	if p.claimScope == umpirespb.CLAIM_SCOPE_MODEL_BOUND && p.requiredObligation &&
		scoped.GetDecision() == umpirespb.EXECUTION_DECISION_PASS {
		scoped.Decision = umpirespb.EXECUTION_DECISION_INCONCLUSIVE
	}
	return scoped
}

func modelProvenanceBinding(plan *umpirespb.PortableTestPlan) ModelProvenanceBinding {
	return ModelProvenanceBinding{
		PlanChecksum:  bytes.Clone(plan.GetPlanChecksum()),
		ModelCompiled: proto.CloneOf(plan.GetModelCompiled()),
	}
}

func cloneModelProvenanceBinding(binding ModelProvenanceBinding) ModelProvenanceBinding {
	return ModelProvenanceBinding{
		PlanChecksum:  bytes.Clone(binding.PlanChecksum),
		ModelCompiled: proto.CloneOf(binding.ModelCompiled),
	}
}

func sameModelProvenanceBinding(left, right ModelProvenanceBinding) bool {
	return bytes.Equal(left.PlanChecksum, right.PlanChecksum) &&
		proto.Equal(left.ModelCompiled, right.ModelCompiled)
}

func hasRequiredObligation(plan *umpirespb.PortableTestPlan) bool {
	for _, obligation := range plan.GetExternalObligations() {
		if obligation.GetKind() == umpirespb.EXTERNAL_VERIFICATION_OBLIGATION_KIND_REQUIRED {
			return true
		}
	}
	return false
}
