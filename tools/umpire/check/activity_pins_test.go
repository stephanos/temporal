package check

// The standalone activity Model, as model/ir/activity-standalone.json carries it: the sizes, rows and answers its
// Scala pins asserted, read off the tables and receipts Go derives from the IR.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/interp"
)

func TestActivityProductTable(t *testing.T) {
	product := built(t, activityModel(t))["activityProduct"]
	tb := product.Table
	// Nine phases, and the five the design ends on.
	require.Equal(t, []int{9, 5}, []int{len(tb.States), len(tb.Ends)})
	state := func(key string) string {
		results := row(t, product, key).Results
		require.Len(t, results, 1, key)
		return results[0].State
	}
	// A canceled answer settles only an activity whose cancellation was requested.
	require.Equal(t, []interp.Result{{Outcome: "rejected-invalidArgument", State: "started", Facts: []string{},
		Because: "cancellation was not requested (chasm/lib/activity/model/model.go:171)"}}, row(t, product, "started-respondCanceled").Results)
	require.Equal(t, "canceled", state("cancelRequested-respondCanceled"))
	// Unlike the Nexus product, a retry is visible: the client reads scheduled again.
	require.Equal(t, []interp.Result{
		{Outcome: "accepted", State: "scheduled", Facts: []string{"statusScheduled"}, Choice: "retryScheduled"},
		{Outcome: "accepted", State: "paused", Facts: []string{"statusPaused"}, Choice: "retryPaused"},
		{Outcome: "accepted", State: "failed", Facts: []string{"statusFailed"}, Choice: "retryExhausted"},
	}, row(t, product, "started-respondFailed-retryable").Results)
	require.Equal(t, "canceled", state("cancelRequested-respondFailed-retryable"))
	// A paused activity is dispatched to no worker: no product row leaves paused for started.
	for _, r := range tb.RowsFrom("paused") {
		for _, res := range r.Results {
			require.NotEqual(t, "started", res.State, r.Key)
		}
	}
	require.Empty(t, stuck(tb))
}

func TestActivityProtocolTable(t *testing.T) {
	protocol := built(t, activityModel(t))["activitySystem"]
	tb := protocol.Table
	const n = 3 // attempts 0..2
	// Thirteen phases, three dispatch values, three attempt counts, four deadlines and three retry policies.
	require.Equal(t, []int{13 * 3 * n * 16 * 3, 5 * 3 * n * 16 * 3}, []int{len(tb.States), len(tb.Ends)})
	// Start requests, attempt start, worker and service answers, controls, the fault and timers.
	require.Len(t, tb.Actions, 32*3+1+5+4+6+1+6)
	require.Equal(t, []string{"unstarted-now-0-unset-unset-unset-unset-unlimited"}, tb.Starts)

	results := func(key string) []interp.Result { return row(t, protocol, key).Results }
	phase := func(key string) string {
		rs := results(key)
		require.Len(t, rs, 1, key)
		p, _, _ := strings.Cut(rs[0].State, "-")
		return p
	}
	// A retryable failure backs the attempt off and records the attempt count with scheduled again.
	require.Equal(t, []interp.Result{{Outcome: "accepted", State: "scheduled-backoff-1-unset-unset-unset-unset-unlimited",
		Facts: []string{"statusScheduled", "attemptCount"}, Because: "a retryable attempt backs off; the client reads scheduled again"}},
		results("started-now-1-unset-unset-unset-unset-unlimited-respondFailed-retryable"))
	// Under a cancel request the same failure settles the activity as canceled; under a pause request
	// it lands in paused.
	require.Equal(t, "canceled", phase("cancelRequested-now-1-unset-unset-unset-unset-unlimited-respondFailed-retryable"))
	require.Equal(t, "paused", phase("pauseRequested-now-1-unset-unset-unset-unset-unlimited-respondFailed-retryable"))
	// A pause of a held attempt is a request; of a scheduled one it takes effect at once.
	require.Equal(t, "pauseRequested", phase("started-now-1-unset-unset-unset-unset-unlimited-pause"))
	require.Equal(t, "paused", phase("scheduled-now-0-unset-unset-unset-unset-unlimited-pause"))
	// A control on an activity that is over is not found.
	require.Equal(t, []interp.Result{{Outcome: "rejected-notFound", State: "completed-now-1-unset-unset-unset-unset-unlimited", Facts: []string{}}},
		results("completed-now-1-unset-unset-unset-unset-unlimited-terminate"))
	// Each deadline covers its own span.
	require.True(t, disabled(protocol, "scheduled-now-0-unset-unset-expires-unset-one", "startToClose"))
	require.Equal(t, "timedOut", phase("pauseRequested-now-1-unset-unset-expires-unset-one-startToClose"))
	require.True(t, disabled(protocol, "scheduled-backoff-1-unset-expires-unset-unset-unlimited", "scheduleToStart"))
	require.Equal(t, []string{"statusTimedOut-scheduleToStart"}, results("scheduled-now-1-unset-expires-unset-unset-unlimited-scheduleToStart")[0].Facts)
	require.Empty(t, stuck(tb))
}

