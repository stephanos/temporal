package worker

import (
	"context"
	"sync"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"google.golang.org/protobuf/proto"
)

type cancelActivation func(context.Context, string, string, string) error

type reservation struct {
	mu              sync.Mutex
	identity        umpire.ReservationIdentity
	done            chan struct{}
	bound           chan struct{}
	consumed        bool
	completed       bool
	cancelRequested bool
	canceling       bool
	cancelSent      bool
	workflowID      string
	temporalRunID   string
	requestID       string
	cancel          cancelActivation
	result          umpire.EffectResult
	err             error
}

func newReservation(identity umpire.ReservationIdentity) *reservation {
	return &reservation{identity: identity, done: make(chan struct{}), bound: make(chan struct{})}
}

func (r *reservation) Identity() umpire.ReservationIdentity { return r.identity }

func (r *reservation) Consume(ctx context.Context) (umpire.Coordinate, error) {
	if err := ctx.Err(); err != nil {
		return umpire.Coordinate{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.completed || r.consumed {
		return umpire.Coordinate{}, ErrClosed
	}
	r.consumed = true
	coordinate := umpire.Coordinate{RunID: r.identity.Origin.RunID, EntrypointID: r.identity.EntrypointID, ActivationID: r.identity.ID, Attempt: r.identity.Origin.Attempt}
	return coordinate, nil
}

func (r *reservation) bindCancellation(workflowID, temporalRunID, requestID string, cancel cancelActivation) error {
	workflow := workflowID != "" || temporalRunID != ""
	nexus := requestID != ""
	if cancel == nil || workflow == nexus || workflow && (workflowID == "" || temporalRunID == "") {
		return ErrInvalid
	}
	r.mu.Lock()
	if !r.consumed || r.completed || r.cancel != nil {
		r.mu.Unlock()
		return ErrClosed
	}
	r.workflowID, r.temporalRunID, r.requestID, r.cancel = workflowID, temporalRunID, requestID, cancel
	close(r.bound)
	r.mu.Unlock()
	return nil
}

func (r *reservation) Wait(ctx context.Context) (umpire.EffectResult, error) {
	select {
	case <-ctx.Done():
		return umpire.EffectResult{}, ctx.Err()
	case <-r.done:
		r.mu.Lock()
		defer r.mu.Unlock()
		return cloneEffectResult(r.result), r.err
	}
}

func (r *reservation) Cancel(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for {
		r.mu.Lock()
		if r.completed || r.cancelSent {
			r.mu.Unlock()
			return nil
		}
		if r.canceling {
			r.mu.Unlock()
			return ErrCancellationInFlight
		}
		r.cancelRequested = true
		if !r.consumed {
			r.completeLocked(umpire.EffectResult{Outcome: &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_CANCELED}}, nil)
			r.mu.Unlock()
			return nil
		}
		cancel := r.cancel
		workflowID, temporalRunID, requestID := r.workflowID, r.temporalRunID, r.requestID
		if cancel != nil {
			r.canceling = true
			r.mu.Unlock()
			return r.sendCancellation(ctx, cancel, workflowID, temporalRunID, requestID)
		}
		bound := r.bound
		r.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-bound:
		}
	}
}

func (r *reservation) Drain(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.done:
		return nil
	}
}

func (r *reservation) sendCancellation(ctx context.Context, cancel cancelActivation, workflowID, temporalRunID, requestID string) error {
	err := cancel(ctx, workflowID, temporalRunID, requestID)
	r.mu.Lock()
	r.canceling = false
	if err == nil {
		r.cancelSent = true
	}
	r.mu.Unlock()
	return err
}

func (r *reservation) finish(result umpire.EffectResult, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.completeLocked(result, err)
}

func (r *reservation) completeLocked(result umpire.EffectResult, err error) {
	if r.completed {
		return
	}
	r.completed = true
	r.result, r.err = cloneEffectResult(result), err
	close(r.done)
}

func cloneEffectResult(result umpire.EffectResult) umpire.EffectResult {
	return umpire.EffectResult{Outcome: cloneOutcome(result.Outcome), Response: proto.Clone(result.Response)}
}
