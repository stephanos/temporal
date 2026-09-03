package testplan

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"google.golang.org/protobuf/proto"
)

func TestAuthorizeExternalPlanIsAlwaysPlanLocal(t *testing.T) {
	authorized := authorizeTestPlan(t, testPlan(), nil)

	result, err := authorized.ScopeResult(&umpirespb.ExecutionResult{
		Decision: umpirespb.EXECUTION_DECISION_PASS,
	})

	require.NoError(t, err)
	require.Equal(t, umpirespb.PROVENANCE_OUTCOME_EXTERNAL, result.GetProvenanceOutcome())
	require.Equal(t, umpirespb.CLAIM_SCOPE_PLAN_LOCAL, result.GetClaimScope())
	require.Equal(t, umpirespb.EXECUTION_DECISION_PASS, result.GetDecision())
	requirePlanResultBindings(t, authorized.Plan(), result)
}

func TestAuthorizeModelPlanRequiresExactHostProvenance(t *testing.T) {
	plan := testModelPlan()
	sealed, err := Seal(plan)
	require.NoError(t, err)
	admitted, err := Admit(sealed)
	require.NoError(t, err)
	var verified ModelProvenanceBinding

	authorized, err := Authorize(context.Background(), admitted, func(
		_ context.Context,
		requested ModelProvenanceBinding,
	) (ModelProvenanceBinding, error) {
		verified = requested
		return requested, nil
	})

	require.NoError(t, err)
	require.Equal(t, sealed.GetPlanChecksum(), verified.PlanChecksum)
	protorequire.ProtoEqual(t, sealed.GetModelCompiled(), verified.ModelCompiled)
	result, err := authorized.ScopeResult(&umpirespb.ExecutionResult{
		Decision: umpirespb.EXECUTION_DECISION_PASS,
	})
	require.NoError(t, err)
	require.Equal(t, umpirespb.PROVENANCE_OUTCOME_MODEL_VERIFIED, result.GetProvenanceOutcome())
	require.Equal(t, umpirespb.CLAIM_SCOPE_MODEL_BOUND, result.GetClaimScope())
	require.Equal(t, umpirespb.EXECUTION_DECISION_PASS, result.GetDecision())
}

func TestAuthorizeRejectsUnverifiedModelProvenance(t *testing.T) {
	errExpired := errors.New("trusted provenance expired")
	errUnsupported := errors.New("compiler contract is unsupported")
	testCases := []struct {
		name     string
		verifier ModelProvenanceVerifier
	}{
		{name: "missing verifier"},
		{
			name: "expired",
			verifier: func(context.Context, ModelProvenanceBinding) (ModelProvenanceBinding, error) {
				return ModelProvenanceBinding{}, errExpired
			},
		},
		{
			name: "unsupported",
			verifier: func(context.Context, ModelProvenanceBinding) (ModelProvenanceBinding, error) {
				return ModelProvenanceBinding{}, errUnsupported
			},
		},
		{
			name: "checksum mismatch",
			verifier: func(_ context.Context, requested ModelProvenanceBinding) (ModelProvenanceBinding, error) {
				requested.PlanChecksum[0]++
				return requested, nil
			},
		},
		{
			name: "binding mismatch",
			verifier: func(_ context.Context, requested ModelProvenanceBinding) (ModelProvenanceBinding, error) {
				requested.ModelCompiled.Test.BehaviorFingerprint = testArtifactBinding("ignored").GetArtifactChecksum()
				return requested, nil
			},
		},
		{
			name: "silent downgrade",
			verifier: func(context.Context, ModelProvenanceBinding) (ModelProvenanceBinding, error) {
				return ModelProvenanceBinding{}, nil
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			sealed, err := Seal(testModelPlan())
			require.NoError(t, err)
			admitted, err := Admit(sealed)
			require.NoError(t, err)

			_, err = Authorize(context.Background(), admitted, testCase.verifier)

			requireAuthorityError(t, err, ErrorProvenance)
		})
	}
}

func TestScopeResultRejectsCallerOwnedAuthority(t *testing.T) {
	authorized := authorizeTestPlan(t, testPlan(), nil)
	testCases := []struct {
		name   string
		result *umpirespb.ExecutionResult
	}{
		{
			name: "forged model scope",
			result: &umpirespb.ExecutionResult{
				ProvenanceOutcome: umpirespb.PROVENANCE_OUTCOME_MODEL_VERIFIED,
				ClaimScope:        umpirespb.CLAIM_SCOPE_MODEL_BOUND,
			},
		},
		{
			name: "crossed plan checksum",
			result: &umpirespb.ExecutionResult{
				PlanChecksum: []byte("another plan"),
			},
		},
		{
			name: "caller supplied obligations",
			result: &umpirespb.ExecutionResult{
				UnresolvedExternalObligations: []*umpirespb.ExternalVerificationObligation{{
					Definition: testBinding("umpire.obligation.forged"),
				}},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := authorized.ScopeResult(testCase.result)
			requireAuthorityError(t, err, ErrorResultAuthority)
		})
	}
}

