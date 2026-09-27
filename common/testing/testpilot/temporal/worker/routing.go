package worker

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"

	"github.com/nexus-rpc/sdk-go/nexus"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
	"google.golang.org/protobuf/proto"
)

// workflowRouteIndex is the key the Driver indexes workflow routes by.
type workflowRouteIndex = delivery.WorkflowBinding

type nexusRouteIndex struct {
	name, value string
}

type nexusDispatchKey struct {
	workflowReservation, sourceInstruction string
}

type workflowAdmissionKey struct {
	route  delivery.WorkflowBinding
	header string
}

type workflowAdmission struct {
	activation    delivery.Activation
	temporalRunID string
	mu            sync.Mutex
	terminal      bool
}

type nexusAdmission struct {
	activation delivery.Activation
	requestID  string
}

func workflowAdmissionKeyFor(input delivery.WorkflowDelivery, maximumBytes int64) (workflowAdmissionKey, error) {
	if input.Header == nil || int64(proto.Size(input.Header)) > maximumBytes {
		return workflowAdmissionKey{}, ErrInvalid
	}
	wire, err := (proto.MarshalOptions{Deterministic: true}).Marshal(input.Header)
	if err != nil {
		return workflowAdmissionKey{}, err
	}
	return workflowAdmissionKey{route: input.Binding(), header: string(wire)}, nil
}

func nexusRouteIndexFromHeader(header nexus.Header) (nexusRouteIndex, error) {
	if len(header) != 1 {
		return nexusRouteIndex{}, ErrInvalid
	}
	for name, value := range header {
		return nexusRouteIndex{name: name, value: value}, nil
	}
	return nexusRouteIndex{}, ErrInvalid
}

func (h *Driver) workflowCandidates(input delivery.WorkflowDelivery) []*Session {
	if h == nil || h.mu.LockContext(context.Background(), ErrInvalid) != nil {
		return nil
	}
	defer h.mu.Unlock()
	return append([]*Session(nil), h.workflowRoutes[input.Binding()]...)
}

func (h *Driver) nexusCandidates(ctx context.Context, header nexus.Header) ([]*Session, error) {
	if h == nil || ctx == nil || int64(primitive.NexusHeaderBytes(header)) > h.options.requestBytes {
		return nil, ErrInvalid
	}
	if err := h.mu.LockContext(ctx, ErrInvalid); err != nil {
		return nil, err
	}
	defer h.mu.Unlock()
	candidates := make(map[*Session]struct{})
	for name, value := range header {
		for _, session := range h.nexusRoutes[nexusRouteIndex{name: name, value: value}] {
			candidates[session] = struct{}{}
		}
	}
	return slices.AppendSeq(make([]*Session, 0, len(candidates)), maps.Keys(candidates)), nil
}

func (h *Driver) admitWorkflow(input delivery.WorkflowDelivery) (routedWorkflow, error) {
	return admitFirst(context.Background(), h.workflowCandidates(input), "workflow_delivery_late", func(session *Session) (routedWorkflow, error) {
		activation, admission, replay, err := session.admitWorkflow(input)
		return routedWorkflow{session: session, activation: activation, admission: admission, replay: replay}, err
	})
}

// admitFirst offers a delivery to each candidate Session in turn and returns the first admission.
// A Session that does not own the route crosses it; any other rejection is the delivery's result
// when no Session admits it, and a stale one is diagnosed as late.
func admitFirst[R any](ctx context.Context, candidates []*Session, lateCode string, admit func(*Session) (R, error)) (R, error) {
	var matched error
	for _, session := range candidates {
		routed, err := admit(session)
		if err == nil {
			return routed, nil
		}
		if !errors.Is(err, delivery.ErrRouteCrossed) {
			matched = err
			if errors.Is(err, delivery.ErrRouteStale) {
				session.lateDiagnostic(ctx, lateCode)
			}
		}
	}
	var none R
	if matched != nil {
		return none, matched
	}
	return none, delivery.ErrRouteCrossed
}

