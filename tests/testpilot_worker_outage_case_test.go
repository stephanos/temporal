//go:build test_dep && integration

package tests

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	testpilotcore "go.temporal.io/server/tests/testcore/testpilot"
)

const workerOutageCleanupTimeout = 10 * time.Second

func workerOutageBinding() CaseBinding {
	return CaseBinding{
		Identity:       "worker-outage-profile",
		Namespace:      "umpire-worker-outage",
		TaskQueue:      "umpire-worker-outage-queue",
		CleanupTimeout: workerOutageCleanupTimeout,
	}
}

// TestTestpilotWorkerOutageCase drives one deliberate outage end to end. The controller stops the
// SDK worker of its own activation queue before starting the workflow, resumes it after, and reads
// the history back; the Contract requires the two recorded outages in order (the outage-order rule
// the Producer derived from the Model's two fault actions) and a workflow that completed anyway
// (the Model's own clause, confirmed by the completed event), so the queued task surviving the
// outage is what the Run proves.
func TestTestpilotWorkerOutageCase(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	run, verdict := runCapturedCaseWithBinding(t, env, testpilotcore.WorkerOutageFixture, workerOutageBinding())

	check := requireRunSatisfied(t, "", run, verdict)
	check.require("outage evidence", func() {
		requireWorkerOutageEvidence(t, run, verdict)
	})
}

// TestTestpilotWorkerOutageCaseLeavesAnotherQueueAlone runs the outage Case beside a plain Nexus
// Case on its own queue. The outage is real for its own queue and invisible to the other: a pooled
// peer worker on the same physical queue would keep polling it, which is why the queues differ.
func TestTestpilotWorkerOutageCaseLeavesAnotherQueueAlone(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	fixtures := map[string]string{"outage": testpilotcore.WorkerOutageFixture, "plain": nexusCallerQueries[1].fixture()}
	outage := bindCase(t, env, loadTestpilotCase(t, fixtures["outage"]), workerOutageBinding())
	plain := bindCase(t, env, loadTestpilotCase(t, fixtures["plain"]), CaseBinding{
		Identity: "nexus-workflow-profile", Namespace: "umpire-worker-outage-peer",
		TaskQueue: "umpire-worker-outage-peer-queue", NexusEndpoint: "umpire-worker-outage-peer-endpoint",
		CreateEndpoint: true,
	})

	type liveResult struct {
		name    string
		run     *testpilotpb.Run
		verdict *testpilotpb.Verdict
		err     error
	}
	results := make(chan liveResult, 2)
	var runs sync.WaitGroup
	lives := map[string]testpilotLiveCase{"outage": outage, "plain": plain}
	for name, live := range lives {
		runs.Go(func() {
			run, verdict, err := live.prepared.Run(env.Context(), live.driver)
			results <- liveResult{name: name, run: run, verdict: verdict, err: err}
		})
	}
	runs.Wait()
	close(results)

	// Both closed Runs are captured before either is asserted on.
	collected := make([]liveResult, 0, 2)
	for result := range results {
		captureRun(t, fixtures[result.name], lives[result.name], result.run)
		collected = append(collected, result)
	}
	seen := map[string]struct{}{}
	for _, result := range collected {
		require.NoError(t, result.err)
		check := requireRunSatisfied(t, result.name, result.run, result.verdict)
		if result.name == "outage" {
			check.require("outage evidence", func() {
				requireWorkerOutageEvidence(t, result.run, result.verdict)
			})
		} else {
			check.require("fault events", func() {
				require.Empty(t, faultEvents(result.run))
			})
		}
		seen[result.name] = struct{}{}
	}
	require.Len(t, seen, 2)
}

// requireWorkerOutageEvidence reads both rules' supporting Observations back out of the Run: the
// ordered stop and resume on this Case's own task-queue role, and the completed workflow the
// history recorded, lifted as the evidence that confirms the Model's four steps at once.
func requireWorkerOutageEvidence(t testing.TB, run *testpilotpb.Run, verdict *testpilotpb.Verdict) {
	t.Helper()
	// The derived outage-order rule, and one rule per scoped clause the Model's completion Property
	// lowered into (the completed phase, the completed fact), each satisfied.
	require.GreaterOrEqual(t, len(verdict.GetRules()), 2)
	var order *testpilotpb.RuleVerdict
	var clauses []*testpilotpb.RuleVerdict
	for _, rule := range verdict.GetRules() {
		if rule.GetRuleId() == testpilotcore.WorkerOutageRuleID {
			order = rule
		} else {
			clauses = append(clauses, rule)
		}
	}
	require.NotNil(t, order, "the derived outage-order rule")
	require.NotEmpty(t, clauses, "the Model's completion clauses")
	require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, order.GetStatus())
	require.Equal(t, "resumed", order.GetTerminalStateId())
	for _, clause := range clauses {
		require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, clause.GetStatus(), clause.GetRuleId())
		require.Equal(t, "correlated.satisfied", clause.GetTerminalStateId(), clause.GetRuleId())
	}
	completed := clauses[0]

	require.Len(t, order.GetSupportingEventSequences(), 2)
	kinds := make([]testpilotpb.FaultKind, 0, 2)
	for _, sequence := range order.GetSupportingEventSequences() {
		event := runEventAt(t, run, sequence)
		require.Equal(t, testpilotpb.RUN_EVENT_KIND_FAULT_INJECTED, event.GetKind())
		require.Equal(t, "temporal.task-queue", event.GetFaultInjected().GetRoleId())
		kinds = append(kinds, event.GetFaultInjected().GetKind())
	}
	require.Equal(t, []testpilotpb.FaultKind{
		testpilotpb.FAULT_KIND_WORKER_STOP, testpilotpb.FAULT_KIND_WORKER_RESUME,
	}, kinds)
	// Both recorded faults are the ones the Program requested, and nothing else was realized.
	require.Equal(t, order.GetSupportingEventSequences(), faultEvents(run))

	require.Len(t, completed.GetSupportingEventSequences(), 1)
	event := runEventAt(t, run, completed.GetSupportingEventSequences()[0])
	require.Equal(t, "history", event.GetCoordinates().GetInstructionId())
	var evidence testpilotpb.CorrelatedEvidence
	require.NoError(t, observationValue(t, event, "correlated-evidence").GetMessageValue().UnmarshalTo(&evidence))
	require.Equal(t, "evidence.workflowExecutionCompleted", evidence.GetKind())
	var historyEvent historypb.HistoryEvent
	require.NoError(t, observationValue(t, event, "history-event").GetMessageValue().UnmarshalTo(&historyEvent))
	require.Equal(t, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED, historyEvent.GetEventType())
	require.Positive(t, historyEvent.GetWorkflowExecutionCompletedEventAttributes().GetWorkflowTaskCompletedEventId())
	// The workflow is named by the task that completed it, which is the key the evidence carries.
	require.Equal(t, strconv.FormatInt(historyEvent.GetWorkflowExecutionCompletedEventAttributes().GetWorkflowTaskCompletedEventId(), 10), evidence.GetOperation())
}

func faultEvents(run *testpilotpb.Run) []int64 {
	sequences := []int64{}
	for _, event := range run.GetEvents() {
		if event.GetKind() == testpilotpb.RUN_EVENT_KIND_FAULT_INJECTED {
			sequences = append(sequences, event.GetSequence())
		}
	}
	return sequences
}
