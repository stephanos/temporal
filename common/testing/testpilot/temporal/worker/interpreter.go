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

// replyKind names how a Nexus handler entrypoint answered its activation: with a result, through a
// completion handle the caller polls, or with an error. It is the Driver's own classification of a
// NexusHandlerReply, not a protocol value.
type replyKind uint8

const (
	replyUnspecified replyKind = iota
	replySynchronous
	replyAsynchronous
	replyError
)

type nexusResult struct {
	done  chan struct{}
	kind  replyKind
	value *testpilotspb.Value
	// raw is a typed synchronous reply's payload, returned unconverted.
	raw   *commonpb.Payload
	token string
	err   error
	// replied marks err as the reply the entrypoint instructed, a handler or operation error the
	// handler returns on purpose, so the activation that produced it completed rather than failed.
	replied bool
	// retryable marks a replied handler error the caller retries: the activation stays open, and the
	// retried delivery resumes the entrypoint at its next reply instruction.
	retryable bool
	// state and next carry an open activation across deliveries: the activation's values and the
	// position in the entrypoint's order the next delivery resumes at.
	state *activation.State
	next  int
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

// terminalOutcome is the outcome of a Finish or a NexusHandlerReply that ended its activation. Its result is
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
	existing := s.nexusResults[key]
	if existing != nil && !existing.retryable {
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
	result := existing
	if result == nil {
		if len(s.nexusResults) >= boundedInt(s.definition.limits.GetMaxActivations()) {
			s.mu.unlock()
			return nil, ErrCapacity
		}
		result = &nexusResult{}
		s.nexusResults[key] = result
	}
	// A retried delivery reopens the activation its retryable reply left open: the same entry, with
	// the reply the last delivery answered behind it.
	result.retryable = false
	result.done = make(chan struct{})
	s.mu.unlock()

	func() {
		defer func() {
			if recover() != nil {
				result.err = errors.New("nexus handler activation panicked")
			}
		}()
		interpreted, err := s.interpretNexus(ctx, delivered, options, result)
		result.kind, result.value, result.raw, result.token, result.err = interpreted.kind, interpreted.value, interpreted.raw, interpreted.token, err
		result.replied = interpreted.replied
	}()
	// Whether the next delivery on this route resumes or replays is read under the lock, so it is
	// written there too, before the waiters are released.
	var handlerErr *nexus.HandlerError
	retryable := result.replied && result.err != nil && errors.As(result.err, &handlerErr) && handlerErr.Retryable()
	if s.mu.lock(context.Background()) == nil {
		result.retryable = retryable
		s.mu.unlock()
	}
	close(result.done)
	return result.response()
}

// interpretNexus runs the handler entrypoint from where its activation stands: from its first
// instruction on the first delivery, and from the instruction after the reply the last delivery
// answered when that reply was a retryable handler error the caller retried. The activation's
// values are carried in `resume`, so the second reply's guards read what the first admitted.
func (s *Session) interpretNexus(ctx context.Context, delivered delivery.Activation, options nexus.StartOperationOptions, resume *nexusResult) (nexusResult, error) {
	entry, exists := s.definition.entries[delivered.Coordinate().EntrypointID]
	if !exists || entry.plan.Kind() != testpilot.NexusHandlerEntrypoint {
		return nexusResult{}, ErrInvalid
	}
	if resume.state == nil {
		state, err := activation.New(entry.plan)
		if err != nil {
			return nexusResult{}, err
		}
		resume.state, resume.next = state, 0
	}
	state := resume.state
	instructions := entry.plan.Instructions()
	order := entry.plan.Order()
	for position := resume.next; position < len(order); position++ {
		index := order[position]
		// A typed reply carries its own message, so the evaluated input is unused; the evaluation
		// still resolves the instruction's guard.
		_, enabled, err := state.Evaluate(ctx, index)
		if err != nil {
			return nexusResult{}, err
		}
		if !enabled {
			continue
		}
		instruction := instructions[index]
		switch instruction.Opcode() {
		case testpilot.NexusHandlerReply:
			reply := instruction.Source().GetInstruction().GetNexusHandlerReply()
			if err := state.Admit(ctx, index, terminalOutcome()); err != nil {
				return nexusResult{}, err
			}
			resume.next = position + 1
			return s.replyTyped(ctx, delivered, reply, options)
		default:
			return nexusResult{}, ErrInvalid
		}
	}
	return nexusResult{}, errors.New("nexus handler entrypoint completed without a reply")
}

// nexusActivationOutcome is what a start reports of its handler activation, and whether it reports
// yet. A reply the entrypoint instructed completes the activation whatever the SDK carries back, an
// error included, since the handler did what the Program said; any other failed start failed the
// activation. A retryable handler error leaves the activation open for the retried delivery, so
// nothing is reported until a later reply settles it.
func (s *Session) nexusActivationOutcome(delivered delivery.Activation, startErr error) (outcome *testpilotspb.InstructionOutcome, open bool, activationErr error) {
	if startErr == nil {
		return terminalOutcome(), false, nil
	}
	if s.mu.lock(context.Background()) == nil {
		result := s.nexusResults[delivered.Reservation().ID]
		s.mu.unlock()
		if result != nil && result.retryable {
			return nil, true, nil
		}
		if result != nil && result.replied {
			return terminalOutcome(), false, nil
		}
	}
	return sdkFailureOutcome(startErr), false, startErr
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
	case replySynchronous:
		if r.raw != nil {
			return &nexus.HandlerStartOperationResultSync[any]{Value: converter.NewRawValue(proto.CloneOf(r.raw))}, nil
		}
		return &nexus.HandlerStartOperationResultSync[any]{Value: proto.CloneOf(r.value)}, nil
	case replyAsynchronous:
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