func (s *Session) admitWorkflow(input delivery.WorkflowDelivery) (delivery.Activation, *workflowAdmission, bool, error) {
	key, err := workflowAdmissionKeyFor(input, s.host.options.requestBytes)
	if err != nil {
		return delivery.Activation{}, nil, false, err
	}
	if err := s.mu.LockContext(context.Background(), ErrInvalid); err != nil {
		return delivery.Activation{}, nil, false, err
	}
	defer s.mu.Unlock()
	if activation, admission, replay, err := s.workflowAdmissionLocked(key, input.TemporalRunID); replay || err != nil {
		return activation, admission, replay, err
	}
	activation, err := s.ledger.AdmitWorkflow(context.Background(), input)
	if err != nil {
		return delivery.Activation{}, nil, false, err
	}
	raw := s.reservations[activation.Reservation().ID]
	if raw == nil {
		return delivery.Activation{}, nil, false, ErrClosed
	}
	err = raw.bindCancellation(input.WorkflowID, input.TemporalRunID, "", func(ctx context.Context, workflowID, runID, requestID string) error {
		if requestID != "" || workflowID != input.WorkflowID || runID != input.TemporalRunID {
			return ErrInvalid
		}
		if primitive.NilValue(s.host.options.client) {
			return ErrInvalid
		}
		return s.host.options.client.CancelWorkflow(ctx, workflowID, runID)
	})
	if err != nil {
		return delivery.Activation{}, nil, false, err
	}
	if err := s.prepareNexusDispatchesLocked(activation); err != nil {
		raw.finish(testpilot.EffectResult{}, err)
		return delivery.Activation{}, nil, false, err
	}
	admitted := &workflowAdmission{activation: activation, temporalRunID: input.TemporalRunID}
	s.workflowAdmissions[key] = admitted
	return activation, admitted, false, nil
}

func (s *Session) workflowAdmissionLocked(key workflowAdmissionKey, temporalRunID string) (delivery.Activation, *workflowAdmission, bool, error) {
	if admitted, exists := s.workflowAdmissions[key]; exists {
		if admitted.temporalRunID != temporalRunID {
			return delivery.Activation{}, nil, false, delivery.ErrRouteConflict
		}
		return admitted.activation, admitted, true, nil
	}
	if s.closed || s.failure != nil {
		return delivery.Activation{}, nil, false, errors.Join(delivery.ErrRouteStale, s.failure)
	}
	if len(s.workflowAdmissions) >= boundedInt(s.definition.limits.GetMaxActivations()) {
		return delivery.Activation{}, nil, false, ErrCapacity
	}
	return delivery.Activation{}, nil, false, nil
}

func (s *Session) prepareNexusDispatchesLocked(activation delivery.Activation) error {
	entry := s.definition.entries[activation.Coordinate().EntrypointID]
	prepared := make(map[nexusDispatchKey]nexus.Header)
	routeKeys := make(map[nexusRouteIndex]struct{})
	for _, instruction := range entry.plan.Instructions() {
		if !startsNexusOperation(instruction) {
			continue
		}
		sourceID := instruction.Source().GetInstructionId()
		key := nexusDispatchKey{workflowReservation: activation.Reservation().ID, sourceInstruction: sourceID}
		if s.nexusDispatch[key] != nil {
			continue
		}
		dispatch, err := s.ledger.PrepareNexus(context.Background(), activation, sourceID)
		if err != nil {
			return err
		}
		header := dispatch.Header()
		routeKey, err := nexusRouteIndexFromHeader(header)
		if err != nil {
			return err
		}
		prepared[key] = header
		routeKeys[routeKey] = struct{}{}
	}
	if len(s.nexusDispatch) > boundedInt(s.definition.limits.GetMaxActivations())-len(prepared) {
		return ErrCapacity
	}
	if err := s.host.addNexusRoutes(context.Background(), s, routeKeys); err != nil {
		return err
	}
	for key, header := range prepared {
		s.nexusDispatch[key] = header
	}
	return nil
}

