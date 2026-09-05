package worker

import (
	"context"
	"errors"
	"maps"
	"sync"

	"github.com/nexus-rpc/sdk-go/nexus"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"go.temporal.io/server/tools/umpire/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
)

type workflowRouteIndex struct {
	namespace, workflowID, workflowType, taskQueue string
}

type nexusRouteIndex struct {
	name, value string
}

type nexusDispatchKey struct {
	workflowReservation, sourceInstruction string
}

type workflowAdmissionKey struct {
	route  workflowRouteIndex
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

func workflowRouteIndexFor(binding WorkflowBinding) workflowRouteIndex {
	return workflowRouteIndex{namespace: binding.Namespace, workflowID: binding.WorkflowID, workflowType: binding.WorkflowType, taskQueue: binding.TaskQueue}
}

func workflowDeliveryIndex(input delivery.WorkflowDelivery) workflowRouteIndex {
	return workflowRouteIndex{namespace: input.Namespace, workflowID: input.WorkflowID, workflowType: input.WorkflowType, taskQueue: input.TaskQueue}
}

func workflowAdmissionKeyFor(input delivery.WorkflowDelivery, maximumBytes int64) (workflowAdmissionKey, error) {
	if input.Header == nil || int64(proto.Size(input.Header)) > maximumBytes {
		return workflowAdmissionKey{}, ErrInvalid
	}
	wire, err := (proto.MarshalOptions{Deterministic: true}).Marshal(input.Header)
	if err != nil {
		return workflowAdmissionKey{}, err
	}
	return workflowAdmissionKey{route: workflowDeliveryIndex(input), header: string(wire)}, nil
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

func (h *Host) workflowCandidates(input delivery.WorkflowDelivery) []*Session {
	if h == nil || h.mu.lock(context.Background()) != nil {
		return nil
	}
	defer h.mu.unlock()
	return append([]*Session(nil), h.workflowRoutes[workflowDeliveryIndex(input)]...)
}

func (h *Host) nexusCandidates(ctx context.Context, header nexus.Header) ([]*Session, error) {
	if h == nil || ctx == nil || nexusHeaderBytes(header) > h.options.requestBytes {
		return nil, ErrInvalid
	}
	if err := h.mu.lock(ctx); err != nil {
		return nil, err
	}
	defer h.mu.unlock()
	candidates := make(map[*Session]struct{})
	for name, value := range header {
		for _, session := range h.nexusRoutes[nexusRouteIndex{name: name, value: value}] {
			candidates[session] = struct{}{}
		}
	}
	result := make([]*Session, 0, len(candidates))
	for session := range candidates {
		result = append(result, session)
	}
	return result, nil
}

func (h *Host) admitWorkflow(input delivery.WorkflowDelivery) (routedWorkflow, error) {
	var matched error
	for _, session := range h.workflowCandidates(input) {
		activation, admission, replay, err := session.admitWorkflow(input)
		if err != nil {
			if !errors.Is(err, delivery.ErrRouteCrossed) {
				matched = err
				if errors.Is(err, delivery.ErrRouteStale) {
					session.lateDiagnostic(context.Background(), "workflow_delivery_late")
				}
			}
			continue
		}
		return routedWorkflow{session: session, activation: activation, admission: admission, replay: replay}, nil
	}
	if matched != nil {
		return routedWorkflow{}, matched
	}
	return routedWorkflow{}, delivery.ErrRouteCrossed
}

func (s *Session) admitWorkflow(input delivery.WorkflowDelivery) (delivery.Activation, *workflowAdmission, bool, error) {
	key, err := workflowAdmissionKeyFor(input, s.host.options.requestBytes)
	if err != nil {
		return delivery.Activation{}, nil, false, err
	}
	if err := s.mu.lock(context.Background()); err != nil {
		return delivery.Activation{}, nil, false, err
	}
	defer s.mu.unlock()
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
		if nilValue(s.host.options.client) {
			return ErrInvalid
		}
		return s.host.options.client.CancelWorkflow(ctx, workflowID, runID)
	})
	if err != nil {
		return delivery.Activation{}, nil, false, err
	}
	if err := s.prepareNexusDispatchesLocked(activation); err != nil {
		raw.finish(umpire.EffectResult{}, err)
		return delivery.Activation{}, nil, false, err
	}
	admitted := &workflowAdmission{activation: activation, temporalRunID: input.TemporalRunID}
	s.workflowAdmissions[key] = admitted
	if carrier := s.carriers[activation.Reservation().Origin]; carrier != nil {
		carrier.admitWorkflow(activation)
	}
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
	if len(s.workflowAdmissions) >= boundedInt(s.definition.snapshot.GetLimits().GetMaxActivations()) {
		return delivery.Activation{}, nil, false, ErrCapacity
	}
	return delivery.Activation{}, nil, false, nil
}

