//go:build test_dep && integration

package tests

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tests/testcore"
	testpilotcore "go.temporal.io/server/tests/testcore/testpilot"
	"google.golang.org/protobuf/proto"
)

const testpilotCleanupTimeout = 5 * time.Second

type testpilotLiveRunResult struct {
	environment int
	run         *testpilotpb.Run
	verdict     *testpilotpb.Verdict
	err         error
}

// switchRun is one Case bound and run under one value of a switch: the Case as bound to one
// of the value's environments, and what one Run of it produced.
type switchRun struct {
	value   testpilotcore.SwitchValue
	binding CaseBinding
	live    testpilotLiveCase
	run     *testpilotpb.Run
	verdict *testpilotpb.Verdict
}

// nexusCallerQuery is one Query of the caller Model's functional set as the live suite runs it: the
// fixture the set produced for it, and the history events its Contract's supporting evidence must
// project, in the order the operation records them.
type nexusCallerQuery struct {
	name       string
	supporting []enumspb.EventType
	// terminal is the history event the operation settles on, read from the Run whether or not
	// the Contract's clause supports it.
	terminal enumspb.EventType
	// timeoutType is the deadline the terminal timed-out event records, for a Query that times out.
	timeoutType enumspb.TimeoutType
	// readsAttempts is whether the Contract's supporting evidence includes the pending operation's
	// attempt count, read after a retryable failure and confirming the backoff.
	readsAttempts bool
	// stopsWorker is whether the Run injects the fault that stops the handler's worker.
	stopsWorker bool
}

func (q nexusCallerQuery) fixture() string { return "nexusCallerTests-" + q.name }

// Every admitted semantic step of the operation supports the clause: the scheduled event, read as
// soon as it exists, confirms the schedule command; then each event the path's side effects record,
// read out of history once the workflow closed; a retryable failure is confirmed by the
// attempt-count read, and a step that records nothing -- the backoff, the worker stop -- by the
// event of the step after it. `supporting` lists the history read's events; the scheduled read
// supports every Query.
var nexusCallerQueries = []nexusCallerQuery{
	{name: "syncCompletion", supporting: []enumspb.EventType{enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED}, terminal: enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED},
	{name: "asyncCompletion", supporting: []enumspb.EventType{enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED, enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED}, terminal: enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED},
	{name: "asyncFailure", supporting: []enumspb.EventType{enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED, enumspb.EVENT_TYPE_NEXUS_OPERATION_FAILED}, terminal: enumspb.EVENT_TYPE_NEXUS_OPERATION_FAILED},
	{name: "handlerError", supporting: []enumspb.EventType{enumspb.EVENT_TYPE_NEXUS_OPERATION_FAILED}, terminal: enumspb.EVENT_TYPE_NEXUS_OPERATION_FAILED},
	{name: "retry", supporting: []enumspb.EventType{enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED}, terminal: enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED, readsAttempts: true},
	{name: "scheduleToStartTimeout", supporting: []enumspb.EventType{enumspb.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT}, terminal: enumspb.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT, timeoutType: enumspb.TIMEOUT_TYPE_SCHEDULE_TO_START, stopsWorker: true},
	{name: "startToCloseTimeout", supporting: []enumspb.EventType{enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED, enumspb.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT}, terminal: enumspb.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT, timeoutType: enumspb.TIMEOUT_TYPE_START_TO_CLOSE},
}

// The seven Queries of the caller Model's functional set, one live test each, run the way the
// upstream Nexus suites run under HSM and CHASM: once per value of the implementation switch, each
// under its own cluster constructed with that value's settings.
func TestTestpilotNexusCallerSyncCompletion(t *testing.T) {
	runNexusCallerQueryUnderEachSwitchValue(t, nexusCallerQueries[0])
}

func TestTestpilotNexusCallerAsyncCompletion(t *testing.T) {
	runNexusCallerQueryUnderEachSwitchValue(t, nexusCallerQueries[1])
}

func TestTestpilotNexusCallerAsyncFailure(t *testing.T) {
	runNexusCallerQueryUnderEachSwitchValue(t, nexusCallerQueries[2])
}

func TestTestpilotNexusCallerHandlerError(t *testing.T) {
	runNexusCallerQueryUnderEachSwitchValue(t, nexusCallerQueries[3])
}

