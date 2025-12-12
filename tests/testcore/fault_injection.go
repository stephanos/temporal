package testcore

import (
	"context"
	"sync/atomic"

	"go.temporal.io/server/common/testing/protoutils"
	"google.golang.org/protobuf/proto"
)

// InjectRPCFault registers a fault injection that applies to all services
// (frontend, history, matching). It matches incoming RPC requests against the
// provided partial proto message and injects the specified error when a match is found.
//
// The partialReq parameter should be a proto message with only the fields set that
// you want to match against. All set fields must match for the fault to be injected.
//
// The test will fail if the fault is never injected. This ensures tests don't
// silently pass when the fault injection didn't trigger.
//
// Example:
//
//	testcore.InjectRPCFault(
//	    s.GetTestCluster(),
//	    &matchingservice.AddWorkflowTaskRequest{
//	        Speculative: true,
//	    },
//	    serviceerror.NewUnavailable("injected fault"),
//	)
func InjectRPCFault(tc *TestCluster, partialReq proto.Message, err error) {
	t := tc.t
	t.Helper()
	generator := tc.Host().GetFaultInjector()
	if generator == nil {
		t.Fatal("fault injector is nil")
		return
	}

	var firedCount atomic.Int64

	cleanup := generator.RegisterCallback(func(ctx context.Context, fullMethod string, req any) (bool, error) {
		protoReq, ok := req.(proto.Message)
		if !ok {
			return false, nil
		}

		if protoutils.MatchesPartialProto(partialReq, protoReq) {
			firedCount.Add(1)
			t.Logf("Fault injection matched: %T", protoReq)
			return true, err
		}
		return false, nil
	})

	t.Cleanup(func() {
		cleanup()
		if firedCount.Load() == 0 {
			t.Error("fault injection was registered but never fired - the fault was never injected")
		}
	})
}
