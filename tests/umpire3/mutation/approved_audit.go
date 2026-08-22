package mutation

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	umpire3execution "go.temporal.io/server/tests/umpire3/execution"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

const approvedMutationSeed int64 = 73

func RunApprovedMutationAudit(
	ctx context.Context,
	source protocolexperiment.Experiment,
) (MutationGateReport, error) {
	encoded, err := source.CanonicalJSON()
	if err != nil {
		return MutationGateReport{}, err
	}
	experiment, err := protocolexperiment.DecodeExperiment(bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit)
	if err != nil {
		return MutationGateReport{}, err
	}
	baseline := "baseline"
	mutationPath := ""
	for index := range experiment.Actions {
		if experiment.Actions[index].Kind != string(protocolcatalog.ActionKindRequestCancellation) {
			continue
		}
		experiment.Actions[index].Arguments = []protocolexperiment.NamedValue{{
			Name: "reason", Value: protocolexperiment.Value{Type: protocolexperiment.ValueString, Text: &baseline},
		}}
		mutationPath = fmt.Sprintf("actions[%d].arguments[reason]", index)
		break
	}
	if mutationPath == "" {
		return MutationGateReport{}, errors.New("approved mutation audit requires a cancellation reason argument")
	}
	seeded := "seeded-adapter-corruption"
	executor := func(_ context.Context, candidate protocolexperiment.Experiment) (umpire3execution.Result, error) {
		digest, digestErr := candidate.Digest()
		if digestErr != nil {
			return umpire3execution.Result{}, digestErr
		}
		result := umpire3execution.Result{
			FormatVersion: umpire3execution.ResultFormatVersion, ExperimentDigest: digest,
			Environment: umpire3execution.EnvironmentIdentity{
				Name: "local-in-process",
				Capabilities: []protocolcatalog.CapabilityID{
					protocolcatalog.CapabilityIDHistoryObservation, protocolcatalog.CapabilityIDNexus,
				},
			},
			Claim: umpire3execution.Claim{
				Kind: umpire3execution.ClaimConforming, Property: candidate.Property.Identifier,
			},
			Cleanup: umpire3execution.CleanupResult{Complete: true},
		}
		if containsTextArgument(candidate, "reason", seeded) {
			result.Claim.Kind = umpire3execution.ClaimViolating
			result.Claim.Checkpoint = "observe-cancellation-won"
			result.Observations = []umpire3execution.Observation{{
				CheckpointID: "seeded-cross-layer", Kind: "cancellation-won", Satisfied: true,
				Source: "history", SourceIdentity: "history", ClockDomain: "history-sequence",
				SourceSequence: 1, ObservedAtUnixNano: 1, Reference: "history/1",
				EntityIdentity: "operation", Lineage: []string{"namespace", "operation"},
			}}
			if traceErr := bindAcceptedSemanticTrace(candidate, &result); traceErr != nil {
				result.Claim.Kind = umpire3execution.ClaimInconclusive
				result.Claim.Reason = traceErr.Error()
				result.Observations = nil
			} else if evidenceErr := result.NormalizeEvidence(protocolexperiment.DefaultDecodeLimit); evidenceErr != nil {
				result.Claim.Kind = umpire3execution.ClaimInconclusive
				result.Claim.Reason = evidenceErr.Error()
				result.Observations = nil
			}
		}
		result.DeriveAssurance()
		return result, nil
	}
	mutationRequest := MutationRequest{
		Experiment: experiment, Seed: approvedMutationSeed, MaxCandidates: 32,
		Values:        []protocolexperiment.Value{{Type: protocolexperiment.ValueString, Text: &seeded}},
		TopologyKinds: []protocolcatalog.EntityKind{protocolcatalog.EntityKindCallback},
	}
	mutations, err := Mutate(mutationRequest)
	if err != nil {
		return MutationGateReport{}, err
	}
	targetCoverage := CoveragePoint{Kind: CoverageProtobuf, Identifier: mutationPath}
	var corpusCoverage []CoveragePoint
	for _, mutation := range mutations.Selected {
		coverage, coverageErr := modelCoverage(mutation.Experiment)
		if coverageErr != nil {
			return MutationGateReport{}, coverageErr
		}
		corpusCoverage = append(corpusCoverage, coverage...)
		mutationPoint, coverageErr := mutationCoverage(mutation.Kind, mutation.Path)
		if coverageErr != nil {
			return MutationGateReport{}, coverageErr
		}
		if mutationPoint != targetCoverage {
			corpusCoverage = append(corpusCoverage, mutationPoint)
		}
	}
	report, err := RunMutationGate(ctx, MutationGateRequest{
		Mutation: mutationRequest, MaxExecutions: 1,
		CorpusCoverage: normalizeCoverage(corpusCoverage),
		Approved: []ApprovedMutation{{
			Identifier: "adapter-response-corruption-v1", Layer: "temporal-adapter",
			Kind: MutationProtobufValue, Path: mutationPath,
		}},
		Executor: executor,
	})
	if err != nil {
		return MutationGateReport{}, err
	}
	return sealMutationGateReport(report, source)
}

func containsTextArgument(experiment protocolexperiment.Experiment, name, expected string) bool {
	for _, action := range experiment.Actions {
		for _, argument := range action.Arguments {
			if argument.Name == name && argument.Value.Text != nil && *argument.Value.Text == expected {
				return true
			}
		}
	}
	return false
}
