// Package executor owns resident admission, execution, portable evaluation, and reuse safety.
package executor

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/evaluationcontract"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/portableevaluation"
	"go.temporal.io/server/tools/umpire/runner"
	"go.temporal.io/server/tools/umpire/temporal/local"
)

type executorState uint32

const (
	stateIdle executorState = iota
	stateActive
	statePoisoned
)

type executeRun func(
	context.Context,
	artifact.AdmittedSet,
	runner.InputBinding,
	string,
	runner.Adapter,
) (runOutcome, error)

type evaluate func(context.Context, portableevaluation.Request) *umpirespb.EvaluationResult

type runOutcome struct {
	rawEvidence       artifactv2.RawEvidence
	runBinding        artifactv2.ArtifactBinding
	sourceClosures    []artifactv2.SourceClosure
	operationalStatus umpirespb.OperationalStatus
	cleanupStatus     umpirespb.CleanupStatus
	reusable          bool
}

// Executor admits one request at a time and permanently rejects reuse after uncertain cleanup.
type Executor struct {
	state           atomic.Uint32
	adapter         runner.Adapter
	run             executeRun
	evaluate        evaluate
	nextRunIdentity func() string
}

// New returns one single-flight resident executor over the supplied runner adapter.
func New(adapter runner.Adapter) *Executor {
	var sequence atomic.Uint64
	prefix := "umpire.executor.run." + uuid.NewString()
	return newExecutor(
		adapter,
		executeWithRunner,
		portableevaluation.Evaluate,
		func() string {
			return prefix + "." + strconv.FormatUint(sequence.Add(1), 10)
		},
	)
}

func newExecutor(
	adapter runner.Adapter,
	run executeRun,
	evaluate evaluate,
	nextRunIdentity func() string,
) *Executor {
	return &Executor{
		adapter: adapter, run: run, evaluate: evaluate, nextRunIdentity: nextRunIdentity,
	}
}

// Execute admits, runs, closes, evaluates, and cleans up one exact contract request.
func (e *Executor) Execute(
	ctx context.Context,
	request *umpirespb.ExecuteRequest,
) (*umpirespb.ExecuteResponse, error) {
	admitted, rejection := e.admit()
	if !admitted {
		return failedResponse(nil, "", rejection), nil
	}
	reusable := false
	defer func() { e.finish(reusable) }()

	if ctx == nil {
		reusable = true
		return failedResponse(nil, "", umpirespb.TOOLING_STATUS_INTERNAL_ERROR), nil
	}
	if ctx.Err() != nil {
		reusable = true
		return failedResponse(nil, "", umpirespb.TOOLING_STATUS_CANCELED), nil
	}
	contract, input, failure := admitRequest(request)
	if failure != umpirespb.TOOLING_STATUS_SUCCEEDED {
		reusable = true
		return failedResponse(contract, "", failure), nil
	}

	executionContext, cancel := context.WithTimeout(
		ctx,
		time.Duration(contract.GetLimits().GetMaxTotalDurationMilliseconds())*time.Millisecond,
	)
	defer cancel()
	runIdentity := e.nextRunIdentity()
	outcome, err := e.run(executionContext, input, runnerBinding(input), runIdentity, e.adapter)
	reusable = outcome.reusable
	if err != nil {
		status := umpirespb.TOOLING_STATUS_INTERNAL_ERROR
		if errors.Is(executionContext.Err(), context.Canceled) ||
			errors.Is(executionContext.Err(), context.DeadlineExceeded) ||
			errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			status = umpirespb.TOOLING_STATUS_CANCELED
		}
		return failedResponse(contract, runIdentity, status), nil
	}
	if outcome.rawEvidence.RunIdentity != runIdentity {
		reusable = false
		return failedResponse(contract, runIdentity, umpirespb.TOOLING_STATUS_INTERNAL_ERROR), nil
	}

	result := e.evaluate(executionContext, portableevaluation.Request{
		Contract:            contract,
		RawEvidence:         outcome.rawEvidence,
		ExpectedRunIdentity: runIdentity,
		ExpectedRun:         outcome.runBinding,
		ExpectedClosures:    outcome.sourceClosures,
		OperationalStatus:   outcome.operationalStatus,
		CleanupStatus:       outcome.cleanupStatus,
	})
	if result == nil {
		reusable = false
		return failedResponse(contract, runIdentity, umpirespb.TOOLING_STATUS_INTERNAL_ERROR), nil
	}
	return &umpirespb.ExecuteResponse{Result: result}, nil
}

