package server

import (
	"context"
	"sync/atomic"

	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
	"google.golang.org/protobuf/proto"
)

type opaqueHandle struct {
	session   *Session
	origin    testpilot.Coordinate
	invoke    testpilot.HandleEffect
	used      bool
	published string
}

type handleSlot struct {
	ready  chan struct{}
	handle *opaqueHandle
	claim  *handleClaim
}

type handleClaim struct {
	handle   *opaqueHandle
	context  context.Context
	released atomic.Bool
}

// NewHandle is the injection seam for composite Driver wiring. The handle remains opaque
// to execution, and minting performs no target I/O.
func (s *Session) NewHandle(ctx context.Context, origin testpilot.Coordinate, invoke testpilot.HandleEffect) (testpilot.OpaqueHandle, error) {
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return nil, err
	}
	if origin.RunID != s.runID || origin.ActivationID == "" || len(origin.ActivationID) > 256 || primitive.NilValue(invoke) {
		return nil, errUnauthorized
	}
	if _, ok := s.entries[origin.EntrypointID]; !ok {
		return nil, errUnauthorized
	}
	if err := s.host.mu.LockContext(ctx, errInvalid); err != nil {
		return nil, err
	}
	defer s.host.mu.Unlock()
	if s.closed {
		return nil, errClosed
	}
	if s.minted >= s.host.profile.ProgramLimits.MaxActivations {
		return nil, errCapacity
	}
	handle := &opaqueHandle{session: s, origin: origin, invoke: invoke}
	s.handles[handle] = struct{}{}
	s.minted++
	return handle, nil
}

func (s *Session) InvokeHandle(ctx context.Context, c testpilot.Coordinate, claimed testpilot.OpaqueHandle, input proto.Message) (testpilot.EffectHandle, error) {
	claim, ok := claimed.(*handleClaim)
	if !ok || claim == nil || claim.handle == nil || claim.handle.session != s {
		return nil, errUnauthorized
	}
	accepted := false
	defer func() {
		if !accepted {
			claim.released.Store(true)
		}
	}()
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return nil, err
	}
	n, err := s.controllerNode(c)
	if err != nil {
		return nil, err
	}
	if primitive.NilValue(input) {
		return nil, errUnauthorized
	}
	if int64(proto.Size(input)) > s.host.profile.ProgramLimits.MaxRequestBytes {
		return nil, errCapacity
	}
	input = proto.Clone(input)
	if err := s.host.mu.LockContext(ctx, errInvalid); err != nil {
		return nil, err
	}
	opaque := claim.handle
	slot := s.slots[opaque.published]
	if opaque.used || primitive.NilValue(opaque.invoke) || claim.released.Load() || claim.context.Err() != nil || slot == nil || slot.claim != claim {
		s.host.mu.Unlock()
		return nil, errUnauthorized
	}
	if _, exists := s.handles[opaque]; !exists {
		s.host.mu.Unlock()
		return nil, errUnauthorized
	}
	invoke := opaque.invoke
	s.host.mu.Unlock()
	if !invoke.Accepts(ctx, n.GetInstruction(), input) {
		return nil, errUnauthorized
	}
	if err := s.host.mu.LockContext(ctx, errInvalid); err != nil {
		return nil, err
	}
	defer s.host.mu.Unlock()
	slot = s.slots[opaque.published]
	if opaque.used || primitive.NilValue(opaque.invoke) || claim.released.Load() || claim.context.Err() != nil || slot == nil || slot.claim != claim {
		return nil, errUnauthorized
	}
	if _, exists := s.handles[opaque]; !exists {
		return nil, errUnauthorized
	}
	handle, err := s.startLocked(ctx, c, n.Limits, func(ctx context.Context) testpilot.EffectResult {
		return invoke.Invoke(ctx, input, s.host.profile.ProgramLimits.MaxInstructionResponseBytes)
	})
	if err != nil {
		return nil, err
	}
	accepted = true
	opaque.used = true
	opaque.invoke = nil
	delete(s.handles, opaque)
	return handle, nil
}

func (s *Session) Bridge(ctx context.Context) (testpilot.HandleBridge, error) {
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return nil, err
	}
	if err := s.host.mu.LockContext(ctx, errInvalid); err != nil {
		return nil, err
	}
	defer s.host.mu.Unlock()
	if s.closed {
		return nil, errClosed
	}
	return s, nil
}

func (s *Session) Publish(ctx context.Context, c testpilot.Coordinate, slotID string, opaque testpilot.OpaqueHandle) error {
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return err
	}
	if err := s.host.mu.LockContext(ctx, errInvalid); err != nil {
		return err
	}
	defer s.host.mu.Unlock()
	if s.closed {
		return errClosed
	}
	slot := s.slots[slotID]
	handle, ok := opaque.(*opaqueHandle)
	if slot == nil || !ok || handle == nil || handle.session != s || handle.origin != c || handle.used {
		return errUnauthorized
	}
	if _, exists := s.handles[handle]; !exists {
		return errUnauthorized
	}
	if slot.claim != nil || slot.handle != nil && slot.handle != handle || handle.published != "" && handle.published != slotID {
		return errInvalid
	}
	if slot.handle == nil {
		slot.handle = handle
		handle.published = slotID
		close(slot.ready)
	}
	return nil
}

func (s *Session) Await(ctx context.Context, slotID string) error {
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return err
	}
	if err := s.host.mu.LockContext(ctx, errInvalid); err != nil {
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
		if err := s.host.mu.LockContext(ctx, errInvalid); err != nil {
			return err
		}
		defer s.host.mu.Unlock()
		if s.closed {
			return errClosed
		}
		if slot.handle.used {
			return errInvalid
		}
		return nil
	}
}

func (s *Session) Consume(ctx context.Context, slotID string) (testpilot.OpaqueHandle, error) {
	if err := s.host.mu.LockContext(ctx, errInvalid); err != nil {
		return nil, err
	}
	defer s.host.mu.Unlock()
	if s.closed {
		return nil, errClosed
	}
	slot := s.slots[slotID]
	if slot == nil || slot.handle == nil || slot.handle.used {
		return nil, errUnauthorized
	}
	if old := slot.claim; old != nil && !old.released.Load() && old.context.Err() == nil {
		return nil, errUnauthorized
	}
	claim := &handleClaim{handle: slot.handle, context: ctx}
	slot.claim = claim
	return claim, nil
}
