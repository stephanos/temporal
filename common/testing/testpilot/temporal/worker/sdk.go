package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/nexus-rpc/sdk-go/nexus"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	sdkworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
)

type sdkManagedWorker struct{ sdkworker.Worker }

func (h *Driver) newSDKWorker(key, queue string, registration queueRegistration) (managedWorker, error) {
	options := h.options.workerOptions
	options.Interceptors = append([]interceptor.WorkerInterceptor(nil), options.Interceptors...)
	options.Interceptors = append(options.Interceptors, &sdkWorkerInterceptor{host: h, queue: queue, registration: registration})
	options.OnFatalError = func(err error) { h.registry.fail(key, err) }
	worker := sdkworker.New(h.options.client, queue, options)
	if err := h.register(worker, queue, registration); err != nil {
		return nil, err
	}
	return &sdkManagedWorker{Worker: worker}, nil
}

// sdkRegistrar is the part of an SDK worker a queue's registration is written to.
type sdkRegistrar interface {
	RegisterDynamicWorkflow(w interface{}, options workflow.DynamicRegisterOptions)
	RegisterDynamicActivity(a interface{}, options activity.DynamicRegisterOptions)
	RegisterNexusService(service *nexus.Service)
}

// register writes one queue's registration to its SDK worker before the worker starts. A queue
// that names no activity type registers no activity, so its worker still has no implementation for
// an activity task it is handed.
func (h *Driver) register(worker sdkRegistrar, queue string, registration queueRegistration) error {
	worker.RegisterDynamicWorkflow(h.dynamicWorkflow, workflow.DynamicRegisterOptions{})
	if len(registration.activities) > 0 {
		worker.RegisterDynamicActivity(h.dynamicActivity, activity.DynamicRegisterOptions{})
	}
	services := make(map[string]*nexus.Service)
	for _, signature := range registration.nexus {
		service := services[signature.service]
		if service == nil {
			service = nexus.NewService(signature.service)
			services[signature.service] = service
		}
		if err := service.Register(&genericNexusOperation{queue: queue, service: signature.service, operation: signature.operation}); err != nil {
			return err
		}
	}
	for _, service := range services {
		worker.RegisterNexusService(service)
	}
	return nil
}

type sdkWorkerInterceptor struct {
	interceptor.WorkerInterceptorBase
	host         *Driver
	queue        string
	registration queueRegistration
}

func (i *sdkWorkerInterceptor) InterceptWorkflow(_ workflow.Context, next interceptor.WorkflowInboundInterceptor) interceptor.WorkflowInboundInterceptor {
	return &workflowInboundInterceptor{WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next}, worker: i}
}

func (i *sdkWorkerInterceptor) InterceptActivity(_ context.Context, next interceptor.ActivityInboundInterceptor) interceptor.ActivityInboundInterceptor {
	return &activityInboundInterceptor{ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next}, worker: i}
}

func (i *sdkWorkerInterceptor) InterceptNexusOperation(_ context.Context, next interceptor.NexusOperationInboundInterceptor) interceptor.NexusOperationInboundInterceptor {
	return &nexusInboundInterceptor{NexusOperationInboundInterceptorBase: interceptor.NexusOperationInboundInterceptorBase{Next: next}, worker: i}
}

type workflowInboundInterceptor struct {
	interceptor.WorkflowInboundInterceptorBase
	worker *sdkWorkerInterceptor
}

func (i *workflowInboundInterceptor) Init(outbound interceptor.WorkflowOutboundInterceptor) error {
	return i.Next.Init(&workflowOutboundInterceptor{WorkflowOutboundInterceptorBase: interceptor.WorkflowOutboundInterceptorBase{Next: outbound}})
}

func (i *workflowInboundInterceptor) ExecuteWorkflow(ctx workflow.Context, input *interceptor.ExecuteWorkflowInput) (interface{}, error) {
	info := workflow.GetInfo(ctx)
	if info == nil || info.TaskQueueName != i.worker.queue || !slices.Contains(i.worker.registration.workflows, info.WorkflowType.Name) {
		return nil, activationError(ErrRegistrationConflict)
	}
	deliveryInput := delivery.WorkflowDelivery{
		Header: &commonpb.Header{Fields: maps.Clone(interceptor.WorkflowHeader(ctx))}, Namespace: info.Namespace,
		WorkflowID: info.WorkflowExecution.ID, WorkflowType: info.WorkflowType.Name, TaskQueue: info.TaskQueueName,
		TemporalRunID: info.WorkflowExecution.RunID,
	}
	routed, err := i.worker.host.admitWorkflow(deliveryInput)
	if err != nil {
		return nil, activationError(err)
	}
	ctx = workflow.WithValue(ctx, workflowRouteKey{}, routed)
	return i.Next.ExecuteWorkflow(ctx, input)
}

