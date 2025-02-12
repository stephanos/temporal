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
)

type (
	Server          Model[Server]
	StartOptions    struct{}
	ServerRequestID struct{}
	ServerRequest   struct {
		Ctx        context.Context
		Request    any
		FullMethod string
	}
	ServerResponse struct {
		*ServerRequest
		Response any
		Error    error
	}
	ServerImportInput struct{}
	ServerStartInput  struct{}
)

var (
	// ==== ModelType

	serverModel = NewModelType[Server]()

	// ==== Inputs

	ServerAddr = NewVar[string]("ServerAddr")

	// ==== Transitions

	//NewTransition(stateLocalServerStarting, StartingWorkflowUpdate, stateLocalServerStarted),

	// ==== States

	stateLocalServerStarting = NewState("Starting")
	stateLocalServerStarted  = NewState("Started")

	// ==== Events

	_ = NewInitHandler("Import",
		func(s Server, input *ServerImportInput) ID {
			// TODO
			//s.NewState(stateLocalServerStarting)
			return uuid.New()
		})

	_ = NewInitHandler("Start",
		func(s Server, input *ServerStartInput) ID {
			//testcore.TestFlags.TestClusterConfigFile = "../tests/testdata/es_cluster.yaml"
			//s := testcore.FunctionalTestBase{Suite: suite.Suite{}}
			//s.Suite.SetT(w.Run().T())
			//s.SetupSuiteWithDefaultCluster(testcore.WithAdditionalGrpcInterceptors(
			//	testcore.NewPropTestInterceptor(
			//		func(ctx context.Context, req any) (any, error) {
			//			return nil, nil
			//		},
			//		func(ctx context.Context, resp any) (any, error) {
			//			return nil, nil
			//		},
			//	).Intercept,
			//))
			//w.Update(stateLocalServerStarting)
			return uuid.New()
		})

	_ = NewUpdateHandler("Request",
		[]*State{stateLocalServerStarted},
		func(s Server, req *ServerRequest) *ServerResponse {
			// TODO
			return nil
		})
)