func TestScopeResultAppliesExternalObligationPolicy(t *testing.T) {
	testCases := []struct {
		name       string
		model      bool
		kind       umpirespb.ExternalVerificationObligationKind
		wantScope  umpirespb.ClaimScope
		wantResult umpirespb.ExecutionDecision
	}{
		{
			name:      "external required remains plan local",
			kind:      umpirespb.EXTERNAL_VERIFICATION_OBLIGATION_KIND_REQUIRED,
			wantScope: umpirespb.CLAIM_SCOPE_PLAN_LOCAL, wantResult: umpirespb.EXECUTION_DECISION_PASS,
		},
		{
			name: "model advisory permits bounded success", model: true,
			kind:      umpirespb.EXTERNAL_VERIFICATION_OBLIGATION_KIND_ADVISORY,
			wantScope: umpirespb.CLAIM_SCOPE_MODEL_BOUND, wantResult: umpirespb.EXECUTION_DECISION_PASS,
		},
		{
			name: "model required prevents complete success", model: true,
			kind:      umpirespb.EXTERNAL_VERIFICATION_OBLIGATION_KIND_REQUIRED,
			wantScope: umpirespb.CLAIM_SCOPE_MODEL_BOUND, wantResult: umpirespb.EXECUTION_DECISION_INCONCLUSIVE,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			plan := testPlan()
			plan.ExternalObligations[0].Kind = testCase.kind
			var verifier ModelProvenanceVerifier
			if testCase.model {
				plan.Provenance = &umpirespb.PortableTestPlan_ModelCompiled{ModelCompiled: testModelProvenance()}
				verifier = acceptExactModelProvenance
			}
			authorized := authorizeTestPlan(t, plan, verifier)

			result, err := authorized.ScopeResult(&umpirespb.ExecutionResult{
				Decision: umpirespb.EXECUTION_DECISION_PASS,
			})

			require.NoError(t, err)
			require.Equal(t, testCase.wantScope, result.GetClaimScope())
			require.Equal(t, testCase.wantResult, result.GetDecision())
			requirePlanResultBindings(t, authorized.Plan(), result)
		})
	}
}

func authorizeTestPlan(
	t *testing.T,
	plan *umpirespb.PortableTestPlan,
	verifier ModelProvenanceVerifier,
) AuthorizedPlan {
	t.Helper()
	sealed, err := Seal(plan)
	require.NoError(t, err)
	admitted, err := Admit(sealed)
	require.NoError(t, err)
	authorized, err := Authorize(context.Background(), admitted, verifier)
	require.NoError(t, err)
	return authorized
}

func acceptExactModelProvenance(
	_ context.Context,
	requested ModelProvenanceBinding,
) (ModelProvenanceBinding, error) {
	return requested, nil
}

func requirePlanResultBindings(
	t *testing.T,
	plan *umpirespb.PortableTestPlan,
	result *umpirespb.ExecutionResult,
) {
	t.Helper()
	protorequire.ProtoEqual(t, plan.GetVersion(), result.GetVersion())
	require.Equal(t, plan.GetPlanChecksum(), result.GetPlanChecksum())
	require.Len(t, result.GetKnownGaps(), len(plan.GetKnownGaps()))
	for index := range plan.GetKnownGaps() {
		protorequire.ProtoEqual(t, plan.GetKnownGaps()[index], result.GetKnownGaps()[index])
	}
	require.Len(t, result.GetUnresolvedExternalObligations(), len(plan.GetExternalObligations()))
	for index := range plan.GetExternalObligations() {
		protorequire.ProtoEqual(t, plan.GetExternalObligations()[index], result.GetUnresolvedExternalObligations()[index])
	}
}

func requireAuthorityError(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	require.Error(t, err)
	code, ok := CodeOf(err)
	require.True(t, ok)
	require.Equal(t, want, code)
}

func testModelPlan() *umpirespb.PortableTestPlan {
	plan := testPlan()
	plan.Provenance = &umpirespb.PortableTestPlan_ModelCompiled{ModelCompiled: testModelProvenance()}
	return proto.CloneOf(plan)
}
