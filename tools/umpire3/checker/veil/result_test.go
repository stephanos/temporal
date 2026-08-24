package veil

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire3/checker/finite"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
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

func TestNormalizeConcreteOutputBindsCompiledVeilDeclarations(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-sound.json")

	result, err := NormalizeConcreteOutput(view, binding,
		strings.NewReader(wrapConcreteOutput(t, binding.Binding, soundConcreteOutput)),
		protocolexperiment.DefaultDecodeLimit, canonicalExecutionLimits(), nil)
	require.NoError(t, err)
	require.Equal(t, binding.ArtifactDigest, result.BindingArtifactDigest)

	wrong := binding.Binding
	wrong.ModuleName = "WrongModule"
	_, err = NormalizeConcreteOutput(view, binding,
		strings.NewReader(wrapConcreteOutput(t, wrong, soundConcreteOutput)),
		protocolexperiment.DefaultDecodeLimit, canonicalExecutionLimits(), nil)
	require.ErrorContains(t, err, "compiled Veil binding")
}

func TestDefaultMutatedArtifactsRemainCanonicallyBound(t *testing.T) {
	view, found, err := finite.DefaultFirstOrderView(
		protocolcatalog.TargetIDNexusCancellation, "stale-completion-guard-removed")
	require.NoError(t, err)
	require.True(t, found)
	binding, err := DefaultMutatedBinding()
	require.NoError(t, err)
	require.NoError(t, binding.ValidateAgainst(view))
	result, err := DefaultMutatedResult()
	require.NoError(t, err)
	require.Equal(t, binding.ArtifactDigest, result.BindingArtifactDigest)
	require.Equal(t, protocolcatalog.ResultClassTraceWitness, result.ResultClass)
}

func TestNormalizeConcreteOutputClassifiesSoundInstanceWithoutCompleteness(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-sound.json")

	result, err := NormalizeConcreteOutput(view, binding, concreteOutputReader(t, binding, soundConcreteOutput),
		protocolexperiment.DefaultDecodeLimit, canonicalExecutionLimits(), nil)
	require.NoError(t, err)
	require.Equal(t, protocolcatalog.ResultClassExternalNoCounterexample, result.ResultClass)
	require.Equal(t, protocolcatalog.TrustBadgeTestedInstance, result.TrustBadge)
	require.Equal(t, binding.ArtifactDigest, result.BindingArtifactDigest)
	require.False(t, result.Exact)
	require.Equal(t, []string{protocolchecker.VeilConcreteCollisionOmission}, result.Omissions)
	require.Nil(t, result.Trace)
	require.NoError(t, result.Validate())
}

func TestCheckConcreteRunsSupervisedBackend(t *testing.T) {
	view := testView()
	binding := testBinding(t, view, ReconstructedSMT)
	receipt := wrapConcreteOutput(t, binding.Binding, `{
  "explored_states": 2,
  "result": "no_violation_found",
  "termination_reason": {"kind": "explored_all_reachable_states"}
}`)

	result, err := CheckConcrete(context.Background(),
		explicitTestEnvironment([]string{
			"UMPIRE3_VEIL_CONCRETE_HELPER=1", "UMPIRE3_VEIL_CONCRETE_RECEIPT=" + receipt,
		}, os.Args[0], "-test.run=^TestConcreteCheckerHelper$", "--"), nil, view, binding)
	require.NoError(t, err)
	require.Equal(t, 2, result.ExploredStates)
	require.Equal(t, protocolcatalog.ResultClassExternalNoCounterexample, result.ResultClass)
}

func TestConcreteCheckerHelper(t *testing.T) {
	if os.Getenv("UMPIRE3_VEIL_CONCRETE_HELPER") != "1" {
		return
	}
	separator := slices.Index(os.Args, "--")
	if separator < 0 || !slices.Equal(os.Args[separator+1:], []string{
		"sha256:0000000000000000000000000000000000000000000000000000000000000000",
	}) {
		//nolint:revive // The subprocess helper reports malformed invocation through its exit status.
		os.Exit(3)
	}
	_, err := os.Stdout.Write([]byte(os.Getenv("UMPIRE3_VEIL_CONCRETE_RECEIPT")))
	require.NoError(t, err)
	//nolint:revive // The helper must not append the Go test runner's PASS output to the receipt.
	os.Exit(0)
}

func TestNormalizeConcreteOutputRejectsExplorationCountMismatch(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-sound.json")

	wrongCount := strings.Replace(soundConcreteOutput, `"explored_states": 26`,
		`"explored_states": 25`, 1)
	_, err := NormalizeConcreteOutput(view, binding, concreteOutputReader(t, binding, wrongCount),
		protocolexperiment.DefaultDecodeLimit, canonicalExecutionLimits(), nil)
	require.ErrorContains(t, err, "invalid Veil no-violation result")

	overLimit := strings.Replace(soundConcreteOutput, `"explored_states": 26`,
		`"explored_states": 513`, 1)
	_, err = NormalizeConcreteOutput(view, binding, concreteOutputReader(t, binding, overLimit),
		protocolexperiment.DefaultDecodeLimit, canonicalExecutionLimits(), nil)
	require.ErrorContains(t, err, "beyond the declared limit 512")
}

