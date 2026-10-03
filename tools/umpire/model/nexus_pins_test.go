package model

// The Nexus caller Model, as model/ir/nexus-caller.json carries it: the sizes, rows and answers its
// Scala pins asserted, read off the tables and receipts Go derives from the IR. The goldens under
// testdata/migration freeze every one of these values; these tests state the ones the Model's
// reader names.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// productClaimProbes are the two Queries the Scala pins asked of the product claim, which no Model
// declares: its verification over every protocol trace within four steps, and a product Property
// about an action the protocol machine lacks. They are written in the lifter's shape, with
// declaration positions only, and read the lifted product claim terminalIsFinal.
const productClaimProbes = `{
  "functions": [
    {"name": "nexusProduct.property.timesOut",
      "position": {"file": "tools/umpire/model/nexus_pins_test.go"},
      "params": [{"name": "after", "type": {"named": "umpire.Step"}}],
      "body": {"binary": {"op": "OP_EQ",
        "left": {"field": {"base": {"field": {"base": {"var": "after"}, "field": "state"}}, "field": "phase"}},
        "right": {"literal": {"enum": {"type": "temporal.nexuscaller.kernel.ProductPhase", "case": "timedOut"}}}}}}],
  "properties": [
    {"machine": "nexusProduct", "name": "timesOut", "holds": "nexusProduct.property.timesOut",
      "whenClass": {"action": "temporal.nexuscaller.Model$package$.timeout"},
      "position": {"file": "tools/umpire/model/nexus_pins_test.go"}}],
  "scenarios": [
    {"machine": "nexusProtocol", "name": "everywhere", "free": true,
      "position": {"file": "tools/umpire/model/nexus_pins_test.go"},
      "start": {"construct": {"type": "temporal.nexuscaller.kernel.ProtocolState", "args": [
        {"literal": {"enum": {"type": "temporal.nexuscaller.kernel.Phase", "case": "unscheduled"}}},
        {"literal": {"int": "0"}},
        {"literal": {"enum": {"type": "temporal.nexuscaller.kernel.Timeout", "case": "unset"}}},
        {"literal": {"enum": {"type": "temporal.nexuscaller.kernel.Timeout", "case": "unset"}}},
        {"literal": {"enum": {"type": "temporal.nexuscaller.kernel.Timeout", "case": "unset"}}}]}}}],
  "queries": [
    {"name": "terminalHoldsEverywhere", "form": "FORM_VERIFY", "through": true,
      "position": {"file": "tools/umpire/model/nexus_pins_test.go"},
      "property": {"machine": "nexusProduct", "name": "terminalIsFinal"},
      "scenario": {"machine": "nexusProtocol", "name": "everywhere"},
      "limits": {"name": "four", "steps": 4, "actions": 4, "search": 32768}}]
}`

const timesOutOnProtocol = `{"queries": [{"name": "timesOutOnProtocol", "form": "FORM_VERIFY", "through": true,
  "position": {"file": "tools/umpire/model/nexus_pins_test.go"},
  "property": {"machine": "nexusProduct", "name": "timesOut"},
  "scenario": {"machine": "nexusProtocol", "name": "asyncThenSucceeded"},
  "limits": {"name": "three", "steps": 3, "actions": 3, "search": 4096}}]}`

// withDeclarations is the Model with these ProtoJSON declarations appended.
func withDeclarations(t *testing.T, m *umpirespb.Model, fragments ...string) *umpirespb.Model {
	t.Helper()
	out := proto.Clone(m).(*umpirespb.Model)
	for _, fragment := range fragments {
		more := &umpirespb.Model{}
		require.NoError(t, protojson.Unmarshal([]byte(fragment), more))
		proto.Merge(out, more)
	}
	return out
}

