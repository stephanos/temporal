package replay

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/execution"
	umpire3fault "go.temporal.io/server/tests/umpire3/fault"
	"go.temporal.io/server/tests/umpire3/observation"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestEncodeRedactsConcreteRuntimeIdentities(t *testing.T) {
	experiment := artifactExperiment(t)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	result := execution.Result{
		FormatVersion: execution.ResultFormatVersion, ExperimentDigest: digest,
		Bindings: execution.Bindings{"operation": "sensitive-operation-id"},
		Actions: []execution.ActionResult{{
			Identifier: "a1",
			Kind:       "schedule-operation",
			Evidence: execution.ActionEvidence{
				Source: "cluster", Outcome: protocol.ActionOutcomeApplied,
				SourceIdentity: "sensitive-action-source", Reference: "sensitive-reference",
				EntityIdentity: "sensitive-entity", Lineage: []string{"sensitive-namespace", "sensitive-entity"},
			},
		}},
		Observations: []execution.Observation{{
			CheckpointID: "checkpoint", Kind: "observation", Source: "history",
			SourceIdentity: "sensitive-observation-source",
			Reference:      "sensitive-observation", EntityIdentity: "sensitive-entity",
			Lineage: []string{"sensitive-namespace", "sensitive-entity"}, SupportingFacts: []string{"raw-fact"},
		}},
		Facts: []observation.Fact{{
			Identifier: "raw-fact",
			Source: observation.Source{
				Identity: "raw-history", ClockDomain: "raw-sequence", Sequence: 1,
				Reference: "sensitive-raw-reference", CausalReferences: []string{"sensitive-raw-cause"},
				EntityIdentity: "sensitive-raw-entity", Lineage: []string{"sensitive-raw-lineage"},
			},
			History: &observation.HistoryEvent{
				EventType: observation.NexusCancellationAccepted, EventID: 1,
				OperationID: "sensitive-raw-operation",
			},
		}},
		Faults: []execution.FaultResult{{
			Identifier: "fault", Kind: "drop", SourceIdentity: "sensitive-fault-source",
			Reference: "sensitive-fault-reference", EntityIdentity: "sensitive-fault-entity",
			Installed: true, Activated: true, Realized: true,
		}},
		Cleanup: execution.CleanupResult{
			Complete:             false,
			RecoverableResources: map[string]string{"operation": "sensitive-operation-id"},
		},
	}
	result.DeriveAssurance()
	result.Footprint, err = learnedFootprintForArtifact()
	require.NoError(t, err)

	encoded, err := EncodeBundle(experiment, result, 1<<20)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "sensitive-operation-id")
	require.NotContains(t, string(encoded), "sensitive-reference")
	require.NotContains(t, string(encoded), "sensitive-observation")
	require.NotContains(t, string(encoded), "sensitive-entity")
	require.NotContains(t, string(encoded), "sensitive-namespace")
	require.NotContains(t, string(encoded), "sensitive-fault-source")
	require.NotContains(t, string(encoded), "sensitive-fault-reference")
	require.NotContains(t, string(encoded), "sensitive-fault-entity")
	require.NotContains(t, string(encoded), "sensitive-action-source")
	require.NotContains(t, string(encoded), "sensitive-observation-source")
	require.NotContains(t, string(encoded), "sensitive-raw-reference")
	require.NotContains(t, string(encoded), "sensitive-raw-cause")
	require.NotContains(t, string(encoded), "sensitive-raw-entity")
	require.NotContains(t, string(encoded), "sensitive-raw-lineage")
	require.NotContains(t, string(encoded), "sensitive-raw-operation")
	require.NotContains(t, string(encoded), "sensitive-footprint-namespace")
	require.NotContains(t, string(encoded), "sensitive-footprint-participant")
	require.NotContains(t, string(encoded), "sensitive-footprint-reference")
	require.Contains(t, string(encoded), "sha256:")
	record, err := DecodeBundle(encoded, 1<<20)
	require.NoError(t, err)
	require.NoError(t, record.Result.ValidateEvidenceDigest())
	require.Equal(t, record.Result.Facts[0].Identifier, record.Result.Observations[0].SupportingFacts[0])
	require.Equal(t, BundleFormatVersion, record.FormatVersion)
	require.Equal(t, experiment.Scope.Seed, record.Replay.Seed)
	require.Equal(t, "umpire3 replay -bundle <bundle.json>", record.Replay.Command)
}

func learnedFootprintForArtifact() (*umpire3fault.Report, error) {
	report, err := umpire3fault.BuildFootprintReport(
		[]umpire3fault.Footprint{{Protocol: "http", Service: "nexus", Route: "/service/operation"}},
		[]umpire3fault.Call{{
			Protocol: "http", Service: "nexus", Route: "/service/operation",
			Direction: umpire3fault.DirectionInbound, Role: umpire3fault.CallRoleInternal,
			Namespace: "sensitive-footprint-namespace", Participant: "sensitive-footprint-participant",
			Attempt: 1, Occurrence: 1, Interval: umpire3fault.Interval{Start: 1, Stop: 2},
			CausalReferences: []string{"sensitive-footprint-reference"},
		}}, nil)
	return &report, err
}