func (s *Session) prepareNexusDispatchesLocked(activation delivery.Activation) error {
	entry := s.definition.entries[activation.Coordinate().EntrypointID]
	prepared := make(map[nexusDispatchKey]nexus.Header)
	routeKeys := make(map[nexusRouteIndex]struct{})
	for _, instruction := range entry.plan.Instructions() {
		if instruction.Opcode() != umpire.StartNexusOperation {
			continue
		}
		sourceID := instruction.Source().GetInstructionId()
		key := nexusDispatchKey{workflowReservation: activation.Reservation().ID, sourceInstruction: sourceID}
		if s.nexusDispatch[key] != nil {
			continue
		}
		dispatch, err := s.ledger.PrepareNexus(context.Background(), activation, sourceID, nil, nil)
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
	if len(s.nexusDispatch) > boundedInt(s.definition.snapshot.GetLimits().GetMaxActivations())-len(prepared) {
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

func (s *Session) preparedNexusDispatch(activation delivery.Activation, sourceID string, header nexus.Header, value *umpirespb.Value) (nexus.Header, *umpirespb.Value, error) {
	if s == nil || sourceID == "" || value == nil || s.mu.lock(context.Background()) != nil {
		return nil, nil, ErrInvalid
	}
	base := maps.Clone(s.nexusDispatch[nexusDispatchKey{workflowReservation: activation.Reservation().ID, sourceInstruction: sourceID}])
	s.mu.unlock()
	if base == nil {
		return nil, nil, delivery.ErrRouteCrossed
	}
	result := maps.Clone(header)
	if result == nil {
		result = make(nexus.Header)
	}
	for name, route := range base {
		if _, collision := result[name]; collision {
			return nil, nil, delivery.ErrReservedHeader
		}
		result[name] = route
	}
	if nexusHeaderBytes(result) > s.host.options.requestBytes {
		return nil, nil, ErrCapacity
	}
	return result, proto.CloneOf(value), nil
}

func (h *Host) admitNexus(ctx context.Context, queue string, input delivery.NexusDelivery, cancel context.CancelFunc) (routedNexus, error) {
	if cancel == nil {
		return routedNexus{}, ErrInvalid
	}
	candidates, err := h.nexusCandidates(ctx, input.Header)
	if err != nil {
		return routedNexus{}, err
	}
	var matched error
	for _, session := range candidates {
		if !session.dependsOnQueue(queue) {
			continue
		}
		activation, replay, err := session.admitNexus(ctx, input, cancel)
		if err != nil {
			if !errors.Is(err, delivery.ErrRouteCrossed) {
				matched = err
				if errors.Is(err, delivery.ErrRouteStale) {
					session.lateDiagnostic(ctx, "nexus_delivery_late")
				}
			}
			continue
		}
		return routedNexus{session: session, activation: activation, replay: replay}, nil
	}
	if matched != nil {
		return routedNexus{}, matched
	}
	return routedNexus{}, delivery.ErrRouteCrossed
}

func (s *Session) admitNexus(ctx context.Context, input delivery.NexusDelivery, cancel context.CancelFunc) (delivery.Activation, bool, error) {
	if err := s.mu.lock(ctx); err != nil {
		return delivery.Activation{}, false, err
	}
	defer s.mu.unlock()
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
	if len(s.nexusAdmissions) >= boundedInt(s.definition.snapshot.GetLimits().GetMaxActivations()) {
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

func (h *Host) checkRouteCapacityLocked(session *Session, key workflowRouteIndex) error {
	if sessionIn(h.workflowRoutes[key], session) {
		return nil
	}
	return h.ensureRouteCapacityLocked(1)
}

func (h *Host) addWorkflowRouteLocked(session *Session, key workflowRouteIndex) {
	if sessionIn(h.workflowRoutes[key], session) {
		return
	}
	if h.workflowRoutes == nil {
		h.workflowRoutes = make(map[workflowRouteIndex][]*Session)
	}
	h.workflowRoutes[key] = append(h.workflowRoutes[key], session)
	session.workflowKeys[key] = struct{}{}
	h.routeAssociations++
}

func (h *Host) addNexusRoutes(ctx context.Context, session *Session, keys map[nexusRouteIndex]struct{}) error {
	if err := h.mu.lock(ctx); err != nil {
		return err
	}
	defer h.mu.unlock()
	if h.sessions[session.runID] != session {
		return ErrClosed
	}
	additional := 0
	for key := range keys {
		if !sessionIn(h.nexusRoutes[key], session) {
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
		if sessionIn(h.nexusRoutes[key], session) {
			continue
		}
		h.nexusRoutes[key] = append(h.nexusRoutes[key], session)
		session.nexusKeys[key] = struct{}{}
		h.routeAssociations++
	}
	return nil
}

func (h *Host) ensureRouteCapacityLocked(additional int) error {
	for h.routeAssociations > h.options.maximum-additional && len(h.tombstones) > 0 {
		h.evictOldestTombstoneLocked()
	}
	if additional < 0 || h.routeAssociations > h.options.maximum-additional {
		return ErrCapacity
	}
	return nil
}

func (h *Host) evictOldestTombstoneLocked() {
	oldest := h.tombstones[0]
	copy(h.tombstones, h.tombstones[1:])
	h.tombstones = h.tombstones[:len(h.tombstones)-1]
	h.removeRouteIndexesLocked(oldest)
}

func (h *Host) removeRouteIndexesLocked(session *Session) {
	for key := range session.workflowKeys {
		h.workflowRoutes[key] = removeSessionFrom(h.workflowRoutes[key], session)
		if len(h.workflowRoutes[key]) == 0 {
			delete(h.workflowRoutes, key)
		}
		h.routeAssociations--
	}
	for key := range session.nexusKeys {
		h.nexusRoutes[key] = removeSessionFrom(h.nexusRoutes[key], session)
		if len(h.nexusRoutes[key]) == 0 {
			delete(h.nexusRoutes, key)
		}
		h.routeAssociations--
	}
}

func sessionIn(sessions []*Session, target *Session) bool {
	for _, session := range sessions {
		if session == target {
			return true
		}
	}
	return false
}

func removeSessionFrom(sessions []*Session, target *Session) []*Session {
	result := sessions[:0]
	for _, session := range sessions {
		if session != target {
			result = append(result, session)
		}
	}
	return result
}

func nexusHeaderBytes(header nexus.Header) int64 {
	var size int64
	for name, value := range header {
		size += int64(len(name) + len(value))
	}
	return size
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
	_ = s.Diagnose(ctx, s.runID, &umpirespb.RunDiagnostic{Kind: umpirespb.RUN_DIAGNOSTIC_KIND_POST_CLOSE_EVENT, Code: code, Detail: "reserved worker delivery rejected after Run closure"})
}

func validCoordinate(coordinate umpire.Coordinate) bool {
	return coordinate.RunID != "" && coordinate.EntrypointID != "" && coordinate.ActivationID != "" && coordinate.Attempt > 0
}
