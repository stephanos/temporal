package replay

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/tests/umpire3/artifact"
	"go.temporal.io/server/tests/umpire3/protocol"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
)

type Executor func(context.Context, protocol.Experiment) (umpire3runtime.Result, error)

type Report struct {
	ExperimentDigest string                 `json:"experimentDigest"`
	Baseline         umpire3runtime.Claim   `json:"baseline"`
	Current          umpire3runtime.Claim   `json:"current"`
	Drift            []umpire3runtime.Drift `json:"drift"`
	Result           umpire3runtime.Result  `json:"result"`
	Reproduced       bool                   `json:"reproduced"`
}

func Run(ctx context.Context, bundle artifact.Record, executor Executor) (Report, error) {
	if executor == nil {
		return Report{}, errors.New("replay executor is required")
	}
	digest, err := bundle.Experiment.Digest()
	if err != nil {
		return Report{}, err
	}
	if bundle.Result.ExperimentDigest != digest {
		return Report{}, errors.New("replay baseline is not bound to the experiment")
	}
	current, err := executor(ctx, bundle.Experiment)
	if err != nil {
		return Report{}, fmt.Errorf("execute replay: %w", err)
	}
	drift := umpire3runtime.CompareReplay(bundle.Result, current)
	if bundle.Replay.Profile != "" && bundle.Replay.Profile != current.Environment.Name {
		drift = append(drift, umpire3runtime.Drift{
			Kind: umpire3runtime.DriftRealization, Detail: "environment profile changed",
		})
	}
	capabilities := append([]string(nil), current.Environment.Capabilities...)
	slices.Sort(capabilities)
	expectedCapabilities := append([]string(nil), bundle.Replay.Capabilities...)
	slices.Sort(expectedCapabilities)
	if !slices.Equal(expectedCapabilities, capabilities) {
		drift = append(drift, umpire3runtime.Drift{
			Kind: umpire3runtime.DriftRealization, Detail: "environment capabilities changed",
		})
	}
	reproduced := len(drift) == 0 && bundle.Result.Claim.Kind == current.Claim.Kind &&
		bundle.Result.Claim.Property == current.Claim.Property
	return Report{
		ExperimentDigest: digest, Baseline: bundle.Result.Claim, Current: current.Claim,
		Drift: drift, Result: current, Reproduced: reproduced,
	}, nil
}
