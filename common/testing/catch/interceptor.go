// The MIT License
//
// Copyright (c) 2025 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package catch

import (
	"context"

	"google.golang.org/grpc"
	"go.temporal.io/server/tools/catch/pitcher"
)

// UnaryServerInterceptor returns a gRPC unary interceptor that injects faults via the global pitcher.
// This interceptor should be installed in the test cluster to enable fault injection for all RPC methods.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Check if pitcher is configured
		p := pitcher.Get()
		if p == nil {
			// No pitcher configured, pass through
			return handler(ctx, req)
		}

		// Try to make a play based on the request type
		playMade, err := p.MakePlay(ctx, req, req)
		if err != nil {
			// Play was made (fault injected), return the error
			// TODO: Send playMade to scorebook for recording
			_ = playMade
			return nil, err
		}

		// No play made, continue with normal execution
		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a gRPC stream interceptor that injects faults via the global pitcher.
// For streams, we inject faults when the stream is first opened.
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Check if pitcher is configured
		p := pitcher.Get()
		if p == nil {
			// No pitcher configured, pass through
			return handler(srv, ss)
		}

		// For streams, we use a placeholder request type based on the method name
		// In practice, we'd need to know the request type for each stream method
		// For now, we skip fault injection for streams (can be enhanced later)

		// No fault injected for streams yet, continue with normal execution
		return handler(srv, ss)
	}
}
