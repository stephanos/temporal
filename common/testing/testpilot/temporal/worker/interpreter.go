package worker

import (
	"context"
	"errors"
	"maps"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
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
	token string
	err   error
}

type workflowInterpreter struct {
	session *Session
	ctx     workflow.Context
	state   *activation.State
	futures map[string]workflow.NexusOperationFuture
}

func (s *Session) executeWorkflow(ctx workflow.Context, delivered delivery.Activation) (*testpilotspb.Value, error) {
	entry, exists := s.definition.entries[delivered.Coordinate().EntrypointID]
	if !exists || entry.plan.Context() != testpilotspb.ENTRYPOINT_KIND_WORKFLOW {
		return nil, ErrInvalid
	}
	state, err := activation.New(entry.plan)
	if err != nil {
		return nil, err
	}
	interpreter := workflowInterpreter{session: s, ctx: ctx, state: state, futures: make(map[string]workflow.NexusOperationFuture)}
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
	case testpilot.Await:
		return nil, false, i.awaitNexus(index, instruction)
	case testpilot.Finish:
		outcome := terminalOutcome(instruction, input)
		if err := i.state.Admit(context.Background(), index, outcome); err != nil {
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
			ScheduleToCloseTimeout: time.Duration(source.GetLimits().GetTimeoutMilliseconds()) * time.Millisecond,
			CancellationType:       workflow.NexusOperationCancellationTypeWaitRequested,
		},
	)
	i.futures[source.GetInstructionId()] = future
	var execution workflow.NexusOperationExecution
	err := future.GetNexusOperationExecution().Get(i.ctx, &execution)
	return i.state.Admit(context.Background(), index, outcomeForError(err))
}

func (i *workflowInterpreter) awaitNexus(index int, instruction testpilot.InstructionPlan) error {
	await := instruction.Source().GetInstruction().GetAwaitOutcome()
	future := i.futures[await.GetInstruction().GetInstructionId()]
	if future == nil {
		return ErrInvalid
	}
	var result testpilotspb.Value
	ready := future.IsReady()
	var err error
	if !ready {
		ready, err = workflow.AwaitWithTimeout(i.ctx, time.Duration(instruction.Source().GetLimits().GetTimeoutMilliseconds())*time.Millisecond, future.IsReady)
	}
	if err == nil {
		if ready {
			err = future.Get(i.ctx, &result)
		} else {
			err = context.DeadlineExceeded
		}
	}
	outcome := outcomeForError(err)
	if err == nil {
		outcome.Value = &result
	}
	return i.state.Admit(context.Background(), index, outcome)
}

func terminalOutcome(instruction testpilot.InstructionPlan, input *testpilotspb.Value) *testpilotspb.InstructionOutcome {
	outcome := &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}
	for _, field := range instruction.Source().GetOutcome().GetFields() {
		if field.GetField() == testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE {
			outcome.Value = proto.CloneOf(input)
			break
		}
	}
	return outcome
}

func outcomeForError(err error) *testpilotspb.InstructionOutcome {
	if err != nil {
		return sdkFailureOutcome(err)
	}
	return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}
}

func (s *Session) executeNexus(ctx context.Context, delivered delivery.Activation, _ *testpilotspb.Value, options nexus.StartOperationOptions) (nexus.HandlerStartOperationResult[*testpilotspb.Value], error) {
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
		result.kind, result.value, result.token, result.err = s.interpretNexus(ctx, delivered, options)
	}()
	close(result.done)
	return result.response()
}

func (s *Session) interpretNexus(ctx context.Context, delivered delivery.Activation, options nexus.StartOperationOptions) (testpilotspb.NexusResponseKind, *testpilotspb.Value, string, error) {
	entry, exists := s.definition.entries[delivered.Coordinate().EntrypointID]
	if !exists || entry.plan.Context() != testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER {
		return 0, nil, "", ErrInvalid
	}
	state, err := activation.New(entry.plan)
	if err != nil {
		return 0, nil, "", err
	}
	instructions := entry.plan.Instructions()
	for _, index := range entry.plan.Order() {
		input, enabled, err := state.Evaluate(ctx, index)
		if err != nil {
			return 0, nil, "", err
		}
		if !enabled {
			continue
		}
		instruction := instructions[index]
		if instruction.Opcode() != testpilot.RespondNexus {
			return 0, nil, "", ErrInvalid
		}
		response := instruction.Source().GetInstruction().GetRespondNexus()
		if err := state.Admit(ctx, index, terminalOutcome(instruction, input)); err != nil {
			return 0, nil, "", err
		}
		return s.respondNexus(ctx, delivered, response, input, options)
	}
	return 0, nil, "", errors.New("nexus handler entrypoint completed without RespondNexus")
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
		if input != nil && input.GetText() != "" {
			detail = input.GetText()
		}
		return response.GetKind(), nil, "", &nexus.HandlerError{Type: nexus.HandlerErrorTypeInternal, Message: boundedText(detail), RetryBehavior: nexus.HandlerErrorRetryBehaviorNonRetryable}
	default:
		return 0, nil, "", ErrInvalid
	}
}

func (s *Session) respondNexusAsync(ctx context.Context, delivered delivery.Activation, response *testpilotspb.RespondNexus, input *testpilotspb.Value, options nexus.StartOperationOptions) (testpilotspb.NexusResponseKind, *testpilotspb.Value, string, error) {
	if input == nil || s.options.NewCapability == nil || nilValue(s.options.Bridge) {
		return 0, nil, "", ErrInvalid
	}
	invoke, err := s.host.options.completion.newEffect(completionInfo{URL: options.CallbackURL, Header: maps.Clone(options.CallbackHeader), OperationToken: delivered.RequestID(), StartTime: s.host.options.now()})
	if err != nil {
		return 0, nil, "", err
	}
	capability, err := s.options.NewCapability(ctx, delivered.Coordinate(), invoke)
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
	if err := s.options.Bridge.Publish(ctx, delivered.Coordinate(), response.GetCapabilitySlotId(), capability); err != nil {
		if errors.Is(s.publicationAllowed(ctx), ErrClosed) {
			s.lateDiagnostic(ctx, "completion_publication_late")
		}
		return 0, nil, "", err
	}
	return response.GetKind(), nil, delivered.RequestID(), nil
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

func (r *nexusResult) response() (nexus.HandlerStartOperationResult[*testpilotspb.Value], error) {
	if r.err != nil {
		return nil, r.err
	}
	switch r.kind {
	case testpilotspb.NEXUS_RESPONSE_KIND_SYNCHRONOUS:
		return &nexus.HandlerStartOperationResultSync[*testpilotspb.Value]{Value: proto.CloneOf(r.value)}, nil
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
