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
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/acceptance/model"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/common/persistence"
	"go.temporal.io/server/common/testing/stamp"
	"go.temporal.io/server/common/testing/testlogger"
	"go.temporal.io/server/tests/testcore"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type (
	testEnv struct {
		mdlEnv  *stamp.ModelEnv
		cluster *model.Cluster
		t       *testing.T
		tl      *testlogger.TestLogger
		tb      *testcore.FunctionalTestBase
	}
)

func NewCluster(t *testing.T) *model.Cluster {
	tl := testlogger.NewTestLogger(t, testlogger.FailOnExpectedErrorOnly)
	testlogger.DontPanicOnError(tl)
	tl.Expect(testlogger.Error, ".*", tag.FailedAssertion)

	tenv := &testEnv{t: t, tl: tl}
	tenv.mdlEnv = stamp.NewModelEnv(tl, *model.TemporalModel, tenv)

	root := tenv.mdlEnv.Root()
	tenv.cluster = stamp.Trigger(root, model.StartCluster{ClusterName: "local"})
	t.Cleanup(func() {
		stamp.Trigger(root, model.StopCluster{Cluster: tenv.cluster})
	})

	return tenv.cluster
}

// ==== Effect handlers

func (t *testEnv) OnStartCluster(action model.StartCluster) {
	monitor := newClusterMonitor(t.mdlEnv, action.ClusterName)
	// TODO: cache and only start one at a time
	t.tb = &testcore.FunctionalTestBase{}
	t.tb.Logger = t.mdlEnv.Logger
	t.tb.SetT(t.t) // TODO: drop this; requires cluster setup to be decoupled from testify
	t.tb.SetupSuiteWithCluster(
		"../tests/testdata/es_cluster.yaml",
		// TODO: need to make this the 1st interceptor so it can observe every request
		testcore.WithAdditionalGrpcInterceptors(monitor.GrpcInterceptor()),
		testcore.WithPersistenceInterceptor(monitor.PersistenceInterceptor()),
	)
}

func (t *testEnv) OnStopCluster(_ model.StopCluster) {
	if t.tl.ResetFailureStatus() {
		panic(`Failing test as unexpected error logs were found.
Look for 'Unexpected Error log encountered'.`)
	}
	t.tb.TearDownCluster()
}

func (t *testEnv) OnCreateNamespace(action *persistence.CreateNamespaceRequest, triggerID stamp.TriggerID) {
	// tagging the request with a trigger ID to match it to the action (see clusterMonitor)
	ctx := context.WithValue(t.context(), triggerIdKey, triggerID)

	_, _ = t.tb.GetTestCluster().TestBase().MetadataManager.CreateNamespace(ctx, action)
}

func (t *testEnv) OnClientRequest(_ *model.WorkflowClient, req proto.Message, triggerID stamp.TriggerID) {
	sendWorkflowRPC(t, t.tb.FrontendClient(), req, triggerID)
}

func (t *testEnv) OnWorkerRequest(_ *model.WorkflowWorker, req proto.Message, triggerID stamp.TriggerID) {
	sendWorkflowRPC(t, t.tb.FrontendClient(), req, triggerID)
}

func sendWorkflowRPC(c *testEnv, client workflowservice.WorkflowServiceClient, req proto.Message, triggerID stamp.TriggerID) {
	// tagging the request with a trigger ID to match it to the action (see clusterMonitor)
	md := metadata.Pairs(triggerIdKey, string(triggerID))
	ctx := metadata.NewOutgoingContext(c.context(), md)

	reflect.ValueOf(client).
		MethodByName(strings.TrimSuffix(string(req.ProtoReflect().Descriptor().Name()), "Request")).
		Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(req)})
}

func (t *testEnv) context() context.Context {
	return context.Background()
}
