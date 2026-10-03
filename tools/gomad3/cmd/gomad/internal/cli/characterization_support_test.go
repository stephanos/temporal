package cli

import (
	"context"
	"errors"

	"go.temporal.io/server/tools/gomad3/qualification"
	"go.temporal.io/server/tools/gomad3/runner"
)

// fakeInstallation is the resolved installation a characterization test hands
// to a command through its dependency seam. It records every explicit
// --toolchain-root the command asked to resolve, so tests can pin both what a
// command requests and how it wires the resolved result into its operation.
// Only this file knows the seam's shape; the characterization tests in
// characterization_test.go build their dependencies through it.
type fakeInstallation struct {
	toolchainRoot string
	executable    string
	runnerBuild   string
	err           error
	requested     *[]string
}

func newFakeInstallation() fakeInstallation {
	return fakeInstallation{toolchainRoot: "/toolchain", executable: "/bin/gomad", runnerBuild: "sha256:runner", requested: new([]string)}
}

func (fake fakeInstallation) failing(err error) fakeInstallation {
	fake.err = err
	return fake
}

func (fake fakeInstallation) install(explicitToolchainRoot string) (installation, error) {
	*fake.requested = append(*fake.requested, explicitToolchainRoot)
	if fake.err != nil {
		return installation{}, fake.err
	}
	return installation{toolchainRoot: fake.toolchainRoot, executable: fake.executable, runnerBuild: fake.runnerBuild}, nil
}

func (fake fakeInstallation) exploreDependencies(explore func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error), plan func(context.Context, runner.CampaignPlanSpec) (runner.CampaignPlanResult, error)) exploreDependencies {
	return exploreDependencies{install: fake.install, workingDirectory: func() (string, error) { return "/workspace", nil }, explore: explore, plan: plan}
}

func (fake fakeInstallation) replayDependencies(replay func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error)) replayDependencies {
	return replayDependencies{install: fake.install, replay: replay}
}

func (fake fakeInstallation) campaignShardDependencies(run func(context.Context, runner.CampaignShardSpec) (runner.CampaignResult, error)) campaignShardDependencies {
	return campaignShardDependencies{install: fake.install, run: run}
}

func (fake fakeInstallation) resumeDependencies(run func(context.Context, runner.ResumeSpec) (runner.CampaignResult, error)) resumeDependencies {
	return resumeDependencies{install: fake.install, run: run}
}

func (fake fakeInstallation) minimizeDependencies(minimize func(context.Context, runner.MinimizeSpec) (runner.MinimizeResult, error)) minimizeDependencies {
	return minimizeDependencies{install: fake.install, minimize: minimize}
}

func (fake fakeInstallation) qualifyDependencies(run func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error), replay func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error), write func(string, qualification.QualificationReport) (string, error)) qualifyDependencies {
	return qualifyDependencies{install: fake.install, workingDirectory: func() (string, error) { return "/workspace", nil }, run: run, replay: replay, write: write}
}

// analyzeToolchain is the toolchain-root seam gomad analyze resolves through.
func (fake fakeInstallation) analyzeToolchain(explicitToolchainRoot string) (string, error) {
	resolved, err := fake.install(explicitToolchainRoot)
	return resolved.toolchainRoot, err
}

var errFakeInstallation = errors.New("resolve Gomad installation: fake installation is unavailable")
