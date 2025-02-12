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
	"go.temporal.io/api/workflowservice/v1"
	. "go.temporal.io/server/common/proptest"
	"go.temporal.io/server/common/proptest/require"
)

type Namespace Model[Namespace]

var (
	// ==== ModelType

	namespaceModel = NewModelType[Namespace]()

	// ==== States

	stateNamespaceRegistering = NewState("Registering")
	stateNamespaceRegistered  = NewState("Registered")

	// ==== Events

	RegisteringNamespace = NewInitHandler("Registering",
		func(ns Namespace, req *workflowservice.RegisterNamespaceRequest) ID {
			require.Zero(ns, req)
			require.NotZero(ns, req)
			NamespaceName.Set(ns, req.Namespace)
			return req.Namespace
		})

	RegisteredNamespace = NewSignalHandler("Registered",
		[]*State{stateNamespaceRegistering},
		func(ns Namespace, resp *workflowservice.RegisterNamespaceResponse) {
			// TODO
		})

	// ==== Variables

	NamespaceName = NewVar[string]("NamespaceName")
)
