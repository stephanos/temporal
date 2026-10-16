package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	backend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

type inspectionBoundaryProvider struct {
	*boundaryProvider
	inspections   int
	inspection    backend.InspectionResult
	inspectionErr error
}

func (provider *inspectionBoundaryProvider) Inspect(_ context.Context, _ *artifact.Opened, _ backend.InspectionRequest) (backend.InspectionResult, error) {
	provider.inspections++
	return provider.inspection, provider.inspectionErr
}

func inspectionReplayArtifact(t *testing.T) (target.Prepared, artifact.Artifact) {
	t.Helper()
	prepared := newFakePreparer(t).prepared
	provenance := []byte("fixture")
	prepared.Backend = &record.BackendMetadata{Name: "fixture-cooperative", ReplayMode: record.ReplayExact, Provenance: record.BackendPayload{Schema: "fixture/v1", File: "backend/provenance.json", SHA256: record.HashBytes(provenance), Bytes: record.Uint64String(len(provenance))}}
	prepared.BuildInfo.Settings = []record.BuildSetting{}
	prepared.CapabilityMode = target.CapabilityModeClosure
	config, _ := testConfig(t, nil, nil, "7", PolicyAll, 1)
	config.Target.Backend = "fixture-cooperative"
	config.Backend = &boundaryProvider{}
	_, environment, err := validateConfig(config)
	require.NoError(t, err)
	completion := runCompletion{job: runJob{seed: 7}, startedAt: time.Unix(1, 0), finishedAt: time.Unix(2, 0), result: processResult(1, "done", "")}
	completion.result.BackendEvidence = []byte("run")
	world := noneWorldBundle()
	manifest, err := manifestForRun(campaignRequestFromSpec(config), prepared, environment, completion, backend.Classify(completion.result, false, world.Manifest.Terminal), "fixture", world.Manifest, nil)
	require.NoError(t, err)
	metadata := manifest.Target.Backend
	published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, artifact.ArtifactInput{Manifest: manifest, TargetPath: prepared.Path, Stdout: []byte("done"), World: world.Payloads, BackendPayloads: []target.BackendPayload{{Reference: metadata.Provenance, Data: provenance}, {Reference: *metadata.Evidence, Data: []byte("run")}}})
	require.NoError(t, err)
	return prepared, published
}

func TestReplayInspectionUsesDedicatedProviderAfterPreflight(t *testing.T) {
	prepared, published := inspectionReplayArtifact(t)
	request := &backend.InspectionRequest{Stop: backend.InspectionStop{Kind: "choice"}, MaximumBytes: 1024, PerDumpBytes: 512}
	provider := &inspectionBoundaryProvider{boundaryProvider: &boundaryProvider{prepared: prepared}, inspection: backend.InspectionResult{Status: backend.InspectionStatusInspectedPrefix, PrefixVerified: true, WitnessMatch: true}}
	result, err := Replay(t.Context(), ReplaySpec{Backend: provider, ArtifactPath: published.Path, Inspection: request})
	require.NoError(t, err)
	require.Equal(t, 1, provider.inspections)
	require.Zero(t, provider.calls)
	require.False(t, result.Match)
	require.Equal(t, ChoiceReplayNone, result.ChoiceReplayStatus)
	require.NotNil(t, result.Inspection)
	require.Equal(t, string(published.Manifest.RecordHash), result.Inspection.ArtifactID)
	require.Equal(t, *request, result.Inspection.Requested)
}

func TestReplayInspectionRejectsMissingWitnessAndPreservesPreflightRefusals(t *testing.T) {
	prepared, published := inspectionReplayArtifact(t)
	for _, test := range []struct {
		name      string
		result    backend.InspectionResult
		err       error
		preflight bool
	}{
		{"missing runtime witness", backend.InspectionResult{Status: backend.InspectionStatusInspectedPrefix, PrefixVerified: true}, nil, false},
		{"unknown registration", backend.InspectionResult{}, &backend.InspectionPreflightError{Err: errors.New("unknown target dump")}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider := &inspectionBoundaryProvider{boundaryProvider: &boundaryProvider{prepared: prepared}, inspection: test.result, inspectionErr: test.err}
			_, err := Replay(t.Context(), ReplaySpec{Backend: provider, ArtifactPath: published.Path, Inspection: &backend.InspectionRequest{Stop: backend.InspectionStop{Kind: "failure"}, MaximumBytes: 1024, PerDumpBytes: 512}})
			require.Error(t, err)
			require.Zero(t, provider.calls)
			var refusal *ReplayPreflightError
			require.Equal(t, test.preflight, errors.As(err, &refusal))
		})
	}
}