type workflowOutboundInterceptor struct {
	interceptor.WorkflowOutboundInterceptorBase
}

func (i *workflowOutboundInterceptor) ExecuteNexusOperation(ctx workflow.Context, input interceptor.ExecuteNexusOperationInput) workflow.NexusOperationFuture {
	routed, ok := ctx.Value(workflowRouteKey{}).(routedWorkflow)
	sourceID, sourceOK := ctx.Value(workflowSourceKey{}).(string)
	if !ok || !sourceOK {
		return failedNexusOperationFuture(ctx, ErrInvalid)
	}
	// A schedule command dispatches the payload it carries unconverted, or none, under the Case's
	// own Nexus header.
	if _, raw := input.Input.(converter.RawValue); !raw && input.Input != nil {
		return failedNexusOperationFuture(ctx, ErrInvalid)
	}
	caseHeader, _ := ctx.Value(caseNexusHeaderKey{}).(nexus.Header)
	prepared, err := routed.session.preparedNexusHeader(routed.activation, sourceID, caseHeader, input.NexusHeader)
	if err != nil {
		return failedNexusOperationFuture(ctx, err)
	}
	input.NexusHeader = prepared
	return i.Next.ExecuteNexusOperation(ctx, input)
}

type activityInboundInterceptor struct {
	interceptor.ActivityInboundInterceptorBase
	worker *sdkWorkerInterceptor
}

func (i *activityInboundInterceptor) ExecuteActivity(ctx context.Context, input *interceptor.ExecuteActivityInput) (interface{}, error) {
	info := activity.GetInfo(ctx)
	deliveryInput := delivery.ActivityDelivery{
		Header: &commonpb.Header{Fields: maps.Clone(interceptor.Header(ctx))}, Namespace: info.Namespace,
		ActivityID: info.ActivityID, ActivityType: info.ActivityType.Name, TaskQueue: info.TaskQueue,
		ActivityRunID: info.ActivityRunID, Attempt: info.Attempt, DeliveryID: deliveryIdentity(info.TaskToken),
	}
	return i.worker.activateActivity(ctx, deliveryInput, func(ctx context.Context) (interface{}, error) {
		return i.Next.ExecuteActivity(ctx, input)
	})
}

// deliveryIdentity is the bounded opaque name of one delivery of an activity attempt: a digest of
// the task token the server issued for it, which is itself unbounded and the server's to read.
func deliveryIdentity(taskToken []byte) string {
	digest := sha256.Sum256(taskToken)
	return hex.EncodeToString(digest[:])
}

// activateActivity runs one delivered activity task as the activation its start reserved. The first
// delivery of an attempt admits it against the reservation of its number, runs the entrypoint under
// a context the reservation's cancellation ends, and settles the reservation; any other delivery of
// that attempt waits for that answer and returns it, so nothing runs or settles twice.
//
// The reservation records, as a typed fact, which attempt this was and what the worker offers
// Temporal for it. That is the worker's answer and not the server's acceptance of it, so no later
// attempt's reservation is touched here: only the server closing the activity releases them. An attempt that did what its script said, completing or failing as declared, is a
// succeeded activation, and the declared failure goes to the SDK as it is, retryable or not as the
// script says. Anything else, a disabled instruction, a cancellation by the Run, an SDK or Driver
// error, is answered with the Driver's own non-retryable application failure, and that refusal,
// with its cause as detail, is the recorded outcome. The reservation settles with no error of its
// own, so the outcome is what the Run is given.
func (i *sdkWorkerInterceptor) activateActivity(ctx context.Context, input delivery.ActivityDelivery, run func(context.Context) (interface{}, error)) (interface{}, error) {
	if input.TaskQueue != i.queue || !slices.Contains(i.registration.activities, input.ActivityType) {
		return nil, activationError(ErrRegistrationConflict)
	}
	activationCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	routed, err := i.host.admitActivity(activationCtx, input, cancel)
	if err != nil {
		return nil, activationError(err)
	}
	answer := routed.answer
	if routed.activation.Replay() {
		select {
		case <-ctx.Done():
			return nil, activationError(ctx.Err())
		case <-answer.done:
			return answer.result, answer.err
		}
	}
	defer close(answer.done)
	func() {
		defer func() {
			if recover() != nil {
				answer.result, answer.err = nil, errors.New("activity activation panicked")
			}
		}()
		answer.result, answer.err = run(context.WithValue(activationCtx, activityRouteKey{}, routed))
	}()
	outcome := &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ActivityAttempt: &testpilotspb.ActivityAttempt{
		ActivityRunId: routed.activation.TemporalRunID(), SdkAttempt: routed.activation.Attempt(), DeliveryId: routed.activation.DeliveryID(),
		Response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED,
	}}
	var declared *declaredFailure
	switch {
	case answer.err == nil:
	case errors.As(answer.err, &declared):
		answer.result, answer.err = nil, declared.failure
		outcome.ActivityAttempt.Response = testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE
		if declared.nonRetryable {
			outcome.ActivityAttempt.Response = testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_NON_RETRYABLE
		}
	default:
		cause := answer.err
		answer.result, answer.err = nil, activationError(cause)
		outcome.Status, outcome.SdkFailureCode, outcome.Detail = testpilotspb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE, activationErrorType, boundedText(cause.Error())
		outcome.ActivityAttempt.Response = testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_REFUSED
	}
	routed.session.finishActivation(routed.activation, outcome, nil)
	routed.session.watchActivityClosure(input, routed.activation)
	return answer.result, answer.err
}

