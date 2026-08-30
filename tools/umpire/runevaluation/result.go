package runevaluation

import (
	"errors"
	"fmt"

	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/internal/runtimeengine"
)

const (
	callerClosureImplementationLinkID          = "temporal.system.nexus.caller-closure.implementation-link"
	callerClosureImplementationLinkFingerprint = "sha256:96b55d0e5a782099f66479c6ced603c08c8046b565f89435b5b2a54848aed777"
	callerClosureSystemTargetID                = "temporal.system.nexus.caller-closure.target"
	callerClosureSystemTargetFingerprint       = "sha256:6729e790d336a96173ffd0ebe0b2b2d2406e6c5444596924f0c06c4ba9652bf8"
)

type evaluationFailure struct {
	kind  string
	phase string
	code  string
	cause error
}

func newEvaluationFailure(kind, phase, code string, cause error) *evaluationFailure {
	return &evaluationFailure{kind: kind, phase: phase, code: code, cause: cause}
}

func (failure *evaluationFailure) Error() string {
	if failure == nil {
		return ""
	}
	return fmt.Sprintf("run evaluation %s/%s: %s", failure.kind, failure.phase, failure.code)
}

func (failure *evaluationFailure) Kind() string {
	if failure == nil {
		return ""
	}
	return failure.kind
}

func (failure *evaluationFailure) Phase() string {
	if failure == nil {
		return ""
	}
	return failure.phase
}

func (failure *evaluationFailure) Code() string {
	if failure == nil {
		return ""
	}
	return failure.code
}

func (failure *evaluationFailure) Unwrap() error {
	if failure == nil {
		return nil
	}
	return failure.cause
}

func constructEvaluation(
	execution artifact.ExecutionSet,
	request checkerRequest,
	response checkerResponse,
) (artifactv2.Evidence, artifactv2.Result, error) {
	evidence, err := sealEvaluationEvidence(request, response)
	if err != nil {
		return artifactv2.Evidence{}, artifactv2.Result{}, newEvaluationFailure(
			"output-invariant", "construction", "umpire.run-evaluation.evidence.invalid", err,
		)
	}
	result, err := sealEvaluationResult(execution, request, response, evidence)
	if err != nil {
		return artifactv2.Evidence{}, artifactv2.Result{}, newEvaluationFailure(
			"output-invariant", "construction", "umpire.run-evaluation.result.invalid", err,
		)
	}
	return evidence, result, nil
}

func sealEvaluationEvidence(
	request checkerRequest,
	response checkerResponse,
) (artifactv2.Evidence, error) {
	evidence := artifactv2.Evidence{
		FormatVersion:               artifactv2.EvidenceFormat,
		RunIdentity:                 request.RunIdentity,
		BehaviorFingerprint:         checkerBehaviorFingerprint,
		Experiment:                  request.Experiment,
		RuntimeConfiguration:        request.RuntimeConfiguration,
		Run:                         request.Run,
		RawEvidence:                 request.RawEvidence,
		ObservationProgram:          artifactDefinitionReference(request.ObservationProgram),
		Mapping:                     artifactDefinitionReference(request.Mapping),
		ObservationEvaluationStatus: response.ObservationEvaluationStatus,
		EvidenceBackedModelTrace:    response.EvidenceBackedModelTrace,
		EvidenceLinks:               response.EvidenceLinks,
		Dispositions:                response.Dispositions,
		Diagnostics:                 response.Diagnostics,
		KnownGaps:                   response.ObservationKnownGaps,
		Provenance:                  evaluationProvenance(),
	}
	return artifactv2.SealEvidence(evidence)
}

func sealEvaluationResult(
	execution artifact.ExecutionSet,
	request checkerRequest,
	response checkerResponse,
	evidence artifactv2.Evidence,
) (artifactv2.Result, error) {
	run := execution.ExperimentRun()
	implementationStatus := "not-evaluated"
	if response.ObservationEvaluationStatus == "accepted" {
		implementationStatus = "applied"
	}
	result := artifactv2.Result{
		FormatVersion:               artifactv2.ResultFormat,
		RunIdentity:                 request.RunIdentity,
		BehaviorFingerprint:         checkerBehaviorFingerprint,
		Experiment:                  request.Experiment,
		RuntimeConfiguration:        request.RuntimeConfiguration,
		Run:                         request.Run,
		RawEvidence:                 request.RawEvidence,
		Evidence:                    artifactv2.EvidenceArtifactBinding(evidence),
		OperationalStatus:           runtimeengine.OperationalStatus(run),
		ObservationEvaluationStatus: response.ObservationEvaluationStatus,
		ImplementationLink:          callerClosureImplementationLink(),
		ImplementationLinkStatus:    implementationStatus,
		PropertyVerdicts:            response.PropertyVerdicts,
		QuerySummary:                response.QuerySummary,
		SemanticStatus:              response.SemanticStatus,
		Limits:                      callerClosureStagedLimits(),
		KnownGaps:                   response.ResultKnownGaps,
		CleanupStatus:               run.Cleanup.Status,
		EvaluationOutcomeChecksum:   response.EvaluationOutcomeChecksum,
		Provenance:                  evaluationProvenance(),
	}
	if result.EvaluationOutcomeChecksum != nil {
		expected, err := artifactv2.ExpectedEvaluationOutcomeChecksum(
			result, evidence, execution.Experiment(),
		)
		if err != nil {
			return artifactv2.Result{}, err
		}
		if *result.EvaluationOutcomeChecksum != expected {
			return artifactv2.Result{}, errors.New("evaluation outcome checksum drifted")
		}
	}
	return artifactv2.SealResult(result)
}

func callerClosureImplementationLink() artifactv2.ImplementationLinkRecord {
	return artifactv2.ImplementationLinkRecord{
		DefinitionID:        callerClosureImplementationLinkID,
		BehaviorFingerprint: callerClosureImplementationLinkFingerprint,
		SourceTarget: artifactv2.ImplementationTargetReference{
			DefinitionID: callerClosureSystemTargetID, Kind: "target",
			BehaviorFingerprint: callerClosureSystemTargetFingerprint,
		},
		DestinationTarget: artifactv2.ImplementationTargetReference{
			DefinitionID: callerClosureTargetID, Kind: "target",
			BehaviorFingerprint: callerClosureTargetFingerprint,
		},
	}
}

func callerClosureStagedLimits() []artifactv2.StagedLimit {
	return []artifactv2.StagedLimit{
		{Stage: "observation-evaluation", Limit: artifactv2.Limit{
			Value: artifactv2.NaturalFromUint64(4096), Unit: "evidence-records",
		}},
		{Stage: "query", Limit: artifactv2.Limit{
			Value: artifactv2.NaturalFromUint64(8), Unit: "candidate-evaluations",
		}},
	}
}

func evaluationProvenance() artifactv2.Provenance {
	one := artifactv2.NaturalFromUint64(1)
	return artifactv2.Provenance{
		SourceDefinitionIDs: []string{"temporal.tool.run-evaluation"},
		SourceLocations: []artifactv2.SourceLocation{{
			Path: "Temporal/Tool/RunEvaluation.lean", Line: one, Column: one, Provenance: "lean-model",
		}},
	}
}
