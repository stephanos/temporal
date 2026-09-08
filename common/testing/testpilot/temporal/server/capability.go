package server

import (
	"context"
	"sync/atomic"

	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
)

type opaqueCapability struct {
	session   *Session
	origin    testpilot.Coordinate
	invoke    testpilot.CapabilityEffect
	used      bool
	published string
}

type capabilitySlot struct {
	ready      chan struct{}
	capability *opaqueCapability
	claim      *capabilityClaim
}

type capabilityClaim struct {
	capability *opaqueCapability
	context    context.Context
	released   atomic.Bool
}

// NewCapability is the injection seam for composite Driver wiring. The capability remains opaque
// to execution, and minting performs no target I/O.
func (s *Session) NewCapability(ctx context.Context, origin testpilot.Coordinate, invoke testpilot.CapabilityEffect) (testpilot.OpaqueCapability, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if origin.RunID != s.runID || origin.ActivationID == "" || len(origin.ActivationID) > 256 || nilValue(invoke) {
		return nil, errUnauthorized
	}
	if _, ok := s.entries[origin.EntrypointID]; !ok {
		return nil, errUnauthorized
	}
	if err := s.host.mu.LockContext(ctx); err != nil {
		return nil, err
	}
	defer s.host.mu.Unlock()
	if s.closed {
		return nil, errClosed
	}
	if s.minted >= s.host.profile.ProgramLimits.MaxActivations {
		return nil, errCapacity
	}
	capability := &opaqueCapability{session: s, origin: origin, invoke: invoke}
	s.capabilities[capability] = struct{}{}
	s.minted++
	return capability, nil
}

func (s *Session) InvokeCapability(ctx context.Context, c testpilot.Coordinate, opaque testpilot.OpaqueCapability, input proto.Message) (testpilot.EffectHandle, error) {
	claim, ok := opaque.(*capabilityClaim)
	if !ok || claim == nil || claim.capability == nil || claim.capability.session != s {
		return nil, errUnauthorized
	}
	accepted := false
	defer func() {
		if !accepted {
			claim.released.Store(true)
		}
	}()
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	n, err := s.controllerNode(c)
	if err != nil {
		return nil, err
	}
	if nilValue(input) {
		return nil, errUnauthorized
	}
	if int64(proto.Size(input)) > s.host.profile.ProgramLimits.MaxRequestBytes {
		return nil, errCapacity
	}
	input = proto.Clone(input)
	if err := s.host.mu.LockContext(ctx); err != nil {
		return nil, err
	}
	capability := claim.capability
	slot := s.slots[capability.published]
	if capability.used || nilValue(capability.invoke) || claim.released.Load() || claim.context.Err() != nil || slot == nil || slot.claim != claim {
		s.host.mu.Unlock()
		return nil, errUnauthorized
	}
	if _, exists := s.capabilities[capability]; !exists {
		s.host.mu.Unlock()
		return nil, errUnauthorized
	}
	invoke := capability.invoke
	s.host.mu.Unlock()
	if !invoke.Accepts(ctx, n.GetInstruction(), input) {
		return nil, errUnauthorized
	}
	if err := s.host.mu.LockContext(ctx); err != nil {
		return nil, err
	}
	defer s.host.mu.Unlock()
	slot = s.slots[capability.published]
	if capability.used || nilValue(capability.invoke) || claim.released.Load() || claim.context.Err() != nil || slot == nil || slot.claim != claim {
		return nil, errUnauthorized
	}
	if _, exists := s.capabilities[capability]; !exists {
		return nil, errUnauthorized
	}
	handle, err := s.startLocked(ctx, c, n.Limits, func(ctx context.Context) testpilot.EffectResult {
		return invoke.Invoke(ctx, input, min(n.Limits.MaxResponseBytes, s.host.profile.ProgramLimits.MaxResponseBytes))
	})
	if err != nil {
		return nil, err
	}
	accepted = true
	capability.used = true
	capability.invoke = nil
	delete(s.capabilities, capability)
	return handle, nil
}

func (s *Session) Bridge(ctx context.Context) (testpilot.CapabilityBridge, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if err := s.host.mu.LockContext(ctx); err != nil {
		return nil, err
	}
	defer s.host.mu.Unlock()
	if s.closed {
		return nil, errClosed
	}
	return s, nil
}

func (s *Session) Publish(ctx context.Context, c testpilot.Coordinate, slotID string, opaque testpilot.OpaqueCapability) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := s.host.mu.LockContext(ctx); err != nil {
		return err
	}
	defer s.host.mu.Unlock()
	if s.closed {
		return errClosed
	}
	slot := s.slots[slotID]
	capability, ok := opaque.(*opaqueCapability)
	if slot == nil || !ok || capability == nil || capability.session != s || capability.origin != c || capability.used {
		return errUnauthorized
	}
	if _, exists := s.capabilities[capability]; !exists {
		return errUnauthorized
	}
	if slot.claim != nil || slot.capability != nil && slot.capability != capability || capability.published != "" && capability.published != slotID {
		return errInvalid
	}
	if slot.capability == nil {
		slot.capability = capability
		capability.published = slotID
		close(slot.ready)
	}
	return nil
}

func (s *Session) Await(ctx context.Context, slotID string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := s.host.mu.LockContext(ctx); err != nil {
		return err
	}
	if s.closed {
		s.host.mu.Unlock()
		return errClosed
	}
	slot := s.slots[slotID]
	s.host.mu.Unlock()
	if slot == nil {
		return errUnauthorized
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closedSignal:
		return errClosed
	case <-slot.ready:
		if err := s.host.mu.LockContext(ctx); err != nil {
			return err
		}
		defer s.host.mu.Unlock()
		if s.closed {
			return errClosed
		}
		if slot.capability.used {
			return errInvalid
		}
		return nil
	}
}

func (s *Session) Consume(ctx context.Context, slotID string) (testpilot.OpaqueCapability, error) {
	if err := s.host.mu.LockContext(ctx); err != nil {
		return nil, err
	}
	defer s.host.mu.Unlock()
	if s.closed {
		return nil, errClosed
	}
	slot := s.slots[slotID]
	if slot == nil || slot.capability == nil || slot.capability.used {
		return nil, errUnauthorized
	}
	if old := slot.claim; old != nil && !old.released.Load() && old.context.Err() == nil {
		return nil, errUnauthorized
	}
	claim := &capabilityClaim{capability: slot.capability, context: ctx}
	slot.claim = claim
	return claim, nil
}
