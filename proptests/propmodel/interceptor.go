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

var _ grpc.UnaryServerInterceptor = (*TestServerInterceptor)(nil).Intercept

type TestServerInterceptor struct {
	run Run
}

func NewPropTestInterceptor(
	run Run,
) *TestServerInterceptor {
	return &TestServerInterceptor{
		run: run,
	}
}

func (i *TestServerInterceptor) Intercept(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	ctx = context.WithValue(ctx, ServerRequestID{}, uuid.New())
	mdl := Get[Server](i.run) // TODO: use "ID"

	reqObj := &ServerRequest{
		Ctx:        ctx,
		Request:    req,
		FullMethod: info.FullMethod,
	}
	if respObj, ok := Update[*ServerResponse](mdl, reqObj); ok {
		return respObj.Response, respObj.Error
	}

	resp, err := handler(ctx, req)
	respObj := &ServerResponse{
		ServerRequest: reqObj,
		Response:      resp,
		Error:         err,
	}
	if respObj, ok := Update[*ServerResponse](mdl, respObj); ok {
		return respObj.Response, respObj.Error
	}
	return resp, err
}
