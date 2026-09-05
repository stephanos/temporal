package worker

import (
	"context"
	"errors"
	"maps"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"go.temporal.io/server/tools/umpire/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
)

type nexusResult struct {
	done  chan struct{}
	kind  umpirespb.NexusResponseKind
	value *umpirespb.Value
	token string
	err   error
}

type workflowInterpreter struct {
	session *Session
	ctx     workflow.Context
	values  *activationValues
	futures map[string]workflow.NexusOperationFuture
}

func (s *Session) executeWorkflow(ctx workflow.Context, activation delivery.Activation) (*umpirespb.Value, error) {
	entry, exists := s.definition.entries[activation.Coordinate().EntrypointID]
	if !exists || entry.plan.Context() != umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW {
		return nil, ErrInvalid
	}
	interpreter := workflowInterpreter{session: s, ctx: ctx, values: newActivationValues(entry.plan.ID(), entry.plan.RuntimeWorkLimit()), futures: make(map[string]workflow.NexusOperationFuture)}
	for _, index := range entry.plan.Order() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		instruction, err := instructionAt(entry.plan, index)
		if err != nil {
			return nil, err
		}
		input, enabled, err := evaluateInstruction(context.Background(), interpreter.values, instruction)
		if err != nil {
			return nil, err
		}
		if !enabled {
			continue
		}
		result, finished, err := interpreter.execute(instruction, input)
		if err != nil || finished {
			return result, err
		}
	}
	return nil, errors.New("workflow entrypoint completed without Finish")
}

func (i *workflowInterpreter) execute(instruction umpire.InstructionPlan, input *umpirespb.Value) (*umpirespb.Value, bool, error) {
	switch instruction.Opcode() {
	case umpire.StartNexusOperation:
		return nil, false, i.startNexus(instruction, input)
	case umpire.Await:
		return nil, false, i.awaitNexus(instruction)
	case umpire.Finish:
		outcome := &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}
		if err := validateAndStore(context.Background(), i.values, instruction, outcome); err != nil {
			return nil, false, err
		}
		return proto.CloneOf(input), true, nil
	default:
		return nil, false, ErrInvalid
	}
}

func (i *workflowInterpreter) startNexus(instruction umpire.InstructionPlan, input *umpirespb.Value) error {
	source := instruction.Source()
	start := source.GetInstruction().GetStartNexusOperation()
	endpoint := i.session.definition.endpoints[start.GetEndpointRoleId()]
	if endpoint == "" || input == nil {
		return ErrInvalid
	}
	operationCtx := workflow.WithValue(i.ctx, workflowSourceKey{}, source.GetInstructionId())
	future := workflow.NewNexusClient(endpoint, start.GetService()).ExecuteOperation(
		operationCtx,
		start.GetOperation(),
		input,
		workflow.NexusOperationOptions{
			ScheduleToCloseTimeout: time.Duration(source.GetBounds().GetTimeoutMilliseconds()) * time.Millisecond,
			CancellationType:       workflow.NexusOperationCancellationTypeWaitRequested,
		},
	)
	i.futures[source.GetInstructionId()] = future
	var execution workflow.NexusOperationExecution
	err := future.GetNexusOperationExecution().Get(i.ctx, &execution)
	return validateAndStore(context.Background(), i.values, instruction, outcomeForError(err))
}

func (i *workflowInterpreter) awaitNexus(instruction umpire.InstructionPlan) error {
	await := instruction.Source().GetInstruction().GetAwaitOutcome()
	future := i.futures[await.GetInstruction().GetInstructionId()]
	if future == nil {
		return ErrInvalid
	}
	var result umpirespb.Value
	err := future.Get(i.ctx, &result)
	outcome := outcomeForError(err)
	if err == nil {
		outcome.Value = &result
	}
	return validateAndStore(context.Background(), i.values, instruction, outcome)
}

func outcomeForError(err error) *umpirespb.InstructionOutcome {
	if err != nil {
		return sdkFailureOutcome(err)
	}
	return &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}
}

