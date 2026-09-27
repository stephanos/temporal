package worker

import (
	"context"
	"errors"

	"go.temporal.io/api/workflowservice/v1"
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

func (s *Session) CreateCarrier(ctx context.Context, origin testpilot.Coordinate, plan testpilot.ReservationCarrierPlan, binding WorkflowBinding, handles []testpilot.ReservationHandle) (*Carrier, error) {
	if s == nil || origin.RunID != s.runID || !s.validCarrierBinding(plan, binding) {
		return nil, ErrInvalid
	}
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
	key := workflowRouteIndexFor(binding)
	if err := s.host.checkRouteCapacityLocked(s, key); err != nil {
		return nil, err
	}
	bundle, err := s.ledger.CreateBundle(ctx, origin, plan, delivery.WorkflowBinding(binding), handles)
	if err != nil {
		return nil, err
	}
	carrier := &Carrier{session: s, bundle: bundle, origin: origin}
	s.carriers[origin] = carrier
	s.host.addWorkflowRouteLocked(s, key)
	return carrier, nil
}

func (s *Session) validCarrierBinding(plan testpilot.ReservationCarrierPlan, binding WorkflowBinding) bool {
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

func (c *Carrier) PinStartResponse(ctx context.Context, response *workflowservice.StartWorkflowExecutionResponse) error {
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