func TestNormalizeConcreteOutputMapsMutationTraceAndRequiresReplay(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation-mutated.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-mutated.json")

	_, err := NormalizeConcreteOutput(view, binding, concreteOutputReader(t, binding, mutatedConcreteOutput),
		protocolexperiment.DefaultDecodeLimit, canonicalExecutionLimits(), nil)
	require.ErrorContains(t, err, "accepted canonical replay")

	result, err := NormalizeConcreteOutput(view, binding, concreteOutputReader(t, binding, mutatedConcreteOutput),
		protocolexperiment.DefaultDecodeLimit, canonicalExecutionLimits(), acceptedReplayReceipt(t, view))
	require.NoError(t, err)
	require.Equal(t, []protocolchecker.TraceStep{
		{Action: protocolcatalog.ActionKindDispatchTask},
		{Action: protocolcatalog.ActionKindAcquireOwnership},
		{Action: protocolcatalog.ActionKindWorkerReturnsSuccess},
		{Action: protocolcatalog.ActionKindPersistSuccess},
	}, result.Trace.Steps)
	require.Equal(t, protocolcatalog.ResultClassTraceWitness, result.ResultClass)
	require.Equal(t, protocolcatalog.TrustBadgeCheckedCertificate, result.TrustBadge)
	require.Equal(t, binding.ArtifactDigest, result.BindingArtifactDigest)
	require.True(t, result.Exact)
	require.NoError(t, result.Validate())
}

