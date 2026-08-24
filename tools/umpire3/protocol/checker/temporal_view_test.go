package checker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTemporalViewsBindProgressProofAndMutation(t *testing.T) {
	t.Parallel()

	sound, found, err := DefaultTemporalView("sound")
	require.NoError(t, err)
	require.True(t, found)
	require.NotNil(t, sound.Proof)
	require.Equal(t, "canonical-model-only", sound.ClaimScope)
	require.Equal(t, ResultClassTemporalProved, sound.Proof.ResultClass)
	require.Equal(t, []string{
		"task-delivery.recovery-responsive",
		"task-delivery.delivery-responsive",
	}, sound.Proof.FairnessAssumptions)

	mutated, found, err := DefaultTemporalView("delivery-fairness-removed")
	require.NoError(t, err)
	require.True(t, found)
	require.Nil(t, mutated.Proof)
	require.Len(t, mutated.Fairness, 1)
}

func TestTemporalViewRejectsDuplicateLiveOnlyAction(t *testing.T) {
	t.Parallel()

	view, found, err := DefaultTemporalView("sound")
	require.NoError(t, err)
	require.True(t, found)
	view.LiveOnlyActions = append(view.LiveOnlyActions, view.LiveOnlyActions[0])

	err = view.Validate()
	require.ErrorContains(t, err, "duplicate temporal live-only action")
}

func TestTemporalLassoChecksCanonicalStepsFairnessAndProgress(t *testing.T) {
	t.Parallel()

	mutated, found, err := DefaultTemporalView("delivery-fairness-removed")
	require.NoError(t, err)
	require.True(t, found)
	lasso := TemporalLasso{
		States:    []string{"unavailable", "ready"},
		Actions:   []ActionKind{ActionKindRecoverOwner, ""},
		LoopStart: 1,
	}
	require.NoError(t, lasso.Validate(mutated))

	sound, found, err := DefaultTemporalView("sound")
	require.NoError(t, err)
	require.True(t, found)
	require.ErrorContains(t, lasso.Validate(sound), "delivery-responsive")

	lasso.Actions[0] = ActionKindProgressEntity
	require.ErrorContains(t, lasso.Validate(mutated), "canonical temporal transition")
}

func TestTemporalLassoReplayReceiptIsDigestBound(t *testing.T) {
	t.Parallel()

	view, found, err := DefaultTemporalView("delivery-fairness-removed")
	require.NoError(t, err)
	require.True(t, found)
	input := TemporalLassoReplayInput{
		FormatVersion: TemporalLassoReplayInputFormatVersion,
		Target:        view.Target,
		Property:      view.Property,
		World:         view.World,
		Variant:       view.Variant,
		SemanticHash:  view.SemanticHash,
		Lasso: TemporalLasso{
			States:    []string{"unavailable", "ready"},
			Actions:   []ActionKind{ActionKindRecoverOwner, ""},
			LoopStart: 1,
		},
	}
	digest, err := input.Digest()
	require.NoError(t, err)
	receipt := TemporalLassoReplayReceipt{
		FormatVersion: TemporalLassoReplayReceiptFormatVersion,
		LassoDigest:   digest,
		Target:        input.Target,
		Property:      input.Property,
		World:         input.World,
		Variant:       input.Variant,
		SemanticHash:  input.SemanticHash,
		Lasso:         input.Lasso,
		Status:        TraceReplayAccepted,
		TrustBadge:    TrustBadgeCheckedCertificate,
		Axioms:        []string{},
	}
	require.NoError(t, receipt.Validate())
	receipt.Lasso.Actions[0] = ActionKindProgressEntity
	require.Error(t, receipt.Validate())
}