func (s *Session) executeNexus(ctx context.Context, activation delivery.Activation, _ *umpirespb.Value, options nexus.StartOperationOptions) (nexus.HandlerStartOperationResult[*umpirespb.Value], error) {
	key := activation.Reservation().ID
	if err := s.mu.lock(ctx); err != nil {
		return nil, err
	}
	if existing := s.nexusResults[key]; existing != nil {
		s.mu.unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-existing.done:
			return existing.response()
		}
	}
	if s.closed || s.failure != nil {
		s.mu.unlock()
		return nil, errors.Join(ErrClosed, s.failure)
	}
	if len(s.nexusResults) >= boundedInt(s.definition.snapshot.GetLimits().GetMaxActivations()) {
		s.mu.unlock()
		return nil, ErrCapacity
	}
	result := &nexusResult{done: make(chan struct{})}
	s.nexusResults[key] = result
	s.mu.unlock()

	func() {
		defer func() {
			if recover() != nil {
				result.err = errors.New("nexus handler activation panicked")
			}
		}()
		result.kind, result.value, result.token, result.err = s.interpretNexus(ctx, activation, options)
	}()
	close(result.done)
	return result.response()
}

func (s *Session) interpretNexus(ctx context.Context, activation delivery.Activation, options nexus.StartOperationOptions) (umpirespb.NexusResponseKind, *umpirespb.Value, string, error) {
	entry, exists := s.definition.entries[activation.Coordinate().EntrypointID]
	if !exists || entry.plan.Context() != umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER {
		return 0, nil, "", ErrInvalid
	}
	values := newActivationValues(entry.plan.ID(), entry.plan.RuntimeWorkLimit())
	for _, index := range entry.plan.Order() {
		instruction, err := instructionAt(entry.plan, index)
		if err != nil {
			return 0, nil, "", err
		}
		input, enabled, err := evaluateInstruction(ctx, values, instruction)
		if err != nil {
			return 0, nil, "", err
		}
		if !enabled {
			continue
		}
		if instruction.Opcode() != umpire.RespondNexus {
			return 0, nil, "", ErrInvalid
		}
		response := instruction.Source().GetInstruction().GetRespondNexus()
		if err := validateAndStore(ctx, values, instruction, &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}); err != nil {
			return 0, nil, "", err
		}
		return s.respondNexus(ctx, activation, response, input, options)
	}
	return 0, nil, "", errors.New("nexus handler entrypoint completed without RespondNexus")
}

func (s *Session) respondNexus(ctx context.Context, activation delivery.Activation, response *umpirespb.RespondNexus, input *umpirespb.Value, options nexus.StartOperationOptions) (umpirespb.NexusResponseKind, *umpirespb.Value, string, error) {
	switch response.GetKind() {
	case umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS:
		if input == nil {
			return 0, nil, "", ErrInvalid
		}
		return response.GetKind(), proto.CloneOf(input), "", nil
	case umpirespb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS:
		return s.respondNexusAsync(ctx, activation, response, input, options)
	case umpirespb.NEXUS_RESPONSE_KIND_ERROR:
		detail := "Nexus handler returned an error"
		if input != nil && input.GetText() != "" {
			detail = input.GetText()
		}
		return response.GetKind(), nil, "", &nexus.HandlerError{Type: nexus.HandlerErrorTypeInternal, Message: boundedText(detail), RetryBehavior: nexus.HandlerErrorRetryBehaviorNonRetryable}
	default:
		return 0, nil, "", ErrInvalid
	}
}

