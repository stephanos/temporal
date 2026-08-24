package veil

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/umpire3/internal/subprocess"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

const backendMemoryLimit = 1 << 30

var backendProcessLimits = subprocess.Limits{
	CPUSeconds:  int(canonicalReplayTimeout.Seconds()),
	MemoryBytes: backendMemoryLimit,
}

func canonicalExecutionLimits() protocolchecker.BackendExecutionLimits {
	return protocolchecker.BackendExecutionLimits{
		TimeoutMillis: canonicalReplayTimeout.Milliseconds(),
		CPUSeconds:    backendProcessLimits.CPUSeconds, MemoryBytes: backendProcessLimits.MemoryBytes,
		MaxOutputBytes: protocolexperiment.DefaultDecodeLimit,
	}
}

func CheckConcrete(
	ctx context.Context,
	command []string,
	replayCommand []string,
	view protocolchecker.FirstOrderView,
	binding BindingArtifact,
) (protocolchecker.BackendResult, error) {
	if len(command) == 0 {
		return protocolchecker.BackendResult{}, errors.New("veil concrete checker command is required")
	}
	checked, err := subprocess.Run(ctx, subprocess.Request{
		Command: append(append([]string(nil), command...), view.SemanticHash), Timeout: canonicalReplayTimeout,
		MaxOutputBytes: protocolexperiment.DefaultDecodeLimit, Limits: backendProcessLimits,
	})
	if err != nil {
		return protocolchecker.BackendResult{}, fmt.Errorf("run Veil concrete checker: %w", err)
	}
	replayInput, err := ConcreteReplayInput(view, binding, bytes.NewReader(checked.Output),
		protocolexperiment.DefaultDecodeLimit)
	if err != nil {
		return protocolchecker.BackendResult{}, err
	}
	var receipt *protocolchecker.TraceReplayReceipt
	if replayInput != nil {
		if len(replayCommand) == 0 {
			return protocolchecker.BackendResult{}, errors.New("canonical replay command is required for a Veil counterexample")
		}
		accepted, err := Replay(ctx, replayCommand, *replayInput)
		if err != nil {
			return protocolchecker.BackendResult{}, err
		}
		receipt = &accepted
	}
	return NormalizeConcreteOutput(view, binding, bytes.NewReader(checked.Output),
		protocolexperiment.DefaultDecodeLimit, canonicalExecutionLimits(), receipt)
}
