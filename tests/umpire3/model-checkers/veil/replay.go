package veil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"go.temporal.io/server/tests/umpire3/process"
	"go.temporal.io/server/tests/umpire3/protocol"
)

const canonicalReplayTimeout = 30 * time.Second

func Replay(
	ctx context.Context,
	command []string,
	input protocol.TraceReplayInput,
) (protocol.TraceReplayReceipt, error) {
	if len(command) == 0 {
		return protocol.TraceReplayReceipt{}, errors.New("canonical Lean replay command is required")
	}
	if err := input.Validate(); err != nil {
		return protocol.TraceReplayReceipt{}, fmt.Errorf("validate trace replay input: %w", err)
	}
	digest, err := input.Digest()
	if err != nil {
		return protocol.TraceReplayReceipt{}, err
	}
	arguments := []string{
		digest,
		string(input.Target),
		string(input.Property),
		input.World,
		input.Variant,
		input.SemanticHash,
	}
	for _, action := range input.Actions {
		arguments = append(arguments, string(action))
	}
	result, err := process.Run(ctx, process.Request{
		Command:        append(append([]string(nil), command...), arguments...),
		Timeout:        canonicalReplayTimeout,
		MaxOutputBytes: protocol.DefaultDecodeLimit,
		Limits:         backendProcessLimits,
	})
	if err != nil {
		return protocol.TraceReplayReceipt{}, fmt.Errorf("run canonical Lean replay: %w", err)
	}
	receipt, err := protocol.DecodeTraceReplayReceipt(bytes.NewReader(result.Output),
		protocol.DefaultDecodeLimit)
	if err != nil {
		return protocol.TraceReplayReceipt{}, err
	}
	if receipt.TraceDigest != digest {
		return protocol.TraceReplayReceipt{}, errors.New("canonical Lean replay receipt has an unexpected trace digest")
	}
	if receipt.Target != input.Target || receipt.Property != input.Property ||
		receipt.World != input.World || receipt.Variant != input.Variant ||
		receipt.SemanticHash != input.SemanticHash || !slices.Equal(receipt.Actions, input.Actions) {
		return protocol.TraceReplayReceipt{}, errors.New("canonical Lean replay receipt does not match the checked trace")
	}
	return receipt, nil
}