type failedNexusFuture struct{ workflow.Future }

func (f failedNexusFuture) GetNexusOperationExecution() workflow.Future { return f.Future }

func failedNexusOperationFuture(ctx workflow.Context, err error) workflow.NexusOperationFuture {
	future, settable := workflow.NewFuture(ctx)
	settable.SetError(activationError(err))
	return failedNexusFuture{Future: future}
}

type nexusInboundInterceptor struct {
	interceptor.NexusOperationInboundInterceptorBase
	worker *sdkWorkerInterceptor
}

func (i *nexusInboundInterceptor) StartOperation(ctx context.Context, input interceptor.NexusStartOperationInput) (nexus.HandlerStartOperationResult[any], error) {
	activationCtx, cancel := context.WithCancel(ctx)
	routed, err := i.worker.host.admitNexus(activationCtx, i.worker.queue, delivery.NexusDelivery{Header: input.Options.Header, RequestID: input.Options.RequestID}, cancel)
	if err != nil {
		cancel()
		return nil, nexusError(err)
	}
	activationCtx = context.WithValue(activationCtx, nexusRouteKey{}, routed)
	result, startErr := i.Next.StartOperation(activationCtx, input)
	// A retried delivery is admitted as a replay of its route and settles the activation the
	// retryable reply left open, so what finishes the activation is the reply, not the delivery.
	if outcome, open, activationErr := routed.session.nexusActivationOutcome(routed.activation, startErr); !open {
		routed.session.finishActivation(routed.activation, outcome, activationErr)
	}
	return result, startErr
}

func (i *nexusInboundInterceptor) CancelOperation(ctx context.Context, input interceptor.NexusCancelOperationInput) error {
	routed, err := i.worker.host.admitNexus(ctx, i.worker.queue, delivery.NexusDelivery{Header: input.Options.Header, RequestID: input.Token}, func() {})
	if err != nil {
		return nexusError(err)
	}
	raw, err := routed.session.rawReservation(routed.activation.Reservation().ID)
	if err != nil {
		return nexusError(err)
	}
	if err := raw.Cancel(ctx); err != nil {
		return nexusError(err)
	}
	return i.Next.CancelOperation(context.WithValue(ctx, nexusRouteKey{}, routed), input)
}

// genericNexusOperation is every registered operation: its input is read unconverted, because a
// schedule command carries any payload and the handler entrypoint reads none of it, and its output
// is whatever the entrypoint's reply carries, an interpreter value or an unconverted payload.
type genericNexusOperation struct {
	nexus.UnimplementedOperation[converter.RawValue, any]
	queue, service, operation string
}

func (o *genericNexusOperation) Name() string { return o.operation }

func (o *genericNexusOperation) Start(ctx context.Context, input converter.RawValue, options nexus.StartOperationOptions) (nexus.HandlerStartOperationResult[any], error) {
	routed, ok := ctx.Value(nexusRouteKey{}).(routedNexus)
	if !ok {
		return nil, nexusError(ErrInvalid)
	}
	entry := routed.session.definition.entries[routed.activation.Coordinate().EntrypointID]
	if entry.queue != o.queue || entry.service != o.service || entry.operation != o.operation {
		return nil, nexusError(ErrRegistrationConflict)
	}
	return routed.session.executeNexus(ctx, routed.activation, input, options)
}

