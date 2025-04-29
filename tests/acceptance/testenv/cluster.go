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

package testenv

import (
	"cmp"
	"context"
	"fmt"
	"sync"

	"github.com/pborman/uuid"
	"go.temporal.io/server/common/persistence"
	"go.temporal.io/server/common/persistence/intercept"
	"go.temporal.io/server/common/softassert"
	"go.temporal.io/server/common/testing/stamp"
	"go.temporal.io/server/tests/acceptance/model"
	"go.temporal.io/server/tests/acceptance/testenv/action"
	"go.temporal.io/server/tests/testcore"
)

var (
	actionIdKey      = "stamp-action-id"
	clusterStartLock sync.Mutex
)

type Cluster struct {
	stamp.ActorModel[*model.Cluster]
	mdlEnv   *stamp.ModelEnv
	scenario *stamp.Scenario
	tbase    *testcore.FunctionalTestBase
}

func newCluster(
	s *stamp.Scenario,
	mdlEnv *stamp.ModelEnv,
) *Cluster {
	// TODO: move reporting here: mdlEnv.SetCreateHook(func() { ... }
	mdlEnv.Root().SetActionHandler(
		func(params stamp.ActionParams) error {
			switch trg := params.Payload.(type) {
			case model.ClusterStarted:
				mdlEnv.Route(&model.IncomingAction[any]{
					TriggerID: params.ActID,
					Cluster:   trg.ClusterName,
					Request:   params.Payload,
				})
			default:
				s.T().Fatalf("unexpected action %T", params.Payload)
			}
			return nil
		})

	c := stamp.Act(mdlEnv.Root(), action.StartCluster{ClusterName: "local"})
	return &Cluster{
		ActorModel: stamp.NewActorModel(c),
		mdlEnv:     mdlEnv,
		scenario:   s,
	}
}

func (c *Cluster) OnAction(params stamp.ActionParams) error {
	switch t := params.Payload.(type) {
	case *persistence.CreateNamespaceRequest:
		// tagging the request with a trigger ID to match it to the action
		ctx := context.WithValue(c.scenario.Context(), actionIdKey, params.ActID)
		_, _ = c.tbase.GetTestCluster().TestBase().MetadataManager.CreateNamespace(ctx, t)
		_ = c.mdlEnv.Route(&model.IncomingAction[model.NamespaceCreated]{
			TriggerID: params.ActID,
			Cluster:   model.ClusterName(c.GetID()),
			Request:   model.NamespaceCreated{Name: t.Namespace.Info.Name},
		})

	case model.NewTaskQueue,
		model.NewWorkflowClient,
		model.NewWorkflowWorker:
		_ = c.mdlEnv.Route(&model.IncomingAction[any]{
			TriggerID: params.ActID,
			Cluster:   model.ClusterName(c.GetID()),
			Request:   t,
		})

	default:
		panic(fmt.Sprintf("unhandled action %T", t))
	}

	return nil
}

func (c *Cluster) dbInterceptor() intercept.PersistenceInterceptor {
	return func(methodName string, fn func() (any, error), params ...any) error {
		defer func() {
			if r := recover(); r != nil {
				softassert.Fail(c.scenario.Logger(), fmt.Sprintf("%v", r))
			}
		}()

		var actID stamp.ActID
		var reqArgs []any
		for _, p := range params {
			if ctx, ok := p.(context.Context); ok {
				if ctxVal, ok := ctx.Value(actionIdKey).(stamp.ActID); ok {
					actID = stamp.ActID(ctxVal)
				}
				continue // no value in adding context to the action
			}
			reqArgs = append(reqArgs, p)
		}
		actID = cmp.Or(actID, stamp.ActID(uuid.New()))

		incAction := &model.IncomingAction[any]{
			TriggerID: actID,
			Cluster:   model.ClusterName(c.GetID()),
			RequestID: uuid.New(),
			Method:    methodName,
		}
		if len(reqArgs) == 1 {
			incAction.Request = reqArgs[0]
		} else {
			incAction.Request = reqArgs
		}

		// handle request in model
		onResp := c.mdlEnv.Route(incAction)

		// process request
		resp, err := fn()

		// handle response in model
		if onResp != nil {
			outAction := model.OutgoingAction[any]{ActID: actID}
			outAction.Response = resp
			outAction.ResponseErr = err
			onResp(outAction)
		}

		return err
	}
}
