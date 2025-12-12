package faultinject

import (
	"context"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc"
)

// RPCReqCallback is a callback function for RPC request fault injection.
// It receives context, method name, and request, and can decide whether to inject a fault.
//
// Parameters:
//   - ctx: the request context
//   - fullMethod: the full gRPC method name
//   - req: the request proto message
//
// Returns:
//   - bool: if true, the callback matched and the error (even if nil) should be used.
//     If false, the callback did not match and other callbacks should be checked.
//   - error: if non-nil, this error will be returned to the caller instead of proceeding.
type RPCReqCallback func(ctx context.Context, fullMethod string, req interface{}) (bool, error)

// rpcReqCallbackEntry represents a registered RPC callback with its ID.
type rpcReqCallbackEntry struct {
	id       uint64
	callback RPCReqCallback
}

// RPCReqGenerator handles fault injection for RPC requests.
type RPCReqGenerator struct {
	mu        sync.RWMutex
	callbacks []rpcReqCallbackEntry
	nextID    atomic.Uint64
}

// NewRPCReqGenerator creates a new RPCReqGenerator instance.
func NewRPCReqGenerator() *RPCReqGenerator {
	return &RPCReqGenerator{
		callbacks: make([]rpcReqCallbackEntry, 0),
	}
}

// RegisterCallback registers an RPC fault injection callback and returns a
// cleanup function that removes the callback when called.
func (r *RPCReqGenerator) RegisterCallback(cb RPCReqCallback) func() {
	if r == nil {
		return func() {}
	}
	id := r.nextID.Add(1)
	entry := rpcReqCallbackEntry{id: id, callback: cb}

	r.mu.Lock()
	r.callbacks = append(r.callbacks, entry)
	r.mu.Unlock()

	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		for i, e := range r.callbacks {
			if e.id == id {
				r.callbacks = append(r.callbacks[:i], r.callbacks[i+1:]...)
				return
			}
		}
	}
}

// Generate checks all registered RPC callbacks for the given request.
// Returns (true, err) if a callback matched, or (false, nil) if no callbacks matched.
func (r *RPCReqGenerator) Generate(ctx context.Context, fullMethod string, req interface{}) (bool, error) {
	if r == nil {
		return false, nil
	}
	r.mu.RLock()
	callbacks := make([]rpcReqCallbackEntry, len(r.callbacks))
	copy(callbacks, r.callbacks)
	r.mu.RUnlock()

	for _, entry := range callbacks {
		if matched, err := entry.callback(ctx, fullMethod, req); matched {
			return true, err
		}
	}
	return false, nil
}

// HasCallbacks returns true if there are any registered callbacks.
func (r *RPCReqGenerator) HasCallbacks() bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.callbacks) > 0
}

// GRPCUnaryServerInterceptor returns a unary server interceptor that checks for
// dynamically registered fault injection callbacks before proceeding with the handler.
//
// This is primarily used for testing, allowing tests to register callbacks that
// can inspect requests and inject faults on demand.
//
// Behavior:
// - If generator is nil, the handler proceeds normally.
// - If a callback matches (returns true), the callback's error is returned.
// - If no callbacks match, the handler proceeds normally.
func GRPCUnaryServerInterceptor(generator *RPCReqGenerator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if generator == nil {
			return handler(ctx, req)
		}
		if matched, err := generator.Generate(ctx, info.FullMethod, req); matched {
			return nil, err
		}
		return handler(ctx, req)
	}
}