func (s *Session) respondNexusAsync(ctx context.Context, activation delivery.Activation, response *umpirespb.RespondNexus, input *umpirespb.Value, options nexus.StartOperationOptions) (umpirespb.NexusResponseKind, *umpirespb.Value, string, error) {
	if input == nil || s.options.NewCompletionCapability == nil || nilValue(s.options.Bridge) {
		return 0, nil, "", ErrInvalid
	}
	info := CompletionInfo{URL: options.CallbackURL, Header: maps.Clone(options.CallbackHeader), OperationToken: activation.RequestID(), StartTime: s.host.options.now()}
	capability, err := s.options.NewCompletionCapability(ctx, activation.Coordinate(), info)
	if err != nil {
		return 0, nil, "", err
	}
	if nilValue(capability) {
		return 0, nil, "", ErrInvalid
	}
	if err := s.publicationAllowed(ctx); err != nil {
		s.lateDiagnostic(ctx, "completion_publication_late")
		return 0, nil, "", err
	}
	if err := s.options.Bridge.Publish(ctx, activation.Coordinate(), response.GetCapabilitySlotId(), capability); err != nil {
		if errors.Is(s.publicationAllowed(ctx), ErrClosed) {
			s.lateDiagnostic(ctx, "completion_publication_late")
		}
		return 0, nil, "", err
	}
	return response.GetKind(), nil, activation.RequestID(), nil
}

func instructionAt(entry umpire.EntrypointPlan, index int) (umpire.InstructionPlan, error) {
	instructions := entry.Instructions()
	if index < 0 || index >= len(instructions) {
		return umpire.InstructionPlan{}, ErrInvalid
	}
	return instructions[index], nil
}

func (s *Session) publicationAllowed(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalid
	}
	lockCtx := ctx
	cancel := func() {}
	if ctx.Err() != nil {
		lockCtx, cancel = s.host.cleanupContext()
	}
	defer cancel()
	if err := s.mu.lock(lockCtx); err != nil {
		return err
	}
	defer s.mu.unlock()
	if s.closed || s.failure != nil {
		return errors.Join(ErrClosed, s.failure)
	}
	return ctx.Err()
}

func (r *nexusResult) response() (nexus.HandlerStartOperationResult[*umpirespb.Value], error) {
	if r.err != nil {
		return nil, r.err
	}
	switch r.kind {
	case umpirespb.NEXUS_RESPONSE_KIND_SYNCHRONOUS:
		return &nexus.HandlerStartOperationResultSync[*umpirespb.Value]{Value: proto.CloneOf(r.value)}, nil
	case umpirespb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS:
		return &nexus.HandlerStartOperationResultAsync{OperationToken: r.token}, nil
	default:
		return nil, ErrInvalid
	}
}

func evaluateInstruction(ctx context.Context, values *activationValues, instruction umpire.InstructionPlan) (*umpirespb.Value, bool, error) {
	input, enabled, work, err := instruction.EvaluateInput(ctx, values.lookup, values.remaining)
	values.remaining -= work
	if values.remaining < 0 {
		return nil, false, ErrCapacity
	}
	return input, enabled, err
}

func validateAndStore(ctx context.Context, values *activationValues, instruction umpire.InstructionPlan, outcome *umpirespb.InstructionOutcome) error {
	snapshot, work, err := instruction.ValidateOutcome(ctx, outcome, values.remaining)
	values.remaining -= work
	if err != nil {
		return err
	}
	if values.remaining < 0 {
		return ErrCapacity
	}
	values.store(instruction.Source().GetInstructionId(), snapshot)
	return nil
}

func sdkFailureOutcome(err error) *umpirespb.InstructionOutcome {
	outcome := &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE, SdkFailureCode: "sdk_failure", Detail: boundedText(err.Error())}
	if temporal.IsCanceledError(err) || errors.Is(err, context.Canceled) {
		outcome.Status = umpirespb.INSTRUCTION_OUTCOME_STATUS_CANCELED
		outcome.SdkFailureCode = "canceled"
	} else if temporal.IsTimeoutError(err) || errors.Is(err, context.DeadlineExceeded) {
		outcome.Status = umpirespb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT
		outcome.SdkFailureCode = "timed_out"
	} else {
		var applicationError *temporal.ApplicationError
		if errors.As(err, &applicationError) && applicationError.Type() != "" {
			outcome.SdkFailureCode = boundedText(applicationError.Type())
		}
	}
	return outcome
}

func cloneCompletionInfo(info CompletionInfo) CompletionInfo {
	return CompletionInfo{URL: info.URL, Header: maps.Clone(info.Header), OperationToken: info.OperationToken, StartTime: info.StartTime}
}
