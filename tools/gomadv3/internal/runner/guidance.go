package runner

import (
	"context"
	"errors"
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
	targetRecord := recordTarget(prepared)
	boundaryVersion, boundarySHA256 := ioprofile.BoundaryManifestIdentity()
	identity, err := guide.IdentityFor(targetRecord, record.Toolchain{
		GoVersion: prepared.GoVersion, BuildKey: prepared.BuildKey, TargetGOOS: prepared.TargetGOOS, TargetGOARCH: prepared.TargetGOARCH,
	}, boundaryVersion, boundarySHA256)
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
	published, err := (artifact.Store{Root: campaign.corpus.CasesPath(), Context: ctx, MaximumBytes: guide.MaximumBytes, Key: artifact.StoreKeyRecord}).Publish(artifact.Input{
		Manifest: manifest, TargetPath: campaign.prepared.Path, Stdout: completion.result.Stdout.Bytes, Stderr: completion.result.Stderr.Bytes,
		IOTranscript: completion.result.IOTranscript.Bytes, ReadOnlyMounts: mountArtifact, World: worldBundle.Payloads,
	})
	if err != nil {
		return false, fmt.Errorf("publish guided corpus case: %w", err)
	}
	features, err := guide.SemanticFeatures(published.Manifest, coverage, completion.result.IOTranscript.Bytes, worldBundle.Payloads.Transitions)
	if err != nil {
		return false, errors.Join(err, campaign.corpus.Discard(published))
	}
	if !campaign.corpus.Interesting(features, published.StoredBytes) {
		return false, campaign.corpus.Discard(published)
	}
	replayConfig := replay.Config{
		ArtifactPath: published.Path, ToolchainRoot: campaign.config.Target.ToolchainRoot,
		SupervisorCommand: append([]string(nil), campaign.config.SupervisorCommand...),
	}
	if len(campaign.config.SupervisorCommand) != 0 {
		replayConfig.BootstrapCommand = []string{campaign.config.SupervisorCommand[0], "__target_bootstrap"}
	}
	replayed, err := campaign.replayer.Replay(ctx, replayConfig)
	if err != nil {
		return false, errors.Join(fmt.Errorf("replay guided corpus case: %w", err), campaign.corpus.Discard(published))
	}
	replayResult := guide.ReplayResult{Verified: replayed.Verified, Match: replayed.Match, Diagnostic: replayed.Diagnostic, Divergence: replayed.Divergence}
	if !replayResult.Verified || !replayResult.Match {
		return false, errors.Join(fmt.Errorf("guided corpus replay diverged at %s", replayResult.Divergence), campaign.corpus.Discard(published))
	}
	return campaign.corpus.Merge(published, coverage, features, replayResult)
}

func recordTarget(prepared target.Prepared) record.Target {
	return record.Target{
		Kind: string(prepared.Kind), Source: prepared.Source, SHA256: record.SHA256(prepared.SHA256), Size: record.Uint64String(prepared.Size),
		Argv: append([]string{}, prepared.Argv...), BuildTags: append([]string{}, prepared.BuildTags...), Adapters: cloneAdapters(prepared.Adapters), Compatibility: cloneCompatibility(prepared.Compatibility), BuildInfo: cloneBuildInfo(prepared.BuildInfo),
	}
}

func guidedCorpusPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve guided corpus path: %w", err)
	}
	return absolute, nil
}
