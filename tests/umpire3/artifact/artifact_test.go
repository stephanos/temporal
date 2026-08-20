package artifact

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/protocol"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
)

func TestEncodeRedactsConcreteRuntimeIdentities(t *testing.T) {
	experiment := artifactExperiment()
	result := umpire3runtime.Result{
		FormatVersion: protocol.FormatVersion,
		Bindings:      environment.Bindings{"operation": "sensitive-operation-id"},
		Actions: []umpire3runtime.ActionResult{{
			Identifier: "a1",
			Kind:       "schedule-operation",
			Evidence: environment.ActionEvidence{
				Source:    "cluster",
				Reference: "sensitive-reference",
			},
		}},
		Cleanup: environment.CleanupResult{
			Complete:             false,
			RecoverableResources: map[string]string{"operation": "sensitive-operation-id"},
		},
	}

	encoded, err := Encode(experiment, result, 1<<20)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "sensitive-operation-id")
	require.NotContains(t, string(encoded), "sensitive-reference")
	require.Contains(t, string(encoded), "sha256:")
}

func TestEncodeEnforcesArtifactLimit(t *testing.T) {
	_, err := Encode(artifactExperiment(), umpire3runtime.Result{}, 16)
	require.ErrorContains(t, err, "exceeds")
}

func TestFileCorpusDeduplicatesByExperimentDigest(t *testing.T) {
	root := t.TempDir()
	store := NewFileCorpus(root)
	experiment := artifactExperiment()
	result := umpire3runtime.Result{FormatVersion: protocol.FormatVersion}

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

func artifactExperiment() protocol.Experiment {
	return protocol.Experiment{
		FormatVersion: protocol.FormatVersion,
		ExperimentID:  "artifact",
		Model: protocol.Model{
			Modules:        []string{"Model"},
			SourceRevision: "revision",
			SemanticHash:   "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			LeanVersion:    "4.33.0",
		},
		Property: protocol.Property{
			Identifier:    "property",
			StatementHash: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Claim:         "implementation-conformance",
		},
		Scope: protocol.Scope{
			Bounds:   protocol.Bounds{MaxDepth: 1, MaxResults: 1},
			Strategy: "curated",
		},
		Resources: []protocol.Resource{{Identifier: "resource", Kind: "nexus-operation"}},
		Actions: []protocol.Action{{
			Identifier:           "a1",
			Kind:                 "schedule-operation",
			RequiredCapabilities: []string{"nexus"},
			PostCheckpoint:       "checkpoint",
		}},
		Checkpoints: []protocol.Checkpoint{{
			Identifier:     "checkpoint",
			Observation:    "created",
			Ordering:       "none",
			OmissionPolicy: "required",
		}},
		Provenance: protocol.Provenance{Kind: "curated-trace", ProofManifest: "proof"},
		Retention:  protocol.Retention{RedactionClass: "semantic-only", MaxArtifactBytes: 1 << 20},
	}
}
