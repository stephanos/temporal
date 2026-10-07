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
	state := func(key string) string { return row(t, product, key).Results[0].State }
	// A canceled answer settles only an activity whose cancellation was requested.
	require.True(t, disabled(product, "started", "respondCanceled"))
	require.Equal(t, "canceled", state("cancelRequested-respondCanceled"))
	// Unlike the Nexus product, a retry is visible: the client reads scheduled again.
	require.Equal(t, "scheduled", state("started-respondFailed-retryable"))
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
	// Twelve phases, three attempt counts and three deadlines: 288 states.
	require.Equal(t, []int{12 * n * 8, 5 * n * 8}, []int{len(tb.States), len(tb.Ends)})
	// Eight start requests, one attempt start, four answers, four controls, the fault and four timers.
	require.Len(t, tb.Actions, 8+1+4+4+1+4)
	require.Equal(t, []string{"unstarted-0-unset-unset-unset"}, tb.Starts)

	results := func(key string) []interp.Result { return row(t, protocol, key).Results }
	phase := func(key string) string {
		rs := results(key)
		require.Len(t, rs, 1, key)
		p, _, _ := strings.Cut(rs[0].State, "-")
		return p
	}
	// A retryable failure backs the attempt off and is read as scheduled again with the count raised.
	require.Equal(t, []interp.Result{{Outcome: "accepted", State: "backingOff-1-unset-unset-unset",
		Facts: []string{"statusScheduled", "attemptCount"}, Because: "a retryable failure backs off; the client reads scheduled again"}},
		results("started-1-unset-unset-unset-respondFailed-retryable"))
	// Under a cancel request the same failure settles the activity as canceled; under a pause request
	// it lands in paused.
	require.Equal(t, "canceled", phase("cancelRequested-1-unset-unset-unset-respondFailed-retryable"))
	require.Equal(t, "paused", phase("pauseRequested-1-unset-unset-unset-respondFailed-retryable"))
	// A pause of a held attempt is a request; of a scheduled one it takes effect at once.
	require.Equal(t, "pauseRequested", phase("started-1-unset-unset-unset-pause"))
	require.Equal(t, "paused", phase("scheduled-0-unset-unset-unset-pause"))
	// A control on an activity that is over is not found.
	require.Equal(t, []interp.Result{{Outcome: "rejected-notFound", State: "completed-1-unset-unset-unset", Facts: []string{}}},
		results("completed-1-unset-unset-unset-terminate"))
	// Each deadline covers its own span.
	require.True(t, disabled(protocol, "scheduled-0-unset-unset-expires", "startToClose"))
	require.Equal(t, "timedOut", phase("pauseRequested-1-unset-unset-expires-startToClose"))
	require.Equal(t, []string{"statusTimedOut-scheduleToStart"}, results("backingOff-1-unset-expires-unset-scheduleToStart")[0].Facts)
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
	// The visible retry is the product's retryable-failure row; the pause request is a stutter; the
	// unpause of a requested pause is a stutter too.
	require.Equal(t, "respondFailed-retryable", lookup["started-1-unset-unset-unset-respondFailed-retryable"])
	require.Contains(t, lookup, "started-1-unset-unset-unset-pause")
	require.Empty(t, lookup["started-1-unset-unset-unset-pause"])
	require.Contains(t, lookup, "pauseRequested-1-unset-unset-unset-unpause")
	require.Empty(t, lookup["pauseRequested-1-unset-unset-unset-unpause"])
	// A retryable failure under a pause request is the product's pause.
	require.Equal(t, "pause", lookup["pauseRequested-1-unset-unset-unset-respondFailed-retryable"])
}

func TestActivityQueries(t *testing.T) {
	r := checked(t, activityModel(t))
	for _, name := range []string{"completion", "nonRetryableFailure", "retry", "cancel", "terminate", "pauseResume",
		"scheduleToStartTimeout", "startToCloseTimeout",
		// The finds the protocol's capabilities generate, one per functional law.
		"activitySystem.terminateSettles", "activitySystem.cancelIsRequested"} {
		require.Equal(t, Found, receiptOf(t, r, "query activitySystem "+name).Kind, name)
	}
	// The product's laws, which its capabilities generate as verifications over a free search of the
	// product (they retire terminalHolds and pauseHolds, which verified them through the protocol's
	// paths), and the uniform rejection of a closed activity, which no hand-written Query asked.
	for _, law := range []string{"terminalStatesAreFinal", "pausedIsNotDispatched", "closedIsRejectedUniformly"} {
		require.Equal(t, Verified, receiptOf(t, r, "query activityProduct activityProduct."+law).Kind, law)
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
