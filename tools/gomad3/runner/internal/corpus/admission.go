package corpus

import (
	"context"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio"
)

type Candidate struct {
	Artifact artifact.ArtifactInput
	Coverage deterministicio.SemanticCoverage
	Choices  *choice.FeatureProjection
}

type ReplayCandidate func(context.Context, string) (ReplayResult, error)

func (corpus *Corpus) Admit(ctx context.Context, candidate Candidate, replay ReplayCandidate) (bool, error) {
	published, err := artifact.PublishArtifact(artifact.Store{Root: corpus.casesPath(), Context: ctx, MaximumBytes: maximumBytes, Key: artifact.StoreKeyRecord}, candidate.Artifact)
	if err != nil {
		return false, fmt.Errorf("publish guided corpus case: %w", err)
	}
	features, err := semanticFeatures(published.Manifest, candidate.Coverage, candidate.Artifact.IOTranscript, candidate.Artifact.World.Transitions, candidate.Choices)
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
