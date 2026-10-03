// Package runnerconsumer is an ordinary consumer of the public Runner
// interface. TestRunnerExternalConsumerCompiles builds it as its own module,
// outside the Runner subtree, so it can name only exported Runner types and
// cannot import runner/internal/execution. It substitutes target preparation
// and artifact replay, the two public seams, and constructs every operation
// request without naming an execution or descriptor type.
package runnerconsumer

import (
	"context"
	"time"

	"go.temporal.io/server/tools/gomad3/runner"
	"go.temporal.io/server/tools/gomad3/target"
)

// cachedPreparer substitutes target preparation with an already prepared
// target.
type cachedPreparer struct {
	prepared target.Prepared
}

func (preparer cachedPreparer) Prepare(_ context.Context, spec target.Spec) (target.Prepared, error) {
	prepared := preparer.prepared
	prepared.Kind, prepared.Source = spec.Kind, spec.Source
	return prepared, nil
}

// matchingReplayer substitutes artifact replay with one that reports a match.
type matchingReplayer struct{}

func (matchingReplayer) Replay(_ context.Context, spec runner.ReplaySpec) (runner.ReplayResult, error) {
	return runner.ReplayResult{Verified: true, Match: !spec.VerifyOnly}, nil
}

var (
	_ runner.Preparer         = cachedPreparer{}
	_ runner.ArtifactReplayer = matchingReplayer{}
)

// Operations runs each Runner operation once with substituted preparation
// and replay where the request accepts them.
func Operations(ctx context.Context, supervisor []string, toolchainRoot string) error {
	campaign := runner.CampaignSpec{
		Strategy: runner.StrategySeed, Seeds: "1-4", Parallel: 2,
		ExecutionTimeout: time.Second, OverallTimeout: time.Minute, TerminateGrace: time.Second,
		OnFailure: runner.PolicyAll, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20,
		Artifacts: "artifacts", Target: target.Spec{Kind: target.KindGoRun, Source: ".", ToolchainRoot: toolchainRoot},
		SupervisorCommand: supervisor, RunnerBuild: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		Progress: func(runner.CampaignEvent) error { return nil },
		Preparer: cachedPreparer{}, Replayer: matchingReplayer{},
	}
	if _, err := runner.Explore(ctx, campaign); err != nil {
		return err
	}
	plan, err := runner.CreateCampaignPlan(ctx, runner.CampaignPlanSpec{Campaign: campaign, Output: "campaign.plan.json"})
	if err != nil {
		return err
	}
	shard, err := runner.RunCampaignShard(ctx, runner.CampaignShardSpec{
		PlanPath: plan.Path, Shard: runner.CampaignShard{Index: 0, Count: 1}, Artifacts: "artifacts", ToolchainRoot: toolchainRoot,
		RunnerBuild: campaign.RunnerBuild, SupervisorCommand: supervisor, Replayer: matchingReplayer{},
	})
	if err != nil {
		return err
	}
	if _, err := runner.Resume(ctx, runner.ResumeSpec{
		CampaignPath: shard.CampaignPath, ToolchainRoot: toolchainRoot, RunnerBuild: campaign.RunnerBuild,
		SupervisorCommand: supervisor, Replayer: matchingReplayer{},
	}); err != nil {
		return err
	}
	if _, err := runner.Replay(ctx, runner.ReplaySpec{ArtifactPath: "artifact", ToolchainRoot: toolchainRoot, SupervisorCommand: supervisor}); err != nil {
		return err
	}
	_, err = runner.Minimize(ctx, runner.MinimizeSpec{
		ArtifactPath: "artifact", OutputRoot: "minimized", AttemptBudget: 16, ToolchainRoot: toolchainRoot,
		SupervisorCommand: supervisor, Replayer: matchingReplayer{},
	})
	return err
}
