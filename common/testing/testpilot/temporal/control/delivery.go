package control

import (
	"context"
	"errors"
	"reflect"
	"sync"
)

var (
	ErrNotHeld = errors.New("delivery has not reached the hold")
	ErrClosed  = errors.New("delivery control is closed")
	ErrInvalid = errors.New("invalid delivery control")
)

// DeliveryGate holds one declared identity, including concurrent redeliveries. Closing it cancels
// blocked deliveries rather than allowing cleanup to dispatch work the controller never released.
type DeliveryGate[K comparable] struct {
	key      K
	mu       sync.Mutex
	held     bool
	released bool
	closed   bool
	arrival  chan struct{}
	release  chan struct{}
	shutdown chan struct{}
}

func NewDeliveryGate[K comparable](key K) *DeliveryGate[K] {
	return &DeliveryGate[K]{key: key, arrival: make(chan struct{}), release: make(chan struct{}), shutdown: make(chan struct{})}
}

func (g *DeliveryGate[K]) Arrive(ctx context.Context, key K) error {
	if g == nil || g.arrival == nil || ctx == nil {
		return ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !comparableIdentity(key) || !comparableIdentity(g.key) {
		return ErrInvalid
	}
	if key != g.key {
		return nil
	}
	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return ErrClosed
	}
	if !g.held {
		g.held = true
		close(g.arrival)
	}
	g.mu.Unlock()
	return g.wait(ctx, g.release)
}

func (g *DeliveryGate[K]) WaitHeld(ctx context.Context) error {
	if g == nil || g.arrival == nil || ctx == nil {
		return ErrInvalid
	}
	return g.wait(ctx, g.arrival)
}

func (g *DeliveryGate[K]) wait(ctx context.Context, ready <-chan struct{}) error {
	select {
	case <-ctx.Done():
	case <-g.shutdown:
	case <-ready:
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return ErrClosed
	}
	return nil
}

func (g *DeliveryGate[K]) Release() error {
	if g == nil || g.arrival == nil {
		return ErrInvalid
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return ErrClosed
	}
	if !g.held {
		return ErrNotHeld
	}
	if !g.released {
		g.released = true
		close(g.release)
	}
	return nil
}

func (g *DeliveryGate[K]) Close() {
	if g == nil || g.arrival == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.closed {
		g.closed = true
		close(g.shutdown)
	}
}

func comparableIdentity(key any) bool {
	value := reflect.ValueOf(key)
	return !value.IsValid() || value.Comparable()
}
