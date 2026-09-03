package executor

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/portableevaluation"
	"go.temporal.io/server/tools/umpire/runner"
	"go.temporal.io/server/tools/umpire/testplan"
)

// PortableErrorCode identifies failures that prevent an admitted execution result.
type PortableErrorCode string

const (
	PortableErrorInvalidArgument    PortableErrorCode = "invalid-argument"
	PortableErrorFailedPrecondition PortableErrorCode = "failed-precondition"
	PortableErrorResourceExhausted  PortableErrorCode = "resource-exhausted"
	PortableErrorInternal           PortableErrorCode = "internal"
)

// PortableError reports the responsible pre-result executor seam.
type PortableError struct {
	Code PortableErrorCode
	err  error
}

func (e *PortableError) Error() string {
	if e == nil {
		return ""
	}
	if e.err == nil {
		return string(e.Code)
	}
	return string(e.Code) + ": " + e.err.Error()
}

func (e *PortableError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// PortableCodeOf returns the stable code for a pre-result executor failure.
func PortableCodeOf(err error) (PortableErrorCode, bool) {
	var portable *PortableError
	if !errors.As(err, &portable) {
		return "", false
	}
	return portable.Code, true
}

type portablePreparation struct {
	plan             testplan.AuthorizedPlan
	input            artifact.AdmittedSet
	binding          runner.InputBinding
	experiment       artifactv2.ArtifactBinding
	runtime          artifactv2.ArtifactBinding
	executionTimeout time.Duration
}

type preparePortable func(
	context.Context,
	*umpirespb.PortableTestPlan,
	testplan.ModelProvenanceVerifier,
) (portablePreparation, error)

type evaluatePortable func(context.Context, portableevaluation.PortableRequest) *umpirespb.ExecutionResult

// PortableExecutor owns one admitted typed-plan execution at a time.
type PortableExecutor struct {
	gate               executionGate
	adapter            runner.Adapter
	provenanceVerifier testplan.ModelProvenanceVerifier
	prepare            preparePortable
	run                executeRun
	evaluate           evaluatePortable
	nextRunIdentity    func() string
}

// NewPortable returns the caller-neutral executor over the existing runner adapter.
func NewPortable(adapter runner.Adapter, verifier testplan.ModelProvenanceVerifier) *PortableExecutor {
	var sequence atomic.Uint64
	prefix := "umpire.executor.portable-run." + uuid.NewString()
	return newPortableExecutor(
		adapter, verifier, preparePortableExecution, executeWithRunner,
		portableevaluation.EvaluatePortable,
		func() string { return prefix + "." + strconv.FormatUint(sequence.Add(1), 10) },
	)
}

func newPortableExecutor(
	adapter runner.Adapter,
	verifier testplan.ModelProvenanceVerifier,
	prepare preparePortable,
	run executeRun,
	evaluate evaluatePortable,
	nextRunIdentity func() string,
) *PortableExecutor {
	return &PortableExecutor{
		adapter: adapter, provenanceVerifier: verifier, prepare: prepare,
		run: run, evaluate: evaluate, nextRunIdentity: nextRunIdentity,
	}
}

// Execute admits, projects, runs, closes, and evaluates one exact typed plan.
func (e *PortableExecutor) Execute(
	ctx context.Context,
	plan *umpirespb.PortableTestPlan,
) (*umpirespb.ExecutionResult, error) {
	if e == nil {
		return nil, portableError(PortableErrorInternal, errors.New("portable executor is required"))
	}
	admitted, rejection := e.gate.admit()
	if !admitted {
		code := PortableErrorResourceExhausted
		if rejection == umpirespb.TOOLING_STATUS_POISONED {
			code = PortableErrorFailedPrecondition
		}
		return nil, portableError(code, fmt.Errorf("executor is %s", rejection.String()))
	}
	reusable := false
	defer func() { e.gate.finish(reusable) }()

	if ctx == nil {
		reusable = true
		return nil, portableError(PortableErrorInvalidArgument, errors.New("context is required"))
	}
	if err := ctx.Err(); err != nil {
		reusable = true
		return nil, err
	}
	prepared, err := e.prepare(ctx, plan, e.provenanceVerifier)
	if err != nil {
		reusable = true
		return nil, err
	}
	executionContext, cancel := context.WithTimeout(
		ctx,
		prepared.executionTimeout,
	)
	defer cancel()
	runIdentity := e.nextRunIdentity()
	outcome, err := e.run(executionContext, prepared.input, prepared.binding, runIdentity, e.adapter)
	reusable = outcome.reusable
	if err != nil {
		if executionOccurred(err) {
			return nil, portableError(PortableErrorInternal, err)
		}
		if cancellation := executionCancellation(executionContext, err); cancellation != nil {
			return nil, cancellation
		}
		return nil, portableError(PortableErrorFailedPrecondition, err)
	}
	if outcome.rawEvidence.RunIdentity != runIdentity {
		reusable = false
		return nil, portableError(PortableErrorInternal, errors.New("runner returned crossed Evidence"))
	}
	result := e.evaluate(executionContext, portableevaluation.PortableRequest{
		Plan: prepared.plan, RawEvidence: outcome.rawEvidence,
		ExpectedRunIdentity:          runIdentity,
		ExpectedExperiment:           prepared.experiment,
		ExpectedRuntimeConfiguration: prepared.runtime,
		ExpectedRun:                  outcome.runBinding, ExpectedClosures: outcome.sourceClosures,
		OperationalStatus: outcome.operationalStatus, CleanupStatus: outcome.cleanupStatus,
	})
	if result == nil {
		reusable = false
		return nil, portableError(PortableErrorInternal, errors.New("portable evaluator returned no result"))
	}
	return result, nil
}

func preparePortableExecution(
	ctx context.Context,
	plan *umpirespb.PortableTestPlan,
	verifier testplan.ModelProvenanceVerifier,
) (portablePreparation, error) {
	admitted, err := testplan.Admit(plan)
	if err != nil {
		return portablePreparation{}, err
	}
	authorized, err := testplan.Authorize(ctx, admitted, verifier)
	if err != nil {
		return portablePreparation{}, err
	}
	checkedPlan := authorized.Plan()
	input, err := projectPortableExecution(checkedPlan)
	if err != nil {
		return portablePreparation{}, portableError(PortableErrorInvalidArgument, err)
	}
	if !portableModelBindingsMatch(checkedPlan, input) {
		return portablePreparation{}, portableError(
			PortableErrorInvalidArgument,
			errors.New("model artifact bindings do not match the typed execution projection"),
		)
	}
	bindings, err := portableInputBindings(
		input,
		checkedPlan.GetExecution().GetRuntimeBindingSlots(),
		portableDefinitionIDs(checkedPlan.GetExecution().GetRuntime().GetAuthorityRequiredCapabilities()),
	)
	if err != nil {
		return portablePreparation{}, portableError(PortableErrorInternal, err)
	}
	return portablePreparation{
		plan: authorized, input: input, binding: bindings.binding,
		experiment: bindings.experiment, runtime: bindings.runtime,
		executionTimeout: time.Duration(
			checkedPlan.GetLimits().GetExecution().GetMaxTotalDurationMilliseconds(),
		) * time.Millisecond,
	}, nil
}

func executionCancellation(ctx context.Context, err error) error {
	if contextErr := ctx.Err(); contextErr != nil {
		return contextErr
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	return nil
}

func portableError(code PortableErrorCode, err error) error {
	return &PortableError{Code: code, err: err}
}
