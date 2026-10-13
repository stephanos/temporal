package corpus

import (
	"context"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/model/choice"
)

type Candidate struct {
	Artifact artifact.ArtifactInput
	Coverage record.SemanticCoverage
	Choices  *choice.FeatureProjection
}

type ReplayCandidate func(context.Context, string) (ReplayResult, error)

// Admit publishes a candidate case, replays it, and commits it when it is
// novel and fits the bounded selection. A case whose record hash is already
// live is a no-op: the published case is the live one, and it is neither
// replayed nor committed again.
func (corpus *Corpus) Admit(ctx context.Context, candidate Candidate, replay ReplayCandidate) (bool, error) {
	if corpus == nil || corpus.writer == nil {
		return false, errors.New("guided corpus is not open")
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: corpus.casesPath(), Context: ctx, MaximumBytes: corpus.limits.bytes, Key: artifact.StoreKeyRecord, TargetPool: corpus.targetPool()}, candidate.Artifact)
	if err != nil {
		return false, fmt.Errorf("publish guided corpus case: %w", err)
	}
	if _, live := corpus.live[published.Manifest.RecordHash]; live {
		return false, nil
	}
	evidence, err := corpus.evidence.Evidence(ctx, backends.EvidenceRequest{Transcript: candidate.Artifact.IOTranscript})
	if err != nil {
		return false, errors.Join(err, corpus.discard(published))
	}
	features, err := semanticFeatures(published.Manifest, candidate.Coverage, evidence.Operations, candidate.Artifact.World.Transitions, candidate.Choices)
	if err != nil {
		return false, errors.Join(err, corpus.discard(published))
	}
	if !corpus.interesting(features, published.StoredBytes) {
		return false, corpus.discard(published)
	}
	if replay == nil {
		return false, errors.Join(errors.New("guided corpus replay is required"), corpus.discard(published))
	}
	replayed, err := replay(ctx, published.Path)
	if err != nil {
		return false, errors.Join(fmt.Errorf("replay guided corpus case: %w", err), corpus.discard(published))
	}
	if !replayed.Verified || !replayed.Match {
		return false, errors.Join(fmt.Errorf("guided corpus replay diverged at %s", replayed.Divergence), corpus.discard(published))
	}
	return corpus.merge(published, candidate.Coverage, features, replayed)
}