// preparedNexusHeader is the header a Nexus dispatch of the named start instruction carries: the
// headers given and the Run's routing header, merged within the request byte ceiling. A name two of
// them carry is refused, so no header can shadow the route.
func (s *Session) preparedNexusHeader(activation delivery.Activation, sourceID string, headers ...nexus.Header) (nexus.Header, error) {
	if s == nil || sourceID == "" || s.mu.LockContext(context.Background(), ErrInvalid) != nil {
		return nil, ErrInvalid
	}
	base := maps.Clone(s.nexusDispatch[nexusDispatchKey{workflowReservation: activation.Reservation().ID, sourceInstruction: sourceID}])
	s.mu.Unlock()
	if base == nil {
		return nil, delivery.ErrRouteCrossed
	}
	result := base
	for _, header := range headers {
		for name, value := range header {
			if _, collision := result[name]; collision {
				return nil, delivery.ErrReservedHeader
			}
			result[name] = value
		}
	}
	if int64(primitive.NexusHeaderBytes(result)) > s.host.options.requestBytes {
		return nil, ErrCapacity
	}
	return result, nil
}

func (h *Driver) admitNexus(ctx context.Context, queue string, input delivery.NexusDelivery, cancel context.CancelFunc) (routedNexus, error) {
	if cancel == nil {
		return routedNexus{}, ErrInvalid
	}
	candidates, err := h.nexusCandidates(ctx, input.Header)
	if err != nil {
		return routedNexus{}, err
	}
	candidates = slices.DeleteFunc(candidates, func(session *Session) bool { return !session.dependsOnQueue(queue) })
	return admitFirst(ctx, candidates, "nexus_delivery_late", func(session *Session) (routedNexus, error) {
		activation, replay, err := session.admitNexus(ctx, input, cancel)
		return routedNexus{session: session, activation: activation, replay: replay}, err
	})
}

func (s *Session) admitNexus(ctx context.Context, input delivery.NexusDelivery, cancel context.CancelFunc) (delivery.Activation, bool, error) {
	if err := s.mu.LockContext(ctx, ErrInvalid); err != nil {
		return delivery.Activation{}, false, err
	}
	defer s.mu.Unlock()
	key, err := s.nexusAdmissionKeyLocked(input.Header)
	if err != nil {
		return delivery.Activation{}, false, err
	}
	if s.closed {
		return delivery.Activation{}, false, delivery.ErrRouteStale
	}
	if admitted, replay := s.nexusAdmissions[key]; replay {
		if admitted.requestID != input.RequestID {
			return delivery.Activation{}, false, delivery.ErrRouteConflict
		}
		return admitted.activation, true, nil
	}
	if s.failure != nil {
		return delivery.Activation{}, false, errors.Join(delivery.ErrRouteStale, s.failure)
	}
	if len(s.nexusAdmissions) >= boundedInt(s.definition.limits.GetMaxActivations()) {
		return delivery.Activation{}, false, ErrCapacity
	}
	activation, err := s.ledger.AdmitNexus(ctx, input)
	if err != nil {
		return delivery.Activation{}, false, err
	}
	raw := s.reservations[activation.Reservation().ID]
	if raw == nil {
		return delivery.Activation{}, false, ErrClosed
	}
	requestID := activation.RequestID()
	err = raw.bindCancellation("", "", requestID, func(_ context.Context, workflowID, runID, providedRequestID string) error {
		if workflowID != "" || runID != "" || providedRequestID != requestID {
			return ErrInvalid
		}
		cancel()
		return nil
	})
	if err != nil {
		return delivery.Activation{}, false, err
	}
	s.nexusAdmissions[key] = nexusAdmission{activation: activation, requestID: input.RequestID}
	return activation, false, nil
}

func (s *Session) nexusAdmissionKeyLocked(header nexus.Header) (nexusRouteIndex, error) {
	var result nexusRouteIndex
	matched := false
	for name, value := range header {
		key := nexusRouteIndex{name: name, value: value}
		if _, exists := s.nexusKeys[key]; !exists {
			continue
		}
		if matched {
			return nexusRouteIndex{}, delivery.ErrRouteConflict
		}
		result, matched = key, true
	}
	if !matched {
		return nexusRouteIndex{}, delivery.ErrRouteCrossed
	}
	return result, nil
}