func TestNexusProductTable(t *testing.T) {
	product := machines(t)["nexusProduct"]
	tb := product.Table
	// Six phases, and the four the design ends on.
	require.Equal(t, []int{6, 4}, []int{len(tb.States), len(tb.Ends)})
	// Six replies, three resolutions, the two faults it cannot see, and the one timer.
	require.Len(t, tb.Actions, 12)
	// A retryable handler error is invisible here: it is the protocol machine that backs off.
	require.True(t, product.Disabled("scheduled", "handlerReply-handlerError-true"))
	require.Equal(t, []string{"scheduled", "canceled", "failed", "succeeded", "started", "timedOut"}, tb.Reachable)
	require.Empty(t, tb.Stuck)
}

func TestNexusProtocolTable(t *testing.T) {
	nexus := machines(t)
	protocol := nexus["nexusProtocol"]
	tb := protocol.Table
	const n = 3 // attempts 0..2
	// Eight phases, three attempt counts and three deadlines, and the four ending phases.
	require.Equal(t, []int{8 * n * 8, 4 * n * 8}, []int{len(tb.States), len(tb.Ends)})
	// Eight schedule commands, six replies, three resolutions, the two faults and the four timers. The
	// catalog is in canonical order, so it opens on the backoff timer.
	require.Len(t, tb.Actions, 8+6+3+1+1+4)
	require.Equal(t, []string{"backoff", "complete-canceled"}, tb.Actions[:2])
	// The machine begins before the operation exists, with every deadline at its first value.
	require.Equal(t, []string{"unscheduled-0-unset-unset-unset"}, tb.Starts)

	results := func(key string) []Result { return row(t, protocol, key).Results }
	// A retryable handler error backs the operation off and raises the attempt count; the count
	// saturates rather than wrapping.
	require.Equal(t, []Result{{Outcome: "accepted", State: "backingOff-1-unset-unset-unset", Facts: []string{"pendingAttempts"}}},
		results("scheduled-0-unset-unset-unset-handlerReply-handlerError-true"))
	require.Equal(t, "backingOff-2-unset-unset-unset", results("scheduled-2-unset-unset-unset-handlerReply-handlerError-true")[0].State)
	// A completion before the start records the Started event first, one after it does not.
	require.Equal(t, []string{"nexusOperationStarted", "nexusOperationCompleted"},
		results("backingOff-1-unset-unset-unset-complete-succeeded")[0].Facts)
	require.Equal(t, []string{"nexusOperationCompleted"}, results("started-0-unset-unset-unset-complete-succeeded")[0].Facts)
	// A completion after the operation is over is not found and changes nothing.
	require.Equal(t, []Result{{Outcome: "notFound", State: "timedOut-0-unset-unset-unset", Facts: []string{}}},
		results("timedOut-0-unset-unset-unset-complete-succeeded"))
	// A timer fires only when the schedule command set it, and each covers its own span.
	require.True(t, protocol.Disabled("scheduled-0-unset-unset-expires", "startToClose"))
	require.Equal(t, "timedOut-0-unset-unset-expires", results("started-0-unset-unset-expires-startToClose")[0].State)
	require.True(t, protocol.Disabled("started-0-unset-unset-unset", "scheduleToClose"))
	// Which timer fired is recorded.
	require.Equal(t, []string{"nexusOperationTimedOut-scheduleToStart"}, results("scheduled-0-unset-expires-unset-scheduleToStart")[0].Facts)
	// The worker stopping keeps the state and records nothing; the product machine does not see it.
	require.Equal(t, []Result{{Outcome: "accepted", State: "scheduled-0-unset-expires-unset", Facts: []string{}}},
		results("scheduled-0-unset-expires-unset-workerStop"))
	require.True(t, nexus["nexusProduct"].Disabled("scheduled", "workerStop"))

	require.Empty(t, tb.Stuck)
	// Not every state is reachable. The Behavior Fingerprint reads the table, so these numbers are part
	// of the Model's identity.
	require.Len(t, tb.Reachable, 158)
	require.Len(t, tb.Rows, 1152)
}

