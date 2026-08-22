package mutation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

func TestRunRetainsEverySemanticMutationStage(t *testing.T) {
	experimentBytes, err := os.ReadFile("../../../testdata/generated/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := protocolexperiment.DecodeExperiment(bytes.NewReader(experimentBytes), protocolexperiment.DefaultDecodeLimit)
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
	require.Equal(t, protocolchecker.SemanticTraceProducerNative, report.NativeTrace.Producer)
	require.Equal(t, protocolchecker.SemanticTraceProducerVeil, report.VeilTrace.Producer)
	require.Equal(t, protocolchecker.SemanticTraceProducerLeanTemporal, report.TemporalTrace.Producer)
	require.Equal(t, report.NativeTrace.Steps, report.VeilTrace.Steps)

	encoded, err := report.CanonicalJSON()
	require.NoError(t, err)
	decoded, err := DecodeReport(encoded, protocolexperiment.DefaultDecodeLimit)
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
		//nolint:revive // The helper process reports malformed protocol input through its exit status.
		os.Exit(3)
	}
	arguments := os.Args[separator+1:]
	var receipt any
	switch mode {
	case "finite":
		if len(arguments) < 7 {
			//nolint:revive // The helper process reports malformed protocol input through its exit status.
			os.Exit(4)
		}
		actions := make([]protocolcatalog.ActionKind, len(arguments)-6)
		for index, action := range arguments[6:] {
			actions[index] = protocolcatalog.ActionKind(action)
		}
		receipt = protocolchecker.TraceReplayReceipt{
			FormatVersion: protocolchecker.TraceReplayReceiptFormatVersion,
			TraceDigest:   arguments[0], Target: protocolcatalog.TargetID(arguments[1]),
			Property: protocolcatalog.PropertyID(arguments[2]), World: arguments[3], Variant: arguments[4],
			SemanticHash: arguments[5], Actions: actions, Status: protocolchecker.TraceReplayAccepted,
			TrustBadge: protocolcatalog.TrustBadgeCheckedCertificate, Axioms: []string{},
		}
	case "temporal":
		if len(arguments) != 12 {
			//nolint:revive // The helper process reports malformed protocol input through its exit status.
			os.Exit(5)
		}
		receipt = protocolchecker.TemporalLassoReplayReceipt{
			FormatVersion: protocolchecker.TemporalLassoReplayReceiptFormatVersion,
			LassoDigest:   arguments[0], Target: protocolcatalog.TargetID(arguments[1]),
			Property: protocolcatalog.PropertyID(arguments[2]), World: arguments[3], Variant: arguments[4],
			SemanticHash: arguments[5],
			Lasso: protocolchecker.TemporalLasso{
				States: []string{arguments[8], arguments[9]},
				Actions: []protocolcatalog.ActionKind{
					protocolcatalog.ActionKind(arguments[10]), protocolcatalog.ActionKind(arguments[11]),
				}, LoopStart: 1,
			},
			Status:     protocolchecker.TraceReplayAccepted,
			TrustBadge: protocolcatalog.TrustBadgeCheckedCertificate, Axioms: []string{},
		}
	default:
		//nolint:revive // The helper process reports an unsupported protocol mode through its exit status.
		os.Exit(6)
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		//nolint:revive // The helper process reports response encoding failure through its exit status.
		os.Exit(7)
	}
	fmt.Print(string(encoded))
	//nolint:revive // The helper process must not append the Go test runner's PASS output to its response.
	os.Exit(0)
}