func (h *Driver) checkRouteCapacityLocked(session *Session, key delivery.WorkflowBinding) error {
	if slices.Contains(h.workflowRoutes[key], session) {
		return nil
	}
	return h.ensureRouteCapacityLocked(1)
}

func (h *Driver) addWorkflowRouteLocked(session *Session, key delivery.WorkflowBinding) {
	if slices.Contains(h.workflowRoutes[key], session) {
		return
	}
	if h.workflowRoutes == nil {
		h.workflowRoutes = make(map[delivery.WorkflowBinding][]*Session)
	}
	h.workflowRoutes[key] = append(h.workflowRoutes[key], session)
	session.workflowKeys[key] = struct{}{}
	h.routeAssociations++
}

func (h *Driver) addNexusRoutes(ctx context.Context, session *Session, keys map[nexusRouteIndex]struct{}) error {
	if err := h.mu.LockContext(ctx, ErrInvalid); err != nil {
		return err
	}
	defer h.mu.Unlock()
	if h.sessions[session.runID] != session {
		return ErrClosed
	}
	additional := 0
	for key := range keys {
		if !slices.Contains(h.nexusRoutes[key], session) {
			additional++
		}
	}
	if err := h.ensureRouteCapacityLocked(additional); err != nil {
		return err
	}
	if h.nexusRoutes == nil {
		h.nexusRoutes = make(map[nexusRouteIndex][]*Session)
	}
	for key := range keys {
		if slices.Contains(h.nexusRoutes[key], session) {
			continue
		}
		h.nexusRoutes[key] = append(h.nexusRoutes[key], session)
		session.nexusKeys[key] = struct{}{}
		h.routeAssociations++
	}
	return nil
}

func (h *Driver) ensureRouteCapacityLocked(additional int) error {
	for h.routeAssociations > h.options.maximum-additional && len(h.tombstones) > 0 {
		h.evictOldestTombstoneLocked()
	}
	if additional < 0 || h.routeAssociations > h.options.maximum-additional {
		return ErrCapacity
	}
	return nil
}

func (h *Driver) evictOldestTombstoneLocked() {
	oldest := h.tombstones[0]
	copy(h.tombstones, h.tombstones[1:])
	h.tombstones = h.tombstones[:len(h.tombstones)-1]
	h.removeRouteIndexesLocked(oldest)
}

func (h *Driver) removeRouteIndexesLocked(session *Session) {
	for key := range session.workflowKeys {
		h.workflowRoutes[key] = slices.DeleteFunc(h.workflowRoutes[key], func(candidate *Session) bool { return candidate == session })
		if len(h.workflowRoutes[key]) == 0 {
			delete(h.workflowRoutes, key)
		}
		h.routeAssociations--
	}
	for key := range session.nexusKeys {
		h.nexusRoutes[key] = slices.DeleteFunc(h.nexusRoutes[key], func(candidate *Session) bool { return candidate == session })
		if len(h.nexusRoutes[key]) == 0 {
			delete(h.nexusRoutes, key)
		}
		h.routeAssociations--
	}
}

func (s *Session) parentTerminal(ctx context.Context, activation delivery.Activation) (int, error) {
	release, err := s.ledger.ParentTerminal(ctx, activation)
	return release.Unused(), err
}

func (s *Session) lateDiagnostic(ctx context.Context, code string) {
	cancel := func() {}
	if ctx == nil || ctx.Err() != nil {
		ctx, cancel = s.host.cleanupContext()
	}
	defer cancel()
	_ = s.Diagnose(ctx, s.runID, &testpilotspb.RunDiagnostic{Kind: testpilotspb.RUN_DIAGNOSTIC_KIND_POST_CLOSE_EVENT, Code: code, Detail: "reserved worker delivery rejected after Run closure"})
}