func (e *Executor) admit() (bool, umpirespb.ToolingStatus) {
	if e == nil {
		return false, umpirespb.TOOLING_STATUS_INTERNAL_ERROR
	}
	for {
		switch state := executorState(e.state.Load()); state {
		case stateIdle:
			if e.state.CompareAndSwap(uint32(stateIdle), uint32(stateActive)) {
				return true, umpirespb.TOOLING_STATUS_SUCCEEDED
			}
		case stateActive:
			return false, umpirespb.TOOLING_STATUS_BUSY
		case statePoisoned:
			return false, umpirespb.TOOLING_STATUS_POISONED
		default:
			return false, umpirespb.TOOLING_STATUS_INTERNAL_ERROR
		}
	}
}

func (e *Executor) finish(reusable bool) {
	if reusable {
		if !e.state.CompareAndSwap(uint32(stateActive), uint32(stateIdle)) {
			e.state.Store(uint32(statePoisoned))
		}
		return
	}
	e.state.Store(uint32(statePoisoned))
}

func admitRequest(
	request *umpirespb.ExecuteRequest,
) (*umpirespb.EvaluationContract, artifact.AdmittedSet, umpirespb.ToolingStatus) {
	if request == nil {
		return nil, artifact.AdmittedSet{}, umpirespb.TOOLING_STATUS_INVALID_INPUT
	}
	contract, err := evaluationcontract.Admit(request.GetEvaluationContract())
	if err != nil {
		return nil, artifact.AdmittedSet{}, umpirespb.TOOLING_STATUS_INVALID_CONTRACT
	}
	if request.GetInput() == nil {
		return contract, artifact.AdmittedSet{}, umpirespb.TOOLING_STATUS_INVALID_INPUT
	}
	input, err := artifact.AdmitSet([]artifact.SetMember{
		{Path: "artifacts/experiment.json", Encoded: request.GetInput().GetExperiment()},
		{Path: "artifacts/runtime-configuration.json", Encoded: request.GetInput().GetRuntimeConfig()},
	})
	if err != nil || !contractMatchesInput(contract, input) {
		return contract, artifact.AdmittedSet{}, umpirespb.TOOLING_STATUS_INVALID_INPUT
	}
	return contract, input, umpirespb.TOOLING_STATUS_SUCCEEDED
}

func contractMatchesInput(
	contract *umpirespb.EvaluationContract,
	input artifact.AdmittedSet,
) bool {
	executable, ok := input.Executable()
	if !ok {
		return false
	}
	experiment, err := artifactv2.ExperimentArtifactBinding(executable.Experiment())
	if err != nil {
		return false
	}
	configuration := artifactv2.RuntimeConfigurationArtifactBinding(executable.RuntimeConfiguration())
	return protoBindingMatches(contract.GetExperiment(), experiment) &&
		protoBindingMatches(contract.GetRuntimeConfig(), configuration)
}

func protoBindingMatches(
	expected *umpirespb.ArtifactBinding,
	actual artifactv2.ArtifactBinding,
) bool {
	return expected != nil &&
		expected.GetFormatVersion() == actual.FormatVersion &&
		expected.GetArtifactChecksum() == actual.ArtifactChecksum &&
		expected.GetBehaviorFingerprint() == actual.BehaviorFingerprint &&
		expected.GetProvenanceChecksum() == actual.ProvenanceChecksum
}