// The retried start is answered by the same handler activation: a retryable error, the server's
// backoff, then a synchronous success, with the attempt count read in between.
func TestTestpilotNexusCallerRetry(t *testing.T) {
	runNexusCallerQueryUnderEachSwitchValue(t, nexusCallerQueries[4])
}

// The handler's worker is stopped before the caller workflow starts, so the schedule-to-start
// deadline the schedule command sets is what settles the operation.
func TestTestpilotNexusCallerScheduleToStartTimeout(t *testing.T) {
	runNexusCallerQueryUnderEachSwitchValue(t, nexusCallerQueries[5])
}

// The handler accepts asynchronously and never completes, so the start-to-close deadline settles
// the operation.
func TestTestpilotNexusCallerStartToCloseTimeout(t *testing.T) {
	runNexusCallerQueryUnderEachSwitchValue(t, nexusCallerQueries[6])
}

// runNexusCallerQueryUnderEachSwitchValue runs one Query's Case once per value of the Nexus
// implementation switch. Under each value the Case is bound to two isolated namespaces, queues and
// endpoints and run twice concurrently against each, so a Run's effects are shown to stay in its
// own namespace on a live cluster. The Case bytes are the same under every value; the Profile is
// not, because it records the configuration it ran under; and a Verdict that differs between the
// values fails naming both, because that is a finding about the implementations, not a flake.
func runNexusCallerQueryUnderEachSwitchValue(t *testing.T, query nexusCallerQuery) {
	t.Helper()
	caseSource := loadTestpilotCase(t, query.fixture())
	caseSnapshot := proto.CloneOf(caseSource)

	runs := make([]switchRun, 0, 8)
	for _, value := range testpilotcore.NexusImplementationSwitch() {
		// A subtest per value: each has its own dedicated cluster, constructed with the value's
		// settings, and its own resources named after the value.
		t.Run(value.Name, func(t *testing.T) {
			options := make([]testcore.TestOption, 0, len(value.Settings))
			for _, setting := range value.Settings {
				options = append(options, testcore.WithDynamicConfig(setting.Setting, setting.Value))
			}
			env := newTestpilotTestEnvironment(t, options...)
			bindings := make([]CaseBinding, 0, 2)
			lives := make([]testpilotLiveCase, 0, 2)
			for _, suffix := range []string{"a", "b"} {
				resource := "umpire-" + query.name + "-" + value.Name + "-" + suffix
				binding := CaseBinding{
					Identity:  query.fixture() + "-profile-" + value.Name,
					Namespace: resource, TaskQueue: resource + "-queue",
					NexusEndpoint: resource + "-endpoint", CreateEndpoint: true,
					DynamicConfig: value.Configuration(),
				}
				live := bindCase(t, env, caseSource, binding)
				require.True(t, proto.Equal(caseSnapshot, caseSource))
				require.Len(t, live.profile.Configuration, len(value.Settings))
				bindings = append(bindings, binding)
				lives = append(lives, live)
			}
			// The two bindings prepare the same bytes under two binding identities.
			require.True(t, proto.Equal(lives[0].prepared.Snapshot(), lives[1].prepared.Snapshot()))
			require.NotEqual(t, lives[0].prepared.Identity().Bindings, lives[1].prepared.Identity().Bindings)

			results := make(chan testpilotLiveRunResult, len(lives)*2)
			var pending sync.WaitGroup
			for index, live := range lives {
				for range 2 {
					pending.Go(func() {
						run, verdict, err := live.prepared.Run(env.Context(), live.driver)
						results <- testpilotLiveRunResult{environment: index, run: run, verdict: verdict, err: err}
					})
				}
			}
			pending.Wait()
			close(results)

			runIDs := make(map[string]struct{}, len(lives)*2)
			for result := range results {
				require.NoError(t, result.err)
				requireNexusCallerVerdict(t, query, result.run, result.verdict, bindings[result.environment].NexusEndpoint)
				require.NotContains(t, runIDs, result.run.GetRunId())
				runIDs[result.run.GetRunId()] = struct{}{}

				// A Run's workflow exists in its own namespace and in no other.
				_, err := lives[result.environment].client.DescribeWorkflowExecution(env.Context(), result.run.GetRunId(), "")
				require.NoError(t, err)
				_, err = lives[1-result.environment].client.DescribeWorkflowExecution(env.Context(), result.run.GetRunId(), "")
				var notFound *serviceerror.NotFound
				require.ErrorAs(t, err, &notFound)
				runs = append(runs, switchRun{value: value, binding: bindings[result.environment], live: lives[result.environment], run: result.run, verdict: result.verdict})
			}
			for _, live := range lives {
				require.Equal(t, live.profile.EnvironmentBindings, live.driver.Snapshot().EnvironmentBindings)
				require.True(t, proto.Equal(caseSnapshot, live.prepared.Snapshot()))
			}
		})
	}
	require.Len(t, runs, 8)

	// Identical Case bytes under every value: the same Case, Program, Contract and provenance; and
	// two Profiles per environment pair, because each records the configuration it ran under.
	first, second := runs[0].live.prepared, runs[len(runs)-1].live.prepared
	require.NotEqual(t, runs[0].value.Name, runs[len(runs)-1].value.Name)
	require.True(t, proto.Equal(first.Snapshot(), second.Snapshot()))
	require.Equal(t, first.Snapshot().GetCaseId(), second.Snapshot().GetCaseId())
	require.NotEqual(t, runs[0].live.profile.Configuration, runs[len(runs)-1].live.profile.Configuration)

	// Every Verdict, reported together: a divergence names the switch, both values and both.
	results := make([]testpilotcore.SwitchVerdict, 0, len(runs))
	for _, run := range runs {
		results = append(results, testpilotcore.SwitchVerdict{Value: run.value.Name, Verdict: run.verdict})
	}
	require.NoError(t, testpilotcore.CheckSwitchAgreement(testpilotcore.NexusImplementationSwitchName, results))
	require.True(t, proto.Equal(caseSnapshot, caseSource))
}

