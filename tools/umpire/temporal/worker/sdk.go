package worker

import (
	"context"
	"errors"
	"maps"
	"strings"

	"github.com/nexus-rpc/sdk-go/nexus"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	sdkworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/temporal/internal/delivery"
)

type sdkManagedWorker struct{ sdkworker.Worker }

func (h *Host) newSDKWorker(queue string, registration queueRegistration) (managedWorker, error) {
	options := h.options.workerOptions
	options.Interceptors = append([]interceptor.WorkerInterceptor(nil), options.Interceptors...)
	options.Interceptors = append(options.Interceptors, &sdkWorkerInterceptor{host: h, queue: queue, registration: registration})
	options.OnFatalError = func(err error) { h.registry.fail(queue, err) }
	worker := sdkworker.New(h.options.client, queue, options)
	worker.RegisterDynamicWorkflow(h.dynamicWorkflow, workflow.DynamicRegisterOptions{})
	services := make(map[string]*nexus.Service)
	for _, signature := range registration.nexus {
		service := services[signature.service]
		if service == nil {
			service = nexus.NewService(signature.service)
			services[signature.service] = service
		}
		if err := service.Register(&genericNexusOperation{queue: queue, service: signature.service, operation: signature.operation}); err != nil {
			return nil, err
		}
	}
	for _, service := range services {
		worker.RegisterNexusService(service)
	}
	return &sdkManagedWorker{Worker: worker}, nil
}

type sdkWorkerInterceptor struct {
	interceptor.WorkerInterceptorBase
	host         *Host
	queue        string
	registration queueRegistration
}

func (i *sdkWorkerInterceptor) InterceptWorkflow(_ workflow.Context, next interceptor.WorkflowInboundInterceptor) interceptor.WorkflowInboundInterceptor {
	return &workflowInboundInterceptor{WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next}, worker: i}
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
	if info == nil || info.TaskQueueName != i.worker.queue || !slicesContains(i.worker.registration.workflows, info.WorkflowType.Name) {
		return nil, workflowError(ErrRegistrationConflict)
	}
	deliveryInput := delivery.WorkflowDelivery{
		Header: &commonpb.Header{Fields: maps.Clone(interceptor.WorkflowHeader(ctx))}, Namespace: info.Namespace,
		WorkflowID: info.WorkflowExecution.ID, WorkflowType: info.WorkflowType.Name, TaskQueue: info.TaskQueueName,
		TemporalRunID: info.WorkflowExecution.RunID,
	}
	routed, err := i.worker.host.admitWorkflow(deliveryInput)
	if err != nil {
		return nil, workflowError(err)
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
	value, valueOK := input.Input.(*umpirespb.Value)
	if !ok || !sourceOK || !valueOK {
		return failedNexusOperationFuture(ctx, ErrInvalid)
	}
	header, preparedValue, err := routed.session.preparedNexusDispatch(routed.activation, sourceID, input.NexusHeader, value)
	if err != nil {
		return failedNexusOperationFuture(ctx, err)
	}
	input.Input = preparedValue
	input.NexusHeader = header
	return i.Next.ExecuteNexusOperation(ctx, input)
}

type failedNexusFuture struct{ workflow.Future }

func (f failedNexusFuture) GetNexusOperationExecution() workflow.Future { return f.Future }

func failedNexusOperationFuture(ctx workflow.Context, err error) workflow.NexusOperationFuture {
	future, settable := workflow.NewFuture(ctx)
	settable.SetError(workflowError(err))
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
	outcome := &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}
	if startErr != nil {
		outcome = sdkFailureOutcome(startErr)
	}
	if !routed.replay {
		routed.session.finishActivation(routed.activation, outcome, startErr)
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

type genericNexusOperation struct {
	nexus.UnimplementedOperation[*umpirespb.Value, *umpirespb.Value]
	queue, service, operation string
}

func (o *genericNexusOperation) Name() string { return o.operation }

func (o *genericNexusOperation) Start(ctx context.Context, input *umpirespb.Value, options nexus.StartOperationOptions) (nexus.HandlerStartOperationResult[*umpirespb.Value], error) {
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
type nexusRouteKey struct{}

type routedWorkflow struct {
	session    *Session
	activation delivery.Activation
	admission  *workflowAdmission
	replay     bool
}

type routedNexus struct {
	session    *Session
	activation delivery.Activation
	replay     bool
}

func (h *Host) dynamicWorkflow(ctx workflow.Context, _ converter.EncodedValues) (*umpirespb.Value, error) {
	routed, ok := ctx.Value(workflowRouteKey{}).(routedWorkflow)
	if !ok {
		return nil, workflowError(ErrInvalid)
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
	outcome := &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}
	if executionErr != nil {
		outcome = sdkFailureOutcome(executionErr)
	}
	s.finishActivation(activation, outcome, executionErr)
	admission.terminal = true
	return executionErr
}

func workflowError(err error) error {
	return temporal.NewNonRetryableApplicationError("umpire worker activation", "umpire_worker", err)
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

func slicesContains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
