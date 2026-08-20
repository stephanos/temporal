package replay

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/execution"
	umpire3fault "go.temporal.io/server/tests/umpire3/fault"
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
				Source: "cluster", Reference: "sensitive-reference",
				EntityIdentity: "sensitive-entity", Lineage: []string{"sensitive-namespace", "sensitive-entity"},
			},
		}},
		Observations: []execution.Observation{{
			CheckpointID: "checkpoint", Kind: "observation", Source: "history",
			Reference: "sensitive-observation", EntityIdentity: "sensitive-entity",
			Lineage: []string{"sensitive-namespace", "sensitive-entity"},
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
	require.NotContains(t, string(encoded), "sensitive-footprint-namespace")
	require.NotContains(t, string(encoded), "sensitive-footprint-participant")
	require.NotContains(t, string(encoded), "sensitive-footprint-reference")
	require.Contains(t, string(encoded), "sha256:")
	record, err := DecodeBundle(encoded, 1<<20)
	require.NoError(t, err)
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

func TestReplayBundleV1BytesRemainStable(t *testing.T) {
	experiment := artifactExperiment(t)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	result := execution.Result{
		FormatVersion:    execution.ResultFormatVersion,
		ExperimentDigest: digest,
		Claim:            execution.Claim{Kind: execution.ClaimInconclusive, Reason: "characterization"},
	}
	result.DeriveAssurance()

	encoded, err := EncodeBundle(experiment, result, experiment.Retention.MaxArtifactBytes)
	require.NoError(t, err)
	sum := sha256.Sum256(encoded)
	require.Equal(t, "20a15821675d50e9abffdbcfb8379c025f2acdfe992c355fbd48f94945dfcc09",
		hex.EncodeToString(sum[:]))
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
