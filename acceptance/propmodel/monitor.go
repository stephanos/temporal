// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
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

package propmodel

import (
	"context"

	"github.com/pborman/uuid"
	. "go.temporal.io/server/common/proptest"
	"google.golang.org/grpc"
)

type (
	Monitor struct {
		env *Env
	}
	requestID   ID
	requestPath string
	responseErr error
)

func NewMonitor(env *Env) *Monitor {
	return &Monitor{
		env: env,
	}
}

func (i *Monitor) Interceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		reqID := requestID(uuid.New())

		// process request event
		i.env.Send(reqID, req, requestPath(info.FullMethod))

		// process request
		resp, err := handler(ctx, req)

		// process response event
		// NOTE: the `req` is not provided again to make only the response observers match
		i.env.Send(reqID, requestPath(info.FullMethod), resp, responseErr(err))

		return resp, err
	}
}
