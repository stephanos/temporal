package veil

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"go.temporal.io/server/tests/umpire3/process"
	"go.temporal.io/server/tests/umpire3/protocol"
)

const backendMemoryLimit = 1 << 30

var backendProcessLimits = process.Limits{
	CPUSeconds:  int(canonicalReplayTimeout.Seconds()),
	MemoryBytes: backendMemoryLimit,
}

func canonicalExecutionLimits() protocol.BackendExecutionLimits {
	return protocol.BackendExecutionLimits{
		TimeoutMillis: canonicalReplayTimeout.Milliseconds(),
		CPUSeconds:    backendProcessLimits.CPUSeconds, MemoryBytes: backendProcessLimits.MemoryBytes,
		MaxOutputBytes: protocol.DefaultDecodeLimit,
	}
}

func CheckConcrete(
	ctx context.Context,
	command []string,
	replayCommand []string,
	view protocol.FirstOrderView,
	generated GeneratedModule,
) (protocol.BackendResult, error) {
	if len(command) == 0 {
		return protocol.BackendResult{}, errors.New("veil concrete checker command is required")
	}
	checked, err := process.Run(ctx, process.Request{
		Command: command, Timeout: canonicalReplayTimeout,
		MaxOutputBytes: protocol.DefaultDecodeLimit, Limits: backendProcessLimits,
	})
	if err != nil {
		return protocol.BackendResult{}, fmt.Errorf("run Veil concrete checker: %w", err)
	}
	replayInput, err := ConcreteReplayInput(view, generated, bytes.NewReader(checked.Output),
		protocol.DefaultDecodeLimit)
	if err != nil {
		return protocol.BackendResult{}, err
	}
	var receipt *protocol.TraceReplayReceipt
	if replayInput != nil {
		if len(replayCommand) == 0 {
			return protocol.BackendResult{}, errors.New("canonical replay command is required for a Veil counterexample")
		}
		accepted, err := Replay(ctx, replayCommand, *replayInput)
		if err != nil {
			return protocol.BackendResult{}, err
		}
		receipt = &accepted
	}
	return NormalizeConcreteOutput(view, generated, bytes.NewReader(checked.Output),
		protocol.DefaultDecodeLimit, canonicalExecutionLimits(), receipt)
}
