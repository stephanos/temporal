package runner

import (
	"context"
	"fmt"
	"path/filepath"

	"go.temporal.io/server/tools/gomadv3/internal/artifact"
	"go.temporal.io/server/tools/gomadv3/internal/guide"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	executionoutcome "go.temporal.io/server/tools/gomadv3/internal/outcome"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/replay"
	"go.temporal.io/server/tools/gomadv3/internal/romount"
	"go.temporal.io/server/tools/gomadv3/internal/target"
	"go.temporal.io/server/tools/gomadv3/internal/worldrecord"
)

type guidanceCampaign struct {
	corpus   *guide.Corpus
	config   Config
	prepared target.Prepared
	baseEnv  []record.Environment
	runID    string
	replayer ArtifactReplayer
}

func openGuidance(ctx context.Context, config Config, prepared target.Prepared, baseEnvironment []record.Environment, runID string) (*guidanceCampaign, error) {
	targetRecord := prepared.RecordTarget()
	boundaryVersion, boundarySHA256 := ioprofile.BoundaryManifestIdentity()
	identity, err := guide.IdentityFor(targetRecord, prepared.RecordToolchain(), boundaryVersion, boundarySHA256)
	if err != nil {
		return nil, err
	}
	corpus, err := guide.Open(ctx, config.Corpus, identity)
	if err != nil {
		return nil, err
	}
	replayer := config.Replayer
	if replayer == nil {
		replayer = artifactReplayer{}
	}
	return &guidanceCampaign{
		corpus: corpus, config: config, prepared: prepared, baseEnv: append([]record.Environment(nil), baseEnvironment...), runID: runID, replayer: replayer,
	}, nil
}

func (campaign *guidanceCampaign) Close() error {
	if campaign == nil {
		return nil
	}
	return campaign.corpus.Close()
}

func (campaign *guidanceCampaign) Snapshot() guide.Snapshot {
	return campaign.corpus.Snapshot()
}

func (campaign *guidanceCampaign) MergeRun(
	ctx context.Context,
	completion runCompletion,
	outcome executionoutcome.Classification,
	worldBundle worldrecord.Bundle,
	mountArtifact *romount.ArtifactRecord,
	coverage ioprofile.SemanticCoverage,
) (bool, error) {
	if !completion.result.IOTranscript.Complete || outcome.ReplayMode == record.ReplayNone {
		return false, nil
	}
	manifest, err := manifestForRun(campaign.config, campaign.prepared, campaign.baseEnv, completion, outcome, campaign.runID, worldBundle.Manifest, mountArtifact)
	if err != nil {
		return false, err
	}
	candidate := guide.Candidate{
		Artifact: artifact.Input{
			Manifest: manifest, TargetPath: campaign.prepared.Path, Stdout: completion.result.Stdout.Bytes, Stderr: completion.result.Stderr.Bytes,
			IOTranscript: completion.result.IOTranscript.Bytes, ReadOnlyMounts: mountArtifact, World: worldBundle.Payloads,
		},
		Coverage: coverage,
	}
	return campaign.corpus.Admit(ctx, candidate, func(ctx context.Context, path string) (guide.ReplayResult, error) {
		replayConfig := replay.Config{
			ArtifactPath: path, ToolchainRoot: campaign.config.Target.ToolchainRoot,
			SupervisorCommand: append([]string(nil), campaign.config.SupervisorCommand...),
		}
		if len(campaign.config.SupervisorCommand) != 0 {
			replayConfig.BootstrapCommand = []string{campaign.config.SupervisorCommand[0], "__target_bootstrap"}
		}
		replayed, err := campaign.replayer.Replay(ctx, replayConfig)
		return guide.ReplayResult{Verified: replayed.Verified, Match: replayed.Match, Diagnostic: replayed.Diagnostic, Divergence: replayed.Divergence}, err
	})
}

func guidedCorpusPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve guided corpus path: %w", err)
	}
	return absolute, nil
}
