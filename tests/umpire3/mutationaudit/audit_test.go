package mutationaudit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestRunRetainsEverySemanticMutationStage(t *testing.T) {
	experimentBytes, err := os.ReadFile("../testdata/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(experimentBytes), protocol.DefaultDecodeLimit)
	require.NoError(t, err)

	report, err := Run(context.Background(), Request{
		Experiment:            experiment,
		FiniteReplayCommand:   mutationAuditHelperCommand("finite"),
		TemporalReplayCommand: mutationAuditHelperCommand("temporal"),
	})
	require.NoError(t, err)
	require.Equal(t, []Stage{
		StageExactExploration,
		StageLeanRefinement,
		StageLeanTemporal,
		StageLiveEvidence,
		StageMinimization,
		StageNativeSearch,
		StagePromotion,
		StageReplay,
		StageVeil,
	}, collectStages(report.Evidence))
	require.Equal(t, protocol.SemanticTraceProducerNative, report.NativeTrace.Producer)
	require.Equal(t, protocol.SemanticTraceProducerVeil, report.VeilTrace.Producer)
	require.Equal(t, protocol.SemanticTraceProducerLeanTemporal, report.TemporalTrace.Producer)
	require.Equal(t, report.NativeTrace.Steps, report.VeilTrace.Steps)

	encoded, err := report.CanonicalJSON()
	require.NoError(t, err)
	decoded, err := DecodeReport(encoded, protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	require.Equal(t, report, decoded)
	retained, err := Default()
	require.NoError(t, err)
	require.Equal(t, collectStages(report.Evidence), collectStages(retained.Evidence))

	decoded.Evidence[0].Digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	decoded.ArtifactDigest = decoded.computedDigest()
	require.Error(t, decoded.Validate())
}

func collectStages(evidence []Evidence) []Stage {
	stages := make([]Stage, len(evidence))
	for index, item := range evidence {
		stages[index] = item.Stage
	}
	return stages
}

func mutationAuditHelperCommand(mode string) []string {
	return []string{"/usr/bin/env", "UMPIRE3_MUTATION_AUDIT_HELPER=" + mode, os.Args[0],
		"-test.run=^TestMutationAuditReplayHelper$", "--"}
}

func TestMutationAuditReplayHelper(t *testing.T) {
	mode := os.Getenv("UMPIRE3_MUTATION_AUDIT_HELPER")
	if mode == "" {
		return
	}
	separator := slices.Index(os.Args, "--")
	if separator < 0 {
		os.Exit(3)
	}
	arguments := os.Args[separator+1:]
	var receipt any
	switch mode {
	case "finite":
		if len(arguments) < 7 {
			os.Exit(4)
		}
		actions := make([]protocol.ActionKind, len(arguments)-6)
		for index, action := range arguments[6:] {
			actions[index] = protocol.ActionKind(action)
		}
		receipt = protocol.TraceReplayReceipt{
			FormatVersion: protocol.TraceReplayReceiptFormatVersion,
			TraceDigest:   arguments[0], Target: protocol.TargetID(arguments[1]),
			Property: protocol.PropertyID(arguments[2]), World: arguments[3], Variant: arguments[4],
			SemanticHash: arguments[5], Actions: actions, Status: protocol.TraceReplayAccepted,
			TrustBadge: protocol.TrustBadgeCheckedCertificate, Axioms: []string{},
		}
	case "temporal":
		if len(arguments) != 12 {
			os.Exit(5)
		}
		receipt = protocol.TemporalLassoReplayReceipt{
			FormatVersion: protocol.TemporalLassoReplayReceiptFormatVersion,
			LassoDigest:   arguments[0], Target: protocol.TargetID(arguments[1]),
			Property: protocol.PropertyID(arguments[2]), World: arguments[3], Variant: arguments[4],
			SemanticHash: arguments[5],
			Lasso: protocol.TemporalLasso{
				States: []string{arguments[8], arguments[9]},
				Actions: []protocol.ActionKind{
					protocol.ActionKind(arguments[10]), protocol.ActionKind(arguments[11]),
				}, LoopStart: 1,
			},
			Status:     protocol.TraceReplayAccepted,
			TrustBadge: protocol.TrustBadgeCheckedCertificate, Axioms: []string{},
		}
	default:
		os.Exit(6)
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		os.Exit(7)
	}
	fmt.Print(string(encoded))
	os.Exit(0)
}
