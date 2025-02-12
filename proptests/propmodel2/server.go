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
	"google.golang.org/grpc"
)

type (
	Server struct {
		env *Env
	}
	serverRequestID struct{}
	serverRequest   struct {
		ctx        context.Context
		Request    any
		FullMethod string
	}
	serverResponse struct {
		*serverRequest
		Response any
		Error    error
	}
)

func NewServer(env *Env) *Server {
	return &Server{
		env: env,
	}
}

func (s *Server) Interceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		ctx = context.WithValue(ctx, serverRequestID{}, uuid.New())

		reqObj := &serverRequest{
			ctx:        ctx,
			Request:    req,
			FullMethod: info.FullMethod,
		}
		if respObj, ok := s.handleRequest(reqObj); ok {
			return respObj.Response, respObj.Error
		}

		resp, err := handler(ctx, req)
		respObj := &serverResponse{
			serverRequest: reqObj,
			Response:      resp,
			Error:         err,
		}
		if respObj, ok := s.handleResponse(respObj); ok {
			return respObj.Response, respObj.Error
		}
		return resp, err
	}
}

func (s *Server) handleRequest(req *serverRequest) (*serverResponse, bool) {
	return nil, false
}

func (s *Server) handleResponse(resp *serverResponse) (*serverResponse, bool) {
	return nil, false
}