func TestEncodeEnforcesArtifactLimit(t *testing.T) {
	experiment := artifactExperiment(t)
	digest, digestErr := experiment.Digest()
	require.NoError(t, digestErr)
	result := execution.Result{FormatVersion: execution.ResultFormatVersion, ExperimentDigest: digest}
	result.DeriveAssurance()
	_, err := EncodeBundle(experiment, result, 16)
	require.ErrorContains(t, err, "exceeds")
}

func TestViolatingBundleRetainsDigestBoundSemanticTrace(t *testing.T) {
	experiment := artifactExperiment(t)
	view, found, err := protocol.DefaultAttemptExecutionView(experiment)
	require.NoError(t, err)
	require.True(t, found)
	attempts := make([]protocol.ObservedAttempt, len(experiment.Actions))
	for index, action := range experiment.Actions {
		attempts[index] = protocol.ObservedAttempt{
			Action: protocol.ActionKind(action.Kind), Outcome: protocol.ActionOutcomeApplied,
		}
	}
	trace, err := protocol.NewLiveSemanticTrace(experiment, view, attempts)
	require.NoError(t, err)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	result := execution.Result{
		FormatVersion: execution.ResultFormatVersion, ExperimentDigest: digest,
		Trace: &trace, Claim: execution.Claim{
			Kind: execution.ClaimViolating, Property: experiment.Property.Identifier,
		},
	}
	result.DeriveAssurance()

	encoded, err := EncodeBundle(experiment, result, experiment.Retention.MaxArtifactBytes)
	require.NoError(t, err)
	record, err := DecodeBundle(encoded, experiment.Retention.MaxArtifactBytes)
	require.NoError(t, err)
	require.NotNil(t, record.Result.Trace)
	require.Equal(t, trace.Replay.Digest, record.Result.Trace.Replay.Digest)
	require.NotEmpty(t, record.Result.EvidenceDigest)

	record.Result.Trace.Steps[0].Outcome = protocol.ActionOutcomeSuppressed
	tampered, err := json.Marshal(record)
	require.NoError(t, err)
	_, err = DecodeBundle(tampered, experiment.Retention.MaxArtifactBytes)
	require.Error(t, err)
}

func TestReplayBundleV3EncodingIsDeterministic(t *testing.T) {
	experiment := artifactExperiment(t)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	result := execution.Result{
		FormatVersion:    execution.ResultFormatVersion,
		ExperimentDigest: digest,
		Claim:            execution.Claim{Kind: execution.ClaimInconclusive, Reason: "characterization"},
	}
	result.DeriveAssurance()

	first, err := EncodeBundle(experiment, result, experiment.Retention.MaxArtifactBytes)
	require.NoError(t, err)
	second, err := EncodeBundle(experiment, result, experiment.Retention.MaxArtifactBytes)
	require.NoError(t, err)
	require.Equal(t, "umpire3/replay-bundle/v3", BundleFormatVersion)
	require.Equal(t, first, second)
	_, err = DecodeBundle(first, experiment.Retention.MaxArtifactBytes)
	require.NoError(t, err)
}

func TestDecodeRejectsV2BundleWithoutSemanticTraceContract(t *testing.T) {
	experiment := artifactExperiment(t)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	result := execution.Result{FormatVersion: execution.ResultFormatVersion, ExperimentDigest: digest}
	result.DeriveAssurance()
	encoded, err := EncodeBundle(experiment, result, experiment.Retention.MaxArtifactBytes)
	require.NoError(t, err)
	encoded = bytes.Replace(encoded,
		[]byte(`"formatVersion":"umpire3/replay-bundle/v3"`),
		[]byte(`"formatVersion":"umpire3/replay-bundle/v2"`), 1)

	_, err = DecodeBundle(encoded, experiment.Retention.MaxArtifactBytes)
	require.ErrorContains(t, err, "unsupported replay bundle format")
}

func TestFileCorpusDeduplicatesByExperimentDigest(t *testing.T) {
	root := t.TempDir()
	store := NewFileCorpus(root)
	experiment := artifactExperiment(t)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	result := execution.Result{FormatVersion: execution.ResultFormatVersion, ExperimentDigest: digest}
	result.DeriveAssurance()

	first, err := store.Save(context.Background(), experiment, result)
	require.NoError(t, err)
	second, err := store.Save(context.Background(), experiment, result)
	require.NoError(t, err)
	require.Equal(t, first, second)

	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.FileExists(t, filepath.Join(root, entries[0].Name()))
}

func artifactExperiment(t *testing.T) protocol.Experiment {
	t.Helper()
	encoded, err := os.ReadFile("../testdata/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	return experiment
}
