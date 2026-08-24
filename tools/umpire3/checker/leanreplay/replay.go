package leanreplay

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"go.temporal.io/server/tools/umpire3/internal/subprocess"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

const replayTimeout = 30 * time.Second

var replayLimits = subprocess.Limits{CPUSeconds: 30, MemoryBytes: 1 << 30}

func ReplayFinite(
	ctx context.Context,
	command []string,
	input protocolchecker.TraceReplayInput,
) (protocolchecker.TraceReplayReceipt, error) {
	if len(command) == 0 {
		return protocolchecker.TraceReplayReceipt{}, errors.New("canonical Lean replay command is required")
	}
	if err := input.Validate(); err != nil {
		return protocolchecker.TraceReplayReceipt{}, fmt.Errorf("validate trace replay input: %w", err)
	}
	digest, err := input.Digest()
	if err != nil {
		return protocolchecker.TraceReplayReceipt{}, err
	}
	arguments := []string{
		digest, string(input.Target), string(input.Property), input.World, input.Variant, input.SemanticHash,
	}
	for _, action := range input.Actions {
		arguments = append(arguments, string(action))
	}
	result, err := run(ctx, command, arguments)
	if err != nil {
		return protocolchecker.TraceReplayReceipt{}, fmt.Errorf("run canonical Lean replay: %w", err)
	}
	receipt, err := protocolchecker.DecodeTraceReplayReceipt(bytes.NewReader(result.Output), protocolexperiment.DefaultDecodeLimit)
	if err != nil {
		return protocolchecker.TraceReplayReceipt{}, err
	}
	if receipt.TraceDigest != digest || receipt.Target != input.Target || receipt.Property != input.Property ||
		receipt.World != input.World || receipt.Variant != input.Variant ||
		receipt.SemanticHash != input.SemanticHash || !slices.Equal(receipt.Actions, input.Actions) {
		return protocolchecker.TraceReplayReceipt{}, errors.New("canonical Lean replay receipt does not match the checked trace")
	}
	return receipt, nil
}

func ReplayTemporal(
	ctx context.Context,
	command []string,
	input protocolchecker.TemporalLassoReplayInput,
) (protocolchecker.TemporalLassoReplayReceipt, error) {
	if len(command) == 0 {
		return protocolchecker.TemporalLassoReplayReceipt{},
			errors.New("canonical Lean temporal replay command is required")
	}
	if err := input.Validate(); err != nil {
		return protocolchecker.TemporalLassoReplayReceipt{},
			fmt.Errorf("validate temporal lasso replay input: %w", err)
	}
	digest, err := input.Digest()
	if err != nil {
		return protocolchecker.TemporalLassoReplayReceipt{}, err
	}
	arguments := []string{
		digest, string(input.Target), string(input.Property), input.World, input.Variant,
		input.SemanticHash, strconv.Itoa(input.Lasso.LoopStart), strconv.Itoa(len(input.Lasso.States)),
	}
	arguments = append(arguments, input.Lasso.States...)
	for _, action := range input.Lasso.Actions {
		arguments = append(arguments, string(action))
	}
	result, err := run(ctx, command, arguments)
	if err != nil {
		return protocolchecker.TemporalLassoReplayReceipt{}, fmt.Errorf("run canonical Lean temporal replay: %w", err)
	}
	receipt, err := protocolchecker.DecodeTemporalLassoReplayReceipt(result.Output)
	if err != nil {
		return protocolchecker.TemporalLassoReplayReceipt{}, err
	}
	if receipt.LassoDigest != digest || receipt.Target != input.Target || receipt.Property != input.Property ||
		receipt.World != input.World || receipt.Variant != input.Variant ||
		receipt.SemanticHash != input.SemanticHash || !slices.Equal(receipt.Lasso.States, input.Lasso.States) ||
		!slices.Equal(receipt.Lasso.Actions, input.Lasso.Actions) ||
		receipt.Lasso.LoopStart != input.Lasso.LoopStart {
		return protocolchecker.TemporalLassoReplayReceipt{},
			errors.New("canonical Lean receipt does not match the checked temporal lasso")
	}
	return receipt, nil
}

func run(ctx context.Context, command, arguments []string) (subprocess.Result, error) {
	return subprocess.Run(ctx, subprocess.Request{
		Command: append(append([]string(nil), command...), arguments...),
		Timeout: replayTimeout, MaxOutputBytes: protocolexperiment.DefaultDecodeLimit, Limits: replayLimits,
	})
}
