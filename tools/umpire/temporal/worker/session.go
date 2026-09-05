package worker

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/nexus-rpc/sdk-go/nexus"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"go.temporal.io/server/tools/umpire/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type programDefinition struct {
	snapshot       *umpirespb.Program
	entries        map[string]entryDefinition
	registrations  []queueRegistration
	endpoints      map[string]string
	queueWorkflows map[string]map[string]struct{}
	hasAsync       bool
}

type entryDefinition struct {
	plan                umpire.EntrypointPlan
	queue, workflowType string
	service, operation  string
}

type Session struct {
	host               *Host
	mu                 contextMutex
	closeMu            contextMutex
	runID              string
	id                 string
	definition         programDefinition
	ledger             *delivery.Ledger
	options            SessionOptions
	reservations       map[string]*reservation
	carriers           map[umpire.Coordinate]*Carrier
	nexusResults       map[string]*nexusResult
	workflowKeys       map[workflowRouteIndex]struct{}
	nexusKeys          map[nexusRouteIndex]struct{}
	nexusDispatch      map[nexusDispatchKey]nexus.Header
	workflowAdmissions map[workflowAdmissionKey]*workflowAdmission
	nexusAdmissions    map[nexusRouteIndex]nexusAdmission
	next               atomic.Uint64
	diagnostics        int
	closed             bool
	failure            error
	releaseWorkers     func(context.Context) error
	stopComplete       bool
	released           bool
	removed            bool
}

func newSession(host *Host, runID, sessionID string, definition programDefinition, options SessionOptions) (*Session, error) {
	if host == nil || definition.snapshot == nil || definition.snapshot.GetLimits() == nil {
		return nil, ErrInvalid
	}
	limits := definition.snapshot.GetLimits()
	ledger, err := delivery.New(delivery.Config{RunID: runID, SessionID: sessionID, Limits: delivery.Limits{
		MaxRoutes: boundedInt(limits.GetMaxActivations()), MaxHandles: boundedInt(limits.GetMaxActivations()),
		MaxHeaderBytes: boundedInt(limits.GetMaxRequestBytes()), MaxDiagnostics: boundedInt(limits.GetMaxRunEvents()),
	}})
	if err != nil {
		return nil, err
	}
	return &Session{host: host, mu: newContextMutex(), closeMu: newContextMutex(), runID: runID, id: sessionID, definition: definition, ledger: ledger, options: options,
		reservations: make(map[string]*reservation), carriers: make(map[umpire.Coordinate]*Carrier), nexusResults: make(map[string]*nexusResult),
		workflowKeys: make(map[workflowRouteIndex]struct{}), nexusKeys: make(map[nexusRouteIndex]struct{}), nexusDispatch: make(map[nexusDispatchKey]nexus.Header),
		workflowAdmissions: make(map[workflowAdmissionKey]*workflowAdmission), nexusAdmissions: make(map[nexusRouteIndex]nexusAdmission)}, nil
}

func (s *Session) Reserve(ctx context.Context, request umpire.ReservationRequest) ([]umpire.ReservationHandle, error) {
	if s == nil || ctx == nil || request.Origin.RunID != s.runID || request.EntrypointID == "" || request.Count <= 0 {
		return nil, ErrInvalid
	}
	entry, exists := s.definition.entries[request.EntrypointID]
	if !exists || entry.plan.Context() != umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW && entry.plan.Context() != umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER {
		return nil, ErrInvalid
	}
	if err := s.mu.lock(ctx); err != nil {
		return nil, err
	}
	defer s.mu.unlock()
	if s.closed || s.failure != nil {
		return nil, errors.Join(ErrClosed, s.failure)
	}
	maximum := boundedInt(s.definition.snapshot.GetLimits().GetMaxActivations())
	if int(request.Count) > maximum-len(s.reservations) {
		return nil, ErrCapacity
	}
	result := make([]umpire.ReservationHandle, 0, request.Count)
	for ordinal := int64(0); ordinal < request.Count; ordinal++ {
		identity := umpire.ReservationIdentity{Origin: request.Origin, EntrypointID: request.EntrypointID, Ordinal: ordinal, ID: fmt.Sprintf("reservation-%d", s.next.Add(1))}
		raw := newReservation(identity)
		retained, err := s.ledger.RetainReservation(ctx, raw)
		if err != nil {
			raw.finish(umpire.EffectResult{}, err)
			cleanupCtx, cancel := s.host.cleanupContext()
			for _, handle := range result {
				_ = handle.Cancel(cleanupCtx)
				delete(s.reservations, handle.Identity().ID)
			}
			cancel()
			return nil, err
		}
		s.reservations[identity.ID] = raw
		result = append(result, retained)
	}
	return result, nil
}

