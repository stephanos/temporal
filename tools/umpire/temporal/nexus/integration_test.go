package nexus

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/runner"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

func TestLiveCallerClosureReturnsAndPublishesOneExactOperationalSet(t *testing.T) {
	input := admitCallerClosureSet(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	output, err := runner.Run(
		ctx,
		input,
		runner.InputBinding{
			ArtifactSetIdentity:                     "umpire.artifact-set.ed3605976ba999ec8e166d4309247e2b711fee18f4a421cfb8c6dc037344f1a2",
			ArtifactSetChecksum:                     "sha256:074356889cda0296b13152f87e57d7b980d76125329a9014ceb5321c3f5bda7b",
			ManifestSHA256:                          "sha256:f381da231395b8fec738837535a8bb8da0dd227a08e3d60bf9c2bda620c46b14",
			ExperimentArtifactChecksum:              "sha256:dde2fb35891dcc0020dbedf301805feda1b5136ec8622dd67fdc47a3d00fb1a8",
			ExperimentBehaviorFingerprint:           "sha256:d393ae60847c8524f3a57de6769478f95fd4a6a90a0fefcad6af118206d458af",
			RuntimeConfigurationArtifactChecksum:    "sha256:21b4f7d0db2f68f939df901c2c5d146b1be3e45e55ad6cc171445fda5f29c1d5",
			RuntimeConfigurationBehaviorFingerprint: "sha256:7c4c35a8031d07ff55ef5e83b90c64e63cbc6b196642c379ed75b5fc461f3a67",
		},
		"umpire.local.caller-closure.integration-1",
		Binding{},
	)
	require.NoError(t, err)
	run := output.ExperimentRun()
	rawEvidence := output.RawEvidence()
	require.Equal(t, "succeeded", run.OperationalStatus)
	require.Equal(t, "closed", rawEvidence.CaptureStatus)
	require.Equal(t, []string{
		umpireruntime.EvidenceSourceCleanup,
		umpireruntime.EvidenceSourceControlReceipt,
		umpireruntime.EvidenceSourceHistory,
		umpireruntime.EvidenceSourceParticipantOutput,
	}, sourceDefinitionIDs(rawEvidence.Sources))
	for _, source := range rawEvidence.Sources {
		require.Equal(t, "closed", source.Status)
	}
	require.Len(t, run.ControlAttempts, 1)
	require.Equal(t, "accepted", run.ControlAttempts[0].Status)
	require.EqualValues(t, "0", run.Cleanup.OpenHandleCount)
	require.Empty(t, run.KnownGaps)
	require.Empty(t, rawEvidence.KnownGaps)

	runBytes, err := artifact.EncodeExperimentRunV2(run)
	require.NoError(t, err)
	rawEvidenceBytes, err := artifact.EncodeRawEvidenceV2(rawEvidence)
	require.NoError(t, err)
	for _, encoded := range [][]byte{runBytes, rawEvidenceBytes} {
		require.True(t, bytes.HasSuffix(encoded, []byte("\n")))
		require.False(t, bytes.HasSuffix(encoded, []byte("\n\n")))
		require.Contains(t, string(encoded), "\n  \"")
	}

	executable, ok := input.Executable()
	require.True(t, ok)
	expected, err := executable.AdmitExecution(run, rawEvidence)
	require.NoError(t, err)
	require.Equal(t, expected.Identity(), output.AdmittedSet().Identity())
	require.Equal(t, expected.ManifestBytes(), output.AdmittedSet().ManifestBytes())

	destination, err := artifact.PublishSet(t.TempDir(), output.AdmittedSet())
	require.NoError(t, err)
	reopened, err := artifact.LoadSet(destination)
	require.NoError(t, err)
	require.Equal(t, output.AdmittedSet().Identity(), reopened.Identity())
	require.Equal(t, output.AdmittedSet().Checksum(), reopened.Checksum())
	require.Equal(t, output.AdmittedSet().ManifestSHA256(), reopened.ManifestSHA256())
	require.Equal(t, output.AdmittedSet().ManifestBytes(), reopened.ManifestBytes())
}

func sourceDefinitionIDs(sources []artifactv2.RawEvidenceSource) []string {
	result := make([]string, len(sources))
	for index, source := range sources {
		result[index] = source.SourceDefinitionID
	}
	return result
}
