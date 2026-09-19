package worker

import (
	"context"
	"errors"
	"maps"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/activation"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
)

type nexusResult struct {
	done  chan struct{}
	kind  testpilotspb.NexusResponseKind
	value *testpilotspb.Value
	// raw is a typed synchronous reply's payload, returned unconverted.
	raw   *commonpb.Payload
	token string
	err   error
}

type workflowInterpreter struct {
	session *Session
	ctx     workflow.Context
	state   *activation.State
	futures map[string]workflow.NexusOperationFuture
	// typed marks the futures a schedule command started, whose results are payloads rather than
	// interpreter values.
	typed map[string]bool
}

func (s *Session) executeWorkflow(ctx workflow.Context, delivered delivery.Activation) (*testpilotspb.Value, error) {
	entry, exists := s.definition.entries[delivered.Coordinate().EntrypointID]
	if !exists || entry.plan.Kind() != testpilot.WorkflowEntrypoint {
		return nil, ErrInvalid
	}
	state, err := activation.New(entry.plan)
	if err != nil {
		return nil, err
	}
	interpreter := workflowInterpreter{session: s, ctx: ctx, state: state, futures: make(map[string]workflow.NexusOperationFuture), typed: make(map[string]bool)}
	instructions := entry.plan.Instructions()
	for _, index := range entry.plan.Order() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		input, enabled, err := state.Evaluate(context.Background(), index)
		if err != nil {
			return nil, err
		}
		if !enabled {
			continue
		}
		result, finished, err := interpreter.execute(index, instructions[index], input)
		if err != nil || finished {
			return result, err
		}
	}
	return nil, errors.New("workflow entrypoint completed without Finish")
}

func (i *workflowInterpreter) execute(index int, instruction testpilot.InstructionPlan, input *testpilotspb.Value) (*testpilotspb.Value, bool, error) {
	switch instruction.Opcode() {
	case testpilot.StartNexusOperation:
		return nil, false, i.startNexus(index, instruction, input)
	case testpilot.WorkflowCommand:
		return nil, false, i.scheduleNexus(index, instruction)
	case testpilot.Await:
		return nil, false, i.awaitNexus(index, instruction)
	case testpilot.Finish:
		if err := i.state.Admit(context.Background(), index, terminalOutcome()); err != nil {
			return nil, false, err
		}
		return proto.CloneOf(input), true, nil
	default:
		return nil, false, ErrInvalid
	}
}

func (i *workflowInterpreter) startNexus(index int, instruction testpilot.InstructionPlan, input *testpilotspb.Value) error {
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
			ScheduleToCloseTimeout: time.Duration(instruction.TimeoutMilliseconds()) * time.Millisecond,
			CancellationType:       workflow.NexusOperationCancellationTypeWaitRequested,
		},
	)
	i.futures[source.GetInstructionId()] = future
	var execution workflow.NexusOperationExecution
	err := future.GetNexusOperationExecution().Get(i.ctx, &execution)
	return i.state.Admit(context.Background(), index, outcomeForError(err))
}

func (i *workflowInterpreter) awaitNexus(index int, instruction testpilot.InstructionPlan) error {
	await := instruction.Source().GetInstruction().GetAwaitInstruction()
	future := i.futures[await.GetInstruction().GetInstructionId()]
	if future == nil {
		return ErrInvalid
	}
	var result *testpilotspb.Value
	ready := future.IsReady()
	var err error
	if !ready {
		ready, err = workflow.AwaitWithTimeout(i.ctx, time.Duration(instruction.TimeoutMilliseconds())*time.Millisecond, future.IsReady)
	}
	if err == nil {
		switch {
		case !ready:
			err = context.DeadlineExceeded
		case i.typed[await.GetInstruction().GetInstructionId()]:
			result, err = awaitedPayload(i.ctx, future)
		default:
			var value testpilotspb.Value
			err = future.Get(i.ctx, &value)
			result = &value
		}
	}
	outcome := outcomeForError(err)
	if err == nil {
		outcome.Value = result
	}
	return i.state.Admit(context.Background(), index, outcome)
}

// terminalOutcome is the outcome of a Finish or RespondNexus that ended its activation. Its result is
// the activation's result, not an outcome value.
func terminalOutcome() *testpilotspb.InstructionOutcome {
	return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}
}

func outcomeForError(err error) *testpilotspb.InstructionOutcome {
	if err != nil {
		return sdkFailureOutcome(err)
	}
	return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}
}

func (s *Session) executeNexus(ctx context.Context, delivered delivery.Activation, _ any, options nexus.StartOperationOptions) (nexus.HandlerStartOperationResult[any], error) {
	key := delivered.Reservation().ID
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
	if len(s.nexusResults) >= boundedInt(s.definition.limits.GetMaxActivations()) {
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
		interpreted, err := s.interpretNexus(ctx, delivered, options)
		result.kind, result.value, result.raw, result.token, result.err = interpreted.kind, interpreted.value, interpreted.raw, interpreted.token, err
	}()
	close(result.done)
	return result.response()
}