func (*Session) InvokeRPC(context.Context, umpire.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (umpire.EffectHandle, error) {
	return nil, ErrUnsupportedOperation
}

func (*Session) CompleteNexusOperation(context.Context, umpire.Coordinate, umpire.OpaqueCapability, *umpirespb.Value) (umpire.EffectHandle, error) {
	return nil, ErrUnsupportedOperation
}

func (s *Session) Bridge(ctx context.Context) (umpire.CapabilityBridge, error) {
	if s == nil || ctx == nil {
		return nil, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if nilValue(s.options.Bridge) {
		return nil, ErrInvalid
	}
	return s.options.Bridge, nil
}

func (s *Session) Quarantine(ctx context.Context, handle umpire.EffectHandle) error {
	if s == nil || s.options.Quarantine == nil {
		return ErrInvalid
	}
	return s.ledger.Quarantine(ctx, handle, func(ctx context.Context, handle umpire.EffectHandle, complete delivery.CompletionFunc) error {
		return s.options.Quarantine(ctx, handle, func() { complete() })
	})
}

func (s *Session) Close(ctx context.Context) error {
	if s == nil || ctx == nil {
		return ErrInvalid
	}
	if err := s.closeMu.lock(ctx); err != nil {
		return err
	}
	defer s.closeMu.unlock()
	if err := s.mu.lock(ctx); err != nil {
		return err
	}
	s.closed = true
	releaseWorkers := s.releaseWorkers
	s.mu.unlock()
	if !s.stopComplete {
		if _, err := s.ledger.Stop(ctx); err != nil {
			return err
		}
		s.stopComplete = true
	}
	if !s.released && releaseWorkers != nil {
		if err := releaseWorkers(ctx); err != nil {
			return err
		}
		s.released = true
	}
	if !s.removed {
		if err := s.host.removeSession(ctx, s, true); err != nil {
			return err
		}
		s.removed = true
	}
	return nil
}

func (s *Session) Diagnose(ctx context.Context, runID string, diagnostic *umpirespb.RunDiagnostic) error {
	if s == nil || ctx == nil || runID != s.runID || diagnostic == nil {
		return ErrInvalid
	}
	if err := s.mu.lock(ctx); err != nil {
		return err
	}
	if s.diagnostics >= boundedInt(s.definition.snapshot.GetLimits().GetMaxRunEvents()) {
		s.mu.unlock()
		return ErrCapacity
	}
	s.diagnostics++
	sink := s.options.Diagnose
	s.mu.unlock()
	if sink == nil {
		return nil
	}
	return sink(ctx, runID, proto.CloneOf(diagnostic))
}

func (s *Session) workerFailed(queue string, failure error) {
	if failure == nil {
		return
	}
	ctx, cancel := s.host.cleanupContext()
	defer cancel()
	if s.mu.lock(ctx) != nil {
		return
	}
	if s.failure == nil && s.dependsOnQueue(queue) {
		s.failure = failure
	}
	s.mu.unlock()
}

func (s *Session) dependsOnQueue(queue string) bool {
	for _, registration := range s.definition.registrations {
		if registration.queue == queue {
			return true
		}
	}
	return false
}

func (s *Session) rawReservation(id string) (*reservation, error) {
	if s.mu.lock(context.Background()) != nil {
		return nil, ErrClosed
	}
	defer s.mu.unlock()
	raw := s.reservations[id]
	if raw == nil {
		return nil, ErrClosed
	}
	return raw, nil
}

func (s *Session) finishActivation(activation delivery.Activation, outcome *umpirespb.InstructionOutcome, err error) {
	raw, lookupErr := s.rawReservation(activation.Reservation().ID)
	if lookupErr == nil {
		raw.finish(umpire.EffectResult{Outcome: outcome}, err)
	}
}

func (h *Host) removeSession(ctx context.Context, session *Session, tombstone bool) error {
	if h == nil || session == nil || ctx == nil {
		return ErrInvalid
	}
	if err := h.mu.lock(ctx); err != nil {
		return err
	}
	if h.sessions[session.runID] != session {
		h.mu.unlock()
		return nil
	}
	delete(h.sessions, session.runID)
	if !tombstone || h.options.diagnostics == 0 {
		h.removeRouteIndexesLocked(session)
	} else {
		if len(h.tombstones) == h.options.diagnostics {
			h.evictOldestTombstoneLocked()
		}
		h.tombstones = append(h.tombstones, session)
	}
	h.mu.unlock()
	return nil
}
