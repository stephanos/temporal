package consumer

import (
	"context"

	"go.temporal.io/server/tools/gomad3/runner"
	"go.temporal.io/server/tools/gomad3/target"
)

type preparer struct{}

func (preparer) Prepare(context.Context, target.Spec) (target.Prepared, error) {
	return target.Prepared{}, nil
}

type replayer struct{}

func (replayer) Replay(context.Context, runner.ReplaySpec) (runner.ReplayResult, error) {
	return runner.ReplayResult{}, nil
}

func ConstructRequests() {
	_ = runner.CampaignSpec{Preparer: preparer{}, Replayer: replayer{}}
	_ = runner.CampaignShardSpec{Replayer: replayer{}}
	_ = runner.ReplaySpec{}
	_ = runner.MinimizeSpec{Replayer: replayer{}}
	_ = runner.ResumeSpec{Replayer: replayer{}}
}
