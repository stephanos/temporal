package worker

import (
	"context"
	"errors"
	"maps"
	"sync"
	"time"

	celpb "cel.dev/expr"
	"github.com/nexus-rpc/sdk-go/nexus"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/activation"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
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
	done chan struct{}
	kind replyKind
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
	// futures are the scheduled commands' results an Await may read, by schedule instruction.
	futures map[string]workflow.Future
}

func (s *Session) executeWorkflow(ctx workflow.Context, delivered delivery.Activation) (*celpb.Value, error) {
	entry, exists := s.definition.entries[delivered.Coordinate().EntrypointID]
	if !exists || entry.plan.Kind() != testpilot.WorkflowEntrypoint {
		return nil, ErrInvalid
	}
	state, err := activation.New(entry.plan)
	if err != nil {
		return nil, err
	}
	interpreter := workflowInterpreter{session: s, ctx: ctx, state: state, futures: make(map[string]workflow.Future)}
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

func (i *workflowInterpreter) execute(index int, instruction testpilot.InstructionPlan, input *celpb.Value) (*celpb.Value, bool, error) {
	switch instruction.Opcode() {
	case testpilot.WorkflowCommand:
		return nil, false, i.issueCommand(index, instruction)
	case testpilot.Await:
		return nil, false, i.await(index, instruction)
	case testpilot.Finish:
		if err := i.state.Admit(context.Background(), index, terminalOutcome()); err != nil {
			return nil, false, err
		}
		return proto.CloneOf(input), true, nil
	default:
		return nil, false, ErrInvalid
	}
}

func (i *workflowInterpreter) await(index int, instruction testpilot.InstructionPlan) error {
	await := instruction.Source().GetInstruction().GetAwaitInstruction()
	future := i.futures[await.GetInstruction().GetInstructionId()]
	if future == nil {
		return ErrInvalid
	}
	var result *celpb.Value
	ready := future.IsReady()
	var err error
	if !ready {
		ready, err = workflow.AwaitWithTimeout(i.ctx, time.Duration(instruction.TimeoutMilliseconds())*time.Millisecond, future.IsReady)
	}
	if err == nil {
		if ready {
			result, err = awaitedPayload(i.ctx, future)
		} else {
			err = context.DeadlineExceeded
		}
	}
	outcome := outcomeForError(err)
	if err == nil {
		outcome.Value = result
	}
	return i.state.Admit(context.Background(), index, outcome)
}

// activityScriptKey names the script of one started activity: the instruction whose start carried
// its reservations, and the entrypoint they activate.
type activityScriptKey struct {
	origin       testpilot.Coordinate
	entrypointID string
}

// activityScript carries one activity's script across its attempts, as a Nexus handler's activation
// is carried across its deliveries: the values earlier attempts admitted, which later guards read.
// Attempts interpret it one at a time.
type activityScript struct {
	mu    sync.Mutex
	state *activation.State
}

// declaredFailure is the failure a script's instruction fails its attempt with, as the error the
// SDK reports for it. The attempt performed its instruction, so it is not an activation failure.
type declaredFailure struct {
	failure      error
	nonRetryable bool
}

func (f *declaredFailure) Error() string { return f.failure.Error() }
func (f *declaredFailure) Unwrap() error { return f.failure }

var errAttemptDisabled = errors.New("the activity attempt's instruction is disabled")

// errDeclaredCancellation is the answer of an attempt whose instruction answers a cancellation the
// server requested. The attempt performed its instruction, so it is not an activation failure.
var errDeclaredCancellation = errors.New("the activity attempt answers its requested cancellation")

// errDeclaredWithholding is the end of an attempt whose instruction withholds its answer, once the
// attempt's deadline ended it. The attempt performed its instruction, so it is not an activation
// failure.
var errDeclaredWithholding = errors.New("the activity attempt withheld its answer until its deadline")

// cancellationPoll is how often an attempt that answers a requested cancellation heartbeats to
// learn whether the server has asked for it.
const cancellationPoll = 100 * time.Millisecond

// awaitCancellationRequest heartbeats the activity attempt whose context ctx is until the server
// answers a heartbeat with the activity's requested cancellation, which the SDK reports by ending
// the context with a canceled error as its cause. A worker may answer canceled only then: the SDK
// tells Temporal of no cancellation the server did not ask that delivery for. Any other end of the
// context is the attempt ending unasked, and is the error returned.
func (h *Driver) awaitCancellationRequest(ctx context.Context) error {
	ticker := time.NewTicker(cancellationPoll)
	defer ticker.Stop()
	for ctx.Err() == nil {
		h.options.heartbeat(ctx)
		select {
		case <-ctx.Done():
		case <-ticker.C:
		}
	}
	if temporal.IsCanceledError(context.Cause(ctx)) {
		return nil
	}
	return ctx.Err()
}

func (s *Session) scriptOf(delivered delivery.Activation) (*activityScript, error) {
	if err := s.mu.LockContext(context.Background(), ErrInvalid); err != nil {
		return nil, err
	}
	defer s.mu.Unlock()
	key := activityScriptKey{origin: delivered.Reservation().Origin, entrypointID: delivered.Coordinate().EntrypointID}
	script := s.activityScripts[key]
	if script == nil {
		script = &activityScript{}
		s.activityScripts[key] = script
	}
	return script, nil
}

// awaitEarlierAttempts blocks an attempt until every earlier attempt of its activity has settled.
// Temporal starts an attempt only after the one before it ended, but the worker may be handed them
// in another order, and an attempt's guard reads what the earlier ones admitted.
func (s *Session) awaitEarlierAttempts(ctx context.Context, delivered delivery.Activation) error {
	own := delivered.Reservation()
	if err := s.mu.LockContext(ctx, ErrInvalid); err != nil {
		return err
	}
	var earlier []*reservation
	for _, raw := range s.reservations {
		if raw.identity.Origin == own.Origin && raw.identity.EntrypointID == own.EntrypointID && raw.identity.Ordinal < own.Ordinal {
			earlier = append(earlier, raw)
		}
	}
	s.mu.Unlock()
	for _, raw := range earlier {
		if err := raw.Drain(ctx); err != nil {
			return err
		}
	}
	return nil
}

// executeActivity interprets an activity entrypoint for one admitted attempt. The script declares
// the activity's attempts in order and the attempt's reservation is the one of its number, so the
// attempt performs the group at its reservation's ordinal and no other, under the attempt's
// context, whose cancellation fails the evaluation. A Finish completes the attempt with its result,
// whatever that value is, an ActivityAttemptFailure fails it with the failure it carries, an
// ActivityAttemptCancellation answers it as canceled once the server has asked for that, and an
// ActivityAttemptWithholding either waits for its deadline or returns the SDK pending sentinel. An attempt
// whose instruction is disabled has nothing declared for it and is a failed activation.
func (s *Session) executeActivity(ctx context.Context, delivered delivery.Activation) (*celpb.Value, error) {
	entry, exists := s.definition.entries[delivered.Coordinate().EntrypointID]
	if !exists || entry.plan.Kind() != testpilot.ActivityEntrypoint {
		return nil, ErrInvalid
	}
	if err := s.awaitEarlierAttempts(ctx, delivered); err != nil {
		return nil, err
	}
	script, err := s.scriptOf(delivered)
	if err != nil {
		return nil, err
	}
	script.mu.Lock()
	defer script.mu.Unlock()
	if script.state == nil {
		state, err := activation.New(entry.plan)
		if err != nil {
			return nil, err
		}
		script.state = state
	}
	groups := entry.plan.ActivityAttempts()
	ordinal := delivered.Reservation().Ordinal
	if ordinal < 0 || ordinal >= int64(len(groups)) {
		return nil, ErrInvalid
	}
	group := groups[ordinal]
	instructions := entry.plan.Instructions()
	terminal := instructions[group[len(group)-1]].Source().GetInstruction().GetActivityAttemptWithholding()
	if terminal != nil && terminal.GetMode() == testpilotspb.ACTIVITY_WITHHOLDING_MODE_SDK_PENDING {
		info := activity.GetInfo(ctx)
		if info.ActivityRunID == "" || terminal.GetExternalSettlement() == nil && info.HeartbeatTimeout <= 0 {
			return nil, ErrInvalid
		}
	}
	for _, index := range group {
		instruction := instructions[index]
		input, enabled, err := script.state.Evaluate(ctx, index)
		if err != nil {
			return nil, err
		}
		if !enabled {
			return nil, errAttemptDisabled
		}
		switch instruction.Opcode() {
		case testpilot.ActivityHeartbeat:
			routed, ok := ctx.Value(activityRouteKey{}).(routedActivity)
			if !ok || routed.answer == nil {
				return nil, ErrInvalid
			}
			details := instruction.Source().GetInstruction().GetActivityHeartbeat().GetDetails()
			values := make([]interface{}, 0, len(details.GetPayloads()))
			for _, payload := range details.GetPayloads() {
				values = append(values, converter.NewRawValue(proto.CloneOf(payload)))
			}
			activity.RecordHeartbeat(ctx, values...)
			routed.answer.heartbeatInvoked = true
			if err := script.state.Admit(ctx, index, terminalOutcome()); err != nil {
				return nil, err
			}
		case testpilot.Finish:
			// The server may never accept the completion and issue the next attempt, whose guard reads
			// this instruction's outcome.
			if err := script.state.Admit(ctx, index, terminalOutcome()); err != nil {
				return nil, err
			}
			return proto.CloneOf(input), nil
		case testpilot.ActivityAttemptFailure:
			if err := script.state.Admit(ctx, index, terminalOutcome()); err != nil {
				return nil, err
			}
			failure := instruction.Source().GetInstruction().GetActivityAttemptFailure().GetFailure()
			return nil, &declaredFailure{
				failure:      temporal.GetDefaultFailureConverter().FailureToError(failure),
				nonRetryable: failure.GetApplicationFailureInfo().GetNonRetryable(),
			}
		case testpilot.ActivityAttemptCancellation:
			if err := s.host.awaitCancellationRequest(ctx); err != nil {
				return nil, err
			}
			// The request ended the attempt's context, and the server may still refuse the answer and
			// issue the next attempt, whose guard reads this instruction's outcome.
			if err := script.state.Admit(context.WithoutCancel(ctx), index, terminalOutcome()); err != nil {
				return nil, err
			}
			return nil, errDeclaredCancellation
		case testpilot.ActivityAttemptWithholding:
			if instruction.Source().GetInstruction().GetActivityAttemptWithholding().GetMode() == testpilotspb.ACTIVITY_WITHHOLDING_MODE_SDK_PENDING {
				if err := script.state.Admit(ctx, index, terminalOutcome()); err != nil {
					return nil, err
				}
				return nil, activity.ErrResultPending
			}
			<-ctx.Done()
			// Only the attempt's deadline ends it as declared; the Run canceling it ends it unasked.
			if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, ctx.Err()
			}
			// The server may still issue the next attempt, whose guard reads this instruction's outcome.
			if err := script.state.Admit(context.WithoutCancel(ctx), index, terminalOutcome()); err != nil {
				return nil, err
			}
			return nil, errDeclaredWithholding
		default:
			return nil, ErrInvalid
		}
	}
	return nil, ErrInvalid
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
	if err := s.mu.LockContext(ctx, ErrInvalid); err != nil {
		return nil, err
	}
	existing := s.nexusResults[key]
	if existing != nil && !existing.retryable {
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-existing.done:
			return existing.response()
		}
	}
	if s.closed || s.failure != nil {
		s.mu.Unlock()
		return nil, errors.Join(ErrClosed, s.failure)
	}
	result := existing
	if result == nil {
		if len(s.nexusResults) >= boundedInt(s.definition.limits.GetMaxActivations()) {
			s.mu.Unlock()
			return nil, ErrCapacity
		}
		result = &nexusResult{}
		s.nexusResults[key] = result
	}
	// A retried delivery reopens the activation its retryable reply left open: the same entry, with
	// the reply the last delivery answered behind it.
	result.retryable = false
	result.done = make(chan struct{})
	s.mu.Unlock()

	func() {
		defer func() {
			if recover() != nil {
				result.err = errors.New("nexus handler activation panicked")
			}
		}()
		interpreted, err := s.interpretNexus(ctx, delivered, options, result)
		result.kind, result.raw, result.token, result.err = interpreted.kind, interpreted.raw, interpreted.token, err
		result.replied = interpreted.replied
	}()
	// Whether the next delivery on this route resumes or replays is read under the lock, so it is
	// written there too, before the waiters are released.
	var handlerErr *nexus.HandlerError
	retryable := result.replied && result.err != nil && errors.As(result.err, &handlerErr) && handlerErr.Retryable()
	if s.mu.LockContext(context.Background(), ErrInvalid) == nil {
		result.retryable = retryable
		s.mu.Unlock()
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
	if s.mu.LockContext(context.Background(), ErrInvalid) == nil {
		result := s.nexusResults[delivered.Reservation().ID]
		s.mu.Unlock()
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
// asynchronously, publishes it as the opaque handle of the named handle Slot, and returns the
// operation token the reply carries: the delivery's own request id.
func (s *Session) publishCompletionAuthority(ctx context.Context, delivered delivery.Activation, handleSlotID string, options nexus.StartOperationOptions) (string, error) {
	if s.options.NewHandle == nil || primitive.NilValue(s.options.Bridge) {
		return "", ErrInvalid
	}
	invoke, err := s.host.options.completion.newEffect(completionInfo{URL: options.CallbackURL, Header: maps.Clone(options.CallbackHeader), OperationToken: delivered.RequestID(), StartTime: s.host.options.now()})
	if err != nil {
		return "", err
	}
	handle, err := s.options.NewHandle(ctx, delivered.Coordinate(), invoke)
	if err != nil {
		return "", err
	}
	if primitive.NilValue(handle) {
		return "", ErrInvalid
	}
	if err := s.publicationAllowed(ctx); err != nil {
		s.lateDiagnostic(ctx, "completion_publication_late")
		return "", err
	}
	if err := s.options.Bridge.Publish(ctx, delivered.Coordinate(), handleSlotID, handle); err != nil {
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
	if err := s.mu.LockContext(lockCtx, ErrInvalid); err != nil {
		return err
	}
	defer s.mu.Unlock()
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
		// A reply without a payload answers nil, which the SDK sends as its nil payload.
		if r.raw == nil {
			return &nexus.HandlerStartOperationResultSync[any]{}, nil
		}
		return &nexus.HandlerStartOperationResultSync[any]{Value: converter.NewRawValue(proto.CloneOf(r.raw))}, nil
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
