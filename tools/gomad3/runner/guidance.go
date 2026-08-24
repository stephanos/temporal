package runner

import (
	"context"
	"fmt"
	"path/filepath"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/record"
	guide "go.temporal.io/server/tools/gomad3/runner/internal/corpus"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/target"
)

type guidanceCampaign struct {
	corpus   *guide.Corpus
	config   CampaignSpec
	prepared target.Prepared
	baseEnv  []record.Environment
	runID    string
	replayer ArtifactReplayer
}

func openGuidance(ctx context.Context, config CampaignSpec, prepared target.Prepared, baseEnvironment []record.Environment, runID string) (*guidanceCampaign, error) {
	targetRecord := prepared.RecordTarget()
	boundaryVersion, boundarySHA256 := deterministicio.BoundaryManifestIdentity()
	identity, err := guide.IdentityFor(targetRecord, prepared.RecordToolchain(), boundaryVersion, record.SHA256(boundarySHA256))
	if coverageHasChoice(config.Coverage) {
		implementation, identityErr := choice.ImplementationIdentity(prepared.BuildKey)
		if identityErr != nil {
			return nil, identityErr
		}
		identity, err = guide.IdentityForChoice(targetRecord, prepared.RecordToolchain(), boundaryVersion, record.SHA256(boundarySHA256), guide.ChoiceProfileIdentity{
			Profile: choice.Profile, ImplementationSHA256: record.SHA256FromSum(implementation), Limit: record.Uint64String(config.ChoiceTraceLimit),
		})
	}
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
	outcome execution.Classification,
	worldBundle execution.Bundle,
	mountArtifact *readonlymount.CapturedInputs,
	coverage deterministicio.SemanticCoverage,
) (bool, error) {
	if !completion.result.IOTranscript.Complete || outcome.ReplayMode == record.ReplayNone {
		return false, nil
	}
	manifest, err := manifestForRun(campaign.config, campaign.prepared, campaign.baseEnv, completion, outcome, campaign.runID, worldBundle.Manifest, mountArtifact)
	if err != nil {
		return false, err
	}
	var choiceFeatures *choice.FeatureProjection
	if coverageHasChoice(campaign.config.Coverage) {
		projected, _, projectErr := projectChoiceFeatures(completion.result.ChoiceTrace, campaign.prepared)
		if projectErr != nil {
			return false, projectErr
		}
		choiceFeatures = &projected
	}
	candidate := guide.Candidate{
		Artifact: artifact.ArtifactInput{
			Manifest: manifest, TargetPath: campaign.prepared.Path, Stdout: completion.result.Stdout.Bytes, Stderr: completion.result.Stderr.Bytes,
			IOTranscript: completion.result.IOTranscript.Bytes, ChoiceTrace: completion.result.ChoiceTrace.Trace.Bytes, ReadOnlyMounts: mountArtifact, World: worldBundle.Payloads,
		},
		Coverage: coverage, Choices: choiceFeatures,
	}
	return campaign.corpus.Admit(ctx, candidate, func(ctx context.Context, path string) (guide.ReplayResult, error) {
		replayConfig := ReplaySpec{
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
