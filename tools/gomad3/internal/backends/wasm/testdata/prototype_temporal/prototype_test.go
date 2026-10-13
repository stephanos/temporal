// Package prototypetemporal holds the fn-151 determinism prototype's Temporal
// workload. It runs in one WASM guest under strict virtual time while the host
// network model resets and partitions connections.
package prototypetemporal

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.temporal.io/server/common/testing/testcontext"
	"go.temporal.io/server/tests/testcore"
)

var started = time.Now()

func virtual() time.Duration { return time.Since(started) }

func flakyActivity(ctx context.Context, step int) (string, error) {
	if activity.GetInfo(ctx).Attempt == 1 {
		return "", errors.New("first attempt fails")
	}
	return fmt.Sprintf("step %d attempt %d", step, activity.GetInfo(ctx).Attempt), nil
}

// PrototypeWorkflow sleeps and retries an activity across a 30-second backoff,
// so it completes only if timers fire on virtual time.
func PrototypeWorkflow(ctx workflow.Context) ([]string, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{InitialInterval: 30 * time.Second, BackoffCoefficient: 1, MaximumAttempts: 3},
	})
	var results []string
	for step := range 2 {
		if err := workflow.Sleep(ctx, time.Minute); err != nil {
			return nil, err
		}
		var result string
		if err := workflow.ExecuteActivity(ctx, flakyActivity, step).Get(ctx, &result); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// TestPrototypeWorkflow runs three minutes of workflow time (fn-151.6). The
// host declares network faults inside that window (fn-151.7); the workflow
// must complete regardless. The first activity attempt fails by design, so its
// error log must not fail the test.
func TestPrototypeWorkflow(t *testing.T) {
	testcontext.For(t, testcontext.WithTimeout(time.Hour))
	env := testcore.NewEnv(t, testcore.WithInMemorySQLitePersistence(), testcore.WithDisableTestloggerFailure())
	env.SdkWorker().RegisterWorkflow(PrototypeWorkflow)
	env.SdkWorker().RegisterActivity(flakyActivity)
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	begin := virtual()
	fmt.Printf("prototype: workflow starting at virtual %v\n", begin)
	run, err := env.SdkClient().ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: "prototype-workflow", TaskQueue: env.WorkerTaskQueue()}, PrototypeWorkflow)
	if err != nil {
		t.Fatal(err)
	}
	var results []string
	if err := run.Get(ctx, &results); err != nil {
		t.Fatal(err)
	}
	took := virtual() - begin
	fmt.Printf("prototype: workflow completed %v after virtual %v\n", results, took)
	if len(results) != 2 || took < 3*time.Minute {
		t.Fatalf("workflow finished too early or incomplete: %v after %v", results, took)
	}
}
