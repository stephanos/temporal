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

package env

import (
	"context"
	"fmt"

	"github.com/pborman/uuid"
	"go.temporal.io/server/acceptance/model"
	"go.temporal.io/server/common/persistence/intercept"
	"go.temporal.io/server/common/softassert"
	"go.temporal.io/server/common/testing/stamp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	triggerIdKey = "trigger-id"
)

type (
	clusterMonitor struct {
		env         *stamp.ModelEnv
		clusterName model.ClusterName
	}
)

func newClusterMonitor(env *stamp.ModelEnv, clusterName model.ClusterName) *clusterMonitor {
	return &clusterMonitor{
		env:         env,
		clusterName: clusterName,
	}
}

func (m *clusterMonitor) GrpcInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		defer func() {
			if r := recover(); r != nil {
				softassert.Fail(m.env.Logger, fmt.Sprintf("%v", r))
			}
		}()

		var triggerID stamp.TriggerID
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			if _, ok := md[triggerIdKey]; ok {
				triggerID = stamp.TriggerID(md.Get(triggerIdKey)[0])
			}
		}

		reqID := model.ActionID(uuid.New())
		baseAction := stamp.NewAction(m.clusterName, reqID, model.Method(info.FullMethod), req.(model.RequestMsg))

		// handle request
		reqAction := baseAction.Append(model.IncomingMarker)
		m.env.Handle(reqAction, triggerID)

		// process request
		resp, err := handler(ctx, req)

		// handle response
		var respMsg model.ResponseMsg
		if resp != nil {
			respMsg = resp.(model.ResponseMsg)
		}
		respAction := baseAction.Append(respMsg, model.ResponseErr(err))
		m.env.Handle(respAction, triggerID)

		return resp, err
	}
}

func (m *clusterMonitor) PersistenceInterceptor() intercept.PersistenceInterceptor {
	return func(methodName string, fn func() (any, error), params ...any) error {
		defer func() {
			if r := recover(); r != nil {
				softassert.Fail(m.env.Logger, fmt.Sprintf("%v", r))
			}
		}()

		var triggerID stamp.TriggerID
		var sendArgs []any
		for _, p := range params {
			if ctx, ok := p.(context.Context); ok {
				if ctxVal, ok := ctx.Value(triggerIdKey).(string); ok {
					triggerID = stamp.TriggerID(ctxVal)
				}
				continue // no value in adding context to action
			}
			sendArgs = append(sendArgs, p)
		}

		reqID := model.ActionID(uuid.New())
		sendArgs = append(sendArgs, m.clusterName, reqID, model.Method(methodName))
		baseAction := stamp.NewAction(sendArgs...)

		// handle request
		reqAction := baseAction.Append(model.IncomingMarker)
		m.env.Handle(reqAction, triggerID)

		// TODO: support modifying the request/response

		// process request
		resp, err := fn()

		// handle response
		sendArgs = append(sendArgs, resp, model.ResponseErr(err))
		respAction := baseAction.Append(resp, model.ResponseErr(err))
		m.env.Handle(respAction, triggerID)

		return err
	}
}
