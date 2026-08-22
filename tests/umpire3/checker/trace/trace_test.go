package trace

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/checker/finite"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

func TestSemanticTraceNormalizesFiniteAndTemporalReceipts(t *testing.T) {
	finiteView, found, err := finite.DefaultFirstOrderView(
		protocolcatalog.TargetIDNexusCancellation, "stale-completion-guard-removed")
	require.NoError(t, err)
	require.True(t, found)
	finiteInput := protocolchecker.TraceReplayInput{
		FormatVersion: protocolchecker.TraceReplayInputFormatVersion,
		Target:        finiteView.Target, Property: finiteView.Property,
		World: finiteView.World, Variant: finiteView.Variant,
		SemanticHash: finiteView.SemanticHash,
		Actions: []protocolcatalog.ActionKind{
			protocolcatalog.ActionKindDispatchTask,
			protocolcatalog.ActionKindAcquireOwnership,
			protocolcatalog.ActionKindWorkerReturnsSuccess,
			protocolcatalog.ActionKindPersistSuccess,
		},
	}
	finiteDigest, err := finiteInput.Digest()
	require.NoError(t, err)
	finiteTrace, err := FromTraceReplayReceipt(protocolchecker.SemanticTraceProducerVeil, protocolchecker.TraceReplayReceipt{
		FormatVersion: protocolchecker.TraceReplayReceiptFormatVersion,
		TraceDigest:   finiteDigest, Target: finiteInput.Target, Property: finiteInput.Property,
		World: finiteInput.World, Variant: finiteInput.Variant,
		SemanticHash: finiteInput.SemanticHash, Actions: finiteInput.Actions,
		Status: protocolchecker.TraceReplayAccepted, TrustBadge: protocolcatalog.TrustBadgeCheckedCertificate, Axioms: []string{},
	})
	require.NoError(t, err)
	require.Equal(t, protocolchecker.SemanticTraceFinite, finiteTrace.Kind)
	require.Equal(t, finiteDigest, finiteTrace.Replay.Digest)
	require.Len(t, finiteTrace.Steps, len(finiteInput.Actions))

	temporalView, found, err := protocolchecker.DefaultTemporalView("delivery-fairness-removed")
	require.NoError(t, err)
	require.True(t, found)
	temporalInput := protocolchecker.TemporalLassoReplayInput{
		FormatVersion: protocolchecker.TemporalLassoReplayInputFormatVersion,
		Target:        temporalView.Target, Property: temporalView.Property,
		World: temporalView.World, Variant: temporalView.Variant,
		SemanticHash: temporalView.SemanticHash,
		Lasso: protocolchecker.TemporalLasso{
			States:    []string{"unavailable", "ready"},
			Actions:   []protocolcatalog.ActionKind{protocolcatalog.ActionKindRecoverOwner, ""},
			LoopStart: 1,
		},
	}
	temporalDigest, err := temporalInput.Digest()
	require.NoError(t, err)
	temporal, err := FromTemporalLassoReplayReceipt(
		protocolchecker.SemanticTraceProducerLeanTemporal, protocolchecker.TemporalLassoReplayReceipt{
			FormatVersion: protocolchecker.TemporalLassoReplayReceiptFormatVersion,
			LassoDigest:   temporalDigest, Target: temporalInput.Target, Property: temporalInput.Property,
			World: temporalInput.World, Variant: temporalInput.Variant,
			SemanticHash: temporalInput.SemanticHash, Lasso: temporalInput.Lasso,
			Status: protocolchecker.TraceReplayAccepted, TrustBadge: protocolcatalog.TrustBadgeCheckedCertificate, Axioms: []string{},
		})
	require.NoError(t, err)
	require.Equal(t, protocolchecker.SemanticTraceTemporal, temporal.Kind)
	require.Equal(t, temporalInput.Lasso.States, temporal.States)
	require.Equal(t, temporalInput.Lasso.LoopStart, temporal.LoopStart)
}

