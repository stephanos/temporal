package mutation

import (
	"context"
	"errors"
	"fmt"

	"go.temporal.io/server/tools/umpire3/checker/finite"
	"go.temporal.io/server/tools/umpire3/checker/leanreplay"
	checkertrace "go.temporal.io/server/tools/umpire3/checker/trace"
	"go.temporal.io/server/tools/umpire3/checker/veil"
	"go.temporal.io/server/tools/umpire3/mutation"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
)

type Request struct {
	Experiment            protocolexperiment.Experiment
	FiniteReplayCommand   []string
	TemporalReplayCommand []string
}

func Run(ctx context.Context, request Request) (Report, error) {
	campaignAudit, err := mutation.RunApprovedMutationAudit(ctx, request.Experiment)
	if err != nil {
		return Report{}, fmt.Errorf("run live mutation audit: %w", err)
	}
	view, found, err := finite.DefaultFirstOrderView(
		protocolcatalog.TargetIDNexusCancellation, "stale-completion-guard-removed")
	if err != nil {
		return Report{}, err
	}
	if !found {
		return Report{}, errors.New("mutated Nexus first-order view is unavailable")
	}
	_, err = finite.Produce(ctx, view, finite.Options{
		Workers: 4, Replicas: 10,
		Limits: finite.SearchLimits{
			MaxDepth: 16, MaxStates: 4096, MaxTransitions: 65536, MaxStateBytes: 4 << 20,
		},
	}, nil)
	var counterexample *finite.CounterexampleError
	if !errors.As(err, &counterexample) {
		return Report{}, fmt.Errorf("native mutation search did not produce a counterexample: %w", err)
	}
	finiteInput := protocolchecker.TraceReplayInput{
		FormatVersion: protocolchecker.TraceReplayInputFormatVersion,
		Target:        view.Target, Property: view.Property, World: view.World,
		Variant: view.Variant, SemanticHash: view.SemanticHash,
		Actions: append([]protocolcatalog.ActionKind{}, counterexample.Actions...),
	}
	finiteReceipt, err := leanreplay.ReplayFinite(ctx, request.FiniteReplayCommand, finiteInput)
	if err != nil {
		return Report{}, err
	}
	nativeTrace, err := finite.NormalizeCounterexample(view, counterexample, finiteReceipt)
	if err != nil {
		return Report{}, err
	}
	backendResult, err := veil.DefaultMutatedResult()
	if err != nil {
		return Report{}, err
	}
	veilTrace, err := checkertrace.FromBackendResult(backendResult)
	if err != nil {
		return Report{}, err
	}

	temporalView, found, err := protocolchecker.DefaultTemporalView("delivery-fairness-removed")
	if err != nil {
		return Report{}, err
	}
	if !found {
		return Report{}, errors.New("mutated Lean temporal view is unavailable")
	}
	temporalInput := protocolchecker.TemporalLassoReplayInput{
		FormatVersion: protocolchecker.TemporalLassoReplayInputFormatVersion,
		Target:        temporalView.Target, Property: temporalView.Property, World: temporalView.World,
		Variant: temporalView.Variant, SemanticHash: temporalView.SemanticHash,
		Lasso: protocolchecker.TemporalLasso{
			States:  []string{"unavailable", "ready"},
			Actions: []protocolcatalog.ActionKind{protocolcatalog.ActionKindRecoverOwner, ""}, LoopStart: 1,
		},
	}
	temporalReceipt, err := leanreplay.ReplayTemporal(ctx, request.TemporalReplayCommand, temporalInput)
	if err != nil {
		return Report{}, err
	}
	temporalTrace, err := checkertrace.FromTemporalLassoReplayReceipt(
		protocolchecker.SemanticTraceProducerLeanTemporal, temporalReceipt)
	if err != nil {
		return Report{}, err
	}
	return seal(Report{
		CampaignSourceDigest: campaignAudit.SourceDigest,
		NativeTrace:          nativeTrace, VeilTrace: veilTrace, TemporalTrace: temporalTrace,
	})
}
