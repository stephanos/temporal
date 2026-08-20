package command

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/evidence"
	"go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestQualificationProducesDigestBoundReceipt(t *testing.T) {
	t.Parallel()

	configuration := qualificationFixture(t, true)
	var output bytes.Buffer
	require.NoError(t, qualifyCompatibility(configuration, &output, defaultBackend{}))
	var result receipt
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	require.Equal(t, "umpire3/1.2", result.Release)
	require.Equal(t, "remote-deployment", result.Profile)
	require.NotEmpty(t, result.ReleaseDigest)
	require.NotEmpty(t, result.EvidenceDigest)
}

func TestQualificationRejectsConformanceWithoutEvidence(t *testing.T) {
	t.Parallel()

	configuration := qualificationFixture(t, false)
	require.ErrorContains(t, qualifyCompatibility(configuration, &bytes.Buffer{}, defaultBackend{}), "no supporting evidence")
}

func TestQualificationRejectsIncompleteCanaryEnvelope(t *testing.T) {
	t.Parallel()

	_, err := decodeResult([]byte(`{"formatVersion":"umpire3/canary/v1","runtime":{},"complete":false}`))
	require.ErrorContains(t, err, "canary result is incomplete")
}

func TestQualificationRejectsMissingFaultRealizationEvidence(t *testing.T) {
	t.Parallel()

	configuration := qualificationFixture(t, true)
	resultBytes, err := os.ReadFile(configuration.resultPath)
	require.NoError(t, err)
	var result execution.Result
	require.NoError(t, json.Unmarshal(resultBytes, &result))
	result.Faults = nil
	resultBytes, err = json.Marshal(result)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configuration.resultPath, resultBytes, 0o600))
	require.ErrorContains(t, qualifyCompatibility(configuration, &bytes.Buffer{}, defaultBackend{}), "every declared fault")
}

func qualificationFixture(t *testing.T, withEvidence bool) qualificationCompatibilityOptions {
	t.Helper()
	experimentBytes, err := os.ReadFile("../../testdata/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(experimentBytes), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	graph := evidence.Graph{FormatVersion: evidence.FormatVersion}
	if withEvidence {
		graph.Facts = []evidence.Fact{{
			Identifier: "qualified", Kind: experiment.Checkpoints[0].Observation, Value: true,
			SourceIdentity: "remote-history", ClockDomain: "remote-history-sequence", SourceSequence: 1,
			ObservedAtUnixNano: time.Now().UnixNano(), Reference: "history/1",
			EntityIdentity: "entity", Lineage: []string{"namespace", "entity"},
		}}
		graph.Claims = []evidence.Claim{{Property: experiment.Property.Identifier, Verdict: "conforming"}}
	}
	result := execution.Result{
		FormatVersion: execution.ResultFormatVersion, ExperimentDigest: digest,
		ResultClass: protocol.ResultClassImplementationConforming,
		TrustBadge:  protocol.TrustBadgeTestedInstance,
		Environment: execution.EnvironmentIdentity{
			Name: "remote-deployment", BuildID: "build", ConfigurationIdentity: "configuration",
			EvidenceProfile: "public-grpc-history",
		},
		Claim: execution.Claim{
			Kind: execution.ClaimConforming, Property: experiment.Property.Identifier,
		},
		Evidence: graph, Cleanup: execution.CleanupResult{Complete: true},
	}
	for _, fault := range experiment.Faults {
		result.Faults = append(result.Faults, execution.FaultResult{
			Identifier: fault.Identifier, Kind: fault.Kind, SourceIdentity: "remote-fault-controller",
			Reference: "remote-fault/fired", EntityIdentity: "fault-scope",
			Installed: true, Activated: true, Realized: true, Released: true, CleanupComplete: true,
		})
	}
	resultBytes, err := json.Marshal(result)
	require.NoError(t, err)
	resultPath := filepath.Join(t.TempDir(), "result.json")
	require.NoError(t, os.WriteFile(resultPath, resultBytes, 0o600))
	return qualificationCompatibilityOptions{
		releasePath: "../../testdata/umpire3-1.2.json", experimentPath: "../../testdata/nexus-cancellation.json",
		resultPath: resultPath, profile: "remote-deployment",
	}
}