func (s *Session) interpretNexus(ctx context.Context, delivered delivery.Activation, options nexus.StartOperationOptions) (nexusResult, error) {
	entry, exists := s.definition.entries[delivered.Coordinate().EntrypointID]
	if !exists || entry.plan.Kind() != testpilot.NexusHandlerEntrypoint {
		return nexusResult{}, ErrInvalid
	}
	state, err := activation.New(entry.plan)
	if err != nil {
		return nexusResult{}, err
	}
	instructions := entry.plan.Instructions()
	for _, index := range entry.plan.Order() {
		input, enabled, err := state.Evaluate(ctx, index)
		if err != nil {
			return nexusResult{}, err
		}
		if !enabled {
			continue
		}
		instruction := instructions[index]
		switch instruction.Opcode() {
		case testpilot.RespondNexus:
			response := instruction.Source().GetInstruction().GetRespondNexus()
			if err := state.Admit(ctx, index, terminalOutcome()); err != nil {
				return nexusResult{}, err
			}
			kind, value, token, err := s.respondNexus(ctx, delivered, response, input, options)
			return nexusResult{kind: kind, value: value, token: token}, err
		case testpilot.NexusHandlerReply:
			reply := instruction.Source().GetInstruction().GetNexusHandlerReply()
			if err := state.Admit(ctx, index, terminalOutcome()); err != nil {
				return nexusResult{}, err
			}
			return s.replyTyped(ctx, delivered, reply, options)
		default:
			return nexusResult{}, ErrInvalid
		}
	}
	return nexusResult{}, errors.New("nexus handler entrypoint completed without a reply")
}

func (s *Session) respondNexus(ctx context.Context, delivered delivery.Activation, response *testpilotspb.RespondNexus, input *testpilotspb.Value, options nexus.StartOperationOptions) (testpilotspb.NexusResponseKind, *testpilotspb.Value, string, error) {
	switch response.GetKind() {
	case testpilotspb.NEXUS_RESPONSE_KIND_SYNCHRONOUS:
		if input == nil {
			return 0, nil, "", ErrInvalid
		}
		return response.GetKind(), proto.CloneOf(input), "", nil
	case testpilotspb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS:
		return s.respondNexusAsync(ctx, delivered, response, input, options)
	case testpilotspb.NEXUS_RESPONSE_KIND_ERROR:
		detail := "Nexus handler returned an error"
		if input != nil && input.GetTextValue() != "" {
			detail = input.GetTextValue()
		}
		return response.GetKind(), nil, "", &nexus.HandlerError{Type: nexus.HandlerErrorTypeInternal, Message: boundedText(detail), RetryBehavior: nexus.HandlerErrorRetryBehaviorNonRetryable}
	default:
		return 0, nil, "", ErrInvalid
	}
}

func (s *Session) respondNexusAsync(ctx context.Context, delivered delivery.Activation, response *testpilotspb.RespondNexus, input *testpilotspb.Value, options nexus.StartOperationOptions) (testpilotspb.NexusResponseKind, *testpilotspb.Value, string, error) {
	if input == nil {
		return 0, nil, "", ErrInvalid
	}
	token, err := s.publishCompletionAuthority(ctx, delivered, response.GetHandleSlotId(), options)
	if err != nil {
		return 0, nil, "", err
	}
	return response.GetKind(), nil, token, nil
}

// publishCompletionAuthority builds the completion effect for the operation the activation answers
// asynchronously, publishes it as the opaque capability of the named handle Slot, and returns the
// operation token the reply carries: the delivery's own request id.
func (s *Session) publishCompletionAuthority(ctx context.Context, delivered delivery.Activation, handleSlotID string, options nexus.StartOperationOptions) (string, error) {
	if s.options.NewCapability == nil || nilValue(s.options.Bridge) {
		return "", ErrInvalid
	}
	invoke, err := s.host.options.completion.newEffect(completionInfo{URL: options.CallbackURL, Header: maps.Clone(options.CallbackHeader), OperationToken: delivered.RequestID(), StartTime: s.host.options.now()})
	if err != nil {
		return "", err
	}
	capability, err := s.options.NewCapability(ctx, delivered.Coordinate(), invoke)
	if err != nil {
		return "", err
	}
	if nilValue(capability) {
		return "", ErrInvalid
	}
	if err := s.publicationAllowed(ctx); err != nil {
		s.lateDiagnostic(ctx, "completion_publication_late")
		return "", err
	}
	if err := s.options.Bridge.Publish(ctx, delivered.Coordinate(), handleSlotID, capability); err != nil {
		if errors.Is(s.publicationAllowed(ctx), ErrClosed) {
			s.lateDiagnostic(ctx, "completion_publication_late")
		}
		return "", err
	}
	return delivered.RequestID(), nil
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

func (r *nexusResult) response() (nexus.HandlerStartOperationResult[any], error) {
	if r.err != nil {
		return nil, r.err
	}
	switch r.kind {
	case testpilotspb.NEXUS_RESPONSE_KIND_SYNCHRONOUS:
		if r.raw != nil {
			return &nexus.HandlerStartOperationResultSync[any]{Value: converter.NewRawValue(proto.CloneOf(r.raw))}, nil
		}
		return &nexus.HandlerStartOperationResultSync[any]{Value: proto.CloneOf(r.value)}, nil
	case testpilotspb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS:
		return &nexus.HandlerStartOperationResultAsync{OperationToken: r.token}, nil
	default:
		return nil, ErrInvalid
	}
}

func sdkFailureOutcome(err error) *testpilotspb.InstructionOutcome {
	outcome := &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE, SdkFailureCode: "sdk_failure", Detail: boundedText(err.Error())}
	if temporal.IsCanceledError(err) || errors.Is(err, context.Canceled) {
		outcome.Status = testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED
		outcome.SdkFailureCode = "canceled"
	} else if temporal.IsTimeoutError(err) || errors.Is(err, context.DeadlineExceeded) {
		outcome.Status = testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT
		outcome.SdkFailureCode = "timed_out"
	} else {
		var applicationError *temporal.ApplicationError
		if errors.As(err, &applicationError) && applicationError.Type() != "" {
			outcome.SdkFailureCode = boundedText(applicationError.Type())
		}
	}
	return outcome
}
