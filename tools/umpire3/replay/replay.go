package replay

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/tools/umpire3/execution"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

type Executor func(context.Context, protocolexperiment.Experiment) (execution.Result, error)

type Report struct {
	ExperimentDigest string           `json:"experimentDigest"`
	Baseline         execution.Claim  `json:"baseline"`
	Current          execution.Claim  `json:"current"`
	Drift            []Drift          `json:"drift"`
	Result           execution.Result `json:"result"`
	Reproduced       bool             `json:"reproduced"`
}

func Reproduce(ctx context.Context, bundle Bundle, executor Executor) (Report, error) {
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
	drift := Compare(bundle.Result, current)
	if bundle.Replay.Profile != "" && bundle.Replay.Profile != current.Environment.Name {
		drift = append(drift, Drift{
			Kind: DriftRealization, Detail: "environment profile changed",
		})
	}
	capabilities := append([]protocolcatalog.CapabilityID(nil), current.Environment.Capabilities...)
	slices.Sort(capabilities)
	expectedCapabilities := append([]protocolcatalog.CapabilityID(nil), bundle.Replay.Capabilities...)
	slices.Sort(expectedCapabilities)
	if !slices.Equal(expectedCapabilities, capabilities) {
		drift = append(drift, Drift{
			Kind: DriftRealization, Detail: "environment capabilities changed",
		})
	}
	reproduced := len(drift) == 0 && bundle.Result.Claim.Kind == current.Claim.Kind &&
		bundle.Result.Claim.Property == current.Claim.Property
	return Report{
		ExperimentDigest: digest, Baseline: bundle.Result.Claim, Current: current.Claim,
		Drift: drift, Result: current, Reproduced: reproduced,
	}, nil
}
