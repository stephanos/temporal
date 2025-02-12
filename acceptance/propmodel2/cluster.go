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
	"reflect"
	"strings"
	"sync"

	"go.temporal.io/server/common/api"
	"go.temporal.io/server/common/persistence"
	. "go.temporal.io/server/common/proptest2"
	"google.golang.org/grpc"
)

type (
	Cluster struct {
		Model[*Cluster]
		*Root

		interceptorLock sync.Mutex
	}
	handlerInfo interface {
		Identity() string
	}
	handlerInfoData struct {
		identity string
	}
	gRPCMethod struct {
		inputType  reflect.Type
		outputType reflect.Type
	}
)

func NewCluster(
	env *Env,
	name string, // TODO: better type
) *Cluster {
	return GetOrCreateModel[*Cluster](
		name,
		&Cluster{
			Root: NewRoot(env),
		},

		WithInputHandler(
			func(f *Cluster, evt string) error {
				f.MustSetID(evt)
				return nil
			}))
}

func (s *Cluster) Interceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		handlerInfo := &handlerInfoData{
			identity: info.Server.(handlerInfo).Identity(),
		}

		// process request event
		s.interceptorLock.Lock()
		var respHandler ResponseHandlerProvider
		switch {
		case strings.HasPrefix(info.FullMethod, api.OperatorServicePrefix),
			strings.HasPrefix(info.FullMethod, api.AdminServicePrefix),
			strings.HasPrefix(info.FullMethod, api.WorkflowServicePrefix):
			respHandler = GetFrontend(s, handlerInfo).handle(req)
		case strings.HasPrefix(info.FullMethod, api.HistoryServicePrefix):
			respHandler = GetHistory(s, handlerInfo).handle(req)
		case strings.HasPrefix(info.FullMethod, api.MatchingServicePrefix):
			respHandler = GetMatching(s, handlerInfo).handle(req)
		default:
			panic(fmt.Sprintf("unknown service method %q", info.FullMethod))
		}
		s.interceptorLock.Unlock()

		// process request
		resp, err := handler(ctx, req)

		// process response event
		s.interceptorLock.Lock()
		if respHandler != nil {
			respHandler.ResponseHandler()(resp, err)
		}
		s.interceptorLock.Unlock()

		return resp, err
	}
}

func (s *Cluster) ImportNamespace(req *persistence.CreateNamespaceRequest) {
	newNamespace(NewParent(s), req)
}

func extractMethodsFromGRPCService(service interface{}) []gRPCMethod {
	t := reflect.TypeOf(service)
	if t.Kind() != reflect.Pointer {
		panic("expected a pointer to a gRPC service")
	}
	t = t.Elem()

	var methods []gRPCMethod
	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		if !method.IsExported() {
			continue
		}
		mType := method.Type
		methods = append(methods, gRPCMethod{
			inputType:  mType.In(1), // context.Context is always first
			outputType: mType.Out(0),
		})
	}
	return methods
}