func runnerBinding(input artifact.AdmittedSet) runner.InputBinding {
	executable, _ := input.Executable()
	experiment := executable.Experiment()
	configuration := executable.RuntimeConfiguration()
	return runner.InputBinding{
		ArtifactSetIdentity:                      input.Identity(),
		ArtifactSetChecksum:                      input.Checksum(),
		ManifestSHA256:                           input.ManifestSHA256(),
		ExperimentArtifactChecksum:               experiment.ArtifactChecksum,
		ExperimentBehaviorFingerprint:            experiment.QueryBehaviorFingerprint,
		RuntimeConfigurationArtifactChecksum:     configuration.ArtifactChecksum,
		RuntimeConfigurationBehaviorFingerprint:  configuration.BehaviorFingerprint,
		AuthorityRequiredCapabilityDefinitionIDs: local.RequiredCapabilityDefinitionIDs(),
	}
}

func executeWithRunner(
	ctx context.Context,
	input artifact.AdmittedSet,
	binding runner.InputBinding,
	runIdentity string,
	adapter runner.Adapter,
) (runOutcome, error) {
	output, err := runner.Run(ctx, input, binding, runIdentity, adapter)
	if err != nil {
		started := executionOccurred(err)
		return runOutcome{reusable: !started}, err
	}
	run := output.ExperimentRun()
	if run.RunIdentity != runIdentity {
		return runOutcome{reusable: false}, errors.New("runner returned a crossed run identity")
	}
	cleanup := cleanupStatus(run.Cleanup.Status)
	return runOutcome{
		rawEvidence:       output.RawEvidence(),
		runBinding:        artifactv2.ExperimentRunArtifactBinding(run),
		sourceClosures:    run.SourceClosures,
		operationalStatus: operationalStatus(run.OperationalStatus),
		cleanupStatus:     cleanup,
		reusable:          cleanup == umpirespb.CLEANUP_STATUS_COMPLETE,
	}, nil
}

func executionOccurred(err error) bool {
	var classified interface {
		ExecutionOccurred() bool
	}
	return errors.As(err, &classified) && classified.ExecutionOccurred()
}

func operationalStatus(status string) umpirespb.OperationalStatus {
	switch status {
	case "succeeded":
		return umpirespb.OPERATIONAL_STATUS_SUCCEEDED
	case "failed":
		return umpirespb.OPERATIONAL_STATUS_FAILED
	default:
		return umpirespb.OPERATIONAL_STATUS_INCOMPLETE
	}
}

func cleanupStatus(status string) umpirespb.CleanupStatus {
	switch status {
	case "complete":
		return umpirespb.CLEANUP_STATUS_COMPLETE
	case "failed":
		return umpirespb.CLEANUP_STATUS_FAILED
	default:
		return umpirespb.CLEANUP_STATUS_INCOMPLETE
	}
}

func failedResponse(
	contract *umpirespb.EvaluationContract,
	runIdentity string,
	status umpirespb.ToolingStatus,
) *umpirespb.ExecuteResponse {
	result := &umpirespb.EvaluationResult{
		RunIdentity:       runIdentity,
		ToolingStatus:     status,
		OperationalStatus: umpirespb.OPERATIONAL_STATUS_INCOMPLETE,
		Observation: &umpirespb.ObservationEvaluationResult{
			Status: umpirespb.OBSERVATION_STATUS_UNKNOWN,
		},
		ImplementationLink: &umpirespb.ImplementationLinkResult{
			Status: umpirespb.IMPLEMENTATION_LINK_STATUS_NOT_EVALUATED,
		},
		SemanticStatus: umpirespb.EVALUATION_STATUS_INCOMPLETE,
		CleanupStatus:  umpirespb.CLEANUP_STATUS_INCOMPLETE,
		Decision:       umpirespb.CANARY_DECISION_INCONCLUSIVE,
	}
	if contract != nil {
		result.Version = contract.GetVersion()
		result.ContractChecksum = append([]byte(nil), contract.GetArtifactChecksum()...)
	}
	return &umpirespb.ExecuteResponse{Result: result}
}
