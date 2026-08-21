package mutationaudit

import (
	"context"
	"errors"
	"fmt"

	"go.temporal.io/server/tests/umpire3/campaign"
	"go.temporal.io/server/tests/umpire3/model-checkers/canonical"
	"go.temporal.io/server/tests/umpire3/model-checkers/native"
	"go.temporal.io/server/tests/umpire3/model-checkers/veil"
	"go.temporal.io/server/tests/umpire3/protocol"
)

type Request struct {
	Experiment            protocol.Experiment
	FiniteReplayCommand   []string
	TemporalReplayCommand []string
}

func Run(ctx context.Context, request Request) (Report, error) {
	campaignAudit, err := campaign.RunApprovedMutationAudit(ctx, request.Experiment)
	if err != nil {
		return Report{}, fmt.Errorf("run live mutation audit: %w", err)
	}
	view, found, err := protocol.DefaultFirstOrderView(
		protocol.TargetIDNexusCancellation, "stale-completion-guard-removed")
	if err != nil {
		return Report{}, err
	}
	if !found {
		return Report{}, errors.New("mutated Nexus first-order view is unavailable")
	}
	_, err = native.Produce(ctx, view, native.Options{
		Workers: 4, Replicas: 10,
		Limits: native.SearchLimits{
			MaxDepth: 16, MaxStates: 4096, MaxTransitions: 65536, MaxStateBytes: 4 << 20,
		},
	}, nil)
	var counterexample *native.CounterexampleError
	if !errors.As(err, &counterexample) {
		return Report{}, fmt.Errorf("native mutation search did not produce a counterexample: %w", err)
	}
	finiteInput := protocol.TraceReplayInput{
		FormatVersion: protocol.TraceReplayInputFormatVersion,
		Target:        view.Target, Property: view.Property, World: view.World,
		Variant: view.Variant, SemanticHash: view.SemanticHash,
		Actions: append([]protocol.ActionKind{}, counterexample.Actions...),
	}
	finiteReceipt, err := canonical.ReplayFinite(ctx, request.FiniteReplayCommand, finiteInput)
	if err != nil {
		return Report{}, err
	}
	nativeTrace, err := native.NormalizeCounterexample(view, counterexample, finiteReceipt)
	if err != nil {
		return Report{}, err
	}
	backendResult, err := veil.DefaultMutatedResult()
	if err != nil {
		return Report{}, err
	}
	veilTrace, err := protocol.SemanticTraceFromBackendResult(backendResult)
	if err != nil {
		return Report{}, err
	}

	temporalView, found, err := protocol.DefaultTemporalView("delivery-fairness-removed")
	if err != nil {
		return Report{}, err
	}
	if !found {
		return Report{}, errors.New("mutated Lean temporal view is unavailable")
	}
	temporalInput := protocol.TemporalLassoReplayInput{
		FormatVersion: protocol.TemporalLassoReplayInputFormatVersion,
		Target:        temporalView.Target, Property: temporalView.Property, World: temporalView.World,
		Variant: temporalView.Variant, SemanticHash: temporalView.SemanticHash,
		Lasso: protocol.TemporalLasso{
			States:  []string{"unavailable", "ready"},
			Actions: []protocol.ActionKind{protocol.ActionKindRecoverOwner, ""}, LoopStart: 1,
		},
	}
	temporalReceipt, err := canonical.ReplayTemporal(ctx, request.TemporalReplayCommand, temporalInput)
	if err != nil {
		return Report{}, err
	}
	temporalTrace, err := protocol.SemanticTraceFromTemporalLassoReplayReceipt(
		protocol.SemanticTraceProducerLeanTemporal, temporalReceipt)
	if err != nil {
		return Report{}, err
	}
	return seal(Report{
		CampaignSourceDigest: campaignAudit.SourceDigest,
		NativeTrace:          nativeTrace, VeilTrace: veilTrace, TemporalTrace: temporalTrace,
	})
}
