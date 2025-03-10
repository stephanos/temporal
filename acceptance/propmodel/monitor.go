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

package propmodel

import (
	"context"
	"fmt"

	"github.com/pborman/uuid"
	"go.temporal.io/server/common/persistence/intercept"
	. "go.temporal.io/server/common/proptest"
	"go.temporal.io/server/common/softassert"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

var (
	incomingMarker = incoming{}
)

type (
	Monitor struct {
		env *Env
	}
	grpcID      ID
	requestMsg  proto.Message
	requestPath string
	responseMsg proto.Message
	responseErr error
	incoming    struct{}
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
		defer func() {
			if r := recover(); r != nil {
				softassert.Fail(m.env.Logger, fmt.Sprintf("%v", r))
			}
		}()

		reqID := grpcID(uuid.New())
		baseEvt := NewInput(reqID, req.(requestMsg), requestPath(info.FullMethod))

		// process request event
		reqEvt := NewExtendedInput(baseEvt, incomingMarker)
		reqEvt.StringFunc = func() string {
			return fmt.Sprintf("[rpc] %v", req.(proto.Message).ProtoReflect().Descriptor().Name())
		}
		err := m.env.Send(reqEvt)
		if err != nil {
			m.env.Fatal(fmt.Sprintf("%v", err))
		}

		// process request
		resp, err := handler(ctx, req)
		if resp == nil {
			resp = new(proto.Message)
		}

		// process response event
		respEvt := NewExtendedInput(baseEvt, resp.(responseMsg), responseErr(err))
		respEvt.StringFunc = func() string {
			return fmt.Sprintf("[rpc] %v", resp.(proto.Message).ProtoReflect().Descriptor().Name())
		}
		err = m.env.Send(respEvt)
		if err != nil {
			m.env.Fatal(fmt.Sprintf("%v", err))
		}

		return resp, err
	}
}

func (m *Monitor) PersistenceInterceptor() intercept.PersistenceInterceptor {
	return func(method string, fn func() (any, error), params ...any) error {
		// process persistence event
		// TODO: add method?
		//sendArgs := params
		//reqEvt := NewInput(sendArgs...)
		//reqEvt.StringFunc = func() string {
		//	return fmt.Sprintf("[db] %v req", method)
		//}
		//report := m.env.Send(reqEvt)
		//if !report.Empty() {
		//	m.env.Fatal(fmt.Sprintf("%v", report))
		//}

		// process request
		_, err := fn()

		// process response event
		//sendArgs = append(sendArgs, resp, responseErr(err))
		//respEvt := NewInput(sendArgs...)
		//respEvt.StringFunc = func() string {
		//	return fmt.Sprintf("[db] %v resp", method)
		//}
		//report = m.env.Send(respEvt)
		//if !report.Empty() {
		//	m.env.Fatal(fmt.Sprintf("%v", report))
		//}

		return err
	}
}