func TestNexusRefinement(t *testing.T) {
	protocol := machines(t)["nexusProtocol"]
	require.NoError(t, protocol.Rejected)
	require.Len(t, protocol.Refinement, len(protocol.Table.Rows))
	lookup := map[string]*string{}
	for _, r := range protocol.Refinement {
		lookup[r.Key] = r.Product
	}
	product := func(key string) string {
		p, ok := lookup[key]
		require.True(t, ok, key)
		if p == nil {
			return ""
		}
		return *p
	}
	// A reply the product machine sees is that reply's step. A retry it cannot see is a stutter, and so
	// are the schedule command and the backoff timer.
	require.Equal(t, "handlerReply-async", product("scheduled-0-unset-unset-unset-handlerReply-async"))
	require.Empty(t, product("scheduled-0-unset-unset-unset-handlerReply-handlerError-true"))
	require.Empty(t, product("unscheduled-0-unset-unset-unset-schedule-unset-unset-expires"))
	require.Empty(t, product("backingOff-1-unset-unset-unset-backoff"))
	// A deadline firing is the product's one timer, whichever deadline it was.
	require.Equal(t, "timeout", product("started-0-unset-unset-expires-startToClose"))
	// A completion before the start records the Started event first and still carries the step.
	require.Equal(t, "complete-succeeded", product("backingOff-1-unset-unset-unset-complete-succeeded"))

	// Every schedule command, every retry, every backoff and every worker stop.
	var stutters []string
	for _, r := range protocol.Refinement {
		if r.Product == nil {
			stutters = append(stutters, r.Key)
		}
	}
	require.Len(t, stutters, 24*8+24+24+24+192)
	// Stutter invariance: every stutter leaves a phase that reads as scheduled, or is a worker stop.
	for _, key := range stutters {
		phase, _, _ := strings.Cut(key, "-")
		require.True(t, phase == "unscheduled" || phase == "scheduled" || phase == "backingOff" || strings.HasSuffix(key, "-workerStop"), key)
	}
	// The product state a protocol state reads as is a field named after the product machine.
	require.Equal(t, []string{"phase", "attempts", "scheduleToClose", "scheduleToStart", "startToClose", "nexusProduct"},
		protocol.Table.StateFields)
}

func TestNexusQueries(t *testing.T) {
	r := checked(t, load(t))
	// Each functional Query finds its claim on its path, and a Scenario names its classed actions with
	// their inputs, a timer like any action, where it fires.
	paths := map[string][]string{
		"syncCompletion":         {"schedule-unset-unset-unset", "handlerReply-syncSuccess"},
		"asyncCompletion":        {"schedule-unset-unset-unset", "handlerReply-async", "complete-succeeded"},
		"asyncFailure":           {"schedule-unset-unset-unset", "handlerReply-async", "complete-failed"},
		"handlerError":           {"schedule-unset-unset-unset", "handlerReply-handlerError-false"},
		"retry":                  {"schedule-unset-unset-unset", "handlerReply-handlerError-true", "backoff", "handlerReply-syncSuccess"},
		"scheduleToStartTimeout": {"schedule-unset-expires-unset", "workerStop", "scheduleToStart"},
		"startToCloseTimeout":    {"schedule-unset-unset-expires", "handlerReply-async", "startToClose"},
	}
	for name, path := range paths {
		found := receiptOf(t, r, "query nexusProtocol "+name)
		require.Equal(t, Found, found.Kind, name)
		require.Equal(t, path, taken(found.Witness), name)
		require.Equal(t, "unscheduled-0-unset-unset-unset", found.Witness.Initial.Value, name)
	}
	// The product claim is verified over every trace of the asynchronous path, and keeps its own
	// identity: it is the product Property and no other.
	terminal := receiptOf(t, r, "query nexusProtocol terminalHolds")
	require.Equal(t, Verified, terminal.Kind)
	require.True(t, terminal.Exercised)
	require.Equal(t, ClaimKey{Family: "temporal.nexus.caller", Owner: "nexusProduct", Name: "terminalIsFinal"}, terminal.Property)
}

