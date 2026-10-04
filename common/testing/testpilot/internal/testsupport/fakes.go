package testsupport

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ErrUnscripted is what a Session method its test did not script returns.
var ErrUnscripted = errors.New("testsupport: unscripted Session call")

// Effect is a scripted EffectHandle. Wait answers a copy of Result with WaitErr once the effect
// completes, or the context's error first; OnWait, when set, answers instead. Cancel honors its
// context, then runs OnCancel, or completes the effect when CancelCompletes is set. Drain returns
// DrainErr at once when set and otherwise waits for completion.
type Effect struct {
	Result          contract.EffectResult
	WaitErr         error
	DrainErr        error
	OnWait          func(context.Context) (contract.EffectResult, error)
	OnCancel        func(context.Context) error
	CancelCompletes bool

	init     sync.Once
	done     chan struct{}
	complete sync.Once
	cancels  atomic.Int64
}

// Completed is an effect that has already completed with result.
func Completed(result contract.EffectResult) *Effect {
	effect := &Effect{Result: result}
	effect.Complete()
	return effect
}

// Complete completes the effect; completing it again does nothing.
func (e *Effect) Complete() {
	done := e.doneChannel()
	e.complete.Do(func() { close(done) })
}

// Done is closed once the effect completes.
func (e *Effect) Done() <-chan struct{} { return e.doneChannel() }

// Cancels is how many calls to Cancel got past their context.
func (e *Effect) Cancels() int64 { return e.cancels.Load() }

func (e *Effect) doneChannel() chan struct{} {
	e.init.Do(func() { e.done = make(chan struct{}) })
	return e.done
}

func (e *Effect) Wait(ctx context.Context) (contract.EffectResult, error) {
	if e.OnWait != nil {
		return e.OnWait(ctx)
	}
	done := e.doneChannel()
	select {
	case <-done:
	case <-ctx.Done():
		select {
		case <-done:
		default:
			return contract.EffectResult{}, ctx.Err()
		}
	}
	return contract.EffectResult{Outcome: proto.CloneOf(e.Result.Outcome), Response: proto.Clone(e.Result.Response)}, e.WaitErr
}

func (e *Effect) Cancel(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e.cancels.Add(1)
	if e.OnCancel != nil {
		return e.OnCancel(ctx)
	}
	if e.CancelCompletes {
		e.Complete()
	}
	return nil
}

func (e *Effect) Drain(ctx context.Context) error {
	if e.DrainErr != nil {
		return e.DrainErr
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.doneChannel():
		return nil
	}
}

// Reservation is a scripted ReservationHandle over an Effect. Consume counts each call and answers
// OnConsume when set, and Activation otherwise.
type Reservation struct {
	*Effect
	ID         contract.ReservationIdentity
	Activation contract.Coordinate
	OnConsume  func(context.Context) (contract.Coordinate, error)

	consumes atomic.Int64
}

// NewReservation is a pending reservation that consumes into the activation its identity names and
// then completes successfully.
func NewReservation(identity contract.ReservationIdentity) *Reservation {
	return &Reservation{
		Effect:     &Effect{Result: contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}},
		ID:         identity,
		Activation: contract.Coordinate{RunID: identity.Origin.RunID, EntrypointID: identity.EntrypointID, ActivationID: identity.ID},
	}
}

func (r *Reservation) Identity() contract.ReservationIdentity { return r.ID }

// Consumes is how many times Consume was called.
func (r *Reservation) Consumes() int64 { return r.consumes.Load() }

func (r *Reservation) Consume(ctx context.Context) (contract.Coordinate, error) {
	r.consumes.Add(1)
	if r.OnConsume != nil {
		return r.OnConsume(ctx)
	}
	return r.Activation, nil
}

