package veil

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

const soundConcreteOutput = `{
  "explored_states": 26,
  "result": "no_violation_found",
  "termination_reason": {"kind": "explored_all_reachable_states"}
}`

const mutatedConcreteOutput = `{
  "result": "found_violation",
  "state_fingerprint": "17207709487085510634",
  "trace": {
    "states": [
      {"fields": "<unrepresentable>", "index": 0, "transition": "after_init"},
      {"fields": "<unrepresentable>", "index": 1, "transition": "DispatchTask"},
      {"fields": "<unrepresentable>", "index": 2, "transition": "AcquireOwnership"},
      {"fields": "<unrepresentable>", "index": 3, "transition": "WorkerReturnsSuccess"},
      {"fields": "<unrepresentable>", "index": 4, "transition": "PersistSuccess"}
    ],
    "theory": "<unrepresentable>"
  },
  "violation": {
    "kind": "safety_failure",
    "violates": ["NexusCancellationWonExcludesSuccess"]
  }
}`

func TestNormalizeConcreteOutputClassifiesSoundInstanceWithoutCompleteness(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	generated, err := Generate(view, Concrete)
	require.NoError(t, err)

	result, err := NormalizeConcreteOutput(view, generated, strings.NewReader(soundConcreteOutput),
		protocol.DefaultDecodeLimit, nil)
	require.NoError(t, err)
	require.Equal(t, protocol.ResultClassExternalNoCounterexample, result.ResultClass)
	require.Equal(t, protocol.TrustBadgeTestedInstance, result.TrustBadge)
	require.False(t, result.Exact)
	require.Equal(t, []string{protocol.VeilConcreteCollisionOmission}, result.Omissions)
	require.Nil(t, result.Trace)
	require.NoError(t, result.Validate())
}

func TestNormalizeConcreteOutputMapsMutationTraceAndRequiresReplay(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation-mutated.first-order.json")
	generated, err := Generate(view, Concrete)
	require.NoError(t, err)

	_, err = NormalizeConcreteOutput(view, generated, strings.NewReader(mutatedConcreteOutput),
		protocol.DefaultDecodeLimit, nil)
	require.ErrorContains(t, err, "accepted canonical replay")

	result, err := NormalizeConcreteOutput(view, generated, strings.NewReader(mutatedConcreteOutput),
		protocol.DefaultDecodeLimit, &protocol.TraceReplayReceipt{
			FormatVersion: protocol.TraceReplayReceiptFormatVersion,
			TraceDigest:   "sha256:1c55f5335caa18361c7033c9d6a49f3affcc1a1250753aa4f4e411885e322654",
			Status:        protocol.TraceReplayAccepted, TrustBadge: protocol.TrustBadgeCheckedCertificate,
			Axioms: []string{},
		})
	require.NoError(t, err)
	require.Equal(t, []protocol.TraceStep{
		{Action: protocol.ActionKindDispatchTask},
		{Action: protocol.ActionKindAcquireOwnership},
		{Action: protocol.ActionKindWorkerReturnsSuccess},
		{Action: protocol.ActionKindPersistSuccess},
	}, result.Trace.Steps)
	require.Equal(t, protocol.ResultClassTraceWitness, result.ResultClass)
	require.Equal(t, protocol.TrustBadgeCheckedCertificate, result.TrustBadge)
	require.True(t, result.Exact)
	require.NoError(t, result.Validate())
}

