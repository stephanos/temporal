package tla

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"go.temporal.io/server/tests/umpire3/process"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func ReplayLasso(
	ctx context.Context,
	command []string,
	input protocol.TemporalLassoReplayInput,
) (protocol.TemporalLassoReplayReceipt, error) {
	if len(command) == 0 {
		return protocol.TemporalLassoReplayReceipt{}, errors.New("canonical Lean temporal replay command is required")
	}
	if err := input.Validate(); err != nil {
		return protocol.TemporalLassoReplayReceipt{}, fmt.Errorf("validate temporal lasso replay input: %w", err)
	}
	digest, err := input.Digest()
	if err != nil {
		return protocol.TemporalLassoReplayReceipt{}, err
	}
	arguments := []string{
		digest,
		string(input.Target),
		string(input.Property),
		input.World,
		input.Variant,
		input.SemanticHash,
		strconv.Itoa(input.Lasso.LoopStart),
		strconv.Itoa(len(input.Lasso.States)),
	}
	arguments = append(arguments, input.Lasso.States...)
	for _, action := range input.Lasso.Actions {
		arguments = append(arguments, string(action))
	}
	result, err := process.Run(ctx, process.Request{
		Command:        append(append([]string(nil), command...), arguments...),
		Timeout:        30 * time.Second,
		MaxOutputBytes: protocol.DefaultDecodeLimit,
		Limits: process.Limits{
			CPUSeconds:  30,
			MemoryBytes: 1 << 30,
		},
	})
	if err != nil {
		return protocol.TemporalLassoReplayReceipt{}, fmt.Errorf("run canonical Lean temporal replay: %w", err)
	}
	receipt, err := protocol.DecodeTemporalLassoReplayReceipt(result.Output)
	if err != nil {
		return protocol.TemporalLassoReplayReceipt{}, err
	}
	if receipt.LassoDigest != digest || receipt.Target != input.Target || receipt.Property != input.Property ||
		receipt.World != input.World || receipt.Variant != input.Variant ||
		receipt.SemanticHash != input.SemanticHash || !slices.Equal(receipt.Lasso.States, input.Lasso.States) ||
		!slices.Equal(receipt.Lasso.Actions, input.Lasso.Actions) || receipt.Lasso.LoopStart != input.Lasso.LoopStart {
		return protocol.TemporalLassoReplayReceipt{}, errors.New("canonical Lean receipt does not match the checked temporal lasso")
	}
	return receipt, nil
}

func AttachReplay(result Result, receipt protocol.TemporalLassoReplayReceipt) (Result, error) {
	if result.Lasso == nil || result.ResultClass != protocol.ResultClassLassoWitness {
		return Result{}, errors.New("only a temporal lasso witness can receive a replay receipt")
	}
	if err := receipt.Validate(); err != nil {
		return Result{}, err
	}
	if receipt.Target != result.Target || receipt.Property != result.Property || receipt.World != result.World ||
		receipt.Variant != result.Variant || receipt.SemanticHash != result.SemanticHash ||
		!slices.Equal(receipt.Lasso.States, result.Lasso.States) ||
		!slices.Equal(receipt.Lasso.Actions, result.Lasso.Actions) || receipt.Lasso.LoopStart != result.Lasso.LoopStart {
		return Result{}, errors.New("temporal replay receipt does not match backend lasso identity")
	}
	result.TrustBadge = protocol.TrustBadgeCheckedCertificate
	result.Axioms = append([]string(nil), receipt.Axioms...)
	result.Replay = &receipt
	if err := result.seal(); err != nil {
		return Result{}, err
	}
	return result, nil
}