// requireNexusCallerVerdict is what every Run of a caller Query must show: the Run completed and
// cleaned up, every scoped clause of the Property is satisfied, and the supporting evidence is the
// history event the Query's claim names, lifted under the operation's scheduled event.
func requireNexusCallerVerdict(t testing.TB, query nexusCallerQuery, run *testpilotpb.Run, verdict *testpilotpb.Verdict, endpoint string) {
	t.Helper()
	require.Equal(t, testpilotpb.RUN_DISPOSITION_COMPLETED, run.GetDisposition(), "diagnostics: %v", run.GetDiagnostics())
	require.Equal(t, testpilotpb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
	require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, verdict.GetStatus(), "diagnostics: %v", run.GetDiagnostics())
	require.True(t, proto.Equal(verdict, run.GetVerdict()))
	// One rule verdict per scoped clause the checked Property lowered into, each satisfied and
	// each supported by evidence the Verdict as a whole lists.
	require.NotEmpty(t, verdict.GetRules())
	for _, rule := range verdict.GetRules() {
		require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_SATISFIED, rule.GetStatus())
		require.Subset(t, verdict.GetSupportingEventSequences(), rule.GetSupportingEventSequences())
	}
	// The scheduled read opens the operation's evidence on every path; the attempt-count read,
	// when the path has one, follows it; the history read's events close it.
	scheduled, reads, history := partitionNexusCallerSupport(t, run, verdict.GetSupportingEventSequences())
	scheduledID := requireCorrelatedNexusHistoryEvidence(t, run, history, endpoint, query.supporting)
	requireReadEvidence(t, run, scheduled, "await-scheduled", "evidence.scheduled", scheduledID)
	if query.readsAttempts {
		require.Len(t, reads, 1)
		requireReadEvidence(t, run, reads[0], "pending-attempts", "evidence.pendingAttempts", scheduledID)
	} else {
		require.Empty(t, reads)
	}
	requireNexusHistoryEvent(t, run, query.terminal)
	if query.timeoutType != enumspb.TIMEOUT_TYPE_UNSPECIFIED {
		requireNexusTimedOutType(t, run, query.timeoutType)
	}
	if query.stopsWorker {
		require.NotEmpty(t, faultEvents(run), "the handler's worker was never stopped")
	} else {
		require.Empty(t, faultEvents(run))
	}
}

