package campaign

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	umpire3runtime "go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/protocol"
)

const approvedMutationSeed int64 = 73

func RunApprovedMutationAudit(
	ctx context.Context,
	source protocol.Experiment,
) (MutationGateReport, error) {
	encoded, err := source.CanonicalJSON()
	if err != nil {
		return MutationGateReport{}, err
	}
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
	if err != nil {
		return MutationGateReport{}, err
	}
	baseline := "baseline"
	mutationPath := ""
	for index := range experiment.Actions {
		if experiment.Actions[index].Kind != string(protocol.ActionKindRequestCancellation) {
			continue
		}
		experiment.Actions[index].Arguments = []protocol.NamedValue{{
			Name: "reason", Value: protocol.Value{Type: protocol.ValueString, Text: &baseline},
		}}
		mutationPath = fmt.Sprintf("actions[%d].arguments[reason]", index)
		break
	}
	if mutationPath == "" {
		return MutationGateReport{}, errors.New("approved mutation audit requires a cancellation reason argument")
	}
	seeded := "seeded-adapter-corruption"
	executor := func(_ context.Context, candidate protocol.Experiment) (umpire3runtime.Result, error) {
		digest, digestErr := candidate.Digest()
		if digestErr != nil {
			return umpire3runtime.Result{}, digestErr
		}
		result := umpire3runtime.Result{
			FormatVersion: umpire3runtime.ResultFormatVersion, ExperimentDigest: digest,
			Environment: umpire3runtime.EnvironmentIdentity{
				Name: "local-in-process",
				Capabilities: []protocol.CapabilityID{
					protocol.CapabilityIDHistoryObservation, protocol.CapabilityIDNexus,
				},
			},
			Claim: umpire3runtime.Claim{
				Kind: umpire3runtime.ClaimConforming, Property: candidate.Property.Identifier,
			},
			Cleanup: umpire3runtime.CleanupResult{Complete: true},
		}
		if containsTextArgument(candidate, "reason", seeded) {
			result.Claim.Kind = umpire3runtime.ClaimViolating
			result.Claim.Checkpoint = "observe-cancellation-won"
			result.Observations = []umpire3runtime.Observation{{
				CheckpointID: "seeded-cross-layer", Kind: "cancellation-won", Satisfied: true,
				Source: "history", SourceIdentity: "history", ClockDomain: "history-sequence",
				SourceSequence: 1, ObservedAtUnixNano: 1, Reference: "history/1",
				EntityIdentity: "operation", Lineage: []string{"namespace", "operation"},
			}}
			if traceErr := bindAcceptedSemanticTrace(candidate, &result); traceErr != nil {
				result.Claim.Kind = umpire3runtime.ClaimInconclusive
				result.Claim.Reason = traceErr.Error()
				result.Observations = nil
			} else if evidenceErr := result.NormalizeEvidence(protocol.DefaultDecodeLimit); evidenceErr != nil {
				result.Claim.Kind = umpire3runtime.ClaimInconclusive
				result.Claim.Reason = evidenceErr.Error()
				result.Observations = nil
			}
		}
		result.DeriveAssurance()
		return result, nil
	}
	mutationRequest := MutationRequest{
		Experiment: experiment, Seed: approvedMutationSeed, MaxCandidates: 32,
		Values:        []protocol.Value{{Type: protocol.ValueString, Text: &seeded}},
		TopologyKinds: []protocol.EntityKind{protocol.EntityKindCallback},
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

func containsTextArgument(experiment protocol.Experiment, name, expected string) bool {
	for _, action := range experiment.Actions {
		for _, argument := range action.Arguments {
			if argument.Name == name && argument.Value.Text != nil && *argument.Value.Text == expected {
				return true
			}
		}
	}
	return false
}
