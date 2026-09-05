package worker

import (
	"context"
	"errors"

	"go.temporal.io/api/workflowservice/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"go.temporal.io/server/tools/umpire/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Carrier struct {
	session  *Session
	mu       contextMutex
	bundle   delivery.Bundle
	origin   umpire.Coordinate
	workflow delivery.Activation
	admitted bool
}

func (s *Session) CreateCarrier(ctx context.Context, origin umpire.Coordinate, plan umpire.ReservationCarrierPlan, binding WorkflowBinding, handles []umpire.ReservationHandle) (*Carrier, error) {
	if s == nil || origin.RunID != s.runID || !s.validCarrierBinding(plan, binding) {
		return nil, ErrInvalid
	}
	if err := s.mu.lock(ctx); err != nil {
		return nil, err
	}
	defer s.mu.unlock()
	if s.closed || s.failure != nil {
		return nil, errors.Join(ErrClosed, s.failure)
	}
	if len(s.carriers) >= boundedInt(s.definition.snapshot.GetLimits().GetMaxActivations()) || s.carriers[origin] != nil {
		return nil, ErrCapacity
	}
	if err := s.host.mu.lock(ctx); err != nil {
		return nil, err
	}
	defer s.host.mu.unlock()
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
	carrier := &Carrier{session: s, mu: newContextMutex(), bundle: bundle, origin: origin}
	s.carriers[origin] = carrier
	s.host.addWorkflowRouteLocked(s, key)
	return carrier, nil
}

func (s *Session) validCarrierBinding(plan umpire.ReservationCarrierPlan, binding WorkflowBinding) bool {
	if binding.Namespace != s.host.options.namespace || binding.WorkflowID == "" {
		return false
	}
	var workflowEntrypoint string
	for _, reservation := range plan.Reservations {
		if reservation.Context != umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW {
			continue
		}
		if workflowEntrypoint != "" || reservation.Count != 1 {
			return false
		}
		workflowEntrypoint = reservation.EntrypointID
	}
	entry, exists := s.definition.entries[workflowEntrypoint]
	return exists && entry.plan.Context() == umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW && entry.workflowType == binding.WorkflowType && entry.queue == binding.TaskQueue
}

func (c *Carrier) Handles() []umpire.EffectHandle {
	if c == nil {
		return nil
	}
	return c.bundle.Handles()
}

func (c *Carrier) PrepareRPC(ctx context.Context, role string, method protoreflect.MethodDescriptor, request proto.Message, maximumBytes int64) (proto.Message, error) {
	if c == nil || c.session == nil {
		return nil, ErrInvalid
	}
	if err := c.session.mu.lock(ctx); err != nil {
		return nil, err
	}
	defer c.session.mu.unlock()
	if c.session.closed || c.session.failure != nil {
		return nil, errors.Join(ErrClosed, c.session.failure)
	}
	return c.session.ledger.PrepareRPC(ctx, &c.bundle, role, method, request, maximumBytes)
}

func (c *Carrier) PinStartResponse(ctx context.Context, response *workflowservice.StartWorkflowExecutionResponse) error {
	if c == nil || c.session == nil {
		return ErrInvalid
	}
	if err := c.session.mu.lock(ctx); err != nil {
		return err
	}
	defer c.session.mu.unlock()
	if c.session.closed {
		return ErrClosed
	}
	return c.session.ledger.PinStartResponse(ctx, c.bundle, response)
}

func (c *Carrier) TriggerTerminal(ctx context.Context, disposition delivery.TriggerDisposition) (int, error) {
	if c == nil || c.session == nil {
		return 0, ErrInvalid
	}
	if err := c.session.mu.lock(ctx); err != nil {
		return 0, err
	}
	defer c.session.mu.unlock()
	if c.session.closed {
		return 0, ErrClosed
	}
	release, err := c.session.ledger.TriggerTerminal(ctx, c.bundle, disposition)
	return release.Unused(), err
}

func (c *Carrier) ParentTerminal(ctx context.Context) (int, error) {
	if c == nil || c.session == nil {
		return 0, ErrInvalid
	}
	if err := c.mu.lock(ctx); err != nil {
		return 0, err
	}
	defer c.mu.unlock()
	if !c.admitted {
		return 0, ErrInvalid
	}
	release, err := c.session.ledger.ParentTerminal(ctx, c.workflow)
	return release.Unused(), err
}

func (c *Carrier) Quarantine(ctx context.Context, handle umpire.EffectHandle) error {
	if c == nil || c.session == nil {
		return ErrInvalid
	}
	return c.session.Quarantine(ctx, handle)
}

func (c *Carrier) admitWorkflow(activation delivery.Activation) {
	if c.mu.lock(context.Background()) != nil {
		return
	}
	defer c.mu.unlock()
	c.workflow = activation
	c.admitted = true
}