func TestActivityRefinement(t *testing.T) {
	protocol := built(t, activityModel(t))["activitySystem"]
	refinement, err := refinementOf(t, activityModel(t), "activitySystem")
	require.NoError(t, err)
	results := 0
	for _, r := range protocol.Table.Rows {
		results += len(r.Results)
	}
	require.Len(t, refinement, results)
	lookup := map[string]string{}
	for _, r := range refinement {
		lookup[r.Key] = ""
		if r.Product != nil {
			lookup[r.Key] = *r.Product
		}
	}
	// The visible retry is the product's retryable-failure row; pause and unpause carry their recorded status.
	require.Equal(t, "respondFailed-retryable", lookup["started-now-1-unset-unset-unset-unset-unlimited-respondFailed-retryable"])
	require.Contains(t, lookup, "started-now-1-unset-unset-unset-unset-unlimited-pause")
	require.Equal(t, "pause", lookup["started-now-1-unset-unset-unset-unset-unlimited-pause"])
	require.Contains(t, lookup, "pauseRequested-now-1-unset-unset-unset-unset-unlimited-unpause")
	require.Equal(t, "unpause", lookup["pauseRequested-now-1-unset-unset-unset-unset-unlimited-unpause"])
	// Dispatch delay expiry and worker stop change nothing the Product reads.
	for _, key := range []string{
		"scheduled-backoff-1-unset-unset-unset-unset-unlimited-backoff",
		"started-now-1-unset-unset-unset-unset-unlimited-stop",
	} {
		require.Contains(t, lookup, key)
		require.Empty(t, lookup[key])
	}
	// A retryable failure under a pause request is one of the product's retry results.
	require.Equal(t, "respondFailed-retryable", lookup["pauseRequested-now-1-unset-unset-unset-unset-unlimited-respondFailed-retryable"])
}

func TestActivityQueries(t *testing.T) {
	r := checked(t, activityModel(t))
	for _, name := range []string{"completion", "nonRetryableFailure", "retry", "cancel", "terminate", "pauseResume",
		"scheduleToStartTimeout", "startToCloseTimeout",
		// The finds the protocol's capabilities generate, one per same-step Property.
		"activitySystem.terminateSettles"} {
		require.Equal(t, Found, receiptOf(t, r, "query activitySystem "+name).Kind, name)
	}
	require.Equal(t, Found, receiptOf(t, r, "query byIDCancellation activitySystem.cancelIsRequested").Kind)
	// The product's Properties, which its capabilities generate as verifications over a free search of the
	// product (they retire terminalHolds and pauseHolds, which verified them through the protocol's
	// paths), and the uniform rejection of a closed activity, which no hand-written Query asked.
	for _, property := range []string{"terminalStatesAreFinal", "pausedIsNotDispatched", "closedIsRejectedUniformly"} {
		require.Equal(t, Verified, receiptOf(t, r, "query activityProduct activityProduct."+property).Kind, property)
	}
	// Not vacuous: the scenario performs an attempt start while the worker polls, so the claim is
	// exercised, not merely never contradicted.
	stopped := receiptOf(t, r, "query standaloneActivity stoppedWorkerStartsNothing")
	require.Equal(t, []any{Verified, true}, []any{stopped.Kind, stopped.Exercised})
}

// Every declaration the IR carries passes the checks: no admission or declaration error, no rejected
// refinement and no counterexample, and every Query answered.
func TestActivityChecksClean(t *testing.T) {
	r := checked(t, activityModel(t))
	require.NotEmpty(t, r.Receipts)
	for _, x := range r.Receipts {
		require.Contains(t, []ReceiptKind{Verified, Found}, x.Kind, "%s: %v", receiptKey(x), x.Cause)
	}
}