// Session is a scripted Session that counts every call by method name. A method runs its On script
// when set. Unscripted, Bridge answers HandleBridge, Close and Diagnose succeed, PollRPC invokes
// through the InvokeRPC script and applies the predicate to the completed response, and every
// other method returns ErrUnscripted.
type Session struct {
	OnReserve      func(context.Context, contract.ReservationRequest) ([]contract.ReservationHandle, error)
	OnInvokeRPC    func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (contract.EffectHandle, error)
	OnPollRPC      func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message, time.Duration, contract.PollPredicate) (contract.EffectHandle, error)
	OnInvokeHandle func(context.Context, contract.Coordinate, contract.OpaqueHandle, proto.Message) (contract.EffectHandle, error)
	OnInjectFault  func(context.Context, contract.Coordinate, string, testpilotspb.FaultKind) (contract.EffectHandle, error)
	OnQuarantine   func(context.Context, contract.EffectHandle) error
	OnClose        func(context.Context) error
	OnDiagnose     func(context.Context, string, *testpilotspb.RunDiagnostic) error
	HandleBridge   contract.HandleBridge

	mu    sync.Mutex
	calls map[string]int
}

// Calls is how many times method was called.
func (s *Session) Calls(method string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[method]
}

func (s *Session) record(method string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.calls == nil {
		s.calls = make(map[string]int)
	}
	s.calls[method]++
}

func (s *Session) Reserve(ctx context.Context, request contract.ReservationRequest) ([]contract.ReservationHandle, error) {
	s.record("Reserve")
	if s.OnReserve == nil {
		return nil, ErrUnscripted
	}
	return s.OnReserve(ctx, request)
}

func (s *Session) InvokeRPC(ctx context.Context, coordinate contract.Coordinate, role string, method protoreflect.MethodDescriptor, request proto.Message) (contract.EffectHandle, error) {
	s.record("InvokeRPC")
	if s.OnInvokeRPC == nil {
		return nil, ErrUnscripted
	}
	return s.OnInvokeRPC(ctx, coordinate, role, method, request)
}

func (s *Session) PollRPC(ctx context.Context, coordinate contract.Coordinate, role string, method protoreflect.MethodDescriptor, request proto.Message, interval time.Duration, satisfied contract.PollPredicate) (contract.EffectHandle, error) {
	s.record("PollRPC")
	if s.OnPollRPC != nil {
		return s.OnPollRPC(ctx, coordinate, role, method, request, interval, satisfied)
	}
	if s.OnInvokeRPC == nil {
		return nil, ErrUnscripted
	}
	if interval < 0 || satisfied == nil {
		return nil, errors.New("testsupport: a poll requires a non-negative interval and a predicate")
	}
	handle, err := s.OnInvokeRPC(ctx, coordinate, role, method, request)
	if err != nil {
		return nil, err
	}
	result, err := handle.Wait(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := satisfied(ctx, result.Response); err != nil {
		return nil, err
	}
	return handle, nil
}

func (s *Session) InvokeHandle(ctx context.Context, coordinate contract.Coordinate, handle contract.OpaqueHandle, input proto.Message) (contract.EffectHandle, error) {
	s.record("InvokeHandle")
	if s.OnInvokeHandle == nil {
		return nil, ErrUnscripted
	}
	return s.OnInvokeHandle(ctx, coordinate, handle, input)
}

func (s *Session) InjectFault(ctx context.Context, coordinate contract.Coordinate, role string, kind testpilotspb.FaultKind) (contract.EffectHandle, error) {
	s.record("InjectFault")
	if s.OnInjectFault == nil {
		return nil, ErrUnscripted
	}
	return s.OnInjectFault(ctx, coordinate, role, kind)
}

func (s *Session) Bridge(context.Context) (contract.HandleBridge, error) {
	s.record("Bridge")
	return s.HandleBridge, nil
}

func (s *Session) Quarantine(ctx context.Context, handle contract.EffectHandle) error {
	s.record("Quarantine")
	if s.OnQuarantine == nil {
		return ErrUnscripted
	}
	return s.OnQuarantine(ctx, handle)
}

func (s *Session) Close(ctx context.Context) error {
	s.record("Close")
	if s.OnClose == nil {
		return nil
	}
	return s.OnClose(ctx)
}

func (s *Session) Diagnose(ctx context.Context, runID string, diagnostic *testpilotspb.RunDiagnostic) error {
	s.record("Diagnose")
	if s.OnDiagnose == nil {
		return nil
	}
	return s.OnDiagnose(ctx, runID, diagnostic)
}

var (
	_ contract.Session           = (*Session)(nil)
	_ contract.EffectHandle      = (*Effect)(nil)
	_ contract.ReservationHandle = (*Reservation)(nil)
)