func TestLiveSemanticTraceRetainsObservedAttemptOutcomes(t *testing.T) {
	encoded, err := os.ReadFile("../../testdata/generated/update-lifecycle.json")
	require.NoError(t, err)
	experiment, err := protocolexperiment.DecodeExperiment(bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	view, found, err := finite.DefaultAttemptExecutionView(experiment)
	require.NoError(t, err)
	require.True(t, found)
	attempts := make([]finite.ObservedAttempt, len(experiment.Actions))
	for index, action := range experiment.Actions {
		attempts[index] = finite.ObservedAttempt{
			Action: protocolcatalog.ActionKind(action.Kind), Outcome: protocolexperiment.ActionOutcomeApplied,
		}
	}

	trace, err := NewLive(experiment, view, attempts)
	require.NoError(t, err)
	require.Equal(t, protocolchecker.SemanticTraceLive, trace.Kind)
	require.Equal(t, protocolchecker.SemanticTraceProducerLive, trace.Producer)
	require.Len(t, trace.Steps, len(attempts))
	require.Equal(t, protocolexperiment.ActionOutcomeApplied, trace.Steps[0].Outcome)
	require.NotNil(t, trace.Experiment)
	require.NoError(t, Validate(trace))

	experiment.Actions[0].Kind = string(protocolcatalog.ActionKindAcquireOwnership)
	require.NoError(t, Validate(trace))
	trace.Experiment.Actions[0].Kind = string(protocolcatalog.ActionKindAcquireOwnership)
	require.Error(t, Validate(trace))
	trace.Experiment.Actions[0].Kind = string(attempts[0].Action)
	trace.Steps[0].Outcome = protocolexperiment.ActionOutcomeRejected
	require.Error(t, Validate(trace))
}

func TestSemanticTraceRejectsProducerAndReplayMismatch(t *testing.T) {
	view, found, err := finite.DefaultFirstOrderView(
		protocolcatalog.TargetIDNexusCancellation, "stale-completion-guard-removed")
	require.NoError(t, err)
	require.True(t, found)
	input := protocolchecker.TraceReplayInput{
		FormatVersion: protocolchecker.TraceReplayInputFormatVersion,
		Target:        view.Target, Property: view.Property, World: view.World, Variant: view.Variant,
		SemanticHash: view.SemanticHash, Actions: []protocolcatalog.ActionKind{protocolcatalog.ActionKindDispatchTask},
	}
	digest, err := input.Digest()
	require.NoError(t, err)
	receipt := protocolchecker.TraceReplayReceipt{
		FormatVersion: protocolchecker.TraceReplayReceiptFormatVersion,
		TraceDigest:   digest, Target: input.Target, Property: input.Property,
		World: input.World, Variant: input.Variant, SemanticHash: input.SemanticHash,
		Actions: input.Actions, Status: protocolchecker.TraceReplayAccepted,
		TrustBadge: protocolcatalog.TrustBadgeCheckedCertificate, Axioms: []string{},
	}
	_, err = FromTraceReplayReceipt(protocolchecker.SemanticTraceProducerLeanTemporal, receipt)
	require.ErrorContains(t, err, "producer")

	trace, err := FromTraceReplayReceipt(protocolchecker.SemanticTraceProducerExact, receipt)
	require.NoError(t, err)
	trace.Replay.Digest =
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	require.ErrorContains(t, Validate(trace), "digest")
}

func TestSemanticTraceStrictDecodeRoundTrip(t *testing.T) {
	encodedResult, err := os.ReadFile("../veil/testdata/retained/nexus-cancellation-mutated-concrete.json")
	require.NoError(t, err)
	result, err := protocolchecker.DecodeBackendResult(bytes.NewReader(encodedResult), protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	trace, err := FromBackendResult(result)
	require.NoError(t, err)
	encoded, err := trace.CanonicalJSON()
	require.NoError(t, err)

	decoded, err := protocolchecker.DecodeSemanticTrace(bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, trace, decoded)

	unknown := append(encoded[:len(encoded)-1], []byte(`,"unknown":true}`)...)
	_, err = protocolchecker.DecodeSemanticTrace(bytes.NewReader(unknown), protocolexperiment.DefaultDecodeLimit)
	require.Error(t, err)
	_, err = protocolchecker.DecodeSemanticTrace(bytes.NewReader(encoded), int64(len(encoded)-1))
	require.ErrorContains(t, err, "exceeds")
}
