package canonical

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestReplayFiniteAndTemporalRequireDigestBoundLeanReceipts(t *testing.T) {
	finiteView, found, err := protocol.DefaultFirstOrderView(
		protocol.TargetIDNexusCancellation, "stale-completion-guard-removed")
	require.NoError(t, err)
	require.True(t, found)
	finiteInput := protocol.TraceReplayInput{
		FormatVersion: protocol.TraceReplayInputFormatVersion,
		Target:        finiteView.Target, Property: finiteView.Property, World: finiteView.World,
		Variant: finiteView.Variant, SemanticHash: finiteView.SemanticHash,
		Actions: []protocol.ActionKind{
			protocol.ActionKindDispatchTask,
			protocol.ActionKindAcquireOwnership,
			protocol.ActionKindWorkerReturnsSuccess,
			protocol.ActionKindPersistSuccess,
		},
	}
	finiteDigest, err := finiteInput.Digest()
	require.NoError(t, err)
	finiteReceipt := protocol.TraceReplayReceipt{
		FormatVersion: protocol.TraceReplayReceiptFormatVersion,
		TraceDigest:   finiteDigest, Target: finiteInput.Target, Property: finiteInput.Property,
		World: finiteInput.World, Variant: finiteInput.Variant, SemanticHash: finiteInput.SemanticHash,
		Actions: finiteInput.Actions, Status: protocol.TraceReplayAccepted,
		TrustBadge: protocol.TrustBadgeCheckedCertificate, Axioms: []string{},
	}
	finiteJSON, err := json.Marshal(finiteReceipt)
	require.NoError(t, err)
	finite, err := ReplayFinite(context.Background(), helperCommand("finite", finiteJSON), finiteInput)
	require.NoError(t, err)
	require.Equal(t, finiteReceipt, finite)

	temporalView, found, err := protocol.DefaultTemporalView("delivery-fairness-removed")
	require.NoError(t, err)
	require.True(t, found)
	temporalInput := protocol.TemporalLassoReplayInput{
		FormatVersion: protocol.TemporalLassoReplayInputFormatVersion,
		Target:        temporalView.Target, Property: temporalView.Property, World: temporalView.World,
		Variant: temporalView.Variant, SemanticHash: temporalView.SemanticHash,
		Lasso: protocol.TemporalLasso{
			States:  []string{"unavailable", "ready"},
			Actions: []protocol.ActionKind{protocol.ActionKindRecoverOwner, ""}, LoopStart: 1,
		},
	}
	temporalDigest, err := temporalInput.Digest()
	require.NoError(t, err)
	temporalReceipt := protocol.TemporalLassoReplayReceipt{
		FormatVersion: protocol.TemporalLassoReplayReceiptFormatVersion,
		LassoDigest:   temporalDigest, Target: temporalInput.Target, Property: temporalInput.Property,
		World: temporalInput.World, Variant: temporalInput.Variant,
		SemanticHash: temporalInput.SemanticHash, Lasso: temporalInput.Lasso,
		Status: protocol.TraceReplayAccepted, TrustBadge: protocol.TrustBadgeCheckedCertificate,
		Axioms: []string{},
	}
	temporalJSON, err := json.Marshal(temporalReceipt)
	require.NoError(t, err)
	temporal, err := ReplayTemporal(context.Background(), helperCommand("temporal", temporalJSON), temporalInput)
	require.NoError(t, err)
	require.Equal(t, temporalReceipt, temporal)
}

func helperCommand(mode string, receipt []byte) []string {
	return []string{"/usr/bin/env", "UMPIRE3_CANONICAL_HELPER=" + mode,
		"UMPIRE3_CANONICAL_RECEIPT=" + string(receipt), os.Args[0],
		"-test.run=^TestCanonicalReplayHelper$", "--"}
}

func TestCanonicalReplayHelper(t *testing.T) {
	mode := os.Getenv("UMPIRE3_CANONICAL_HELPER")
	if mode == "" {
		return
	}
	separator := slices.Index(os.Args, "--")
	if separator < 0 {
		os.Exit(3)
	}
	arguments := os.Args[separator+1:]
	switch mode {
	case "finite":
		if len(arguments) != 10 || arguments[1] != string(protocol.TargetIDNexusCancellation) ||
			arguments[6] != string(protocol.ActionKindDispatchTask) {
			os.Exit(4)
		}
	case "temporal":
		if len(arguments) != 12 || arguments[1] != string(protocol.TargetIDFoundationDeliverySafety) ||
			arguments[8] != "unavailable" || arguments[10] != string(protocol.ActionKindRecoverOwner) {
			os.Exit(5)
		}
	default:
		os.Exit(6)
	}
	fmt.Print(os.Getenv("UMPIRE3_CANONICAL_RECEIPT"))
	os.Exit(0)
}