// partitionNexusCallerSupport splits a Verdict's supporting sequences by the controller instruction
// that recorded each: the scheduled read's one, the attempt-count poll's, and the history read's.
func partitionNexusCallerSupport(t testing.TB, run *testpilotpb.Run, sequences []int64) (scheduled int64, reads, history []int64) {
	t.Helper()
	for _, sequence := range sequences {
		switch instruction := runEventAt(t, run, sequence).GetCoordinates().GetInstructionId(); instruction {
		case "await-scheduled":
			require.Zero(t, scheduled, "the scheduled event is read once")
			scheduled = sequence
		case "pending-attempts":
			reads = append(reads, sequence)
		case "history":
			history = append(history, sequence)
		default:
			require.Fail(t, "supporting evidence from an unexpected instruction", "%s", instruction)
		}
	}
	require.NotZero(t, scheduled, "the scheduled read supports every Query")
	return scheduled, reads, history
}

// requireReadEvidence reads the evidence one poll lifted: the kind the read declares, under the
// operation key every history event of the operation carries, with no field.
func requireReadEvidence(t testing.TB, run *testpilotpb.Run, sequence int64, instruction, kind string, scheduledID int64) {
	t.Helper()
	event := runEventAt(t, run, sequence)
	require.Equal(t, instruction, event.GetCoordinates().GetInstructionId())
	var evidence testpilotpb.CorrelatedEvidence
	require.NoError(t, observationValue(t, event, "correlated-evidence").GetMessageValue().UnmarshalTo(&evidence))
	require.Equal(t, kind, evidence.GetKind())
	require.Equal(t, strconv.FormatInt(scheduledID, 10), evidence.GetOperation())
	require.Empty(t, evidence.GetFields())
}

// requireNexusTimedOutType reads which deadline the timed-out event records.
func requireNexusTimedOutType(t testing.TB, run *testpilotpb.Run, timeoutType enumspb.TimeoutType) {
	t.Helper()
	for _, event := range run.GetEvents() {
		for _, observation := range event.GetObservations() {
			if observation.GetObservationId() != "history-event" {
				continue
			}
			var historyEvent historypb.HistoryEvent
			require.NoError(t, observation.GetValue().GetMessageValue().UnmarshalTo(&historyEvent))
			if attributes := historyEvent.GetNexusOperationTimedOutEventAttributes(); attributes != nil {
				require.Equal(t, timeoutType, attributes.GetFailure().GetCause().GetTimeoutFailureInfo().GetTimeoutType())
				return
			}
		}
	}
	require.FailNow(t, "timed-out event not recorded")
}

func TestTestpilotNexusCallerCaseMissingRemoteEndpoint(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	caseSource := loadTestpilotCase(t, nexusCallerQueries[1].fixture())
	// The endpoint the Case binds is deliberately not created, so the schedule command is rejected
	// at the workflow task, the scheduled event the controller polls for never appears, and the
	// Run closes incomplete and inconclusive at that poll.
	live := bindCase(t, env, caseSource, CaseBinding{
		Identity: "async-completion-profile", Namespace: "umpire-async-completion-missing",
		TaskQueue: "umpire-async-completion-queue-missing", NexusEndpoint: "umpire-async-completion-endpoint-missing",
		CreateEndpoint: false,
	})

	run, verdict, err := live.prepared.Run(env.Context(), live.driver)
	require.NoError(t, err)
	require.Equal(t, testpilotpb.RUN_DISPOSITION_INCOMPLETE, run.GetDisposition())
	require.Equal(t, testpilotpb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
	require.Equal(t, testpilotpb.VERDICT_STATUS_INCONCLUSIVE, verdict.GetStatus())
	requireRunHasOutcome(t, run, "await-scheduled", testpilotpb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT)
	_, err = live.client.DescribeWorkflowExecution(env.Context(), run.GetRunId(), "")
	require.NoError(t, err)
}

// TestTestpilotNexusCallerCaseRunsFromItsFixtureNameAlone is what a new live test costs: name the
// fixture, assert the Verdict. Everything else -- the namespace, the queue, the Nexus endpoint, the
// derived Profile, the Driver -- follows from the Case's own bytes.
func TestTestpilotNexusCallerCaseRunsFromItsFixtureNameAlone(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	query := nexusCallerQueries[1]

	run, verdict := runCase(t, env, query.fixture())

	requireNexusCallerVerdict(t, query, run, verdict, "umpire-"+query.fixture()+"-endpoint")
}

func loadTestpilotCase(t testing.TB, name string) *testpilotpb.Case {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testcore", "testpilot", "testdata", name+"-case.json"))
	require.NoError(t, err)
	decoded, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	return decoded
}