func TestConcreteReplayInputBindsNormalizedMutationTrace(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation-mutated.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-mutated.json")

	input, err := ConcreteReplayInput(view, binding, concreteOutputReader(t, binding, mutatedConcreteOutput),
		protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, &protocolchecker.TraceReplayInput{
		FormatVersion: protocolchecker.TraceReplayInputFormatVersion,
		Target:        protocolcatalog.TargetIDNexusCancellation,
		Property:      protocolcatalog.PropertyIDNexusCancellationWonExcludesSuccess,
		World:         "smoke",
		Variant:       "stale-completion-guard-removed",
		SemanticHash:  view.SemanticHash,
		Actions: []protocolcatalog.ActionKind{
			protocolcatalog.ActionKindDispatchTask,
			protocolcatalog.ActionKindAcquireOwnership,
			protocolcatalog.ActionKindWorkerReturnsSuccess,
			protocolcatalog.ActionKindPersistSuccess,
		},
	}, input)

	soundView := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	soundBinding := readBindingArtifact(t, "nexus-cancellation-sound.json")
	input, err = ConcreteReplayInput(soundView, soundBinding,
		concreteOutputReader(t, soundBinding, soundConcreteOutput),
		protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Nil(t, input)
}

func TestNormalizeConcreteOutputRejectsUnknownTransitionAndViolation(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation-mutated.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-mutated.json")
	replay := acceptedReplayReceipt(t, view)

	unknownTransition := strings.Replace(mutatedConcreteOutput, "PersistSuccess", "UnknownAction", 1)
	_, err := NormalizeConcreteOutput(view, binding, concreteOutputReader(t, binding, unknownTransition),
		protocolexperiment.DefaultDecodeLimit, canonicalExecutionLimits(), replay)
	require.ErrorContains(t, err, `unknown Veil transition "UnknownAction"`)

	wrongViolation := strings.Replace(mutatedConcreteOutput,
		"NexusCancellationWonExcludesSuccess", "DifferentProperty", 1)
	_, err = NormalizeConcreteOutput(view, binding, concreteOutputReader(t, binding, wrongViolation),
		protocolexperiment.DefaultDecodeLimit, canonicalExecutionLimits(), replay)
	require.ErrorContains(t, err, "does not match first-order property")

	wrongDigest := *replay
	wrongDigest.TraceDigest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	_, err = NormalizeConcreteOutput(view, binding, concreteOutputReader(t, binding, mutatedConcreteOutput),
		protocolexperiment.DefaultDecodeLimit, canonicalExecutionLimits(), &wrongDigest)
	require.ErrorContains(t, err, "accepted canonical replay")

	representableState := strings.Replace(mutatedConcreteOutput, "<unrepresentable>", "state", 1)
	_, err = NormalizeConcreteOutput(view, binding, concreteOutputReader(t, binding, representableState),
		protocolexperiment.DefaultDecodeLimit, canonicalExecutionLimits(), replay)
	require.ErrorContains(t, err, "unexpected Veil trace state representation")
}

func acceptedReplayReceipt(t *testing.T, view protocolchecker.FirstOrderView) *protocolchecker.TraceReplayReceipt {
	t.Helper()
	input := protocolchecker.TraceReplayInput{
		FormatVersion: protocolchecker.TraceReplayInputFormatVersion,
		Target:        view.Target, Property: view.Property, World: view.World, Variant: view.Variant,
		SemanticHash: view.SemanticHash,
		Actions: []protocolcatalog.ActionKind{
			protocolcatalog.ActionKindDispatchTask,
			protocolcatalog.ActionKindAcquireOwnership,
			protocolcatalog.ActionKindWorkerReturnsSuccess,
			protocolcatalog.ActionKindPersistSuccess,
		},
	}
	digest, err := input.Digest()
	require.NoError(t, err)
	return &protocolchecker.TraceReplayReceipt{
		FormatVersion: protocolchecker.TraceReplayReceiptFormatVersion,
		TraceDigest:   digest, Target: input.Target, Property: input.Property,
		World: input.World, Variant: input.Variant, SemanticHash: input.SemanticHash,
		Actions: input.Actions, Status: protocolchecker.TraceReplayAccepted,
		TrustBadge: protocolcatalog.TrustBadgeCheckedCertificate, Axioms: []string{},
	}
}

func TestNormalizeConcreteOutputRejectsUnknownAndTrailingJSON(t *testing.T) {
	view := readFirstOrderView(t, "nexus-cancellation.first-order.json")
	binding := readBindingArtifact(t, "nexus-cancellation-sound.json")

	unknown := strings.Replace(soundConcreteOutput, `"result":`, `"unknown": true, "result":`, 1)
	_, err := NormalizeConcreteOutput(view, binding, concreteOutputReader(t, binding, unknown),
		protocolexperiment.DefaultDecodeLimit, canonicalExecutionLimits(), nil)
	require.ErrorContains(t, err, "unknown field")

	_, err = NormalizeConcreteOutput(view, binding,
		strings.NewReader(wrapConcreteOutput(t, binding.Binding, soundConcreteOutput)+` {}`),
		protocolexperiment.DefaultDecodeLimit, canonicalExecutionLimits(), nil)
	require.ErrorContains(t, err, "multiple JSON values")
}

func TestGeneratedBackendResultsRetainTheirTrustClasses(t *testing.T) {
	tests := map[string]struct {
		resultClass protocolcatalog.ResultClass
		trustBadge  protocolcatalog.TrustBadge
	}{
		"nexus-cancellation-sound-concrete.json": {
			resultClass: protocolcatalog.ResultClassExternalNoCounterexample,
			trustBadge:  protocolcatalog.TrustBadgeTestedInstance,
		},
		"nexus-cancellation-mutated-concrete.json": {
			resultClass: protocolcatalog.ResultClassTraceWitness,
			trustBadge:  protocolcatalog.TrustBadgeCheckedCertificate,
		},
		"nexus-cancellation-sound-symbolic.json": {
			resultClass: protocolcatalog.ResultClassBoundedSafe,
			trustBadge:  protocolcatalog.TrustBadgeTrustedSolver,
		},
		"nexus-cancellation-sound-invariant.json": {
			resultClass: protocolcatalog.ResultClassInvariantProved,
			trustBadge:  protocolcatalog.TrustBadgeReconstructedSolverProof,
		},
		"nexus-cancellation-sound-invariant-trusted.json": {
			resultClass: protocolcatalog.ResultClassInvariantProved,
			trustBadge:  protocolcatalog.TrustBadgeTrustedSolver,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			encoded, err := os.ReadFile("testdata/retained/" + name)
			require.NoError(t, err)
			result, err := protocolchecker.DecodeBackendResult(bytes.NewReader(encoded),
				protocolexperiment.DefaultDecodeLimit)
			require.NoError(t, err)
			require.Equal(t, test.resultClass, result.ResultClass)
			require.Equal(t, test.trustBadge, result.TrustBadge)
		})
	}
}

func readFirstOrderView(t *testing.T, name string) protocolchecker.FirstOrderView {
	t.Helper()
	encoded, err := os.ReadFile("../finite/testdata/generated/" + name)
	require.NoError(t, err)
	view, err := protocolchecker.DecodeFirstOrderView(bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	return view
}

func readBindingArtifact(t *testing.T, name string) BindingArtifact {
	t.Helper()
	encoded, err := os.ReadFile("testdata/generated/" + name)
	require.NoError(t, err)
	binding, err := DecodeBindingArtifact(bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	return binding
}

func wrapConcreteOutput(t *testing.T, binding CompiledBinding, result string) string {
	t.Helper()
	encoded, err := json.Marshal(struct {
		Binding CompiledBinding `json:"binding"`
		Result  json.RawMessage `json:"result"`
	}{Binding: binding, Result: json.RawMessage(result)})
	require.NoError(t, err)
	return string(encoded)
}

func concreteOutputReader(t *testing.T, binding BindingArtifact, result string) *strings.Reader {
	t.Helper()
	return strings.NewReader(wrapConcreteOutput(t, binding.Binding, result))
}