func TestConcreteReplayInputBindsNormalizedMutationTrace(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation-mutated.first-order.json")
	generated, err := Generate(view, Concrete)
	require.NoError(t, err)

	input, err := ConcreteReplayInput(view, generated, strings.NewReader(mutatedConcreteOutput),
		protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, &protocol.TraceReplayInput{
		FormatVersion: protocol.TraceReplayInputFormatVersion,
		Target:        protocol.TargetIDNexusCancellation,
		Property:      protocol.PropertyIDNexusCancellationWonExcludesSuccess,
		World:         "smoke",
		Variant:       "stale-completion-guard-removed",
		SemanticHash:  "sha256:91939fb7d186499518ed05a76483a9c378a8fe55ca07d8104ad7d1f9e9380e1a",
		Actions: []protocol.ActionKind{
			protocol.ActionKindDispatchTask,
			protocol.ActionKindAcquireOwnership,
			protocol.ActionKindWorkerReturnsSuccess,
			protocol.ActionKindPersistSuccess,
		},
	}, input)

	input, err = ConcreteReplayInput(view, generated, strings.NewReader(soundConcreteOutput),
		protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Nil(t, input)
}

func TestNormalizeConcreteOutputRejectsUnknownTransitionAndViolation(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation-mutated.first-order.json")
	generated, err := Generate(view, Concrete)
	require.NoError(t, err)
	replay := &protocol.TraceReplayReceipt{
		FormatVersion: protocol.TraceReplayReceiptFormatVersion,
		TraceDigest:   "sha256:1c55f5335caa18361c7033c9d6a49f3affcc1a1250753aa4f4e411885e322654",
		Status:        protocol.TraceReplayAccepted, TrustBadge: protocol.TrustBadgeCheckedCertificate,
		Axioms: []string{},
	}

	unknownTransition := strings.Replace(mutatedConcreteOutput, "PersistSuccess", "UnknownAction", 1)
	_, err = NormalizeConcreteOutput(view, generated, strings.NewReader(unknownTransition),
		protocol.DefaultDecodeLimit, replay)
	require.ErrorContains(t, err, `unknown Veil transition "UnknownAction"`)

	wrongViolation := strings.Replace(mutatedConcreteOutput,
		"NexusCancellationWonExcludesSuccess", "DifferentProperty", 1)
	_, err = NormalizeConcreteOutput(view, generated, strings.NewReader(wrongViolation),
		protocol.DefaultDecodeLimit, replay)
	require.ErrorContains(t, err, "does not match first-order property")

	wrongDigest := *replay
	wrongDigest.TraceDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	_, err = NormalizeConcreteOutput(view, generated, strings.NewReader(mutatedConcreteOutput),
		protocol.DefaultDecodeLimit, &wrongDigest)
	require.ErrorContains(t, err, "not bound to normalized trace")

	representableState := strings.Replace(mutatedConcreteOutput, "<unrepresentable>", "state", 1)
	_, err = NormalizeConcreteOutput(view, generated, strings.NewReader(representableState),
		protocol.DefaultDecodeLimit, replay)
	require.ErrorContains(t, err, "unexpected Veil trace state representation")
}

func TestNormalizeConcreteOutputRejectsUnknownAndTrailingJSON(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	generated, err := Generate(view, Concrete)
	require.NoError(t, err)

	unknown := strings.Replace(soundConcreteOutput, `"result":`, `"unknown": true, "result":`, 1)
	_, err = NormalizeConcreteOutput(view, generated, strings.NewReader(unknown),
		protocol.DefaultDecodeLimit, nil)
	require.ErrorContains(t, err, "unknown field")

	_, err = NormalizeConcreteOutput(view, generated, strings.NewReader(soundConcreteOutput+` {}`),
		protocol.DefaultDecodeLimit, nil)
	require.ErrorContains(t, err, "multiple JSON values")
}

func TestCompareReachableStatesUsesGeneratedNexusViews(t *testing.T) {
	require.NoError(t, CompareReachableStates(readFirstOrderView(t,
		"nexus-cancellation.first-order.json")))
	require.NoError(t, CompareReachableStates(readFirstOrderView(t,
		"nexus-cancellation-mutated.first-order.json")))
}

func TestCompareReachableStatesCatchesGeneratedNexusTranslationMutation(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	for index := range view.Actions {
		if view.Actions[index].Identifier == "persist-success" {
			view.Actions[index].Guard = protocol.FirstOrderFormula{
				Kind: protocol.FirstOrderFormulaNot,
				Operand: &protocol.FirstOrderFormula{
					Kind: protocol.FirstOrderFormulaTrue,
				},
			}
		}
	}

	err := CompareReachableStates(view)
	require.ErrorContains(t, err, "first-order translation is missing reachable state")
}

func TestGeneratedBackendResultsRetainTheirTrustClasses(t *testing.T) {
	tests := map[string]struct {
		resultClass protocol.ResultClass
		trustBadge  protocol.TrustBadge
	}{
		"nexus-cancellation-sound-concrete.json": {
			resultClass: protocol.ResultClassExternalNoCounterexample,
			trustBadge:  protocol.TrustBadgeTestedInstance,
		},
		"nexus-cancellation-mutated-concrete.json": {
			resultClass: protocol.ResultClassTraceWitness,
			trustBadge:  protocol.TrustBadgeCheckedCertificate,
		},
		"nexus-cancellation-sound-symbolic.json": {
			resultClass: protocol.ResultClassBoundedSafe,
			trustBadge:  protocol.TrustBadgeReconstructedSolverProof,
		},
		"nexus-cancellation-sound-invariant.json": {
			resultClass: protocol.ResultClassInvariantProved,
			trustBadge:  protocol.TrustBadgeReconstructedSolverProof,
		},
		"nexus-cancellation-sound-invariant-trusted.json": {
			resultClass: protocol.ResultClassInvariantProved,
			trustBadge:  protocol.TrustBadgeTrustedSolver,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			encoded, err := os.ReadFile("results/" + name)
			require.NoError(t, err)
			result, err := protocol.DecodeBackendResult(bytes.NewReader(encoded),
				protocol.DefaultDecodeLimit)
			require.NoError(t, err)
			require.Equal(t, test.resultClass, result.ResultClass)
			require.Equal(t, test.trustBadge, result.TrustBadge)
		})
	}
}

func readFirstOrderView(t *testing.T, name string) protocol.FirstOrderView {
	t.Helper()
	encoded, err := os.ReadFile("../../protocol/generated/" + name)
	require.NoError(t, err)
	view, err := protocol.DecodeFirstOrderView(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	return view
}