func (*genericNexusOperation) Cancel(context.Context, string, nexus.CancelOperationOptions) error {
	return nil
}

type workflowRouteKey struct{}
type workflowSourceKey struct{}
type activityRouteKey struct{}
type nexusRouteKey struct{}

type routedWorkflow struct {
	session    *Session
	activation delivery.Activation
	admission  *workflowAdmission
	replay     bool
}

type routedActivity struct {
	session    *Session
	activation delivery.Activation
	answer     *activityAnswer
}

// activityAnswer is what one admitted activity attempt answers the SDK with. It is written once,
// by the delivery that was admitted, before done closes, and read by every later delivery of the
// attempt.
type activityAnswer struct {
	done   chan struct{}
	result interface{}
	err    error
}

type routedNexus struct {
	session    *Session
	activation delivery.Activation
	replay     bool
}

// pollActivityClosed long-polls the server for the outcome of a standalone activity run and returns
// the run the outcome is of, as the server names it. The server answers an expired poll with no
// outcome, which invites the next one.
func pollActivityClosed(sdk client.Client) func(context.Context, string, string, string) (string, error) {
	return func(ctx context.Context, namespace, activityID, activityRunID string) (string, error) {
		for {
			response, err := sdk.WorkflowService().PollActivityExecution(ctx, &workflowservice.PollActivityExecutionRequest{Namespace: namespace, ActivityId: activityID, RunId: activityRunID})
			if err != nil {
				return "", err
			}
			if response.GetOutcome() != nil {
				return response.GetRunId(), nil
			}
		}
	}
}

func (h *Driver) dynamicActivity(ctx context.Context, _ converter.EncodedValues) (*testpilotspb.Value, error) {
	routed, ok := ctx.Value(activityRouteKey{}).(routedActivity)
	if !ok {
		return nil, ErrInvalid
	}
	return routed.session.executeActivity(ctx, routed.activation)
}

func (h *Driver) dynamicWorkflow(ctx workflow.Context, _ converter.EncodedValues) (*testpilotspb.Value, error) {
	routed, ok := ctx.Value(workflowRouteKey{}).(routedWorkflow)
	if !ok {
		return nil, activationError(ErrInvalid)
	}
	result, err := routed.session.executeWorkflow(ctx, routed.activation)
	err = routed.session.completeWorkflow(routed.admission, routed.activation, err)
	return result, err
}

func (s *Session) completeWorkflow(admission *workflowAdmission, activation delivery.Activation, executionErr error) error {
	if admission == nil || admission.activation.Coordinate() != activation.Coordinate() {
		return ErrInvalid
	}
	admission.mu.Lock()
	defer admission.mu.Unlock()
	if admission.terminal {
		return executionErr
	}
	_, terminalErr := s.parentTerminal(context.Background(), activation)
	if executionErr == nil && terminalErr != nil {
		executionErr = terminalErr
	}
	outcome := &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}
	if executionErr != nil {
		outcome = sdkFailureOutcome(executionErr)
	}
	s.finishActivation(activation, outcome, executionErr)
	admission.terminal = true
	return executionErr
}

// activationErrorType is the application failure type the Driver answers the SDK with for an
// activation it refuses or fails itself.
const activationErrorType = "umpire_worker"

// activationError is what the SDK is answered with when an activation is refused or fails: an
// application error it does not retry, carrying the cause.
func activationError(err error) error {
	return temporal.NewNonRetryableApplicationError("testpilot worker activation", activationErrorType, err)
}

func nexusError(err error) error {
	if err == nil {
		return nil
	}
	kind := nexus.HandlerErrorTypeInternal
	if errors.Is(err, delivery.ErrRouteCrossed) || errors.Is(err, delivery.ErrBindingMismatch) || errors.Is(err, ErrRegistrationConflict) || errors.Is(err, ErrInvalid) {
		kind = nexus.HandlerErrorTypeBadRequest
	} else if errors.Is(err, delivery.ErrRouteConflict) {
		kind = nexus.HandlerErrorTypeConflict
	}
	return &nexus.HandlerError{Type: kind, Message: boundedText(err.Error()), RetryBehavior: nexus.HandlerErrorRetryBehaviorNonRetryable}
}

func boundedText(value string) string {
	const maximum = 1024
	if len(value) > maximum {
		value = value[:maximum]
	}
	return strings.ToValidUTF8(value, "")
}
