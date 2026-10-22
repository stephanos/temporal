package wasi

import (
	"context"
	"errors"

	"go.temporal.io/server/tools/gomad/internal/model/world"
)

// Prepare owns host continuation waits and transport dispatch only. Model
// ordering, request generations and state transitions remain in broker.go.
func (s *BrokerSession) Prepare(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		ready, changed, err := s.acquirePreparation()
		if err != nil || ready {
			return err
		}
		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *BrokerSession) wait(ctx context.Context) error {
	b := s.broker
	for {
		b.mu.Lock()
		if s.err != nil {
			err := s.err
			b.mu.Unlock()
			return err
		}
		if b.sessions[s.Node.ID] != s || s.state == "finished" || s.state == "revoked" {
			b.mu.Unlock()
			return invalid("stale broker incarnation")
		}
		if b.active == s {
			b.mu.Unlock()
			return nil
		}
		changed := b.changed
		b.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *BrokerSession) Activate(ctx context.Context) error { return s.wait(ctx) }

func (s *BrokerSession) Handle(ctx context.Context, e *Environment, call Call) (Reply, error) {
	if err := s.wait(ctx); err != nil {
		return Reply{}, err
	}
	if err := s.beginCall(e, call); err != nil {
		return Reply{}, err
	}
	if call.Op == "runtime_idle" {
		_, deadline, err := e.runtimeIdleInput(call.Input)
		if err != nil {
			return Reply{}, err
		}
		now, err := s.idle(ctx, deadline)
		if err != nil {
			return Reply{}, err
		}
		e.brokerTime = &now
	}
	reply, err := e.Handle(call)
	if err != nil {
		return Reply{}, err
	}
	if s.afterHandle != nil {
		s.afterHandle()
	}
	return reply, s.finishCall(e, call)
}

func (s *BrokerSession) idle(ctx context.Context, deadline uint64) (uint64, error) {
	request, err := s.prepareIdle(deadline)
	if err != nil {
		return 0, err
	}
	response, err := s.broker.clock.Quiesce(ctx, s.participant, request)
	err = s.resumeIdle(response, err)
	if err != nil {
		return 0, err
	}
	if err = s.wait(ctx); err != nil {
		return 0, err
	}
	if response.Kind == 3 {
		return 0, errors.New("WASM broker global deadlock")
	}
	return uint64(response.Time - int64(world.InitialTime)), nil
}
