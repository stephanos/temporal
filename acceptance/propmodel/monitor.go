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
	"fmt"

	"github.com/pborman/uuid"
	"go.temporal.io/server/common/persistence/intercept"
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

func (m *Monitor) GrpcInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		reqID := requestID(uuid.New())

		// process request event
		report := m.env.Send(reqID, req, requestPath(info.FullMethod))
		if !report.Empty() {
			m.env.Fatal(fmt.Sprintf("%v", report))
		}

		// process request
		resp, err := handler(ctx, req)

		// process response event
		// NOTE: the `req` is not provided again to make only the response observers match
		report = m.env.Send(reqID, requestPath(info.FullMethod), resp, responseErr(err))
		if !report.Empty() {
			m.env.Fatal(fmt.Sprintf("%v", report))
		}

		return resp, err
	}
}

func (m *Monitor) PersistenceInterceptor() intercept.PersistenceInterceptor {
	return func(method string, fn func() (any, error), params ...any) error {
		reqID := requestID(uuid.New())

		// process persistence event
		sendArgs := append([]any{reqID}, params...)
		report := m.env.Send(sendArgs...)
		if !report.Empty() {
			m.env.Fatal(fmt.Sprintf("%v", report))
		}

		// process request
		resp, err := fn()

		// process response event
		sendArgs = append(sendArgs, resp, responseErr(err))
		report = m.env.Send(sendArgs...)
		if !report.Empty() {
			m.env.Fatal(fmt.Sprintf("%v", report))
		}

		return err
	}
}