// The search keeps one fired bit per Property and visits 111 product states. The count is a search
// statistic.
func TestNexusProductPropertyOverEveryTraceWithinFour(t *testing.T) {
	r := checked(t, withDeclarations(t, load(t), productClaimProbes))
	everywhere := receiptOf(t, r, "query nexusProtocol terminalHoldsEverywhere")
	require.Equal(t, []any{Verified, 111}, []any{everywhere.Kind, everywhere.Explored})
}

// A product Property about an action the protocol machine does not have cannot be read there.
func TestNexusProductPropertyOnAMissingAction(t *testing.T) {
	r := checked(t, withDeclarations(t, load(t), productClaimProbes, timesOutOnProtocol))
	refused := receiptOf(t, r, "query nexusProtocol timesOutOnProtocol")
	require.Equal(t, DeclarationError, refused.Kind)
	require.EqualError(t, refused.Cause, "query timesOutOnProtocol: the Property names the action 'timeout' of "+
		"'nexusProduct', and 'nexusProtocol' has no action of that name; a Property on the refined machine is read on "+
		"the refining one through the values of the same name, and a state through its map")
}

func TestNexusCallerComposition(t *testing.T) {
	m := load(t)
	require.Equal(t, []string{"serve", "workerStop"}, built(t, m)["handlerWorker"].Table.Actions)
	realizer, err := NewRealizer(m, DefaultScope)
	require.NoError(t, err)
	composed, err := realizer.Composition("nexusCaller")
	require.NoError(t, err)
	tb := composed.Table
	// Every reachable protocol state under both worker phases.
	require.Len(t, tb.States, 316)
	stopped := 0
	for _, s := range tb.States {
		if strings.HasSuffix(s, "_stopped") {
			stopped++
		}
	}
	require.Equal(t, 158, stopped)
	require.Len(t, tb.Rows, 1468)
	require.Equal(t, []string{
		"handlerReply-async",
		"handlerReply-handlerError-false",
		"handlerReply-handlerError-true",
		"handlerReply-operationCanceled",
		"handlerReply-operationFailed",
		"handlerReply-syncSuccess",
		"operation_backoff",
		"operation_complete-canceled",
		"operation_complete-failed",
		"operation_complete-succeeded",
		"operation_schedule-expires-expires-expires",
		"operation_schedule-expires-expires-unset",
		"operation_schedule-expires-unset-expires",
		"operation_schedule-expires-unset-unset",
		"operation_schedule-unset-expires-expires",
		"operation_schedule-unset-expires-unset",
		"operation_schedule-unset-unset-expires",
		"operation_schedule-unset-unset-unset",
		"operation_scheduleToClose",
		"operation_scheduleToStart",
		"operation_startToClose",
		"operation_transportFault",
		"workerStop",
	}, tb.Actions)
	// A reply has a row only where the worker polls.
	replies := 0
	for _, row := range tb.Rows {
		if strings.Contains(row.Key, "-handlerReply") {
			replies++
			require.NotContains(t, row.Key, "_stopped-")
		}
	}
	require.Equal(t, 144, replies)
	// Verified over the path: the one reply comes before the stop.
	stoppedWorker := receiptOf(t, checked(t, m), "query nexusCaller stoppedWorkerRepliesNothing")
	require.Equal(t, []any{Verified, true}, []any{stoppedWorker.Kind, stoppedWorker.Exercised})
}

// Every declaration the IR carries passes the checks: no admission or declaration error, no rejected
// refinement and no counterexample, and every Query answered.
func TestNexusCallerChecksClean(t *testing.T) {
	r := checked(t, load(t))
	require.NotEmpty(t, r.Receipts)
	for _, x := range r.Receipts {
		require.Contains(t, []ReceiptKind{Verified, Found}, x.Kind, "%s: %v", receiptKey(x), x.Cause)
	}
}
