package backend

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/runner"
	runnerbackend "go.temporal.io/server/tools/gomad3/runner/backend"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestObservedCampaignResumeRetainsBackendAndOrdinals(t *testing.T) {
	provider := integrationProvider(t)
	fixture, err := filepath.Abs("../../gomad3/internal/gomadtool/conformance/testdata")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	config := runner.CampaignSpec{Backend: &resumeBoundaryProvider{Provider: provider}, Seeds: "7-8", Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 3 * time.Minute, TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyAll, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, Artifacts: t.TempDir(), RunnerBuild: "task4-integration", Coverage: runner.CoverageNone, ProgressInterval: time.Millisecond, Target: target.Spec{Backend: Name, Kind: target.KindGoTest, Source: "./io_failure", WorkingDir: fixture, BuildTags: []string{"gomad_fixture", "test_dep"}, Args: []string{"-test.run=^TestDeterministicIOFailure$", "-test.v"}}, Progress: func(event runner.CampaignEvent) error {
		if event.Attempted == 1 {
			cancel()
			return nil
		}
		return nil
	}}
	partial, err := runner.Explore(ctx, config)
	if !errors.Is(err, context.Canceled) || partial.Failures != 1 {
		t.Fatalf("partial campaign: %#v %v", partial, err)
	}
	resumed, err := runner.Resume(t.Context(), runner.ResumeSpec{Backend: provider, CampaignPath: partial.CampaignPath, RunnerBuild: config.RunnerBuild})
	if err != nil || resumed.Attempted != 2 || resumed.Failures != 2 || resumed.StopReason != runner.StopSeedsExhausted {
		t.Fatalf("resumed observed campaign: %#v %v", resumed, err)
	}
}

type resumeBoundaryProvider struct{ *Provider }

func (p *resumeBoundaryProvider) Run(ctx context.Context, request runnerbackend.Request) (runnerbackend.Result, error) {
	if request.Seed == 8 {
		<-ctx.Done()
		return runnerbackend.Result{Termination: runnerbackend.Infrastructure, Cancelled: true, Reaped: true}, nil
	}
	return p.Provider.Run(ctx, request)
}
