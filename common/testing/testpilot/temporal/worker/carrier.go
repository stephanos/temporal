package worker

import (
	"context"
	"errors"

	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Carrier struct {
	session *Session
	bundle  delivery.Bundle
	origin  testpilot.Coordinate
}

func (s *Session) CreateCarrier(ctx context.Context, origin testpilot.Coordinate, plan testpilot.ReservationCarrierPlan, binding delivery.WorkflowBinding, handles []testpilot.ReservationHandle) (*Carrier, error) {
	if s == nil || origin.RunID != s.runID || !s.validCarrierBinding(plan, binding) {
		return nil, ErrInvalid
	}
	return s.createCarrier(ctx, origin, carrierRoute{
		capacity: func() error { return s.host.checkRouteCapacityLocked(s, binding) },
		bundle:   func() (delivery.Bundle, error) { return s.ledger.CreateBundle(ctx, origin, plan, binding, handles) },
		index:    func() { s.host.addWorkflowRouteLocked(s, binding) },
	})
}

// CreateActivityCarrier is CreateCarrier for a StartActivityExecution: the carrier of the one
// activation of the standalone activity the request starts.
func (s *Session) CreateActivityCarrier(ctx context.Context, origin testpilot.Coordinate, plan testpilot.ReservationCarrierPlan, binding delivery.ActivityBinding, handles []testpilot.ReservationHandle) (*Carrier, error) {
	if s == nil || origin.RunID != s.runID || !s.validActivityCarrierBinding(plan, binding) {
		return nil, ErrInvalid
	}
	return s.createCarrier(ctx, origin, carrierRoute{
		capacity: func() error { return s.host.checkActivityRouteCapacityLocked(s, binding) },
		bundle: func() (delivery.Bundle, error) {
			return s.ledger.CreateActivityBundle(ctx, origin, plan, binding, handles)
		},
		index: func() { s.host.addActivityRouteLocked(s, binding) },
	})
}

// carrierRoute is what differs between the carrier of a workflow start and of an activity start:
// the Driver route its binding needs room for, the ledger bundle it holds, and the index entry that
// makes its deliveries reach this Session. Each runs under the Session and Driver locks.
type carrierRoute struct {
	capacity func() error
	bundle   func() (delivery.Bundle, error)
	index    func()
}

func (s *Session) createCarrier(ctx context.Context, origin testpilot.Coordinate, route carrierRoute) (*Carrier, error) {
	if err := s.mu.LockContext(ctx, ErrInvalid); err != nil {
		return nil, err
	}
	defer s.mu.Unlock()
	if s.closed || s.failure != nil {
		return nil, errors.Join(ErrClosed, s.failure)
	}
	if len(s.carriers) >= boundedInt(s.definition.limits.GetMaxActivations()) || s.carriers[origin] != nil {
		return nil, ErrCapacity
	}
	if err := s.host.mu.LockContext(ctx, ErrInvalid); err != nil {
		return nil, err
	}
	defer s.host.mu.Unlock()
	if s.host.sessions[s.runID] != s {
		return nil, ErrClosed
	}
	if err := route.capacity(); err != nil {
		return nil, err
	}
	bundle, err := route.bundle()
	if err != nil {
		return nil, err
	}
	carrier := &Carrier{session: s, bundle: bundle, origin: origin}
	s.carriers[origin] = carrier
	route.index()
	return carrier, nil
}

func (s *Session) validCarrierBinding(plan testpilot.ReservationCarrierPlan, binding delivery.WorkflowBinding) bool {
	if binding.WorkflowID == "" {
		return false
	}
	var workflowEntrypoint string
	for _, reservation := range plan.Reservations {
		if reservation.Kind == testpilot.WorkflowEntrypoint {
			workflowEntrypoint = reservation.EntrypointID
		}
	}
	entry, exists := s.definition.entries[workflowEntrypoint]
	return exists && entry.plan.Kind() == testpilot.WorkflowEntrypoint && entry.namespace == binding.Namespace && entry.workflowType == binding.WorkflowType && entry.queue == binding.TaskQueue
}

func (s *Session) validActivityCarrierBinding(plan testpilot.ReservationCarrierPlan, binding delivery.ActivityBinding) bool {
	if binding.ActivityID == "" {
		return false
	}
	var activityEntrypoint string
	for _, reservation := range plan.Reservations {
		if reservation.Kind == testpilot.ActivityEntrypoint {
			activityEntrypoint = reservation.EntrypointID
		}
	}
	entry, exists := s.definition.entries[activityEntrypoint]
	return exists && entry.plan.Kind() == testpilot.ActivityEntrypoint && entry.namespace == binding.Namespace && entry.activityType == binding.ActivityType && entry.queue == binding.TaskQueue
}

func (c *Carrier) PrepareRPC(ctx context.Context, role string, method protoreflect.MethodDescriptor, request proto.Message, maximumBytes int64) (proto.Message, error) {
	if c == nil || c.session == nil {
		return nil, ErrInvalid
	}
	if err := c.session.mu.LockContext(ctx, ErrInvalid); err != nil {
		return nil, err
	}
	defer c.session.mu.Unlock()
	if c.session.closed || c.session.failure != nil {
		return nil, errors.Join(ErrClosed, c.session.failure)
	}
	return c.session.ledger.PrepareRPC(ctx, &c.bundle, role, method, request, maximumBytes)
}

func (c *Carrier) PinStartResponse(ctx context.Context, response delivery.StartResponse) error {
	if c == nil || c.session == nil {
		return ErrInvalid
	}
	if err := c.session.mu.LockContext(ctx, ErrInvalid); err != nil {
		return err
	}
	defer c.session.mu.Unlock()
	if c.session.closed {
		return ErrClosed
	}
	return c.session.ledger.PinStartResponse(ctx, c.bundle, response)
}

func (c *Carrier) TriggerTerminal(ctx context.Context, disposition delivery.TriggerStatus) (int, error) {
	if c == nil || c.session == nil {
		return 0, ErrInvalid
	}
	if err := c.session.mu.LockContext(ctx, ErrInvalid); err != nil {
		return 0, err
	}
	defer c.session.mu.Unlock()
	if c.session.closed {
		return 0, ErrClosed
	}
	release, err := c.session.ledger.TriggerTerminal(ctx, c.bundle, disposition)
	return release.Unused(), err
}
