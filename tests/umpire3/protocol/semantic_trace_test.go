package protocol

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSemanticTraceNormalizesFiniteAndTemporalReceipts(t *testing.T) {
	finiteView, found, err := DefaultFirstOrderView(
		TargetIDNexusCancellation, "stale-completion-guard-removed")
	require.NoError(t, err)
	require.True(t, found)
	finiteInput := TraceReplayInput{
		FormatVersion: TraceReplayInputFormatVersion,
		Target:        finiteView.Target, Property: finiteView.Property,
		World: finiteView.World, Variant: finiteView.Variant,
		SemanticHash: finiteView.SemanticHash,
		Actions: []ActionKind{
			ActionKindDispatchTask,
			ActionKindAcquireOwnership,
			ActionKindWorkerReturnsSuccess,
			ActionKindPersistSuccess,
		},
	}
	finiteDigest, err := finiteInput.Digest()
	require.NoError(t, err)
	finite, err := SemanticTraceFromTraceReplayReceipt(SemanticTraceProducerVeil, TraceReplayReceipt{
		FormatVersion: TraceReplayReceiptFormatVersion,
		TraceDigest:   finiteDigest, Target: finiteInput.Target, Property: finiteInput.Property,
		World: finiteInput.World, Variant: finiteInput.Variant,
		SemanticHash: finiteInput.SemanticHash, Actions: finiteInput.Actions,
		Status: TraceReplayAccepted, TrustBadge: TrustBadgeCheckedCertificate, Axioms: []string{},
	})
	require.NoError(t, err)
	require.Equal(t, SemanticTraceFinite, finite.Kind)
	require.Equal(t, finiteDigest, finite.Replay.Digest)
	require.Len(t, finite.Steps, len(finiteInput.Actions))

	temporalView, found, err := DefaultTemporalView("delivery-fairness-removed")
	require.NoError(t, err)
	require.True(t, found)
	temporalInput := TemporalLassoReplayInput{
		FormatVersion: TemporalLassoReplayInputFormatVersion,
		Target:        temporalView.Target, Property: temporalView.Property,
		World: temporalView.World, Variant: temporalView.Variant,
		SemanticHash: temporalView.SemanticHash,
		Lasso: TemporalLasso{
			States:    []string{"unavailable", "ready"},
			Actions:   []ActionKind{ActionKindRecoverOwner, ""},
			LoopStart: 1,
		},
	}
	temporalDigest, err := temporalInput.Digest()
	require.NoError(t, err)
	temporal, err := SemanticTraceFromTemporalLassoReplayReceipt(
		SemanticTraceProducerLeanTemporal, TemporalLassoReplayReceipt{
			FormatVersion: TemporalLassoReplayReceiptFormatVersion,
			LassoDigest:   temporalDigest, Target: temporalInput.Target, Property: temporalInput.Property,
			World: temporalInput.World, Variant: temporalInput.Variant,
			SemanticHash: temporalInput.SemanticHash, Lasso: temporalInput.Lasso,
			Status: TraceReplayAccepted, TrustBadge: TrustBadgeCheckedCertificate, Axioms: []string{},
		})
	require.NoError(t, err)
	require.Equal(t, SemanticTraceTemporal, temporal.Kind)
	require.Equal(t, temporalInput.Lasso.States, temporal.States)
	require.Equal(t, temporalInput.Lasso.LoopStart, temporal.LoopStart)
}

func TestLiveSemanticTraceRetainsObservedAttemptOutcomes(t *testing.T) {
	encoded, err := os.ReadFile("../testdata/update-lifecycle.json")
	require.NoError(t, err)
	experiment, err := DecodeExperiment(bytes.NewReader(encoded), DefaultDecodeLimit)
	require.NoError(t, err)
	view, found, err := DefaultAttemptExecutionView(experiment)
	require.NoError(t, err)
	require.True(t, found)
	attempts := make([]ObservedAttempt, len(experiment.Actions))
	for index, action := range experiment.Actions {
		attempts[index] = ObservedAttempt{
			Action: ActionKind(action.Kind), Outcome: ActionOutcomeApplied,
		}
	}

	trace, err := NewLiveSemanticTrace(experiment, view, attempts)
	require.NoError(t, err)
	require.Equal(t, SemanticTraceLive, trace.Kind)
	require.Equal(t, SemanticTraceProducerLive, trace.Producer)
	require.Len(t, trace.Steps, len(attempts))
	require.Equal(t, ActionOutcomeApplied, trace.Steps[0].Outcome)
	require.NotNil(t, trace.Experiment)
	require.NoError(t, trace.Validate())

	experiment.Actions[0].Kind = string(ActionKindAcquireOwnership)
	require.NoError(t, trace.Validate())
	trace.Experiment.Actions[0].Kind = string(ActionKindAcquireOwnership)
	require.Error(t, trace.Validate())
	trace.Experiment.Actions[0].Kind = string(attempts[0].Action)
	trace.Steps[0].Outcome = ActionOutcomeRejected
	require.Error(t, trace.Validate())
}

func TestSemanticTraceRejectsProducerAndReplayMismatch(t *testing.T) {
	view, found, err := DefaultFirstOrderView(
		TargetIDNexusCancellation, "stale-completion-guard-removed")
	require.NoError(t, err)
	require.True(t, found)
	input := TraceReplayInput{
		FormatVersion: TraceReplayInputFormatVersion,
		Target:        view.Target, Property: view.Property, World: view.World, Variant: view.Variant,
		SemanticHash: view.SemanticHash, Actions: []ActionKind{ActionKindDispatchTask},
	}
	digest, err := input.Digest()
	require.NoError(t, err)
	receipt := TraceReplayReceipt{
		FormatVersion: TraceReplayReceiptFormatVersion,
		TraceDigest:   digest, Target: input.Target, Property: input.Property,
		World: input.World, Variant: input.Variant, SemanticHash: input.SemanticHash,
		Actions: input.Actions, Status: TraceReplayAccepted,
		TrustBadge: TrustBadgeCheckedCertificate, Axioms: []string{},
	}
	_, err = SemanticTraceFromTraceReplayReceipt(SemanticTraceProducerLeanTemporal, receipt)
	require.ErrorContains(t, err, "producer")

	trace, err := SemanticTraceFromTraceReplayReceipt(SemanticTraceProducerExact, receipt)
	require.NoError(t, err)
	trace.Replay.Digest =
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	require.ErrorContains(t, trace.Validate(), "digest")
}

func TestSemanticTraceStrictDecodeRoundTrip(t *testing.T) {
	encodedResult, err := os.ReadFile("../model-checkers/veil/results/nexus-cancellation-mutated-concrete.json")
	require.NoError(t, err)
	result, err := DecodeBackendResult(bytes.NewReader(encodedResult), DefaultDecodeLimit)
	require.NoError(t, err)
	trace, err := SemanticTraceFromBackendResult(result)
	require.NoError(t, err)
	encoded, err := trace.CanonicalJSON()
	require.NoError(t, err)

	decoded, err := DecodeSemanticTrace(bytes.NewReader(encoded), DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, trace, decoded)

	unknown := append(encoded[:len(encoded)-1], []byte(`,"unknown":true}`)...)
	_, err = DecodeSemanticTrace(bytes.NewReader(unknown), DefaultDecodeLimit)
	require.Error(t, err)
	_, err = DecodeSemanticTrace(bytes.NewReader(encoded), int64(len(encoded)-1))
	require.ErrorContains(t, err, "exceeds")
}
